package flow

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net/http"
	"path"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/Kxiandaoyan/Memoh-v2/internal/conversation"
	"github.com/Kxiandaoyan/Memoh-v2/internal/db"
	"github.com/Kxiandaoyan/Memoh-v2/internal/db/sqlc"
	"github.com/Kxiandaoyan/Memoh-v2/internal/heartbeat"
	"github.com/Kxiandaoyan/Memoh-v2/internal/memory"
	messagepkg "github.com/Kxiandaoyan/Memoh-v2/internal/message"
	"github.com/Kxiandaoyan/Memoh-v2/internal/models"
	"github.com/Kxiandaoyan/Memoh-v2/internal/processlog"
	"github.com/Kxiandaoyan/Memoh-v2/internal/schedule"
	"github.com/Kxiandaoyan/Memoh-v2/internal/settings"
)

const (
	defaultMaxContextMinutes   = 24 * 60
	memoryContextLimitPerScope = 4
	memoryContextMaxItems      = 8
	memoryContextItemMaxChars  = 220
	memoryMinScoreThreshold    = 0.1
	memoryDecayHalfLifeDays    = 30.0
	sharedMemoryNamespace      = "bot"

	solutionsContextLimit          = 2
	solutionsMinScoreThreshold     = 0.3
	solutionsNamespace             = "solutions"
	solutionsScopeID               = "global"
)

func applyTemporalDecay(items []memoryContextItem) {
	now := time.Now()
	ln2 := math.Ln2
	for i := range items {
		updatedStr := items[i].Item.UpdatedAt
		if updatedStr == "" {
			updatedStr = items[i].Item.CreatedAt
		}
		if updatedStr == "" {
			continue
		}
		t, err := time.Parse(time.RFC3339, updatedStr)
		if err != nil {
			t, err = time.Parse("2006-01-02T15:04:05", updatedStr)
			if err != nil {
				continue
			}
		}
		ageDays := now.Sub(t).Hours() / 24.0
		if ageDays < 0 {
			ageDays = 0
		}
		decay := math.Exp(-ln2 / memoryDecayHalfLifeDays * ageDays)
		items[i].Item.Score *= decay
	}
}

func textJaccardSimilarity(a, b string) float64 {
	setA := map[string]struct{}{}
	setB := map[string]struct{}{}
	for _, w := range strings.Fields(strings.ToLower(a)) {
		setA[w] = struct{}{}
	}
	for _, w := range strings.Fields(strings.ToLower(b)) {
		setB[w] = struct{}{}
	}
	return jaccardSimilarity(setA, setB)
}

func applyMMR(items []memoryContextItem, lambda float64) []memoryContextItem {
	if len(items) <= 1 {
		return items
	}

	selected := []memoryContextItem{items[0]}
	remaining := make([]memoryContextItem, len(items)-1)
	copy(remaining, items[1:])

	for len(remaining) > 0 && len(selected) < len(items) {
		bestIdx := -1
		bestMMR := -math.MaxFloat64

		for i, cand := range remaining {
			maxSim := 0.0
			for _, sel := range selected {
				sim := textJaccardSimilarity(cand.Item.Memory, sel.Item.Memory)
				if sim > maxSim {
					maxSim = sim
				}
			}
			mmr := lambda*cand.Item.Score - (1-lambda)*maxSim
			if mmr > bestMMR {
				bestMMR = mmr
				bestIdx = i
			}
		}

		if bestIdx < 0 {
			break
		}
		selected = append(selected, remaining[bestIdx])
		remaining = append(remaining[:bestIdx], remaining[bestIdx+1:]...)
	}

	return selected
}

// generateTraceID creates a new UUID trace ID for request tracking
func generateTraceID() string {
	return uuid.New().String()
}

// SkillEntry represents a skill loaded from the container.
type SkillEntry struct {
	Name        string
	Description string
	Content     string
	Metadata    map[string]any
}

// SkillLoader loads skills for a given bot from its container.
type SkillLoader interface {
	LoadSkills(ctx context.Context, botID string) ([]SkillEntry, error)
}

// ConversationSettingsReader defines settings lookup behavior needed by flow resolution.
type ConversationSettingsReader interface {
	GetSettings(ctx context.Context, conversationID string) (conversation.Settings, error)
}

// OVSessionExtractor commits a conversation to OpenViking for memory extraction.
type OVSessionExtractor interface {
	ExtractSession(ctx context.Context, botID, chatID string, messages []conversation.ModelMessage) (output string, err error)
}

// OVContextLoader loads lightweight OpenViking context for conversation injection.
type OVContextLoader interface {
	LoadContext(ctx context.Context, botID, query string) string
}

// TriggerMessageSender is an optional fallback used by executeTrigger to
// deliver a text message when the LLM did not call the send MCP tool.
type TriggerMessageSender interface {
	SendText(ctx context.Context, botID, platform, target, text string) error
}

// Resolver orchestrates chat with the agent gateway.
type Resolver struct {
	modelsService    *models.Service
	queries          *sqlc.Queries
	memoryService    *memory.Service
	conversationSvc ConversationSettingsReader
	messageService   messagepkg.Service
	settingsService  *settings.Service
	processLogService *processlog.Service
	skillLoader      SkillLoader
	ovSessionExtractor OVSessionExtractor
	ovContextLoader    OVContextLoader
	triggerSender    TriggerMessageSender
	gatewayBaseURL   string
	timezone         string
	timeout          time.Duration
	logger           *slog.Logger
	httpClient       *http.Client
	streamingClient  *http.Client
}

// NewResolver creates a Resolver that communicates with the agent gateway.
func NewResolver(
	log *slog.Logger,
	modelsService *models.Service,
	queries *sqlc.Queries,
	memoryService *memory.Service,
	conversationSvc ConversationSettingsReader,
	messageService messagepkg.Service,
	settingsService *settings.Service,
	processLogService *processlog.Service,
	gatewayBaseURL string,
	timeout time.Duration,
) *Resolver {
	if strings.TrimSpace(gatewayBaseURL) == "" {
		gatewayBaseURL = "http://127.0.0.1:8081"
	}
	gatewayBaseURL = strings.TrimRight(gatewayBaseURL, "/")
	if timeout <= 0 {
		timeout = 60 * time.Second
	}
	return &Resolver{
		modelsService:      modelsService,
		queries:            queries,
		memoryService:      memoryService,
		conversationSvc:   conversationSvc,
		messageService:     messageService,
		settingsService:    settingsService,
		processLogService:   processLogService,
		gatewayBaseURL:    gatewayBaseURL,
		timeout:           timeout,
		logger:            log.With(slog.String("service", "conversation_resolver")),
		httpClient:      &http.Client{Timeout: timeout},
		streamingClient: &http.Client{},
	}
}

// SetSkillLoader sets the skill loader used to populate usable skills in gateway requests.
func (r *Resolver) SetSkillLoader(sl SkillLoader) {
	r.skillLoader = sl
}

// SetTimezone sets the IANA timezone name used in gateway requests.
func (r *Resolver) SetTimezone(tz string) {
	r.timezone = tz
}

// SetOVSessionExtractor sets the optional OpenViking session extractor for
// post-conversation memory extraction.
func (r *Resolver) SetOVSessionExtractor(e OVSessionExtractor) {
	r.ovSessionExtractor = e
}

// SetOVContextLoader sets the optional OpenViking context loader for
// injecting relevant knowledge base context into conversations.
func (r *Resolver) SetOVContextLoader(l OVContextLoader) {
	r.ovContextLoader = l
}

// SetTriggerSender sets the fallback channel sender used when a schedule/heartbeat
// trigger's LLM response contains text but no explicit send tool call.
func (r *Resolver) SetTriggerSender(s TriggerMessageSender) {
	r.triggerSender = s
}

// --- Process Logging Helpers ---

// logProcessStep records a process log entry for debugging and monitoring
func (r *Resolver) logProcessStep(
	ctx context.Context,
	botID, chatID, traceID, userID, channel string,
	step processlog.ProcessLogStep,
	level processlog.ProcessLogLevel,
	message string,
	data map[string]any,
	durationMs int,
) {
	if r.processLogService == nil {
		return
	}
	// Run in background to not block main flow
	go func() {
		_, _ = r.processLogService.Create(ctx, processlog.CreateProcessLogParams{
			BotID:      botID,
			ChatID:     chatID,
			TraceID:    traceID,
			UserID:     userID,
			Channel:    channel,
			Step:       step,
			Level:      level,
			Message:    message,
			Data:       data,
			DurationMs: durationMs,
		})
	}()
}

// --- gateway payload ---

type gatewayModelConfig struct {
	ModelID    string   `json:"modelId"`
	ClientType string   `json:"clientType"`
	Input      []string `json:"input"`
	APIKey     string   `json:"apiKey"`
	BaseURL    string   `json:"baseUrl"`
	Reasoning  bool     `json:"reasoning,omitempty"`
	MaxTokens  int      `json:"maxTokens,omitempty"`
}

type gatewayIdentity struct {
	BotID             string `json:"botId"`
	ContainerID       string `json:"containerId"`
	ChannelIdentityID string `json:"channelIdentityId"`
	DisplayName       string `json:"displayName"`
	CurrentPlatform   string `json:"currentPlatform"`
	ConversationType  string `json:"conversationType,omitempty"`
	ReplyTarget       string `json:"replyTarget"`
	SessionToken      string `json:"sessionToken,omitempty"`
}

type gatewaySkill struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Content     string         `json:"content"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

type gatewayRequest struct {
	Model              gatewayModelConfig          `json:"model"`
	BackgroundModel    *gatewayModelConfig         `json:"backgroundModel,omitempty"`
	ActiveContextTime  int                         `json:"activeContextTime"`
	Language           string                      `json:"language,omitempty"`
	Timezone           string                      `json:"timezone"`
	Channels           []string                    `json:"channels"`
	CurrentChannel     string                      `json:"currentChannel"`
	AllowedActions     []string                    `json:"allowedActions,omitempty"`
	Messages           []conversation.ModelMessage `json:"messages"`
	Skills             []string                    `json:"skills"`
	UsableSkills       []gatewaySkill              `json:"usableSkills"`
	Query              string                      `json:"query,omitempty"`
	Identity           gatewayIdentity             `json:"identity"`
	Attachments        []any                       `json:"attachments"`
	BotIdentity        string                      `json:"botIdentity,omitempty"`
	BotSoul            string                      `json:"botSoul,omitempty"`
	BotTask            string                      `json:"botTask,omitempty"`
	AllowSelfEvolution bool                        `json:"allowSelfEvolution"`
}

type gatewayResponse struct {
	Messages    []conversation.ModelMessage `json:"messages"`
	Skills      []string                    `json:"skills"`
	Usage       *gatewayUsage               `json:"usage,omitempty"`
	Attachments []gatewayFileAttachment     `json:"attachments,omitempty"`
}

type gatewayFileAttachment struct {
	Type string `json:"type"`
	Path string `json:"path"`
}

type gatewayUsage struct {
	PromptTokens     int `json:"promptTokens"`
	CompletionTokens int `json:"completionTokens"`
	TotalTokens      int `json:"totalTokens"`
}

// gatewaySchedule matches the agent gateway ScheduleModel for /chat/trigger-schedule.
type gatewaySchedule struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Pattern     string `json:"pattern,omitempty"`
	MaxCalls    *int   `json:"maxCalls,omitempty"`
	Command     string `json:"command"`
	TriggerType string `json:"triggerType,omitempty"`
}

// triggerScheduleRequest is the payload for POST /chat/trigger-schedule.
type triggerScheduleRequest struct {
	gatewayRequest
	Schedule gatewaySchedule `json:"schedule"`
}

// --- resolved context (shared by Chat / StreamChat / TriggerSchedule) ---

type resolvedContext struct {
	payload  gatewayRequest
	model    models.GetResponse
	provider sqlc.LlmProvider
	traceID  string
}

func (r *Resolver) resolve(ctx context.Context, req conversation.ChatRequest) (resolvedContext, error) {
	// Generate trace ID for this request
	traceID := generateTraceID()

	// Log user message received
	r.logProcessStep(ctx, req.BotID, req.ChatID, traceID, req.UserID, req.CurrentChannel,
		processlog.StepUserMessageReceived, processlog.LevelInfo, "User message received",
		map[string]any{
			"query":             req.Query,
			"channels":          req.Channels,
			"platform":          req.CurrentChannel,
			"identity_id":       req.SourceChannelIdentityID,
			"conversation_type": req.ConversationType,
			"timestamp":         time.Now().Unix(),
		}, 0)

	if strings.TrimSpace(req.Query) == "" {
		return resolvedContext{}, fmt.Errorf("query is required")
	}
	if strings.TrimSpace(req.BotID) == "" {
		return resolvedContext{}, fmt.Errorf("bot id is required")
	}
	if strings.TrimSpace(req.ChatID) == "" {
		return resolvedContext{}, fmt.Errorf("chat id is required")
	}

	skipHistory := req.MaxContextLoadTime < 0

	botSettings, err := r.loadBotSettings(ctx, req.BotID)
	if err != nil {
		return resolvedContext{}, err
	}

	// Check chat-level model override.
	var chatSettings conversation.Settings
	if r.conversationSvc != nil {
		chatSettings, err = r.conversationSvc.GetSettings(ctx, req.ChatID)
		if err != nil {
			return resolvedContext{}, err
		}
	}

	chatModel, provider, err := r.selectChatModel(ctx, req, botSettings, chatSettings)
	if err != nil {
		return resolvedContext{}, err
	}
	clientType, err := normalizeClientType(provider.ClientType)
	if err != nil {
		return resolvedContext{}, err
	}
	maxCtx := coalescePositiveInt(req.MaxContextLoadTime, botSettings.MaxContextLoadTime, defaultMaxContextMinutes)

	// Determine history limit based on conversation type (DM vs Channel/Group)
	historyLimit := settings.DefaultChannelHistoryLimit
	if isDirectConversationType(req.ConversationType) {
		historyLimit = settings.DefaultDMHistoryLimit
	}
	// Allow bot settings to override defaults
	if botSettings.DMHistoryLimit > 0 && isDirectConversationType(req.ConversationType) {
		historyLimit = botSettings.DMHistoryLimit
	}
	if botSettings.ChannelHistoryLimit > 0 && !isDirectConversationType(req.ConversationType) {
		historyLimit = botSettings.ChannelHistoryLimit
	}
	// Allow explicit override (used by evolution heartbeat)
	if req.HistoryLimitOverride > 0 {
		historyLimit = req.HistoryLimitOverride
	}

	var messages []conversation.ModelMessage
	if !skipHistory && r.conversationSvc != nil {
		msgs, err := r.loadMessages(ctx, req.ChatID, maxCtx)
		if err != nil {
			return resolvedContext{}, err
		}
		messages = limitHistoryTurns(msgs, historyLimit)

		r.logProcessStep(ctx, req.BotID, req.ChatID, traceID, req.UserID, req.CurrentChannel,
			processlog.StepHistoryLoaded, processlog.LevelInfo, "History loaded",
			map[string]any{
				"message_count": len(msgs),
				"after_limit":   len(messages),
				"history_limit": historyLimit,
			}, 0)

		proactiveThreshold := int(float64(historyLimit) * 0.8)
		if proactiveThreshold < 4 {
			proactiveThreshold = 4
		}
		if len(messages) >= proactiveThreshold && len(messages) > 6 {
			halfIdx := len(messages) / 2
			olderHalf := messages[:halfIdx]
			r.asyncSummarize(req.BotID, req.ChatID, olderHalf, chatModel, provider, req.Token)
			r.logProcessStep(ctx, req.BotID, req.ChatID, traceID, req.UserID, req.CurrentChannel,
				processlog.StepSummaryRequested, processlog.LevelInfo, "Proactive summarization triggered",
				map[string]any{
					"message_count":   len(messages),
					"history_limit":   historyLimit,
					"threshold":       proactiveThreshold,
					"messages_to_sum": halfIdx,
				}, 0)
		}
	}

	// Inject existing conversation summary as the first message.
	if summary := r.loadSummary(ctx, req.BotID, req.ChatID); summary != "" {
		summaryText := "[Previous conversation summary]\n\n" + summary
		encodedSummary, _ := json.Marshal(summaryText)
		summaryMsg := conversation.ModelMessage{
			Role:    "user",
			Content: json.RawMessage(encodedSummary),
		}
		messages = append([]conversation.ModelMessage{summaryMsg}, messages...)
		r.logProcessStep(ctx, req.BotID, req.ChatID, traceID, req.UserID, req.CurrentChannel,
			processlog.StepSummaryLoaded, processlog.LevelInfo, "Conversation summary loaded",
			map[string]any{
				"summary_length": len(summary),
			}, 0)
	}

	memSearchStart := time.Now()
	if memoryMsg := r.loadMemoryContextMessage(ctx, req, chatModel.ContextWindow, traceID); memoryMsg != nil {
		memDur := int(time.Since(memSearchStart).Milliseconds())
		contentStr := string(memoryMsg.Content)
		r.logProcessStep(ctx, req.BotID, req.ChatID, traceID, req.UserID, req.CurrentChannel,
			processlog.StepMemorySearched, processlog.LevelInfo, "Memory searched",
			map[string]any{
				"query":       truncate(req.Query, 200),
				"has_results": true,
				"duration":    memDur,
			}, memDur)
		r.logProcessStep(ctx, req.BotID, req.ChatID, traceID, req.UserID, req.CurrentChannel,
			processlog.StepMemoryLoaded, processlog.LevelInfo, "Memory loaded",
			map[string]any{
				"memory_content": truncate(contentStr, 1000),
			}, 0)
		messages = append(messages, *memoryMsg)
	} else {
		memDur := int(time.Since(memSearchStart).Milliseconds())
		r.logProcessStep(ctx, req.BotID, req.ChatID, traceID, req.UserID, req.CurrentChannel,
			processlog.StepMemorySearched, processlog.LevelInfo, "Memory searched (no results)",
			map[string]any{
				"query":       truncate(req.Query, 200),
				"has_results": false,
				"duration":    memDur,
			}, memDur)
	}
	if r.ovContextLoader != nil {
		ovStart := time.Now()
		if ovText := r.ovContextLoader.LoadContext(ctx, req.BotID, req.Query); ovText != "" {
			ovDur := int(time.Since(ovStart).Milliseconds())
			messages = append(messages, conversation.ModelMessage{
				Role:    "system",
				Content: conversation.NewTextContent(ovText),
			})
			r.logProcessStep(ctx, req.BotID, req.ChatID, traceID, req.UserID, req.CurrentChannel,
				processlog.StepOpenVikingContext, processlog.LevelInfo, "OpenViking context loaded",
				map[string]any{
					"context_length": len(ovText),
					"duration":       ovDur,
				}, ovDur)
		} else {
			ovDur := int(time.Since(ovStart).Milliseconds())
			r.logProcessStep(ctx, req.BotID, req.ChatID, traceID, req.UserID, req.CurrentChannel,
				processlog.StepOpenVikingContext, processlog.LevelInfo, "OpenViking context (no results)",
				map[string]any{
					"duration": ovDur,
				}, ovDur)
		}
	}

	messages = append(messages, req.Messages...)
	messages = sanitizeMessages(messages)

	allMessages := make([]conversation.ModelMessage, len(messages))
	copy(allMessages, messages)
	messages, trimDiags, budgetDiag := pruneMessagesByTokenBudget(messages, chatModel.ContextWindow)

	r.logProcessStep(ctx, req.BotID, req.ChatID, traceID, req.UserID, req.CurrentChannel,
		processlog.StepTokenBudgetCalculated, processlog.LevelInfo, "Token budget calculated",
		map[string]any{
			"context_window":      chatModel.ContextWindow,
			"system_tokens":       budgetDiag.SystemTokens,
			"gateway_estimate":    budgetDiag.GatewayEstimate,
			"total_system_tokens": budgetDiag.TotalSystemTokens,
			"budget":              budgetDiag.Budget,
			"estimated_before":    budgetDiag.EstimatedTotalBefore,
			"estimated_after":     budgetDiag.EstimatedTotalAfter,
			"protected_tail":      budgetDiag.ProtectedTail,
			"pruned":              budgetDiag.Pruned,
		}, 0)

	if len(trimDiags) > 0 {
		trimData := make([]map[string]any, 0, len(trimDiags))
		for _, td := range trimDiags {
			trimData = append(trimData, map[string]any{
				"tool_call_id":  td.ToolCallID,
				"original_chars": td.OriginalChars,
				"trimmed_chars":  td.TrimmedChars,
			})
		}
		r.logProcessStep(ctx, req.BotID, req.ChatID, traceID, req.UserID, req.CurrentChannel,
			processlog.StepToolResultTrimmed, processlog.LevelInfo,
			fmt.Sprintf("%d tool result(s) soft-trimmed", len(trimDiags)),
			map[string]any{"trimmed_tools": trimData}, 0)
	}

	if len(messages) < len(allMessages) {
		droppedCount := len(allMessages) - len(messages)
		dropped := allMessages[:droppedCount]
		r.logProcessStep(ctx, req.BotID, req.ChatID, traceID, req.UserID, req.CurrentChannel,
			processlog.StepTokenTrimmed, processlog.LevelInfo, "Messages pruned by token budget",
			map[string]any{
				"before":         len(allMessages),
				"after":          len(messages),
				"dropped":        droppedCount,
				"context_window": chatModel.ContextWindow,
			}, 0)

		var syncSummary string
		var syncErr error
		for attempt := 1; attempt <= 3; attempt++ {
			syncSummary, _, syncErr = r.postSummarize(ctx, gatewayModelConfig{
				ModelID:    chatModel.ModelID,
				ClientType: mustNormalizeClientType(provider.ClientType),
				Input:      chatModel.Input,
				APIKey:     provider.ApiKey,
				BaseURL:    provider.BaseUrl,
				Reasoning:  chatModel.Reasoning,
				MaxTokens:  chatModel.MaxTokens,
			}, dropped, req.Token)
			if syncErr == nil && strings.TrimSpace(syncSummary) != "" {
				break
			}
			if attempt < 3 {
				r.logger.Warn("sync summarize retry",
					slog.Int("attempt", attempt),
					slog.Any("error", syncErr),
				)
			}
		}

		if syncErr == nil && strings.TrimSpace(syncSummary) != "" {
			r.logProcessStep(ctx, req.BotID, req.ChatID, traceID, req.UserID, req.CurrentChannel,
				processlog.StepSummaryRequested, processlog.LevelInfo, "Sync summarization completed",
				map[string]any{
					"dropped_messages": droppedCount,
					"summary_length":  len(syncSummary),
				}, 0)
			if pgBotID, parseErr := db.ParseUUID(req.BotID); parseErr == nil {
				if _, upsertErr := r.queries.UpsertConversationSummary(ctx, sqlc.UpsertConversationSummaryParams{
					BotID:        pgBotID,
					ChatID:       req.ChatID,
					Summary:      syncSummary,
					MessageCount: int32(droppedCount),
				}); upsertErr != nil {
					r.logger.Warn("sync upsert summary failed", slog.Any("error", upsertErr))
				}
			}
		} else {
			r.logProcessStep(ctx, req.BotID, req.ChatID, traceID, req.UserID, req.CurrentChannel,
				processlog.StepSummaryRequested, processlog.LevelInfo, "Sync summarize failed, falling back to async",
				map[string]any{
					"dropped_messages": droppedCount,
					"error":           fmt.Sprintf("%v", syncErr),
				}, 0)
			r.asyncSummarize(req.BotID, req.ChatID, dropped, chatModel, provider, req.Token)
		}
	}

	skills := dedup(req.Skills)
	containerID := r.resolveContainerID(ctx, req.BotID, req.ContainerID)

	skillStart := time.Now()
	var usableSkills []gatewaySkill
	if r.skillLoader != nil {
		entries, err := r.skillLoader.LoadSkills(ctx, req.BotID)
		if err != nil {
			r.logger.Warn("failed to load usable skills", slog.String("bot_id", req.BotID), slog.Any("error", err))
		} else {
			usableSkills = make([]gatewaySkill, 0, len(entries))
			for _, e := range entries {
				desc := e.Description
				if strings.TrimSpace(desc) == "" {
					desc = e.Name
				}
				usableSkills = append(usableSkills, gatewaySkill{
					Name:        e.Name,
					Description: desc,
					Content:     e.Content,
					Metadata:    e.Metadata,
				})
			}
		}
	}
	if usableSkills == nil {
		usableSkills = []gatewaySkill{}
	}

	preFilterCount := len(usableSkills)
	skillDur := int(time.Since(skillStart).Milliseconds())
	r.logProcessStep(ctx, req.BotID, req.ChatID, traceID, req.UserID, req.CurrentChannel,
		processlog.StepSkillsLoaded, processlog.LevelInfo, "Skills loaded",
		map[string]any{
			"total_skills":    preFilterCount,
			"after_filter":    len(usableSkills),
			"filtered_out":    preFilterCount - len(usableSkills),
			"duration":        skillDur,
		}, skillDur)
	if preFilterCount > len(usableSkills) {
		skillNames := make([]string, 0, len(usableSkills))
		for _, sk := range usableSkills {
			skillNames = append(skillNames, sk.Name)
		}
		r.logProcessStep(ctx, req.BotID, req.ChatID, traceID, req.UserID, req.CurrentChannel,
			processlog.StepSkillsFiltered, processlog.LevelInfo,
			fmt.Sprintf("Skills filtered: %d → %d", preFilterCount, len(usableSkills)),
			map[string]any{
				"kept_skills": skillNames,
			}, 0)
	}

	// Load bot persona/prompt configuration.
	var botIdentity, botSoul, botTask string
	allowSelfEvolution := true
	if r.queries != nil {
		botUUID, parseErr := db.ParseUUID(req.BotID)
		if parseErr == nil {
			if promptRow, promptErr := r.queries.GetBotPrompts(ctx, botUUID); promptErr == nil {
				botIdentity = promptRow.Identity.String
				botSoul = promptRow.Soul.String
				botTask = promptRow.Task.String
				allowSelfEvolution = promptRow.AllowSelfEvolution
			}
		}
	}

	tz := r.timezone
	if tz == "" {
		tz = "UTC"
	}
	r.logger.Info("resolve: timezone for gateway",
		slog.String("bot_id", req.BotID),
		slog.String("timezone", tz),
	)

	payload := gatewayRequest{
		Model: gatewayModelConfig{
			ModelID:    chatModel.ModelID,
			ClientType: clientType,
			Input:      chatModel.Input,
			APIKey:     provider.ApiKey,
			BaseURL:    provider.BaseUrl,
			Reasoning:  chatModel.Reasoning,
			MaxTokens:  chatModel.MaxTokens,
		},
		ActiveContextTime:  maxCtx,
		Language:            botSettings.Language,
		Timezone:           tz,
		Channels:           nonNilStrings(req.Channels),
		CurrentChannel:     req.CurrentChannel,
		AllowedActions:     req.AllowedActions,
		Messages:           nonNilModelMessages(messages),
		Skills:             nonNilStrings(skills),
		UsableSkills:       usableSkills,
		Query:              req.Query,
		Identity: gatewayIdentity{
			BotID:             req.BotID,
			ContainerID:       containerID,
			ChannelIdentityID: strings.TrimSpace(req.SourceChannelIdentityID),
			DisplayName:       r.resolveDisplayName(ctx, req),
			CurrentPlatform:   req.CurrentChannel,
			ConversationType:  strings.TrimSpace(req.ConversationType),
			ReplyTarget:       strings.TrimSpace(req.ReplyTarget),
			SessionToken:      req.ChatToken,
		},
		Attachments:        []any{},
		BotIdentity:        botIdentity,
		BotSoul:            botSoul,
		BotTask:            botTask,
		AllowSelfEvolution: allowSelfEvolution,
	}

	// Log prompt built
	systemPromptLen := len(botIdentity) + len(botSoul) + len(botTask)
	r.logProcessStep(ctx, req.BotID, req.ChatID, traceID, req.UserID, req.CurrentChannel,
		processlog.StepPromptBuilt, processlog.LevelInfo, "Prompt built",
		map[string]any{
			"model":                chatModel.ModelID,
			"provider":             provider.ClientType,
			"message_count":        len(messages),
			"has_identity":         botIdentity != "",
			"has_soul":             botSoul != "",
			"has_task":             botTask != "",
			"skills_count":         len(usableSkills),
			"system_prompt_length": systemPromptLen,
			"context_window":       chatModel.ContextWindow,
			"timezone":             tz,
			"language":             botSettings.Language,
		}, 0)

	// Resolve the background model for subagent dispatch (if configured and different from primary).
	if bgID := strings.TrimSpace(botSettings.BackgroundModelID); bgID != "" && bgID != chatModel.ModelID {
		if bgModel, bgProv, bgErr := r.fetchChatModel(ctx, bgID); bgErr == nil {
			bgClientType, ctErr := normalizeClientType(bgProv.ClientType)
			if ctErr == nil {
				payload.BackgroundModel = &gatewayModelConfig{
					ModelID:    bgModel.ModelID,
					ClientType: bgClientType,
					Input:      bgModel.Input,
					APIKey:     bgProv.ApiKey,
					BaseURL:    bgProv.BaseUrl,
					Reasoning:  bgModel.Reasoning,
					MaxTokens:  bgModel.MaxTokens,
				}
			}
		}
	}

	return resolvedContext{payload: payload, model: chatModel, provider: provider, traceID: traceID}, nil
}

// --- Chat ---

// Chat sends a synchronous chat request to the agent gateway and stores the result.
func (r *Resolver) Chat(ctx context.Context, req conversation.ChatRequest) (conversation.ChatResponse, error) {
	rc, err := r.resolve(ctx, req)
	if err != nil {
		return conversation.ChatResponse{}, err
	}

	// Log LLM request being sent
	r.logProcessStep(ctx, req.BotID, req.ChatID, rc.traceID, req.UserID, req.CurrentChannel,
		processlog.StepLLMRequestSent, processlog.LevelInfo, "LLM request sent",
		map[string]any{
			"model":         rc.model.ModelID,
			"provider":      rc.provider.ClientType,
			"message_count": len(rc.payload.Messages),
		}, 0)

	syncStart := time.Now()
	var resp gatewayResponse
	err = withGatewayRetry(ctx, func() error {
		var callErr error
		resp, callErr = r.postChat(ctx, rc.payload, req.Token)
		return callErr
	})
	if err != nil {
		// Log LLM error
		r.logProcessStep(ctx, req.BotID, req.ChatID, rc.traceID, req.UserID, req.CurrentChannel,
			processlog.StepLLMResponseReceived, processlog.LevelError, "LLM request failed",
			map[string]any{"error": err.Error()}, 0)

		// Attempt failover if primary model fails
		if fbCtx, fbErr := r.tryFallback(ctx, rc); fbErr == nil {
			r.logProcessStep(ctx, req.BotID, req.ChatID, rc.traceID, req.UserID, req.CurrentChannel,
				processlog.StepModelFallback, processlog.LevelWarn, "Primary model failed, switching to fallback",
				map[string]any{
					"primary_model":    rc.model.ModelID,
					"primary_provider": rc.provider.ClientType,
					"primary_error":    err.Error(),
					"fallback_model":   fbCtx.model.ModelID,
					"fallback_provider": fbCtx.provider.ClientType,
				}, 0)
			err = withGatewayRetry(ctx, func() error {
				var callErr error
				resp, callErr = r.postChat(ctx, fbCtx.payload, req.Token)
				return callErr
			})
			if err == nil {
				rc = fbCtx
			}
		}
		if err != nil {
			return conversation.ChatResponse{}, err
		}
	}

	// Context overflow recovery: if the LLM rejected the prompt as too long,
	// re-prune more aggressively (0.4 budget ratio) and retry once.
	if isContextOverflowError(err) {
		r.logger.Warn("Chat: context overflow detected, retrying with reduced context",
			slog.String("bot_id", req.BotID),
			slog.String("original_error", err.Error()))
		rc.payload.Messages = repruneWithLowerBudget(rc.payload.Messages, rc.model.ContextWindow, 0.4)
		resp, err = r.postChat(ctx, rc.payload, req.Token)
		if err != nil {
			r.logProcessStep(ctx, req.BotID, req.ChatID, rc.traceID, req.UserID, req.CurrentChannel,
				processlog.StepLLMResponseReceived, processlog.LevelError, "LLM request failed after overflow recovery",
				map[string]any{"error": err.Error()}, 0)
			return conversation.ChatResponse{}, err
		}
	}

	// Log LLM response received
	syncDur := int(time.Since(syncStart).Milliseconds())
	responsePreview := extractAssistantPreview(resp.Messages, 300)
	r.logProcessStep(ctx, req.BotID, req.ChatID, rc.traceID, req.UserID, req.CurrentChannel,
		processlog.StepLLMResponseReceived, processlog.LevelInfo, "LLM response received",
		map[string]any{
			"model":            rc.model.ModelID,
			"provider":         rc.provider.ClientType,
			"response_length":  len(resp.Messages),
			"response_preview": responsePreview,
			"usage":            resp.Usage,
		}, syncDur)

	// Extract file attachments from sync gateway response.
	req.FileAttachments = extractGatewayAttachments(resp.Attachments)
	if len(req.FileAttachments) > 0 {
		r.logger.Debug("Chat: collected file attachments from sync response",
			slog.Int("count", len(req.FileAttachments)),
			slog.String("bot_id", req.BotID),
			slog.String("chat_id", req.ChatID),
		)
	}

	if err := r.storeRoundWithTrace(ctx, req, rc.traceID, resp.Messages, resp.Usage); err != nil {
		return conversation.ChatResponse{}, err
	}
	r.recordTokenUsage(ctx, req.BotID, resp.Usage, rc.model.ModelID, "chat")

	// Log response sent
	r.logProcessStep(ctx, req.BotID, req.ChatID, rc.traceID, req.UserID, req.CurrentChannel,
		processlog.StepResponseSent, processlog.LevelInfo, "Response sent",
		map[string]any{
			"message_count":    len(resp.Messages),
			"response_preview": responsePreview,
		}, 0)

	return conversation.ChatResponse{
		Messages: resp.Messages,
		Skills:   resp.Skills,
		Model:    rc.model.ModelID,
		Provider: rc.provider.ClientType,
		Usage:    toTokenUsage(resp.Usage),
	}, nil
}

// --- TriggerSchedule / TriggerHeartbeat ---

// triggerParams holds the unified parameters for executeTrigger.
type triggerParams struct {
	botID                string
	query                string
	ownerUserID          string
	displayName          string          // "Scheduler" or "Heartbeat"
	schedule             gatewaySchedule // gateway schedule metadata
	usageType            string          // "schedule" or "heartbeat"
	evolutionLogID       string          // non-empty only for evolution heartbeats
	historyLimitOverride int             // when > 0, override default history turn limit
	platform             string          // channel platform for message delivery
	replyTarget          string          // chat/group target for message delivery
}

// executeTrigger is the shared execution path for both schedule and heartbeat triggers.
// It resolves the conversation context, posts to the agent gateway, records token usage,
// and stores the conversation round.
func (r *Resolver) executeTrigger(ctx context.Context, p triggerParams, token string) error {
	if strings.TrimSpace(p.schedule.ID) == "" {
		return fmt.Errorf("trigger pre-validation: schedule id is required")
	}
	if strings.TrimSpace(p.schedule.Command) == "" {
		return fmt.Errorf("trigger pre-validation: schedule command is required")
	}

	r.logger.Info("executeTrigger: channel routing",
		slog.String("bot_id", p.botID),
		slog.String("usage_type", p.usageType),
		slog.String("platform", p.platform),
		slog.String("reply_target", p.replyTarget),
		slog.String("query", truncate(p.query, 200)),
	)
	taskType := "schedule"
	if p.schedule.TriggerType == "heartbeat" {
		taskType = "heartbeat"
	}
	req := conversation.ChatRequest{
		BotID:                p.botID,
		ChatID:               p.botID,
		Query:                p.query,
		UserID:               p.ownerUserID,
		Token:                token,
		HistoryLimitOverride: p.historyLimitOverride,
		CurrentChannel:       p.platform,
		ReplyTarget:          p.replyTarget,
		TaskType:             taskType,
	}
	if p.platform != "" {
		req.Channels = []string{p.platform}
	}
	rc, err := r.resolve(ctx, req)
	if err != nil {
		r.logger.Warn("executeTrigger: resolve failed", slog.String("bot_id", p.botID), slog.Any("error", err))
		r.completeEvolutionLogOnError(ctx, p.evolutionLogID, err)
		return err
	}

	gwPayload := rc.payload
	gwPayload.Identity.ChannelIdentityID = strings.TrimSpace(p.ownerUserID)
	gwPayload.Identity.DisplayName = p.displayName
	if p.platform != "" {
		gwPayload.Identity.CurrentPlatform = p.platform
		gwPayload.CurrentChannel = p.platform
		if len(gwPayload.Channels) == 0 {
			gwPayload.Channels = []string{p.platform}
		}
	}
	if p.replyTarget != "" {
		gwPayload.Identity.ReplyTarget = p.replyTarget
	}

	triggerReq := triggerScheduleRequest{
		gatewayRequest: gwPayload,
		Schedule:       p.schedule,
	}

	r.logProcessStep(ctx, p.botID, p.botID, rc.traceID, p.ownerUserID, p.platform,
		processlog.StepLLMRequestSent, processlog.LevelInfo, fmt.Sprintf("Trigger request sent (%s)", p.usageType),
		map[string]any{
			"model":         rc.model.ModelID,
			"provider":      rc.provider.ClientType,
			"message_count": len(gwPayload.Messages),
			"schedule_id":   p.schedule.ID,
			"platform":      p.platform,
			"reply_target":  p.replyTarget,
		}, 0)

	triggerStart := time.Now()
	var resp gatewayResponse
	err = withGatewayRetry(ctx, func() error {
		var callErr error
		resp, callErr = r.postTriggerSchedule(ctx, triggerReq, token)
		return callErr
	})
	triggerDur := int(time.Since(triggerStart).Milliseconds())
	if err != nil {
		r.logger.Warn("executeTrigger: postTriggerSchedule failed", slog.String("bot_id", p.botID), slog.Any("error", err))
		r.logProcessStep(ctx, p.botID, p.botID, rc.traceID, p.ownerUserID, p.platform,
			processlog.StepLLMResponseReceived, processlog.LevelError, "Trigger request failed: "+err.Error(),
			map[string]any{"error": err.Error()}, triggerDur)
		r.completeEvolutionLogOnError(ctx, p.evolutionLogID, err)
		return err
	}

	responsePreview := extractAssistantPreview(resp.Messages, 300)
	r.logProcessStep(ctx, p.botID, p.botID, rc.traceID, p.ownerUserID, p.platform,
		processlog.StepLLMResponseReceived, processlog.LevelInfo, fmt.Sprintf("Trigger response received (%s)", p.usageType),
		map[string]any{
			"model":            rc.model.ModelID,
			"response_length":  len(resp.Messages),
			"response_preview": responsePreview,
			"usage":            resp.Usage,
		}, triggerDur)

	r.recordTokenUsage(ctx, p.botID, resp.Usage, rc.model.ModelID, p.usageType)
	r.completeEvolutionLogFromResponse(ctx, p.evolutionLogID, resp)

	// Fallback delivery: if the LLM never called the send MCP tool, push the
	// response to the channel directly so the user receives it.
	// Skip for heartbeat triggers — their maintenance reports are noise for the user.
	// Priority: (1) LLM text response, (2) last tool-result readable content.
	isHeartbeat := p.schedule.TriggerType == "heartbeat"
	if r.triggerSender != nil &&
		!isHeartbeat &&
		strings.TrimSpace(p.platform) != "" &&
		strings.TrimSpace(p.replyTarget) != "" {
		if !hasSendToolCallInMessages(resp.Messages) {
			var fallbackText string
			var fallbackSource string
			if text := extractFullAssistantText(resp.Messages); strings.TrimSpace(text) != "" {
				fallbackText = text
				fallbackSource = "assistant_text"
			} else if text := extractLastToolResultSummary(resp.Messages); strings.TrimSpace(text) != "" {
				fallbackText = text
				fallbackSource = "tool_result"
			}
			if fallbackText != "" {
				r.logger.Info("executeTrigger: LLM skipped send tool – delivering via fallback",
					slog.String("bot_id", p.botID),
					slog.String("platform", p.platform),
					slog.String("reply_target", p.replyTarget),
					slog.String("source", fallbackSource),
					slog.String("text_preview", truncate(fallbackText, 100)),
				)
				if sendErr := r.triggerSender.SendText(ctx, p.botID, p.platform, p.replyTarget, fallbackText); sendErr != nil {
					r.logger.Warn("executeTrigger: fallback send failed",
						slog.String("bot_id", p.botID),
						slog.Any("error", sendErr),
					)
				}
			}
		}
	}

	r.logProcessStep(ctx, p.botID, p.botID, rc.traceID, p.ownerUserID, p.platform,
		processlog.StepResponseSent, processlog.LevelInfo, fmt.Sprintf("Trigger round stored (%s)", p.usageType),
		map[string]any{
			"message_count":    len(resp.Messages),
			"response_preview": responsePreview,
		}, 0)

	// Extract file attachments from sync trigger response.
	req.FileAttachments = extractGatewayAttachments(resp.Attachments)

	if err := r.storeRound(ctx, req, resp.Messages, resp.Usage); err != nil {
		r.logger.Warn("executeTrigger: storeRound failed", slog.String("bot_id", p.botID), slog.Any("error", err))
		return err
	}
	return nil
}

// hasSendToolCallInMessages returns true if any assistant message in the slice
// contains a tool call to the "send" MCP tool (checks both the ToolCalls field
// and Vercel AI SDK-style content parts).
func hasSendToolCallInMessages(messages []conversation.ModelMessage) bool {
	for _, msg := range messages {
		if msg.Role != "assistant" {
			continue
		}
		for _, tc := range msg.ToolCalls {
			if tc.Function.Name == "send" {
				return true
			}
		}
		if len(msg.Content) > 0 {
			var parts []struct {
				Type     string `json:"type"`
				ToolName string `json:"toolName"`
			}
			if err := json.Unmarshal(msg.Content, &parts); err == nil {
				for _, part := range parts {
					if part.Type == "tool-call" && part.ToolName == "send" {
						return true
					}
				}
			}
		}
	}
	return false
}

// extractFullAssistantText returns the full text of the last assistant message
// (not truncated), skipping reasoning/thinking parts.
func extractFullAssistantText(messages []conversation.ModelMessage) string {
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == "assistant" {
			text := messages[i].TextContent()
			if text != "" {
				return text
			}
		}
	}
	return ""
}

// extractLastToolResultSummary scans the last tool-role message in the
// conversation and attempts to return a human-readable excerpt from the tool
// result. It is used as a secondary fallback when the LLM produced no text
// response (e.g., it only made tool calls without a final summary).
//
// It handles two serialization formats emitted by the Vercel AI SDK:
//  1. Plain string content: returned directly.
//  2. Array of parts with type "tool-result": the inner result is inspected;
//     well-known text fields (stdout, output, text, message, result, content)
//     are preferred, otherwise the raw JSON is returned as a last resort.
func extractLastToolResultSummary(messages []conversation.ModelMessage) string {
	const maxChars = 400
	for i := len(messages) - 1; i >= 0; i-- {
		msg := messages[i]
		if msg.Role != "tool" {
			continue
		}
		// Standard path: plain string content.
		if text := strings.TrimSpace(msg.TextContent()); text != "" {
			return truncate(text, maxChars)
		}
		// Vercel AI SDK path: [{type:"tool-result", toolCallId:"...", result:{...}}]
		if len(msg.Content) == 0 {
			continue
		}
		var parts []struct {
			Type   string          `json:"type"`
			Result json.RawMessage `json:"result"`
		}
		if err := json.Unmarshal(msg.Content, &parts); err != nil {
			continue
		}
		// Walk parts in reverse to get the last tool-result.
		for j := len(parts) - 1; j >= 0; j-- {
			p := parts[j]
			if p.Type != "tool-result" || len(p.Result) == 0 {
				continue
			}
			// Try as plain string.
			var s string
			if err := json.Unmarshal(p.Result, &s); err == nil {
				if s = strings.TrimSpace(s); s != "" {
					return truncate(s, maxChars)
				}
			}
			// Try as object; prefer well-known text-bearing fields.
			var obj map[string]json.RawMessage
			if err := json.Unmarshal(p.Result, &obj); err == nil {
				for _, key := range []string{"stdout", "output", "text", "message", "result", "content"} {
					if raw, ok := obj[key]; ok {
						var field string
						if err2 := json.Unmarshal(raw, &field); err2 == nil {
							if field = strings.TrimSpace(field); field != "" {
								return truncate(field, maxChars)
							}
						}
					}
				}
				// Nothing useful in known fields — fall back to the raw JSON.
				raw := strings.TrimSpace(string(p.Result))
				if raw != "" && raw != "null" && raw != "{}" {
					return truncate(raw, maxChars)
				}
			}
		}
	}
	return ""
}

// TriggerSchedule executes a scheduled command through the agent gateway trigger-schedule endpoint.
func (r *Resolver) TriggerSchedule(ctx context.Context, botID string, payload schedule.TriggerPayload, token string) error {
	r.logger.Info("TriggerSchedule: starting",
		slog.String("bot_id", botID),
		slog.String("schedule_id", payload.ID),
		slog.String("platform", payload.Platform),
		slog.String("reply_target", payload.ReplyTarget),
	)
	if strings.TrimSpace(botID) == "" {
		return fmt.Errorf("bot id is required")
	}
	if strings.TrimSpace(payload.Command) == "" {
		return fmt.Errorf("schedule command is required")
	}
	err := r.executeTrigger(ctx, triggerParams{
		botID:       botID,
		query:       payload.Command,
		ownerUserID: payload.OwnerUserID,
		displayName: "Scheduler",
		schedule: gatewaySchedule{
			ID:          payload.ID,
			Name:        payload.Name,
			Description: payload.Description,
			Pattern:     payload.Pattern,
			MaxCalls:    payload.MaxCalls,
			Command:     payload.Command,
			TriggerType: "schedule",
		},
		usageType:   "schedule",
		platform:    payload.Platform,
		replyTarget: payload.ReplyTarget,
	}, token)
	if err != nil {
		r.logger.Warn("TriggerSchedule: failed", slog.String("bot_id", botID), slog.Any("error", err))
		return err
	}
	return nil
}

// TriggerHeartbeat executes a heartbeat command through the agent gateway trigger-schedule endpoint.
// It reuses the schedule trigger pathway since a heartbeat is functionally identical to a scheduled command.
func (r *Resolver) TriggerHeartbeat(ctx context.Context, botID string, payload heartbeat.TriggerPayload, token string) error {
	r.logger.Info("TriggerHeartbeat: starting", slog.String("bot_id", botID))
	if strings.TrimSpace(botID) == "" {
		return fmt.Errorf("bot id is required")
	}
	if strings.TrimSpace(payload.Prompt) == "" {
		return fmt.Errorf("heartbeat prompt is required")
	}
	err := r.executeTrigger(ctx, triggerParams{
		botID:       botID,
		query:       payload.Prompt,
		ownerUserID: payload.OwnerUserID,
		displayName: "Heartbeat",
		schedule: gatewaySchedule{
			ID:          payload.HeartbeatID,
			Name:        "heartbeat",
			Description: fmt.Sprintf("Heartbeat trigger (reason: %s)", payload.Reason),
			Pattern:     payload.IntervalPattern,
			Command:     payload.Prompt,
			TriggerType: "heartbeat",
		},
		usageType:            "heartbeat",
		evolutionLogID:       payload.EvolutionLogID,
		historyLimitOverride: settings.DefaultEvolutionHistoryLimit,
	}, token)
	if err != nil {
		r.logger.Warn("TriggerHeartbeat: failed", slog.String("bot_id", botID), slog.Any("error", err))
		return err
	}
	return nil
}

// completeEvolutionLogFromResponse parses the agent response and completes the evolution log.
func (r *Resolver) completeEvolutionLogFromResponse(ctx context.Context, logID string, resp gatewayResponse) {
	if logID == "" || r.queries == nil {
		return
	}
	pgLogID, err := db.ParseUUID(logID)
	if err != nil {
		return
	}

	// Extract the assistant's text from the response messages.
	var textParts []string
	for _, msg := range resp.Messages {
		if msg.Role == "assistant" {
			if t := msg.TextContent(); t != "" {
				textParts = append(textParts, t)
			}
		}
	}
	agentText := strings.TrimSpace(strings.Join(textParts, "\n"))

	// Determine status based on the agent response text.
	status := "completed"
	lowerText := strings.ToLower(agentText)
	if strings.Contains(lowerText, "no evolution needed") || strings.Contains(lowerText, "no changes needed") {
		status = "skipped"
	}

	// Extract a brief summary (first paragraph or first 500 chars).
	summary := agentText
	if idx := strings.Index(summary, "\n\n"); idx > 0 && idx < 500 {
		summary = summary[:idx]
	} else if len(summary) > 500 {
		summary = summary[:500] + "..."
	}

	_, completeErr := r.queries.CompleteEvolutionLog(ctx, sqlc.CompleteEvolutionLogParams{
		ID:             pgLogID,
		Status:         status,
		ChangesSummary: pgtype.Text{String: summary, Valid: summary != ""},
		FilesModified:  nil,
		AgentResponse:  pgtype.Text{String: agentText, Valid: agentText != ""},
	})
	if completeErr != nil {
		r.logger.Warn("failed to complete evolution log",
			slog.String("log_id", logID), slog.Any("error", completeErr))
	}
}

// completeEvolutionLogOnError marks an evolution log as failed.
func (r *Resolver) completeEvolutionLogOnError(ctx context.Context, logID string, triggerErr error) {
	if logID == "" || r.queries == nil {
		return
	}
	pgLogID, parseErr := db.ParseUUID(logID)
	if parseErr != nil {
		r.logger.Warn("completeEvolutionLogOnError: UUID parse failed", slog.String("log_id", logID), slog.Any("error", parseErr))
		return
	}
	errMsg := triggerErr.Error()
	_, dbErr := r.queries.CompleteEvolutionLog(ctx, sqlc.CompleteEvolutionLogParams{
		ID:             pgLogID,
		Status:         "failed",
		ChangesSummary: pgtype.Text{String: "Error: " + errMsg, Valid: true},
		FilesModified:  nil,
		AgentResponse:  pgtype.Text{Valid: false},
	})
	if dbErr != nil {
		r.logger.Warn("completeEvolutionLogOnError: DB update failed", slog.String("log_id", logID), slog.Any("error", dbErr))
	}
}

// --- StreamChat ---

// StreamChat sends a streaming chat request to the agent gateway.
func (r *Resolver) StreamChat(ctx context.Context, req conversation.ChatRequest) (<-chan conversation.StreamChunk, <-chan error) {
	chunkCh := make(chan conversation.StreamChunk)
	errCh := make(chan error, 1)
	r.logger.Info("gateway stream start",
		slog.String("bot_id", req.BotID),
		slog.String("chat_id", req.ChatID),
	)

	go func() {
		defer close(chunkCh)
		defer close(errCh)

		streamReq := req
		rc, err := r.resolve(ctx, streamReq)
		if err != nil {
			r.logger.Error("gateway stream resolve failed",
				slog.String("bot_id", streamReq.BotID),
				slog.String("chat_id", streamReq.ChatID),
				slog.Any("error", err),
			)
			errCh <- err
			return
		}

		// Log LLM request sent (stream)
		r.logProcessStep(ctx, req.BotID, req.ChatID, rc.traceID, req.UserID, req.CurrentChannel,
			processlog.StepLLMRequestSent, processlog.LevelInfo, "LLM request sent (stream)",
			map[string]any{
				"model":         rc.model.ModelID,
				"provider":      rc.provider.ClientType,
				"message_count": len(rc.payload.Messages),
			}, 0)

		streamStartTime := time.Now()

		// Log stream started
		r.logProcessStep(ctx, req.BotID, req.ChatID, rc.traceID, req.UserID, req.CurrentChannel,
			processlog.StepStreamStarted, processlog.LevelInfo, "Stream started",
			map[string]any{
				"model": rc.model.ModelID,
			}, 0)

		if !streamReq.UserMessagePersisted {
			if err := r.persistUserMessage(context.WithoutCancel(ctx), streamReq); err != nil {
				r.logger.Error("gateway stream persist user message failed",
					slog.String("bot_id", streamReq.BotID),
					slog.String("chat_id", streamReq.ChatID),
					slog.Any("error", err),
				)
				errCh <- err
				return
			}
			streamReq.UserMessagePersisted = true
		}

		// doStreamAttempt runs one streaming attempt: launches streamChat in a
		// goroutine, forwards chunks to chunkCh, and returns (error, chunksForwarded).
		doStreamAttempt := func(attempt resolvedContext) (error, int) {
			wrapped := make(chan conversation.StreamChunk)
			streamErr := make(chan error, 1)
			go func() {
				defer close(wrapped)
				streamErr <- r.streamChat(ctx, attempt.payload, streamReq, wrapped, attempt.model.ModelID, attempt.traceID)
			}()
			forwarded := 0
			for chunk := range wrapped {
				forwarded++
				select {
				case chunkCh <- chunk:
				case <-ctx.Done():
					return ctx.Err(), forwarded
				}
			}
			return <-streamErr, forwarded
		}

		streamErr, forwarded := doStreamAttempt(rc)

		// Failover: if the primary model failed before sending any data, try
		// the fallback model (mirrors the sync Chat() failover logic).
		if streamErr != nil && forwarded == 0 {
			if fbCtx, fbErr := r.tryFallback(ctx, rc); fbErr == nil {
				r.logProcessStep(ctx, req.BotID, req.ChatID, rc.traceID, req.UserID, req.CurrentChannel,
					processlog.StepModelFallback, processlog.LevelWarn, "Primary stream failed, switching to fallback",
					map[string]any{
						"primary_model":     rc.model.ModelID,
						"primary_provider":  rc.provider.ClientType,
						"primary_error":     streamErr.Error(),
						"fallback_model":    fbCtx.model.ModelID,
						"fallback_provider": fbCtx.provider.ClientType,
					}, 0)
				fbErr, fbForwarded := doStreamAttempt(fbCtx)
				if fbErr == nil || fbForwarded > 0 {
					rc = fbCtx
					streamErr = fbErr
				}
			}
		}

		streamDurMs := int(time.Since(streamStartTime).Milliseconds())
		if streamErr != nil {
			r.logProcessStep(ctx, req.BotID, req.ChatID, rc.traceID, req.UserID, req.CurrentChannel,
				processlog.StepStreamError, processlog.LevelError, "Stream error",
				map[string]any{"error": streamErr.Error()}, streamDurMs)
			errCh <- streamErr
		} else {
			r.logProcessStep(ctx, req.BotID, req.ChatID, rc.traceID, req.UserID, req.CurrentChannel,
				processlog.StepStreamCompleted, processlog.LevelInfo, "Stream completed",
				map[string]any{
					"model": rc.model.ModelID,
				}, streamDurMs)
			r.logProcessStep(ctx, req.BotID, req.ChatID, rc.traceID, req.UserID, req.CurrentChannel,
				processlog.StepResponseSent, processlog.LevelInfo, "Response sent (stream)",
				nil, 0)
		}
	}()
	return chunkCh, errCh
}

// --- HTTP helpers ---

func (r *Resolver) postChat(ctx context.Context, payload gatewayRequest, token string) (gatewayResponse, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return gatewayResponse{}, err
	}
	url := r.gatewayBaseURL + "/chat/"
	r.logger.Info("gateway request", slog.String("url", url), slog.String("body_prefix", truncate(string(body), 200)))

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return gatewayResponse{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if strings.TrimSpace(token) != "" {
		httpReq.Header.Set("Authorization", token)
	}

	resp, err := r.httpClient.Do(httpReq)
	if err != nil {
		return gatewayResponse{}, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return gatewayResponse{}, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		safe := sanitizeGatewayError(string(respBody))
		r.logger.Error("gateway error", slog.String("url", url), slog.Int("status", resp.StatusCode), slog.String("error_summary", safe))
		return gatewayResponse{}, &gatewayHTTPError{StatusCode: resp.StatusCode, Message: safe}
	}

	var parsed gatewayResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		r.logger.Error("gateway response parse failed", slog.String("body_prefix", truncate(string(respBody), 300)), slog.Any("error", err))
		return gatewayResponse{}, fmt.Errorf("failed to parse gateway response: %w", err)
	}
	return parsed, nil
}

// postTriggerSchedule sends a trigger-schedule request to the agent gateway.
func (r *Resolver) postTriggerSchedule(ctx context.Context, payload triggerScheduleRequest, token string) (gatewayResponse, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return gatewayResponse{}, err
	}
	url := r.gatewayBaseURL + "/chat/trigger-schedule"
	r.logger.Info("gateway trigger-schedule request", slog.String("url", url), slog.String("schedule_id", payload.Schedule.ID))

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return gatewayResponse{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if strings.TrimSpace(token) != "" {
		httpReq.Header.Set("Authorization", token)
	}

	resp, err := r.httpClient.Do(httpReq)
	if err != nil {
		return gatewayResponse{}, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return gatewayResponse{}, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		safe := sanitizeGatewayError(string(respBody))
		r.logger.Error("gateway trigger-schedule error", slog.String("url", url), slog.Int("status", resp.StatusCode), slog.String("error_summary", safe))
		return gatewayResponse{}, &gatewayHTTPError{StatusCode: resp.StatusCode, Message: safe}
	}

	var parsed gatewayResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		r.logger.Error("gateway trigger-schedule response parse failed", slog.String("body_prefix", truncate(string(respBody), 300)), slog.Any("error", err))
		return gatewayResponse{}, fmt.Errorf("failed to parse gateway response: %w", err)
	}
	return parsed, nil
}

// sanitizeGatewayError extracts the user-safe error message from a gateway
// error response body, stripping sensitive fields like API keys.
func sanitizeGatewayError(body string) string {
	var obj struct {
		Error   string `json:"error"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal([]byte(body), &obj); err == nil {
		if obj.Error != "" {
			return obj.Error
		}
		if obj.Message != "" {
			return obj.Message
		}
	}
	return truncate(body, 200)
}

func (r *Resolver) streamChat(ctx context.Context, payload gatewayRequest, req conversation.ChatRequest, chunkCh chan<- conversation.StreamChunk, modelID, traceID string) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	url := r.gatewayBaseURL + "/chat/stream"
	r.logger.Info("gateway stream request", slog.String("url", url), slog.String("body_prefix", truncate(string(body), 200)))
	httpReq, err := http.NewRequestWithContext(context.WithoutCancel(ctx), http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")
	if strings.TrimSpace(req.Token) != "" {
		httpReq.Header.Set("Authorization", req.Token)
	}

	resp, err := r.streamingClient.Do(httpReq)
	if err != nil {
		r.logger.Error("gateway stream connect failed", slog.String("url", url), slog.Any("error", err))
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		errBody, _ := io.ReadAll(resp.Body)
		safe := sanitizeGatewayError(string(errBody))
		r.logger.Error("gateway stream error", slog.String("url", url), slog.Int("status", resp.StatusCode), slog.String("error_summary", safe))
		return fmt.Errorf("agent gateway error (status %d): %s", resp.StatusCode, safe)
	}

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 16*1024*1024)

	currentEvent := ""
	stored := false
	receivedChunks := 0
	toolCallTimers := map[string]time.Time{}
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "event:") {
			currentEvent = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
			continue
		}
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "" || data == "[DONE]" {
			continue
		}
		receivedChunks++
		chunkCh <- conversation.StreamChunk([]byte(data))

		// Parse tool call events for process logging.
		r.logToolCallEvent(ctx, req, traceID, data, toolCallTimers)

		// Collect file attachments from attachment_delta events.
		r.collectStreamAttachments(&req, data)

		if stored {
			continue
		}
		// Use a context that survives client disconnect so messages are always persisted.
		if handled, storeErr := r.tryStoreStream(context.WithoutCancel(ctx), req, currentEvent, data, modelID, traceID); storeErr != nil {
			return storeErr
		} else if handled {
			stored = true
		}
	}
	if !stored {
		r.logger.Warn("streamChat: stream ended without storing round",
			slog.String("bot_id", req.BotID),
			slog.String("chat_id", req.ChatID),
			slog.Int("chunks", receivedChunks),
		)
	}
	scanErr := scanner.Err()
	if scanErr != nil && stored {
		r.logger.Warn("gateway stream scanner error after store (ignored)",
			slog.Any("error", scanErr), slog.Int("chunks", receivedChunks))
		return nil
	}
	if scanErr != nil && (errors.Is(scanErr, io.ErrUnexpectedEOF) || errors.Is(scanErr, io.EOF)) {
		if receivedChunks > 0 {
			r.logger.Warn("gateway stream ended with EOF after receiving data",
				slog.Any("error", scanErr), slog.Int("chunks", receivedChunks))
			return fmt.Errorf("agent gateway stream interrupted — the model may have returned a partial response")
		}
		r.logger.Error("gateway stream EOF with no data", slog.Any("error", scanErr))
		return fmt.Errorf("agent gateway connection lost before any response was received")
	}
	return scanErr
}

// extractGatewayAttachments converts gateway file attachments to conversation
// FileAttachments, deduplicating by path.
func extractGatewayAttachments(raw []gatewayFileAttachment) []conversation.FileAttachment {
	if len(raw) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(raw))
	var out []conversation.FileAttachment
	for _, a := range raw {
		if a.Type != "file" {
			continue
		}
		p := strings.TrimSpace(a.Path)
		if p == "" {
			continue
		}
		if _, ok := seen[p]; ok {
			continue
		}
		seen[p] = struct{}{}
		out = append(out, conversation.FileAttachment{
			Path: p,
			Name: path.Base(p),
		})
	}
	return out
}

// collectStreamAttachments parses attachment_delta events from an SSE chunk
// and appends file attachments to req.FileAttachments, deduplicating by path.
func (r *Resolver) collectStreamAttachments(req *conversation.ChatRequest, data string) {
	var env struct {
		Type        string `json:"type"`
		Attachments []struct {
			Type string `json:"type"`
			Path string `json:"path"`
		} `json:"attachments"`
	}
	if json.Unmarshal([]byte(data), &env) != nil || env.Type != "attachment_delta" {
		return
	}
	for _, a := range env.Attachments {
		if a.Type != "file" {
			continue
		}
		p := strings.TrimSpace(a.Path)
		if p == "" {
			continue
		}
		dup := false
		for _, existing := range req.FileAttachments {
			if existing.Path == p {
				dup = true
				break
			}
		}
		if dup {
			continue
		}
		req.FileAttachments = append(req.FileAttachments, conversation.FileAttachment{
			Path: p,
			Name: path.Base(p),
		})
		r.logger.Debug("collectStreamAttachments: collected file",
			slog.String("path", p),
			slog.String("bot_id", req.BotID),
			slog.String("chat_id", req.ChatID),
		)
	}
}

// logToolCallEvent parses a stream chunk and logs tool_call_started / tool_call_completed.
func (r *Resolver) logToolCallEvent(ctx context.Context, req conversation.ChatRequest, traceID, data string, timers map[string]time.Time) {
	if r.processLogService == nil {
		return
	}
	var envelope struct {
		Type       string `json:"type"`
		ToolName   string `json:"toolName"`
		ToolCallID string `json:"toolCallId"`
		Input      any    `json:"input,omitempty"`
		Result     any    `json:"result,omitempty"`
	}
	if err := json.Unmarshal([]byte(data), &envelope); err != nil {
		return
	}

	switch envelope.Type {
	case "tool_call_start":
		timers[envelope.ToolCallID] = time.Now()
		inputStr := truncate(fmt.Sprintf("%v", envelope.Input), 2000)
		r.logProcessStep(ctx, req.BotID, req.ChatID, traceID, req.UserID, req.CurrentChannel,
			processlog.StepToolCallStarted, processlog.LevelInfo,
			fmt.Sprintf("Tool call: %s", envelope.ToolName),
			map[string]any{
				"tool_name":    envelope.ToolName,
				"tool_call_id": envelope.ToolCallID,
				"input":        inputStr,
			}, 0)

	case "tool_call_end":
		var durationMs int
		if start, ok := timers[envelope.ToolCallID]; ok {
			durationMs = int(time.Since(start).Milliseconds())
			delete(timers, envelope.ToolCallID)
		}
		resultStr := truncate(fmt.Sprintf("%v", envelope.Result), 2000)
		r.logProcessStep(ctx, req.BotID, req.ChatID, traceID, req.UserID, req.CurrentChannel,
			processlog.StepToolCallCompleted, processlog.LevelInfo,
			fmt.Sprintf("Tool completed: %s (%dms)", envelope.ToolName, durationMs),
			map[string]any{
				"tool_name":    envelope.ToolName,
				"tool_call_id": envelope.ToolCallID,
				"result":       resultStr,
			}, durationMs)
	}
}

// tryStoreStream attempts to extract final messages from a stream event and persist them.
func (r *Resolver) tryStoreStream(ctx context.Context, req conversation.ChatRequest, eventType, data, modelID, traceID string) (bool, error) {
	record := func(usage *gatewayUsage) {
		r.recordTokenUsage(ctx, req.BotID, usage, modelID, "chat")
	}

	// event: done + data: {messages: [...]}
	if eventType == "done" {
		var resp gatewayResponse
		if err := json.Unmarshal([]byte(data), &resp); err == nil && len(resp.Messages) > 0 {
			record(resp.Usage)
			return true, r.storeRoundWithTrace(ctx, req, traceID, resp.Messages, resp.Usage)
		}
	}

	// data: {"type":"text_delta"|"agent_end"|"done", ...}
	var envelope struct {
		Type     string                      `json:"type"`
		Data     json.RawMessage             `json:"data"`
		Messages []conversation.ModelMessage `json:"messages"`
		Skills   []string                    `json:"skills"`
		Usage    *gatewayUsage               `json:"usage,omitempty"`
	}
	if err := json.Unmarshal([]byte(data), &envelope); err == nil {
		if (envelope.Type == "agent_end" || envelope.Type == "done") && len(envelope.Messages) > 0 {
			record(envelope.Usage)
			return true, r.storeRoundWithTrace(ctx, req, traceID, envelope.Messages, envelope.Usage)
		}
		if envelope.Type == "done" && len(envelope.Data) > 0 {
			var resp gatewayResponse
			if err := json.Unmarshal(envelope.Data, &resp); err == nil && len(resp.Messages) > 0 {
				record(resp.Usage)
				return true, r.storeRoundWithTrace(ctx, req, traceID, resp.Messages, resp.Usage)
			}
		}
	}

	// fallback: data: {messages: [...]}
	var resp gatewayResponse
	if err := json.Unmarshal([]byte(data), &resp); err == nil && len(resp.Messages) > 0 {
		record(resp.Usage)
		return true, r.storeRoundWithTrace(ctx, req, traceID, resp.Messages, resp.Usage)
	}
	return false, nil
}

// --- container resolution ---

func (r *Resolver) resolveContainerID(ctx context.Context, botID, explicit string) string {
	if strings.TrimSpace(explicit) != "" {
		return explicit
	}
	if r.queries != nil {
		pgBotID, err := parseResolverUUID(botID)
		if err == nil {
			row, err := r.queries.GetContainerByBotID(ctx, pgBotID)
			if err == nil && strings.TrimSpace(row.ContainerID) != "" {
				return row.ContainerID
			}
		}
	}
	r.logger.Warn("no container found for bot, using fallback", slog.String("bot_id", botID))
	return "mcp-" + botID
}

// --- message loading ---

func (r *Resolver) loadMessages(ctx context.Context, chatID string, maxContextMinutes int) ([]conversation.ModelMessage, error) {
	if r.messageService == nil {
		return nil, nil
	}
	since := time.Now().UTC().Add(-time.Duration(maxContextMinutes) * time.Minute)
	msgs, err := r.messageService.ListSince(ctx, chatID, since)
	if err != nil {
		return nil, err
	}
	var result []conversation.ModelMessage
	for _, m := range msgs {
		var mm conversation.ModelMessage
		if err := json.Unmarshal(m.Content, &mm); err != nil {
			r.logger.Warn("loadMessages: content unmarshal failed, treating as raw text",
				slog.String("chat_id", chatID), slog.Any("error", err))
			mm = conversation.ModelMessage{Role: m.Role, Content: m.Content}
		} else {
			mm.Role = m.Role
		}
		result = append(result, mm)
	}
	return result, nil
}

type memoryContextItem struct {
	Namespace string
	Item      memory.MemoryItem
}

var commonStopWords = map[string]struct{}{
	"the": {}, "a": {}, "an": {}, "is": {}, "are": {}, "was": {}, "were": {},
	"be": {}, "been": {}, "being": {}, "have": {}, "has": {}, "had": {},
	"do": {}, "does": {}, "did": {}, "will": {}, "would": {}, "could": {},
	"should": {}, "may": {}, "might": {}, "shall": {}, "can": {},
	"to": {}, "of": {}, "in": {}, "for": {}, "on": {}, "with": {}, "at": {},
	"by": {}, "from": {}, "as": {}, "into": {}, "about": {}, "like": {},
	"and": {}, "or": {}, "but": {}, "if": {}, "then": {}, "else": {},
	"it": {}, "its": {}, "this": {}, "that": {}, "these": {}, "those": {},
	"i": {}, "me": {}, "my": {}, "we": {}, "our": {}, "you": {}, "your": {},
	"he": {}, "she": {}, "they": {}, "them": {}, "his": {}, "her": {},
	"what": {}, "which": {}, "who": {}, "when": {}, "where": {}, "how": {},
	"not": {}, "no": {}, "so": {}, "up": {}, "out": {}, "just": {},
	"的": {}, "了": {}, "是": {}, "在": {}, "我": {}, "有": {}, "和": {},
	"就": {}, "不": {}, "人": {}, "都": {}, "一": {}, "一个": {}, "上": {},
	"也": {}, "很": {}, "到": {}, "说": {}, "要": {}, "去": {}, "你": {},
	"会": {}, "着": {}, "没有": {}, "看": {}, "好": {}, "自己": {}, "这": {},
	"他": {}, "她": {}, "吗": {}, "吧": {}, "呢": {}, "啊": {}, "哦": {},
	"嗯": {}, "把": {}, "被": {}, "让": {}, "跟": {}, "给": {}, "从": {},
}

func extractKeywords(query string) string {
	words := strings.Fields(strings.ToLower(query))
	var keywords []string
	for _, w := range words {
		w = strings.Trim(w, ".,!?;:\"'()[]{}"+"\u3002\uff0c\uff01\uff1f\uff1b\uff1a\u201c\u201d\u2018\u2019\uff08\uff09\u3010\u3011")
		if len(w) < 2 {
			continue
		}
		if _, stop := commonStopWords[w]; stop {
			continue
		}
		keywords = append(keywords, w)
	}
	if len(keywords) == 0 {
		return ""
	}
	return strings.Join(keywords, " ")
}

func rrfMerge(primary, secondary []memory.MemoryItem, k int) []memory.MemoryItem {
	const rrfK = 60.0
	scores := map[string]float64{}
	itemMap := map[string]memory.MemoryItem{}

	for rank, item := range primary {
		id := item.ID
		if id == "" {
			id = item.Memory
		}
		scores[id] += 1.0 / (rrfK + float64(rank+1))
		itemMap[id] = item
	}
	for rank, item := range secondary {
		id := item.ID
		if id == "" {
			id = item.Memory
		}
		scores[id] += 1.0 / (rrfK + float64(rank+1))
		if _, exists := itemMap[id]; !exists {
			itemMap[id] = item
		}
	}

	type scored struct {
		id    string
		score float64
	}
	var ranked []scored
	for id, s := range scores {
		ranked = append(ranked, scored{id, s})
	}
	sort.Slice(ranked, func(i, j int) bool {
		return ranked[i].score > ranked[j].score
	})
	if len(ranked) > k {
		ranked = ranked[:k]
	}

	result := make([]memory.MemoryItem, 0, len(ranked))
	for _, r := range ranked {
		item := itemMap[r.id]
		item.Score = r.score * 100
		result = append(result, item)
	}
	return result
}

const trivialQueryMaxRunes = 15

func isTrivialQuery(query string) bool {
	q := strings.TrimSpace(query)
	if utf8.RuneCountInString(q) > trivialQueryMaxRunes {
		return false
	}
	lower := strings.ToLower(q)
	trivials := []string{
		"hi", "hello", "hey", "yo",
		"ok", "okay", "k",
		"yes", "no", "yep", "nope", "yeah", "nah",
		"thanks", "thank you", "thx", "ty",
		"bye", "goodbye",
		"好", "好的", "嗯", "嗯嗯", "哦", "是的", "对",
		"你好", "谢谢", "再见", "行", "可以",
		"👍", "👌", "🙏",
	}
	for _, t := range trivials {
		if lower == t {
			return true
		}
	}
	return false
}

func (r *Resolver) loadMemoryContextMessage(ctx context.Context, req conversation.ChatRequest, contextWindow int, traceID string) *conversation.ModelMessage {
	if r.memoryService == nil {
		return nil
	}
	if strings.TrimSpace(req.Query) == "" || strings.TrimSpace(req.BotID) == "" || strings.TrimSpace(req.ChatID) == "" {
		return nil
	}
	if isTrivialQuery(req.Query) {
		return nil
	}

	filters := map[string]any{
		"namespace": sharedMemoryNamespace,
		"scopeId":   req.BotID,
		"bot_id":    req.BotID,
	}

	resp, err := r.memoryService.Search(ctx, memory.SearchRequest{
		Query:   req.Query,
		BotID:   req.BotID,
		Limit:   memoryContextLimitPerScope,
		Filters: filters,
		NoStats: true,
	})
	if err != nil {
		r.logger.Warn("memory search for context failed",
			slog.String("namespace", sharedMemoryNamespace),
			slog.String("query", truncate(req.Query, 100)),
			slog.Any("error", err),
		)
		return nil
	}

	primaryCount := len(resp.Results)
	allItems := resp.Results

	expandedKeywords := ""
	kwCount := 0
	if kw := extractKeywords(req.Query); kw != "" && kw != strings.ToLower(req.Query) {
		expandedKeywords = kw
		kwResp, kwErr := r.memoryService.Search(ctx, memory.SearchRequest{
			Query:   kw,
			BotID:   req.BotID,
			Limit:   memoryContextLimitPerScope,
			Filters: filters,
			NoStats: true,
		})
		if kwErr == nil && len(kwResp.Results) > 0 {
			kwCount = len(kwResp.Results)
			allItems = rrfMerge(resp.Results, kwResp.Results, memoryContextLimitPerScope*2)
		}
	}

	if expandedKeywords != "" {
		r.logProcessStep(ctx, req.BotID, req.ChatID, traceID, req.UserID, req.CurrentChannel,
			processlog.StepQueryExpanded, processlog.LevelInfo, "Query expanded with keywords",
			map[string]any{
				"original_query":    truncate(req.Query, 200),
				"expanded_keywords": expandedKeywords,
				"primary_results":   primaryCount,
				"keyword_results":   kwCount,
				"merged_total":      len(allItems),
			}, 0)
	}

	filteredByScore := 0
	results := make([]memoryContextItem, 0, memoryContextLimitPerScope)
	seen := map[string]struct{}{}
	var scoreDistribution []float64
	for _, item := range allItems {
		if item.Score < memoryMinScoreThreshold {
			filteredByScore++
			continue
		}
		key := strings.TrimSpace(item.ID)
		if key == "" {
			key = sharedMemoryNamespace + ":" + strings.TrimSpace(item.Memory)
		}
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		scoreDistribution = append(scoreDistribution, item.Score)
		results = append(results, memoryContextItem{Namespace: sharedMemoryNamespace, Item: item})
	}

	// Search global SOLUTIONS area
	solFilters := map[string]any{
		"namespace": solutionsNamespace,
		"scopeId":   solutionsScopeID,
	}
	solResp, solErr := r.memoryService.Search(ctx, memory.SearchRequest{
		Query:   req.Query,
		Limit:   solutionsContextLimit,
		Filters: solFilters,
		NoStats: true,
	})
	if solErr == nil {
		for _, item := range solResp.Results {
			if item.Score < solutionsMinScoreThreshold {
				continue
			}
			key := strings.TrimSpace(item.ID)
			if key == "" {
				continue
			}
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			results = append(results, memoryContextItem{
				Namespace: solutionsNamespace,
				Item:      item,
			})
		}
	}

	if filteredByScore > 0 || len(results) > 0 {
		r.logProcessStep(ctx, req.BotID, req.ChatID, traceID, req.UserID, req.CurrentChannel,
			processlog.StepMemoryFiltered, processlog.LevelInfo, "Memory relevance filtering applied",
			map[string]any{
				"total_candidates":    len(allItems),
				"filtered_by_score":   filteredByScore,
				"deduplicated":        len(allItems) - filteredByScore - len(results),
				"passed":              len(results),
				"min_score_threshold": memoryMinScoreThreshold,
				"score_distribution":  formatScoreDistribution(scoreDistribution),
			}, 0)
	}

	if len(results) == 0 {
		return nil
	}

	preDecayCount := len(results)
	applyTemporalDecay(results)
	results = applyMMR(results, 0.7)

	if contextWindow <= 0 {
		contextWindow = 128000
	}
	memoryBudgetTokens := int(float64(contextWindow) * 0.05)
	if memoryBudgetTokens < 500 {
		memoryBudgetTokens = 500
	}
	memoryBudgetChars := int(float64(memoryBudgetTokens) * 2.5)

	maxItems := memoryContextMaxItems
	if len(results) > maxItems {
		results = results[:maxItems]
	}

	perItemChars := memoryBudgetChars / max(len(results), 1)
	if perItemChars < 80 {
		perItemChars = 80
	}
	if perItemChars > 500 {
		perItemChars = 500
	}

	var sb strings.Builder
	sb.WriteString("Relevant memory context (use when helpful):\n")
	totalChars := 0
	injectedCount := 0
	var injectedItems []map[string]any
	for _, entry := range results {
		text := strings.TrimSpace(entry.Item.Memory)
		if text == "" {
			continue
		}
		snippet := truncateMemorySnippet(text, perItemChars)
		lineLen := len(snippet) + len(entry.Namespace) + 6
		if totalChars+lineLen > memoryBudgetChars {
			break
		}
		sb.WriteString("- [")
		sb.WriteString(entry.Namespace)
		sb.WriteString("] ")
		sb.WriteString(snippet)
		sb.WriteString("\n")
		totalChars += lineLen
		injectedCount++
		injectedItems = append(injectedItems, map[string]any{
			"id":       entry.Item.ID,
			"score":    entry.Item.Score,
			"preview":  truncate(text, 100),
		})
	}
	payload := strings.TrimSpace(sb.String())
	if payload == "" {
		return nil
	}

	r.logProcessStep(ctx, req.BotID, req.ChatID, traceID, req.UserID, req.CurrentChannel,
		processlog.StepMemoryFiltered, processlog.LevelInfo, "Memory pipeline completed",
		map[string]any{
			"pre_decay_count":    preDecayCount,
			"post_mmr_count":     len(results),
			"injected_count":     injectedCount,
			"memory_budget_tokens": memoryBudgetTokens,
			"memory_budget_chars":  memoryBudgetChars,
			"per_item_chars":       perItemChars,
			"total_chars_used":     totalChars,
			"injected_items":       injectedItems,
		}, 0)

	msg := conversation.ModelMessage{
		Role:    "system",
		Content: conversation.NewTextContent(payload),
	}
	return &msg
}

// --- store helpers ---

func (r *Resolver) persistUserMessage(ctx context.Context, req conversation.ChatRequest) error {
	if r.messageService == nil {
		return nil
	}
	if strings.TrimSpace(req.BotID) == "" {
		return fmt.Errorf("bot id is required for persistence")
	}
	text := strings.TrimSpace(req.Query)
	if text == "" {
		return nil
	}

	message := conversation.ModelMessage{
		Role:    "user",
		Content: conversation.NewTextContent(text),
	}
	content, err := json.Marshal(message)
	if err != nil {
		return err
	}
	senderChannelIdentityID, senderUserID := r.resolvePersistSenderIDs(ctx, req)
	_, err = r.messageService.Persist(ctx, messagepkg.PersistInput{
		BotID:                   req.BotID,
		RouteID:                 req.RouteID,
		SenderChannelIdentityID: senderChannelIdentityID,
		SenderUserID:            senderUserID,
		Platform:                req.CurrentChannel,
		ExternalMessageID:       req.ExternalMessageID,
		Role:                    "user",
		Content:                 content,
		Metadata:                buildRouteMetadata(req),
	})
	return err
}

// memoryTraceCtx carries process-log context into the storeMemory goroutine.
type memoryTraceCtx struct {
	traceID string
	chatID  string
	userID  string
	channel string
}

func (r *Resolver) storeRound(ctx context.Context, req conversation.ChatRequest, messages []conversation.ModelMessage, usage ...*gatewayUsage) error {
	return r.storeRoundWithTrace(ctx, req, "", messages, usage...)
}

func (r *Resolver) storeRoundWithTrace(ctx context.Context, req conversation.ChatRequest, traceID string, messages []conversation.ModelMessage, usage ...*gatewayUsage) error {
	// Sanitize before storing so non-standard items (e.g. item_reference) are never persisted.
	messages = sanitizeMessages(messages)
	// Add user query as the first message if not already present in the round.
	// This ensures the user's prompt is persisted alongside the assistant's response.
	fullRound := make([]conversation.ModelMessage, 0, len(messages)+1)
	hasUserQuery := false
	for _, m := range messages {
		if m.Role == "user" && m.TextContent() == req.Query {
			hasUserQuery = true
			break
		}
	}
	if !req.UserMessagePersisted && !hasUserQuery && strings.TrimSpace(req.Query) != "" {
		fullRound = append(fullRound, conversation.ModelMessage{
			Role:    "user",
			Content: conversation.NewTextContent(req.Query),
		})
	}
	for _, m := range messages {
		if req.UserMessagePersisted && m.Role == "user" && strings.TrimSpace(m.TextContent()) == strings.TrimSpace(req.Query) {
			// User message was already persisted before streaming; skip duplicate copy in round payload.
			continue
		}
		fullRound = append(fullRound, m)
	}
	if len(fullRound) == 0 {
		r.logger.Warn("storeRound: fullRound is empty, skipping",
			slog.String("bot_id", req.BotID),
			slog.String("chat_id", req.ChatID),
		)
		return nil
	}

	r.logger.Info("storeRound: persisting round",
		slog.String("bot_id", req.BotID),
		slog.String("chat_id", req.ChatID),
		slog.Int("message_count", len(fullRound)),
	)

	var usagePtr *gatewayUsage
	if len(usage) > 0 {
		usagePtr = usage[0]
	}
	r.storeMessages(ctx, req, fullRound, usagePtr)

	// For memory extraction, always include the user's query so the LLM
	// can extract facts from what the user said. fullRound may have
	// excluded it when UserMessagePersisted is true (to avoid duplicate
	// persistence), but memory extraction needs the full conversation.
	memoryRound := fullRound
	if req.UserMessagePersisted && strings.TrimSpace(req.Query) != "" {
		memoryRound = make([]conversation.ModelMessage, 0, len(fullRound)+1)
		memoryRound = append(memoryRound, conversation.ModelMessage{
			Role:    "user",
			Content: conversation.NewTextContent(req.Query),
		})
		memoryRound = append(memoryRound, fullRound...)
	}

	mtc := memoryTraceCtx{
		traceID: traceID,
		chatID:  req.ChatID,
		userID:  req.UserID,
		channel: req.CurrentChannel,
	}
	go func() {
		defer func() {
			if rec := recover(); rec != nil {
				r.logger.Error("storeMemory panic recovered",
					slog.String("bot_id", req.BotID),
					slog.Any("panic", rec),
				)
			}
		}()
		r.storeMemory(context.WithoutCancel(ctx), req.BotID, memoryRound, mtc)
	}()

	if r.ovSessionExtractor != nil {
		r.logProcessStep(ctx, req.BotID, req.ChatID, traceID, req.UserID, req.CurrentChannel,
			processlog.StepOpenVikingSession, processlog.LevelInfo, "OpenViking session extraction queued",
			map[string]any{
				"message_count": len(memoryRound),
			}, 0)
		ovMsgs := make([]conversation.ModelMessage, len(memoryRound))
		copy(ovMsgs, memoryRound)
		ovCtx := context.WithoutCancel(ctx)
		ovBotID := req.BotID
		ovChatID := req.ChatID
		ovUserID := req.UserID
		ovChannel := req.CurrentChannel
		ovTraceID := traceID
		go func() {
			defer func() {
				if rec := recover(); rec != nil {
					r.logger.Error("OV session extraction panic recovered",
						slog.String("bot_id", ovBotID),
						slog.Any("panic", rec),
					)
				}
			}()
			start := time.Now()
			output, err := r.ovSessionExtractor.ExtractSession(ovCtx, ovBotID, ovChatID, ovMsgs)
			dur := int(time.Since(start).Milliseconds())
			if err != nil {
				r.logProcessStep(ovCtx, ovBotID, ovChatID, ovTraceID, ovUserID, ovChannel,
					processlog.StepOpenVikingSessionFailed, processlog.LevelError,
					"OpenViking session extraction failed: "+err.Error(),
					map[string]any{"error": err.Error()}, dur)
			} else if output != "" {
				r.logProcessStep(ovCtx, ovBotID, ovChatID, ovTraceID, ovUserID, ovChannel,
					processlog.StepOpenVikingSessionCompleted, processlog.LevelInfo,
					"OpenViking session extraction completed",
					map[string]any{"output": output}, dur)
			}
		}()
	}

	return nil
}

func (r *Resolver) storeMessages(ctx context.Context, req conversation.ChatRequest, messages []conversation.ModelMessage, usage *gatewayUsage) {
	if r.messageService == nil {
		return
	}
	if strings.TrimSpace(req.BotID) == "" {
		return
	}
	meta := buildRouteMetadata(req)

	// Find the index of the last assistant message to attach token usage and file attachments.
	hasUsage := usage != nil && usage.TotalTokens > 0
	hasAttachments := len(req.FileAttachments) > 0
	lastAssistantIdx := -1
	if hasUsage || hasAttachments {
		for i := len(messages) - 1; i >= 0; i-- {
			if messages[i].Role == "assistant" {
				lastAssistantIdx = i
				break
			}
		}
	}

	senderChannelIdentityID, senderUserID := r.resolvePersistSenderIDs(ctx, req)
	for i, msg := range messages {
		content, err := json.Marshal(msg)
		if err != nil {
			r.logger.Warn("storeMessages: marshal failed", slog.Any("error", err))
			continue
		}
		messageSenderChannelIdentityID := ""
		messageSenderUserID := ""
		externalMessageID := ""
		sourceReplyToMessageID := ""
		if msg.Role == "user" {
			messageSenderChannelIdentityID = senderChannelIdentityID
			messageSenderUserID = senderUserID
			externalMessageID = req.ExternalMessageID
		} else if strings.TrimSpace(req.ExternalMessageID) != "" {
			// Assistant/tool/system outputs are linked to the inbound source message for cross-channel reply threading.
			sourceReplyToMessageID = req.ExternalMessageID
		}

		// Build per-message metadata; embed token usage and file attachments on the last assistant message.
		msgMeta := meta
		if i == lastAssistantIdx {
			msgMeta = copyMap(meta)
			if hasUsage {
				msgMeta["token_usage"] = map[string]any{
					"prompt_tokens":     usage.PromptTokens,
					"completion_tokens": usage.CompletionTokens,
					"total_tokens":      usage.TotalTokens,
				}
			}
			if hasAttachments {
				atts := make([]map[string]string, len(req.FileAttachments))
				for j, a := range req.FileAttachments {
					atts[j] = map[string]string{"type": "file", "path": a.Path, "name": a.Name}
				}
				msgMeta["file_attachments"] = atts
			}
		}

		if _, err := r.messageService.Persist(ctx, messagepkg.PersistInput{
			BotID:                   req.BotID,
			RouteID:                 req.RouteID,
			SenderChannelIdentityID: messageSenderChannelIdentityID,
			SenderUserID:            messageSenderUserID,
			Platform:                req.CurrentChannel,
			ExternalMessageID:       externalMessageID,
			SourceReplyToMessageID:  sourceReplyToMessageID,
			Role:                    msg.Role,
			Content:                 content,
			Metadata:                msgMeta,
		}); err != nil {
			r.logger.Warn("persist message failed", slog.Any("error", err))
		}
	}
}

// copyMap returns a shallow copy of the input map (nil-safe).
func copyMap(m map[string]any) map[string]any {
	if m == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(m)+1)
	for k, v := range m {
		out[k] = v
	}
	return out
}

func buildRouteMetadata(req conversation.ChatRequest) map[string]any {
	if strings.TrimSpace(req.RouteID) == "" && strings.TrimSpace(req.CurrentChannel) == "" {
		return nil
	}
	meta := map[string]any{}
	if strings.TrimSpace(req.RouteID) != "" {
		meta["route_id"] = req.RouteID
	}
	if strings.TrimSpace(req.CurrentChannel) != "" {
		meta["platform"] = req.CurrentChannel
	}
	return meta
}

func (r *Resolver) resolvePersistSenderIDs(ctx context.Context, req conversation.ChatRequest) (string, string) {
	channelIdentityID := strings.TrimSpace(req.SourceChannelIdentityID)
	userID := strings.TrimSpace(req.UserID)

	senderChannelIdentityID := ""
	if r.isExistingChannelIdentityID(ctx, channelIdentityID) {
		senderChannelIdentityID = channelIdentityID
	}

	senderUserID := ""
	if r.isExistingUserID(ctx, userID) {
		senderUserID = userID
	}
	if senderUserID == "" && senderChannelIdentityID != "" {
		if linked := r.linkedUserIDFromChannelIdentity(ctx, senderChannelIdentityID); linked != "" {
			senderUserID = linked
		}
	}
	return senderChannelIdentityID, senderUserID
}

func (r *Resolver) isExistingChannelIdentityID(ctx context.Context, id string) bool {
	if r.queries == nil {
		return false
	}
	pgID, err := parseResolverUUID(id)
	if err != nil {
		return false
	}
	_, err = r.queries.GetChannelIdentityByID(ctx, pgID)
	return err == nil
}

func (r *Resolver) isExistingUserID(ctx context.Context, id string) bool {
	if r.queries == nil {
		return false
	}
	pgID, err := parseResolverUUID(id)
	if err != nil {
		return false
	}
	_, err = r.queries.GetUserByID(ctx, pgID)
	return err == nil
}

func (r *Resolver) linkedUserIDFromChannelIdentity(ctx context.Context, channelIdentityID string) string {
	if r.queries == nil {
		return ""
	}
	pgID, err := parseResolverUUID(channelIdentityID)
	if err != nil {
		return ""
	}
	row, err := r.queries.GetChannelIdentityByID(ctx, pgID)
	if err != nil || !row.UserID.Valid {
		return ""
	}
	return row.UserID.String()
}

// resolveDisplayName returns the best available display name for the request identity:
// req.DisplayName if set, else channel identity's display_name, else linked user's display_name, else "User".
func (r *Resolver) resolveDisplayName(ctx context.Context, req conversation.ChatRequest) string {
	if name := strings.TrimSpace(req.DisplayName); name != "" {
		return name
	}
	if r.queries == nil {
		return "User"
	}
	channelIdentityID := strings.TrimSpace(req.SourceChannelIdentityID)
	if channelIdentityID == "" {
		return "User"
	}
	pgID, err := parseResolverUUID(channelIdentityID)
	if err != nil {
		return "User"
	}
	ci, err := r.queries.GetChannelIdentityByID(ctx, pgID)
	if err == nil && ci.DisplayName.Valid {
		if name := strings.TrimSpace(ci.DisplayName.String); name != "" {
			return name
		}
	}
	linkedUserID := r.linkedUserIDFromChannelIdentity(ctx, channelIdentityID)
	if linkedUserID == "" {
		return "User"
	}
	userPgID, err := parseResolverUUID(linkedUserID)
	if err != nil {
		return "User"
	}
	u, err := r.queries.GetUserByID(ctx, userPgID)
	if err != nil || !u.DisplayName.Valid {
		return "User"
	}
	if name := strings.TrimSpace(u.DisplayName.String); name != "" {
		return name
	}
	return "User"
}

func (r *Resolver) storeMemory(ctx context.Context, botID string, messages []conversation.ModelMessage, mtc memoryTraceCtx) {
	if r.memoryService == nil {
		if r.logger != nil {
			r.logger.Warn("storeMemory: memoryService is nil, skipping")
		}
		r.logProcessStep(ctx, botID, mtc.chatID, mtc.traceID, mtc.userID, mtc.channel,
			processlog.StepMemoryExtractFailed, processlog.LevelWarn, "Memory service not configured",
			nil, 0)
		return
	}
	if strings.TrimSpace(botID) == "" {
		if r.logger != nil {
			r.logger.Warn("storeMemory: botID is empty, skipping")
		}
		return
	}
	if r.logger != nil {
		r.logger.Info("storeMemory: called",
			slog.String("bot_id", botID),
			slog.Int("input_messages", len(messages)),
		)
	}
	memMsgs := make([]memory.Message, 0, len(messages))
	for _, msg := range messages {
		text := strings.TrimSpace(msg.TextContent())
		if text == "" {
			if r.logger != nil {
				r.logger.Debug("storeMemory: skipping empty text message",
					slog.String("role", msg.Role),
					slog.Int("content_len", len(msg.Content)),
				)
			}
			continue
		}
		role := msg.Role
		if strings.TrimSpace(role) == "" {
			role = "assistant"
		}
		memMsgs = append(memMsgs, memory.Message{Role: role, Content: text})
	}
	if len(memMsgs) == 0 {
		if r.logger != nil {
			r.logger.Warn("storeMemory: no text messages after filtering, skipping",
				slog.String("bot_id", botID),
				slog.Int("input_messages", len(messages)),
			)
		}
		r.logProcessStep(ctx, botID, mtc.chatID, mtc.traceID, mtc.userID, mtc.channel,
			processlog.StepMemoryExtractFailed, processlog.LevelWarn, "No text messages to extract",
			map[string]any{"input_messages": len(messages)}, 0)
		return
	}

	// Inject bot-specific memory model into context so the LLM client
	// uses the model configured in bot settings instead of the global default.
	preferredModel := ""
	if r.queries != nil {
		if pgBotID, parseErr := db.ParseUUID(botID); parseErr == nil {
			if settingsRow, sErr := r.queries.GetSettingsByBotID(ctx, pgBotID); sErr == nil {
				if mid := strings.TrimSpace(settingsRow.MemoryModelID.String); mid != "" {
					ctx = memory.WithPreferredModel(ctx, mid)
					preferredModel = mid
				}
			}
		}
	}

	// Log memory extraction started
	r.logProcessStep(ctx, botID, mtc.chatID, mtc.traceID, mtc.userID, mtc.channel,
		processlog.StepMemoryExtractStarted, processlog.LevelInfo, "Memory extraction started",
		map[string]any{
			"message_count":   len(memMsgs),
			"preferred_model": preferredModel,
		}, 0)

	if r.logger != nil {
		r.logger.Info("storing memory",
			slog.String("bot_id", botID),
			slog.Int("message_count", len(memMsgs)),
			slog.String("namespace", sharedMemoryNamespace),
			slog.String("preferred_model", memory.PreferredModelFromCtx(ctx)),
		)
	}

	r.addMemory(ctx, botID, memMsgs, sharedMemoryNamespace, botID, mtc)

	// Async: extract and store global SOLUTIONS
	go func() {
		defer func() { recover() }()
		r.addSolutions(context.WithoutCancel(ctx), botID, memMsgs, mtc)
	}()
}

func (r *Resolver) addMemory(ctx context.Context, botID string, msgs []memory.Message, namespace, scopeID string, mtc memoryTraceCtx) {
	start := time.Now()
	filters := map[string]any{
		"namespace": namespace,
		"scopeId":   scopeID,
		"bot_id":    botID,
	}
	result, err := r.memoryService.Add(ctx, memory.AddRequest{
		Messages: msgs,
		BotID:    botID,
		Filters:  filters,
	})
	durationMs := int(time.Since(start).Milliseconds())
	if err != nil {
		if r.logger != nil {
			r.logger.Warn("store memory failed",
				slog.String("namespace", namespace),
				slog.String("scope_id", scopeID),
				slog.Any("error", err),
			)
		}
		r.logProcessStep(ctx, botID, mtc.chatID, mtc.traceID, mtc.userID, mtc.channel,
			processlog.StepMemoryExtractFailed, processlog.LevelError, "Memory extraction failed: "+err.Error(),
			map[string]any{"error": err.Error()}, durationMs)
		return
	}

	if r.logger != nil {
		r.logger.Info("memory processed",
			slog.String("bot_id", botID),
			slog.String("namespace", namespace),
			slog.Int("messages_processed", len(msgs)),
			slog.Int("results_count", len(result.Results)),
		)
	}
	var extractedPreview string
	for _, item := range result.Results {
		if extractedPreview != "" {
			extractedPreview += " | "
		}
		extractedPreview += item.Memory
		if len(extractedPreview) > 500 {
			extractedPreview = extractedPreview[:500] + "..."
			break
		}
	}
	r.logProcessStep(ctx, botID, mtc.chatID, mtc.traceID, mtc.userID, mtc.channel,
		processlog.StepMemoryExtractCompleted, processlog.LevelInfo, "Memory extraction completed",
		map[string]any{
			"messages_processed": len(msgs),
			"results_count":     len(result.Results),
			"extracted_preview":  extractedPreview,
		}, durationMs)
}

func (r *Resolver) addSolutions(ctx context.Context, botID string, msgs []memory.Message, mtc memoryTraceCtx) {
	if r.memoryService == nil {
		return
	}
	start := time.Now()
	filters := map[string]any{
		"namespace": solutionsNamespace,
		"scopeId":   solutionsScopeID,
	}
	result, err := r.memoryService.Add(ctx, memory.AddRequest{
		Messages: msgs,
		Filters:  filters,
	})
	durationMs := int(time.Since(start).Milliseconds())
	if err != nil {
		if r.logger != nil {
			r.logger.Warn("addSolutions failed",
				slog.String("bot_id", botID),
				slog.Any("error", err),
			)
		}
		return
	}
	if r.logger != nil {
		r.logger.Info("solutions extracted",
			slog.String("bot_id", botID),
			slog.Int("results_count", len(result.Results)),
			slog.Int("duration_ms", durationMs),
		)
	}
}

// --- model failover ---

// tryFallback attempts to build a resolvedContext using the fallback model.
func (r *Resolver) tryFallback(ctx context.Context, primary resolvedContext) (resolvedContext, error) {
	if primary.model.FallbackModelID == "" {
		return resolvedContext{}, fmt.Errorf("no fallback model configured")
	}
	fbModel, err := r.modelsService.GetByID(ctx, primary.model.FallbackModelID)
	if err != nil {
		return resolvedContext{}, fmt.Errorf("failed to load fallback model: %w", err)
	}
	fbProvider, err := models.FetchProviderByID(ctx, r.queries, fbModel.LlmProviderID)
	if err != nil {
		return resolvedContext{}, fmt.Errorf("failed to load fallback provider: %w", err)
	}
	clientType, err := normalizeClientType(fbProvider.ClientType)
	if err != nil {
		return resolvedContext{}, err
	}
	fbPayload := primary.payload
	fbPayload.Model = gatewayModelConfig{
		ModelID:    fbModel.ModelID,
		ClientType: clientType,
		Input:      fbModel.Input,
		APIKey:     fbProvider.ApiKey,
		BaseURL:    fbProvider.BaseUrl,
		Reasoning:  fbModel.Reasoning,
		MaxTokens:  fbModel.MaxTokens,
	}
	return resolvedContext{payload: fbPayload, model: fbModel, provider: fbProvider, traceID: primary.traceID}, nil
}

// --- model selection ---

func (r *Resolver) selectChatModel(ctx context.Context, req conversation.ChatRequest, botSettings settings.Settings, cs conversation.Settings) (models.GetResponse, sqlc.LlmProvider, error) {
	if r.modelsService == nil {
		return models.GetResponse{}, sqlc.LlmProvider{}, fmt.Errorf("models service not configured")
	}
	modelID := strings.TrimSpace(req.Model)
	providerFilter := strings.TrimSpace(req.Provider)

	// For background tasks (heartbeat/schedule/subagent), prefer the cheaper background model.
	isBackground := req.TaskType == "heartbeat" || req.TaskType == "schedule" || req.TaskType == "subagent"
	if isBackground && modelID == "" && providerFilter == "" {
		if bgID := strings.TrimSpace(botSettings.BackgroundModelID); bgID != "" {
			modelID = bgID
		}
	}

	// Priority: request model > chat settings > bot settings.
	if modelID == "" && providerFilter == "" {
		if value := strings.TrimSpace(cs.ModelID); value != "" {
			modelID = value
		} else if value := strings.TrimSpace(botSettings.ChatModelID); value != "" {
			modelID = value
		}
	}

	if modelID == "" {
		return models.GetResponse{}, sqlc.LlmProvider{}, fmt.Errorf("chat model not configured: specify model in request or bot settings")
	}

	if providerFilter == "" {
		return r.fetchChatModel(ctx, modelID)
	}

	candidates, err := r.listCandidates(ctx, providerFilter)
	if err != nil {
		return models.GetResponse{}, sqlc.LlmProvider{}, err
	}
	for _, m := range candidates {
		if m.ModelID == modelID {
			prov, err := models.FetchProviderByID(ctx, r.queries, m.LlmProviderID)
			if err != nil {
				return models.GetResponse{}, sqlc.LlmProvider{}, err
			}
			return m, prov, nil
		}
	}
	return models.GetResponse{}, sqlc.LlmProvider{}, fmt.Errorf("chat model %q not found for provider %q", modelID, providerFilter)
}

func (r *Resolver) fetchChatModel(ctx context.Context, modelID string) (models.GetResponse, sqlc.LlmProvider, error) {
	model, err := r.modelsService.GetByModelID(ctx, modelID)
	if err != nil {
		return models.GetResponse{}, sqlc.LlmProvider{}, err
	}
	if model.Type != models.ModelTypeChat {
		return models.GetResponse{}, sqlc.LlmProvider{}, fmt.Errorf("model is not a chat model")
	}
	prov, err := models.FetchProviderByID(ctx, r.queries, model.LlmProviderID)
	if err != nil {
		return models.GetResponse{}, sqlc.LlmProvider{}, err
	}
	return model, prov, nil
}

func (r *Resolver) listCandidates(ctx context.Context, providerFilter string) ([]models.GetResponse, error) {
	var all []models.GetResponse
	var err error
	if providerFilter != "" {
		all, err = r.modelsService.ListByClientType(ctx, models.ClientType(providerFilter))
	} else {
		all, err = r.modelsService.ListByType(ctx, models.ModelTypeChat)
	}
	if err != nil {
		return nil, err
	}
	filtered := make([]models.GetResponse, 0, len(all))
	for _, m := range all {
		if m.Type == models.ModelTypeChat {
			filtered = append(filtered, m)
		}
	}
	return filtered, nil
}

// --- settings ---

func (r *Resolver) loadBotSettings(ctx context.Context, botID string) (settings.Settings, error) {
	if r.settingsService == nil {
		return settings.Settings{}, fmt.Errorf("settings service not configured")
	}
	return r.settingsService.GetBot(ctx, botID)
}

// --- utility ---

func mustNormalizeClientType(clientType string) string {
	ct, err := normalizeClientType(clientType)
	if err != nil {
		return "openai-compat"
	}
	return ct
}

func normalizeClientType(clientType string) (string, error) {
	ct := strings.ToLower(strings.TrimSpace(clientType))
	switch ct {
	case "openai", "openai-compat", "anthropic", "google",
		"azure", "bedrock", "mistral", "xai", "ollama", "dashscope",
		"deepseek", "zai-global", "zai-cn", "zai-coding-global", "zai-coding-cn",
		"minimax-global", "minimax-cn", "moonshot-global", "moonshot-cn",
		"volcengine", "volcengine-coding", "qianfan",
		"groq", "openrouter", "together", "fireworks", "perplexity":
		return ct, nil
	default:
		return "", fmt.Errorf("unsupported agent gateway client type: %s", clientType)
	}
}

func sanitizeMessages(messages []conversation.ModelMessage) []conversation.ModelMessage {
	supportedRoles := map[string]bool{
		"user": true, "assistant": true, "system": true, "tool": true,
	}
	cleaned := make([]conversation.ModelMessage, 0, len(messages))
	for _, msg := range messages {
		role := strings.TrimSpace(msg.Role)
		if role == "" {
			continue
		}
		// Drop messages with non-standard roles (e.g. item_reference from OpenAI Responses API).
		if !supportedRoles[role] {
			continue
		}
		if !msg.HasContent() && strings.TrimSpace(msg.ToolCallID) == "" && len(collectAssistantToolIDs(msg)) == 0 {
			continue
		}
		cleaned = append(cleaned, msg)
	}
	return cleaned
}

func dedup(items []string) []string {
	seen := make(map[string]struct{}, len(items))
	result := make([]string, 0, len(items))
	for _, s := range items {
		trimmed := strings.TrimSpace(s)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		result = append(result, trimmed)
	}
	return result
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func coalescePositiveInt(values ...int) int {
	for _, v := range values {
		if v > 0 {
			return v
		}
	}
	return defaultMaxContextMinutes
}

func nonNilStrings(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

func nonNilModelMessages(m []conversation.ModelMessage) []conversation.ModelMessage {
	if m == nil {
		return []conversation.ModelMessage{}
	}
	return m
}

func formatScoreDistribution(scores []float64) map[string]any {
	if len(scores) == 0 {
		return nil
	}
	minS, maxS, sum := scores[0], scores[0], 0.0
	for _, s := range scores {
		if s < minS {
			minS = s
		}
		if s > maxS {
			maxS = s
		}
		sum += s
	}
	return map[string]any{
		"count": len(scores),
		"min":   math.Round(minS*1000) / 1000,
		"max":   math.Round(maxS*1000) / 1000,
		"avg":   math.Round((sum/float64(len(scores)))*1000) / 1000,
	}
}

func truncateMemorySnippet(s string, n int) string {
	trimmed := strings.TrimSpace(s)
	if len(trimmed) <= n {
		return trimmed
	}
	return strings.TrimSpace(trimmed[:n]) + "..."
}

// truncate returns the first n characters of s, adding "..." if truncated.
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

func extractAssistantPreview(messages []conversation.ModelMessage, maxLen int) string {
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == "assistant" {
			text := messages[i].TextContent()
			if text != "" {
				return truncate(text, maxLen)
			}
		}
	}
	return ""
}

func parseResolverUUID(id string) (pgtype.UUID, error) {
	if strings.TrimSpace(id) == "" {
		return pgtype.UUID{}, fmt.Errorf("empty id")
	}
	return db.ParseUUID(id)
}

// isDirectConversationType returns true for DM/private conversations.
func isDirectConversationType(conversationType string) bool {
	ct := strings.ToLower(strings.TrimSpace(conversationType))
	return ct == "" || ct == "p2p" || ct == "private" || ct == "direct"
}

// limitHistoryTurns limits conversation history to the last N user turns (and their associated
// assistant responses). This reduces token usage for long-running sessions.
// Based on OpenClaw's implementation.
func limitHistoryTurns(messages []conversation.ModelMessage, limit int) []conversation.ModelMessage {
	if limit <= 0 || len(messages) == 0 {
		return messages
	}

	userCount := 0
	lastUserIndex := len(messages)

	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == "user" {
			userCount++
			if userCount > limit {
				return messages[lastUserIndex:]
			}
			lastUserIndex = i
		}
	}
	return messages
}

// pruneMessagesByTokenBudget trims oldest messages to fit within a token budget.
// Budget is 50% of the model's context window (the other 50% is reserved for
// system prompt, current turn, and generation).
// After pruning, orphaned tool_result messages (whose tool_use was dropped) are
// also removed to avoid API errors (Anthropic requires paired tool_use/tool_result).
const (
	toolResultContextShare = 0.3
	toolResultHardMaxChars = 400000
	toolResultHeadChars    = 1500
	toolResultTailChars    = 1500
	// toolResultMinKeepChars is the floor applied after computing maxChars from the
	// context window. It must be at least head+tail so we never set a budget lower
	// than what a single trimmed result already occupies.
	toolResultMinKeepChars = toolResultHeadChars + toolResultTailChars // 3000
)

type toolTrimDiag struct {
	ToolCallID    string `json:"tool_call_id"`
	OriginalChars int    `json:"original_chars"`
	TrimmedChars  int    `json:"trimmed_chars"`
}

// estimateCharsPerToken samples up to 500 runes of text to determine the CJK
// ratio and returns a weighted blend of 1.5 chars/token (CJK) and 3.5
// chars/token (Latin/other). This corrects the previous hardcoded 3.5 that
// caused CJK tool results to be over-truncated by up to 2.3x.
func estimateCharsPerToken(text string) float64 {
	runes := []rune(text)
	if len(runes) > 500 {
		runes = runes[:500]
	}
	if len(runes) == 0 {
		return 3.5
	}
	cjk := 0
	for _, r := range runes {
		if isCJK(r) {
			cjk++
		}
	}
	ratio := float64(cjk) / float64(len(runes))
	return 1.5*ratio + 3.5*(1-ratio)
}

// sampleAllToolResults concatenates the first 500 chars from every tool-role
// message so estimateCharsPerToken gets a representative content sample.
func sampleAllToolResults(messages []conversation.ModelMessage) string {
	const samplePerMsg = 500
	var sb strings.Builder
	for _, msg := range messages {
		if msg.Role != "tool" {
			continue
		}
		t := msg.TextContent()
		if len(t) > samplePerMsg {
			t = t[:samplePerMsg]
		}
		sb.WriteString(t)
		if sb.Len() >= 1500 {
			break
		}
	}
	return sb.String()
}

func softTrimToolResults(messages []conversation.ModelMessage, contextWindow int) ([]conversation.ModelMessage, []toolTrimDiag) {
	if contextWindow <= 0 {
		contextWindow = 128000
	}
	cpt := estimateCharsPerToken(sampleAllToolResults(messages))
	maxChars := int(float64(contextWindow) * toolResultContextShare * cpt)
	if maxChars > toolResultHardMaxChars {
		maxChars = toolResultHardMaxChars
	}
	if maxChars < toolResultMinKeepChars {
		maxChars = toolResultMinKeepChars
	}
	out := make([]conversation.ModelMessage, len(messages))
	copy(out, messages)
	var diags []toolTrimDiag
	for i, msg := range out {
		if msg.Role != "tool" {
			continue
		}
		text := msg.TextContent()
		if len(text) <= maxChars {
			continue
		}
		headN := toolResultHeadChars
		tailN := toolResultTailChars
		if headN+tailN >= len(text) {
			continue
		}
		head := text[:headN]
		tail := text[len(text)-tailN:]
		trimmed := head + fmt.Sprintf("\n\n[... content trimmed, original %d chars ...]\n\n", len(text)) + tail
		diags = append(diags, toolTrimDiag{
			ToolCallID:    msg.ToolCallID,
			OriginalChars: len(text),
			TrimmedChars:  len(trimmed),
		})
		out[i] = conversation.ModelMessage{
			Role:       msg.Role,
			Content:    conversation.NewTextContent(trimmed),
			ToolCallID: msg.ToolCallID,
			Name:       msg.Name,
		}
	}
	return out, diags
}

type pruneDiag struct {
	SystemTokens        int `json:"system_tokens"`
	GatewayEstimate     int `json:"gateway_estimate"`
	TotalSystemTokens   int `json:"total_system_tokens"`
	Budget              int `json:"budget"`
	EstimatedTotalBefore int `json:"estimated_total_before"`
	EstimatedTotalAfter  int `json:"estimated_total_after"`
	ProtectedTail       int `json:"protected_tail"`
	Pruned              bool `json:"pruned"`
}

// repruneWithLowerBudget re-runs context pruning with a custom budgetRatio
// (0.0–1.0) instead of the default 0.6. Used for context overflow recovery
// where we need a more aggressive cut to satisfy the model's context limit.
func repruneWithLowerBudget(messages []conversation.ModelMessage, contextWindow int, budgetRatio float64) []conversation.ModelMessage {
	if contextWindow <= 0 {
		contextWindow = 128000
	}
	if budgetRatio <= 0 || budgetRatio > 1 {
		budgetRatio = 0.4
	}

	messages, _ = softTrimToolResults(messages, contextWindow)

	systemTokens := 0
	for _, msg := range messages {
		if msg.Role == "system" {
			raw, err := json.Marshal(msg)
			if err == nil {
				systemTokens += estimateStringTokens(string(raw))
			}
		}
	}
	systemTokens = int(float64(systemTokens) * tokenSafetyMargin)
	agentGatewaySystemPromptEstimate := 2000
	totalSystemTokens := systemTokens + agentGatewaySystemPromptEstimate
	budget := int(float64(contextWindow-totalSystemTokens) * budgetRatio)
	if budget < 4096 {
		budget = 4096
	}

	if estimateTokens(messages) <= budget {
		return messages
	}

	protectedTail := 6
	if protectedTail > len(messages) {
		protectedTail = len(messages)
	}
	splitIdx := len(messages) - protectedTail
	protected := messages[splitIdx:]
	droppable := messages[:splitIdx]

	protectedTokens := estimateTokens(protected)
	if protectedTokens >= budget {
		for len(protected) > 1 && estimateTokens(protected) > budget {
			protected = protected[1:]
		}
		return repairToolPairing(protected)
	}

	remainBudget := budget - protectedTokens
	for len(droppable) > 0 && estimateTokens(droppable) > remainBudget {
		droppable = droppable[1:]
	}

	result := append(droppable, protected...)
	return repairToolPairing(result)
}

func pruneMessagesByTokenBudget(messages []conversation.ModelMessage, contextWindow int) ([]conversation.ModelMessage, []toolTrimDiag, pruneDiag) {
	if contextWindow <= 0 {
		contextWindow = 128000
	}

	messages, trimDiags := softTrimToolResults(messages, contextWindow)

	systemTokens := 0
	for _, msg := range messages {
		if msg.Role == "system" {
			raw, err := json.Marshal(msg)
			if err == nil {
				systemTokens += estimateStringTokens(string(raw))
			}
		}
	}
	systemTokens = int(float64(systemTokens) * tokenSafetyMargin)

	agentGatewaySystemPromptEstimate := 2000
	totalSystemTokens := systemTokens + agentGatewaySystemPromptEstimate
	budget := int(float64(contextWindow-totalSystemTokens) * 0.6)
	if budget < 4096 {
		budget = 4096
	}

	total := estimateTokens(messages)
	diag := pruneDiag{
		SystemTokens:         systemTokens,
		GatewayEstimate:      agentGatewaySystemPromptEstimate,
		TotalSystemTokens:    totalSystemTokens,
		Budget:               budget,
		EstimatedTotalBefore: total,
	}

	if total <= budget {
		diag.EstimatedTotalAfter = total
		return messages, trimDiags, diag
	}

	diag.Pruned = true

	protectedTail := 6
	if protectedTail > len(messages) {
		protectedTail = len(messages)
	}
	diag.ProtectedTail = protectedTail

	splitIdx := len(messages) - protectedTail
	droppable := messages[:splitIdx]
	protected := messages[splitIdx:]

	protectedTokens := estimateTokens(protected)
	if protectedTokens >= budget {
		for len(protected) > 1 && estimateTokens(protected) > budget {
			protected = protected[1:]
		}
		result := repairToolPairing(protected)
		diag.EstimatedTotalAfter = estimateTokens(result)
		return result, trimDiags, diag
	}

	remainBudget := budget - protectedTokens
	if len(droppable) > 0 {
		mid := len(droppable) / 2
		oldHalf := droppable[:mid]
		newHalf := droppable[mid:]

		oldTokens := estimateTokens(oldHalf)
		newTokens := estimateTokens(newHalf)

		if oldTokens+newTokens <= remainBudget {
			result := make([]conversation.ModelMessage, 0, len(messages))
			result = append(result, oldHalf...)
			result = append(result, newHalf...)
			result = append(result, protected...)
			result = repairToolPairing(result)
			diag.EstimatedTotalAfter = estimateTokens(result)
			return result, trimDiags, diag
		}

		for len(oldHalf) > 0 && estimateTokens(oldHalf)+newTokens > remainBudget {
			oldHalf = oldHalf[1:]
		}

		result := make([]conversation.ModelMessage, 0, len(oldHalf)+len(newHalf)+len(protected))
		result = append(result, oldHalf...)
		result = append(result, newHalf...)
		result = append(result, protected...)

		if estimateTokens(result) > budget {
			for len(result) > protectedTail+1 && estimateTokens(result) > budget {
				result = result[1:]
			}
		}
		result = repairToolPairing(result)
		diag.EstimatedTotalAfter = estimateTokens(result)
		return result, trimDiags, diag
	}

	result := repairToolPairing(protected)
	diag.EstimatedTotalAfter = estimateTokens(result)
	return result, trimDiags, diag
}

// estimateTokens gives a rough token count for a message list.
// Detects CJK content and uses ~1.5 chars/token for CJK, ~3.5 chars/token for Latin.
// Applies a 1.2x safety margin to avoid under-estimation.
func estimateTokens(messages []conversation.ModelMessage) int {
	total := 0
	for _, msg := range messages {
		raw, err := json.Marshal(msg)
		if err != nil {
			total += 100
			continue
		}
		total += estimateStringTokens(string(raw))
	}
	return int(float64(total) * tokenSafetyMargin)
}

const tokenSafetyMargin = 1.2

func estimateStringTokens(s string) int {
	cjk, other := 0, 0
	for _, r := range s {
		if isCJK(r) {
			cjk++
		} else {
			other++
		}
	}
	cjkTokens := float64(cjk) / 1.5
	otherTokens := float64(other) / 3.5
	return int(cjkTokens + otherTokens)
}

func isCJK(r rune) bool {
	return (r >= 0x4E00 && r <= 0x9FFF) ||
		(r >= 0x3400 && r <= 0x4DBF) ||
		(r >= 0x3000 && r <= 0x303F) ||
		(r >= 0xFF00 && r <= 0xFFEF) ||
		(r >= 0xAC00 && r <= 0xD7AF) ||
		(r >= 0x3040 && r <= 0x309F) ||
		(r >= 0x30A0 && r <= 0x30FF)
}

// repairToolPairing removes tool_result messages whose matching tool_use was
// dropped, and tool_use messages whose matching tool_result is missing.
// This prevents API errors with providers that require strict pairing.
// Handles both OpenAI format (top-level ToolCalls) and Vercel AI SDK format
// (tool-call parts inside content array).
func repairToolPairing(messages []conversation.ModelMessage) []conversation.ModelMessage {
	toolUseIDs := make(map[string]struct{})
	toolResultIDs := make(map[string]struct{})

	for _, msg := range messages {
		if msg.Role == "tool" && strings.TrimSpace(msg.ToolCallID) != "" {
			toolResultIDs[msg.ToolCallID] = struct{}{}
		}
		if msg.Role == "assistant" {
			for _, tc := range msg.ToolCalls {
				if strings.TrimSpace(tc.ID) != "" {
					toolUseIDs[tc.ID] = struct{}{}
				}
			}
			// Vercel AI SDK format: tool-call parts inside content array
			extractContentToolCallIDs(msg.Content, toolUseIDs)
		}
	}

	repaired := make([]conversation.ModelMessage, 0, len(messages))
	for _, msg := range messages {
		// Drop orphaned tool_result (no matching tool_use).
		if msg.Role == "tool" && strings.TrimSpace(msg.ToolCallID) != "" {
			if _, ok := toolUseIDs[msg.ToolCallID]; !ok {
				continue
			}
		}
		repaired = append(repaired, msg)
	}

	final := make([]conversation.ModelMessage, 0, len(repaired))
	for _, msg := range repaired {
		if msg.Role == "assistant" {
			ids := collectAssistantToolIDs(msg)
			if len(ids) > 0 {
				allOrphaned := true
				for _, id := range ids {
					if _, ok := toolResultIDs[id]; ok {
						allOrphaned = false
						break
					}
				}
				if allOrphaned && !msg.HasContent() {
					continue
				}
			}
		}
		final = append(final, msg)
	}

	return final
}

func extractContentToolCallIDs(content json.RawMessage, ids map[string]struct{}) {
	if len(content) == 0 {
		return
	}
	var parts []struct {
		Type       string `json:"type"`
		ToolCallID string `json:"toolCallId"`
	}
	if err := json.Unmarshal(content, &parts); err != nil {
		return
	}
	for _, p := range parts {
		if p.Type == "tool-call" && strings.TrimSpace(p.ToolCallID) != "" {
			ids[p.ToolCallID] = struct{}{}
		}
	}
}

func collectAssistantToolIDs(msg conversation.ModelMessage) []string {
	var ids []string
	for _, tc := range msg.ToolCalls {
		if strings.TrimSpace(tc.ID) != "" {
			ids = append(ids, tc.ID)
		}
	}
	if len(msg.Content) > 0 {
		var parts []struct {
			Type       string `json:"type"`
			ToolCallID string `json:"toolCallId"`
		}
		if err := json.Unmarshal(msg.Content, &parts); err == nil {
			for _, p := range parts {
				if p.Type == "tool-call" && strings.TrimSpace(p.ToolCallID) != "" {
					ids = append(ids, p.ToolCallID)
				}
			}
		}
	}
	return ids
}

// ── Context summarization ─────────────────────────────────────────────

// loadSummary fetches the stored conversation summary for a (bot, chat) pair.
// Returns empty string if no summary exists or on error.
func (r *Resolver) loadSummary(ctx context.Context, botID, chatID string) string {
	if r.queries == nil {
		return ""
	}
	pgBotID, err := db.ParseUUID(botID)
	if err != nil {
		r.logger.Debug("loadSummary: UUID parse failed", slog.String("bot_id", botID), slog.Any("error", err))
		return ""
	}
	row, err := r.queries.GetConversationSummary(ctx, sqlc.GetConversationSummaryParams{
		BotID:  pgBotID,
		ChatID: chatID,
	})
	if err != nil {
		r.logger.Debug("loadSummary: query failed", slog.String("bot_id", botID), slog.Any("error", err))
		return ""
	}
	summary := strings.TrimSpace(row.Summary)
	if summary != "" {
		r.logger.Info("loadSummary: found",
			slog.String("bot_id", botID),
			slog.Int("length", len(summary)),
		)
	}
	return summary
}

// asyncSummarize sends dropped messages to the Agent Gateway /chat/summarize
// endpoint in a background goroutine, then upserts the result into DB.
func (r *Resolver) asyncSummarize(
	botID, chatID string,
	dropped []conversation.ModelMessage,
	chatModel models.GetResponse,
	provider sqlc.LlmProvider,
	token string,
) {
	if len(dropped) == 0 {
		return
	}
	droppedCopy := make([]conversation.ModelMessage, len(dropped))
	copy(droppedCopy, dropped)
	droppedCount := int32(len(droppedCopy))

	existingSummary := r.loadSummary(context.Background(), botID, chatID)

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()

		clientType, err := normalizeClientType(provider.ClientType)
		if err != nil {
			r.logger.Warn("summarize: invalid client type", slog.Any("error", err))
			return
		}

		msgsToSummarize := droppedCopy
		if strings.TrimSpace(existingSummary) != "" {
			summaryMsg := conversation.ModelMessage{
				Role:    "system",
				Content: conversation.NewTextContent("Previous conversation summary:\n" + existingSummary),
			}
			msgsToSummarize = append([]conversation.ModelMessage{summaryMsg}, droppedCopy...)
		}

		summary, usage, err := r.postSummarize(ctx, gatewayModelConfig{
			ModelID:    chatModel.ModelID,
			ClientType: clientType,
			Input:      chatModel.Input,
			APIKey:     provider.ApiKey,
			BaseURL:    provider.BaseUrl,
			Reasoning:  chatModel.Reasoning,
			MaxTokens:  chatModel.MaxTokens,
		}, msgsToSummarize, token)
		if err != nil {
			r.logger.Warn("summarize request failed", slog.String("bot_id", botID), slog.Any("error", err))
			return
		}
		r.recordTokenUsage(ctx, botID, usage, chatModel.ModelID, "summarize")
		if strings.TrimSpace(summary) == "" {
			return
		}
		pgBotID, err := db.ParseUUID(botID)
		if err != nil {
			return
		}
		if _, err := r.queries.UpsertConversationSummary(ctx, sqlc.UpsertConversationSummaryParams{
			BotID:        pgBotID,
			ChatID:       chatID,
			Summary:      summary,
			MessageCount: droppedCount,
		}); err != nil {
			r.logger.Warn("upsert summary failed", slog.String("bot_id", botID), slog.Any("error", err))
		}
	}()
}

type summarizeRequest struct {
	Model    gatewayModelConfig          `json:"model"`
	Messages []conversation.ModelMessage `json:"messages"`
}

type summarizeResponse struct {
	Summary string        `json:"summary"`
	Usage   *gatewayUsage `json:"usage,omitempty"`
}

// postSummarize calls the Agent Gateway /chat/summarize endpoint.
func (r *Resolver) postSummarize(ctx context.Context, model gatewayModelConfig, messages []conversation.ModelMessage, token string) (string, *gatewayUsage, error) {
	r.logger.Info("postSummarize: calling gateway", slog.Int("message_count", len(messages)), slog.String("model", model.ModelID))
	payload := summarizeRequest{Model: model, Messages: messages}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", nil, err
	}
	url := r.gatewayBaseURL + "/chat/summarize"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if strings.TrimSpace(token) != "" {
		httpReq.Header.Set("Authorization", token)
	}
	resp, err := r.httpClient.Do(httpReq)
	if err != nil {
		r.logger.Warn("postSummarize: http request failed", slog.Any("error", err))
		return "", nil, err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		r.logger.Warn("postSummarize: gateway error", slog.Int("status", resp.StatusCode))
		return "", nil, fmt.Errorf("summarize gateway error %d: %s", resp.StatusCode, truncate(string(respBody), 200))
	}
	var parsed summarizeResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		r.logger.Warn("postSummarize: parse failed", slog.Any("error", err))
		return "", nil, fmt.Errorf("parse summarize response: %w", err)
	}
	r.logger.Info("postSummarize: completed", slog.Int("summary_length", len(parsed.Summary)))
	return parsed.Summary, parsed.Usage, nil
}

// --- Token Usage ---

// recordTokenUsage persists token usage asynchronously so it never blocks the response.
func (r *Resolver) recordTokenUsage(ctx context.Context, botID string, usage *gatewayUsage, model, source string) {
	if usage == nil || usage.TotalTokens == 0 {
		return
	}
	go func() {
		bgCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		botUUID, err := db.ParseUUID(botID)
		if err != nil {
			return
		}
		if _, err := r.queries.RecordTokenUsage(bgCtx, sqlc.RecordTokenUsageParams{
			BotID:            botUUID,
			PromptTokens:     int32(usage.PromptTokens),
			CompletionTokens: int32(usage.CompletionTokens),
			TotalTokens:      int32(usage.TotalTokens),
			Model:            model,
			Source:           source,
		}); err != nil {
			r.logger.Warn("record token usage failed", slog.String("bot_id", botID), slog.Any("error", err))
		}
	}()
}

// toTokenUsage converts internal usage to the public type.
func toTokenUsage(u *gatewayUsage) *conversation.TokenUsage {
	if u == nil {
		return nil
	}
	return &conversation.TokenUsage{
		PromptTokens:     u.PromptTokens,
		CompletionTokens: u.CompletionTokens,
		TotalTokens:      u.TotalTokens,
	}
}

// --- skill context filtering ---

// filterSkillsByRelevance returns the top-N most relevant skills based on
// keyword overlap between the user query and skill name+description.
// If the total count is <= maxCandidates, all skills are returned (no filtering).
// Skills with metadata["enabled"]=true are always kept.
func filterSkillsByRelevance(skills []gatewaySkill, query string, maxCandidates int) []gatewaySkill {
	if len(skills) <= maxCandidates || query == "" {
		return skills
	}

	queryTokens := tokenize(query)
	if len(queryTokens) == 0 {
		return skills
	}

	type scored struct {
		idx   int
		score float64
		keep  bool // enabled skills are always kept
	}

	items := make([]scored, len(skills))
	for i, sk := range skills {
		enabled := false
		if v, ok := sk.Metadata["enabled"]; ok {
			if b, ok2 := v.(bool); ok2 && b {
				enabled = true
			}
		}
		text := sk.Name + " " + sk.Description
		skillTokens := tokenize(text)
		items[i] = scored{idx: i, score: jaccardSimilarity(queryTokens, skillTokens), keep: enabled}
	}

	// Sort by: keep first, then score descending
	sort.Slice(items, func(a, b int) bool {
		if items[a].keep != items[b].keep {
			return items[a].keep
		}
		return items[a].score > items[b].score
	})

	result := make([]gatewaySkill, 0, maxCandidates)
	for _, it := range items {
		if len(result) >= maxCandidates && !it.keep {
			break
		}
		result = append(result, skills[it.idx])
	}
	return result
}

func tokenize(s string) map[string]struct{} {
	s = strings.ToLower(s)
	tokens := make(map[string]struct{})
	start := -1
	for i, r := range s {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r >= 0x4e00 && r <= 0x9fff {
			if start < 0 {
				start = i
			}
		} else {
			if start >= 0 {
				tokens[s[start:i]] = struct{}{}
				start = -1
			}
		}
	}
	if start >= 0 {
		tokens[s[start:]] = struct{}{}
	}
	return tokens
}

func jaccardSimilarity(a, b map[string]struct{}) float64 {
	if len(a) == 0 || len(b) == 0 {
		return 0
	}
	intersection := 0
	for k := range a {
		if _, ok := b[k]; ok {
			intersection++
		}
	}
	union := len(a) + len(b) - intersection
	if union == 0 {
		return 0
	}
	return float64(intersection) / float64(union)
}
