package imagegen

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"google.golang.org/genai"

	"github.com/Kxiandaoyan/Memoh-v2/internal/channel"
	"github.com/Kxiandaoyan/Memoh-v2/internal/db/sqlc"
	mcpgw "github.com/Kxiandaoyan/Memoh-v2/internal/mcp"
	"github.com/Kxiandaoyan/Memoh-v2/internal/models"
	"github.com/Kxiandaoyan/Memoh-v2/internal/providers"
	"github.com/Kxiandaoyan/Memoh-v2/internal/settings"
)

const (
	toolGenerateImage = "generate_image"
	fallbackModel     = "gemini-2.0-flash-preview-image-generation"
	generateTimeout   = 120 * time.Second
)

type Executor struct {
	logger         *slog.Logger
	settings       *settings.Service
	models         *models.Service
	queries        *sqlc.Queries
	channelManager *channel.Manager
	dataRoot       string

	wg     sync.WaitGroup
	ctx    context.Context
	cancel context.CancelFunc
}

func NewExecutor(
	log *slog.Logger,
	settingsSvc *settings.Service,
	modelsSvc *models.Service,
	queries *sqlc.Queries,
	channelMgr *channel.Manager,
	dataRoot string,
) *Executor {
	if log == nil {
		log = slog.Default()
	}
	ctx, cancel := context.WithCancel(context.Background())
	return &Executor{
		logger:         log.With(slog.String("provider", "imagegen")),
		settings:       settingsSvc,
		models:         modelsSvc,
		queries:        queries,
		channelManager: channelMgr,
		dataRoot:       dataRoot,
		ctx:            ctx,
		cancel:         cancel,
	}
}

func (e *Executor) Stop() {
	e.cancel()
	e.wg.Wait()
}

func (e *Executor) ListTools(_ context.Context, _ mcpgw.ToolSessionContext) ([]mcpgw.ToolDescriptor, error) {
	return []mcpgw.ToolDescriptor{
		{
			Name:        toolGenerateImage,
			Description: "Generate an image from a text prompt using an AI model. The generated image will be sent to the user automatically when ready. Returns immediately with a status message.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"prompt": map[string]any{
						"type":        "string",
						"description": "A detailed description of the image to generate",
					},
					"size": map[string]any{
						"type":        "string",
						"enum":        []string{"1K", "2K", "4K"},
						"description": "Output resolution. Default 1K.",
					},
				},
				"required": []string{"prompt"},
			},
		},
	}, nil
}

func (e *Executor) CallTool(ctx context.Context, session mcpgw.ToolSessionContext, toolName string, arguments map[string]any) (map[string]any, error) {
	if toolName != toolGenerateImage {
		return nil, mcpgw.ErrToolNotFound
	}

	prompt := strings.TrimSpace(mcpgw.StringArg(arguments, "prompt"))
	if prompt == "" {
		return mcpgw.BuildToolErrorResult("prompt is required"), nil
	}
	size := strings.TrimSpace(mcpgw.StringArg(arguments, "size"))
	if size == "" {
		size = "1K"
	}

	botID := strings.TrimSpace(session.BotID)
	if botID == "" {
		return mcpgw.BuildToolErrorResult("bot_id is required"), nil
	}
	platform := strings.TrimSpace(session.CurrentPlatform)
	target := strings.TrimSpace(session.ReplyTarget)
	if platform == "" || target == "" {
		return mcpgw.BuildToolErrorResult("channel context is required (platform and target)"), nil
	}

	modelName, apiKey, baseURL, err := e.resolveImageModel(ctx, botID)
	if err != nil {
		return mcpgw.BuildToolErrorResult(fmt.Sprintf("cannot resolve image model: %v", err)), nil
	}

	e.wg.Add(1)
	go func() {
		defer e.wg.Done()
		bgCtx, bgCancel := context.WithTimeout(e.ctx, generateTimeout)
		defer bgCancel()
		e.generateAndSend(bgCtx, generateRequest{
			modelName: modelName,
			apiKey:    apiKey,
			baseURL:   baseURL,
			botID:     botID,
			prompt:    prompt,
			size:      size,
			platform:  platform,
			target:    target,
		})
	}()

	return mcpgw.BuildToolSuccessResult(map[string]any{
		"status":  "generating",
		"message": "Image generation started. The image will be sent to the user automatically when ready (typically 5-30 seconds).",
	}), nil
}

type generateRequest struct {
	modelName string
	apiKey    string
	baseURL   string
	botID     string
	prompt    string
	size      string
	platform  string
	target    string
}

func (e *Executor) generateAndSend(ctx context.Context, req generateRequest) {
	imageBytes, err := e.callImageAPI(ctx, req)
	if err != nil {
		e.logger.Error("image generation failed",
			slog.String("bot_id", req.botID),
			slog.String("prompt_prefix", truncate(req.prompt, 60)),
			slog.Any("error", err))
		e.sendErrorNotification(ctx, req, err)
		return
	}

	filename := fmt.Sprintf("gen_%d_%s.png", time.Now().UnixMilli(), randomHex(4))
	mediaDir := filepath.Join(e.dataRoot, "bots", req.botID, "media")
	if mkErr := os.MkdirAll(mediaDir, 0o755); mkErr != nil {
		e.logger.Error("failed to create media dir", slog.Any("error", mkErr))
	}
	fullPath := filepath.Join(mediaDir, filename)
	if writeErr := os.WriteFile(fullPath, imageBytes, 0o644); writeErr != nil {
		e.logger.Error("failed to save generated image", slog.Any("error", writeErr))
	}

	sendErr := e.channelManager.Send(ctx, req.botID, channel.ChannelType(req.platform), channel.SendRequest{
		Target: req.target,
		Message: channel.Message{
			Attachments: []channel.Attachment{{
				Type: channel.AttachmentImage,
				Data: imageBytes,
				Name: filename,
				Mime: "image/png",
			}},
		},
	})
	if sendErr != nil {
		e.logger.Error("failed to send generated image",
			slog.String("bot_id", req.botID),
			slog.Any("error", sendErr))
	} else {
		e.logger.Info("image generated and sent",
			slog.String("bot_id", req.botID),
			slog.String("file", filename),
			slog.String("platform", req.platform))
	}
}

// callImageAPI routes to either the OpenAI-compatible API (when baseURL is set,
// e.g. OpenRouter) or the native Gemini SDK (when using the Gemini API directly).
func (e *Executor) callImageAPI(ctx context.Context, req generateRequest) ([]byte, error) {
	if strings.TrimSpace(req.baseURL) != "" {
		return callOpenAICompatibleImageAPI(ctx, req)
	}
	return callNativeGeminiAPI(ctx, req)
}

// ---------------------------------------------------------------------------
// Path A: OpenAI-compatible API (OpenRouter, custom proxies)
// ---------------------------------------------------------------------------

type openAIChatRequest struct {
	Model    string           `json:"model"`
	Messages []openAIMessage  `json:"messages"`
}

type openAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIChatResponse struct {
	Choices []struct {
		Message struct {
			// Content can be a plain string or an array of content parts.
			Content json.RawMessage `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
		Code    any    `json:"code"`
	} `json:"error,omitempty"`
}

type contentPart struct {
	Type     string    `json:"type"`
	Text     string    `json:"text,omitempty"`
	ImageURL *imageURL `json:"image_url,omitempty"`
}

type imageURL struct {
	URL string `json:"url"`
}

func callOpenAICompatibleImageAPI(ctx context.Context, req generateRequest) ([]byte, error) {
	baseURL := strings.TrimRight(req.baseURL, "/")
	endpoint := baseURL + "/chat/completions"

	payload := openAIChatRequest{
		Model: req.modelName,
		Messages: []openAIMessage{
			{Role: "user", Content: req.prompt},
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to build request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+req.apiKey)

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		preview := truncate(string(respBody), 300)
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, preview)
	}

	var chatResp openAIChatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	if chatResp.Error != nil {
		return nil, fmt.Errorf("API error: %s", chatResp.Error.Message)
	}
	if len(chatResp.Choices) == 0 {
		return nil, fmt.Errorf("API returned no choices")
	}

	raw := chatResp.Choices[0].Message.Content

	// Try array of content parts first (multimodal response).
	var parts []contentPart
	if err := json.Unmarshal(raw, &parts); err == nil {
		for _, part := range parts {
			if part.Type == "image_url" && part.ImageURL != nil {
				return decodeDataURI(part.ImageURL.URL)
			}
		}
	}

	// Fallback: plain string — check if it's a data URI itself.
	var text string
	if err := json.Unmarshal(raw, &text); err == nil {
		if strings.HasPrefix(text, "data:image/") {
			return decodeDataURI(text)
		}
	}

	return nil, fmt.Errorf("API response contained no image data (model: %s)", req.modelName)
}

// decodeDataURI extracts raw bytes from a data URI like "data:image/png;base64,..."
// or returns the URL directly if it's a plain https URL (caller should fetch it).
func decodeDataURI(uri string) ([]byte, error) {
	if !strings.HasPrefix(uri, "data:") {
		return nil, fmt.Errorf("unsupported image URL format (expected data URI, got: %s)", truncate(uri, 80))
	}
	// Format: data:<mime>;base64,<data>
	comma := strings.Index(uri, ",")
	if comma < 0 {
		return nil, fmt.Errorf("malformed data URI")
	}
	encoded := uri[comma+1:]
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		// Some providers omit padding — try RawStdEncoding.
		decoded, err = base64.RawStdEncoding.DecodeString(encoded)
		if err != nil {
			return nil, fmt.Errorf("failed to decode base64 image: %w", err)
		}
	}
	return decoded, nil
}

// ---------------------------------------------------------------------------
// Path B: Native Gemini SDK (direct Gemini API, no proxy)
// ---------------------------------------------------------------------------

func callNativeGeminiAPI(ctx context.Context, req generateRequest) ([]byte, error) {
	cc := &genai.ClientConfig{
		APIKey:  req.apiKey,
		Backend: genai.BackendGeminiAPI,
	}
	client, err := genai.NewClient(ctx, cc)
	if err != nil {
		return nil, fmt.Errorf("failed to create Gemini client: %w", err)
	}

	config := &genai.GenerateContentConfig{
		ResponseModalities: []string{"TEXT", "IMAGE"},
	}

	result, err := client.Models.GenerateContent(ctx, req.modelName, genai.Text(req.prompt), config)
	if err != nil {
		return nil, fmt.Errorf("Gemini API call failed: %w", err)
	}
	if result == nil || len(result.Candidates) == 0 {
		return nil, fmt.Errorf("Gemini returned no candidates")
	}

	for _, part := range result.Candidates[0].Content.Parts {
		if part.InlineData != nil && strings.HasPrefix(part.InlineData.MIMEType, "image/") {
			return part.InlineData.Data, nil
		}
	}
	return nil, fmt.Errorf("Gemini response contained no image data")
}

// ---------------------------------------------------------------------------
// resolveImageModel
// ---------------------------------------------------------------------------

// resolveImageModel returns (modelName, apiKey, baseURL) for image generation.
// Priority: bot's image_model_id setting -> chat model (Google provider only) with fallback model name.
func (e *Executor) resolveImageModel(ctx context.Context, botID string) (string, string, string, error) {
	botSettings, err := e.settings.GetBot(ctx, botID)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to get bot settings: %w", err)
	}

	// If a dedicated image model is configured, use it directly.
	if imageModelID := strings.TrimSpace(botSettings.ImageModelID); imageModelID != "" {
		model, err := e.models.GetByModelID(ctx, imageModelID)
		if err != nil {
			return "", "", "", fmt.Errorf("failed to get image model %q: %w", imageModelID, err)
		}
		provider, err := models.FetchProviderByID(ctx, e.queries, model.LlmProviderID)
		if err != nil {
			return "", "", "", fmt.Errorf("failed to get image model provider: %w", err)
		}
		apiKey := strings.TrimSpace(provider.ApiKey)
		if apiKey == "" {
			return "", "", "", fmt.Errorf("image model provider has no API key configured")
		}
		return model.ModelID, apiKey, strings.TrimSpace(provider.BaseUrl), nil
	}

	// Fallback: use the chat model's provider credentials with the default Gemini model name.
	chatModelID := strings.TrimSpace(botSettings.ChatModelID)
	if chatModelID == "" {
		return "", "", "", fmt.Errorf("bot has no image model or chat model configured")
	}
	model, err := e.models.GetByModelID(ctx, chatModelID)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to get model %q: %w", chatModelID, err)
	}
	provider, err := models.FetchProviderByID(ctx, e.queries, model.LlmProviderID)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to get provider: %w", err)
	}
	if provider.ClientType != string(providers.ClientTypeGoogle) {
		return "", "", "", fmt.Errorf("image generation requires a Google/Gemini provider or a dedicated Image Model in bot settings (current chat provider: %s)", provider.ClientType)
	}
	apiKey := strings.TrimSpace(provider.ApiKey)
	if apiKey == "" {
		return "", "", "", fmt.Errorf("Gemini provider has no API key configured")
	}
	return fallbackModel, apiKey, "", nil
}

func (e *Executor) sendErrorNotification(ctx context.Context, req generateRequest, genErr error) {
	msg := fmt.Sprintf("Image generation failed: %v", genErr)
	sendErr := e.channelManager.Send(ctx, req.botID, channel.ChannelType(req.platform), channel.SendRequest{
		Target: req.target,
		Message: channel.Message{
			Text: msg,
		},
	})
	if sendErr != nil {
		e.logger.Error("failed to send error notification", slog.Any("error", sendErr))
	}
}

func randomHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
