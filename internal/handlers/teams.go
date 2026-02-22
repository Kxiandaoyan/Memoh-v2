package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/labstack/echo/v4"

	"github.com/Kxiandaoyan/Memoh-v2/internal/accounts"
	"github.com/Kxiandaoyan/Memoh-v2/internal/bots"
	"github.com/Kxiandaoyan/Memoh-v2/internal/db"
	"github.com/Kxiandaoyan/Memoh-v2/internal/db/sqlc"
	"github.com/Kxiandaoyan/Memoh-v2/internal/templates"
)

// TeamsHandler manages agent team CRUD and cross-bot relationships.
type TeamsHandler struct {
	queries        *sqlc.Queries
	botService     *bots.Service
	accountService *accounts.Service
	logger         *slog.Logger
}

// NewTeamsHandler creates a new TeamsHandler.
func NewTeamsHandler(log *slog.Logger, queries *sqlc.Queries, botService *bots.Service, accountService *accounts.Service) *TeamsHandler {
	if log == nil {
		log = slog.Default()
	}
	return &TeamsHandler{
		queries:        queries,
		botService:     botService,
		accountService: accountService,
		logger:         log.With(slog.String("handler", "teams")),
	}
}

// Register mounts team routes on the Echo instance.
func (h *TeamsHandler) Register(e *echo.Echo) {
	g := e.Group("/teams")
	g.POST("", h.CreateTeam)
	g.GET("", h.ListTeams)
	g.GET("/:team_id", h.GetTeam)
	g.DELETE("/:team_id", h.DeleteTeam)
	g.POST("/:team_id/members", h.AddTeamMember)
	g.DELETE("/:team_id/members/:member_id", h.RemoveTeamMember)
	g.GET("/:team_id/members", h.ListTeamMembers)

	e.GET("/bots/:bot_id/team-context", h.GetBotTeamContext)
	e.GET("/bots/:bot_id/call-logs", h.ListBotCallLogs)
}

// ------------------------------------------------------------------
// Request / Response types
// ------------------------------------------------------------------

// TeamMemberInput describes one member bot and its role in a team.
type TeamMemberInput struct {
	BotID           string `json:"bot_id"`
	RoleDescription string `json:"role_description"`
}

// CreateTeamRequest is the body for POST /teams.
type CreateTeamRequest struct {
	Name            string            `json:"name"`
	Members         []TeamMemberInput `json:"members"`
	HeartbeatPrompt string            `json:"heartbeat_prompt"`
}

// TeamResponse is the API representation of a bot_team row.
type TeamResponse struct {
	ID           string    `json:"id"`
	OwnerUserID  string    `json:"owner_user_id"`
	Name         string    `json:"name"`
	ManagerBotID string    `json:"manager_bot_id"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// TeamMemberResponse is the API representation of a bot_team_members row.
type TeamMemberResponse struct {
	ID                string    `json:"id"`
	TeamID            string    `json:"team_id"`
	SourceBotID       string    `json:"source_bot_id"`
	TargetBotID       string    `json:"target_bot_id"`
	RoleDescription   string    `json:"role_description"`
	TargetDisplayName string    `json:"target_display_name"`
	CreatedAt         time.Time `json:"created_at"`
}

// AddMemberRequest is the body for POST /teams/:team_id/members.
type AddMemberRequest struct {
	SourceBotID     string `json:"source_bot_id"`
	TargetBotID     string `json:"target_bot_id"`
	RoleDescription string `json:"role_description"`
}

// ------------------------------------------------------------------
// Handlers
// ------------------------------------------------------------------

// CreateTeam godoc
// @Summary Create an agent team
// @Description Create a new team. A manager bot is auto-generated using the solo-company template.
// @Tags teams
// @Param payload body CreateTeamRequest true "Team payload"
// @Success 201 {object} TeamResponse
// @Failure 400 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /teams [post]
func (h *TeamsHandler) CreateTeam(c echo.Context) error {
	userID, err := RequireChannelIdentityID(c)
	if err != nil {
		return err
	}

	var req CreateTeamRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "team name is required")
	}
	if len(req.Members) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "at least one team member is required")
	}

	ctx := c.Request().Context()

	// Validate all member bots belong to the current user.
	for _, m := range req.Members {
		bot, err := h.botService.Get(ctx, m.BotID)
		if err != nil {
			if isNotFound(err) {
				return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("bot %s not found", m.BotID))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
		if bot.OwnerUserID != userID {
			return echo.NewHTTPError(http.StatusForbidden, fmt.Sprintf("bot %s does not belong to you", m.BotID))
		}
	}

	// Load solo-company template as the base for the manager bot.
	tmpl, tmplErr := templates.Get("solo-company")
	identity := ""
	soul := ""
	task := ""
	if tmplErr == nil {
		identity = tmpl.Identity
		soul = tmpl.Soul
		task = tmpl.Task
	}

	// Append team routing table to the task prompt.
	task = appendTeamRoutingTable(task, req.Name, req.Members)

	// Create the manager bot.
	managerMetadata := map[string]any{
		"team_role": "manager",
		"team_name": req.Name,
	}
	managerBot, err := h.botService.Create(ctx, userID, bots.CreateBotRequest{
		Type:        "personal",
		DisplayName: req.Name + " 总管",
		Metadata:    managerMetadata,
	})
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to create manager bot: "+err.Error())
	}

	// Set manager bot prompts from template + team routing table.
	if identity != "" || soul != "" || task != "" {
		if _, err := h.botService.UpdatePrompts(ctx, managerBot.ID, bots.UpdatePromptsRequest{
			Identity: &identity,
			Soul:     &soul,
			Task:     &task,
		}); err != nil {
			h.logger.Warn("failed to set manager bot prompts", slog.String("bot_id", managerBot.ID), slog.Any("error", err))
		}
	}

	// Parse UUIDs for DB operations.
	ownerUUID, err := db.ParseUUID(userID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	managerUUID, err := db.ParseUUID(managerBot.ID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// Create bot_teams record.
	team, err := h.queries.CreateTeam(ctx, sqlc.CreateTeamParams{
		OwnerUserID:  ownerUUID,
		Name:         req.Name,
		ManagerBotID: managerUUID,
	})
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to create team record: "+err.Error())
	}

	// Create directed edges: manager → each member.
	teamUUID := team.ID
	for _, m := range req.Members {
		targetUUID, err := db.ParseUUID(m.BotID)
		if err != nil {
			continue
		}
		roleDesc := strings.TrimSpace(m.RoleDescription)
		if _, addErr := h.queries.AddTeamMember(ctx, sqlc.AddTeamMemberParams{
			TeamID:          teamUUID,
			SourceBotID:     managerUUID,
			TargetBotID:     targetUUID,
			RoleDescription: roleDesc,
		}); addErr != nil {
			h.logger.Warn("failed to add team member edge", slog.String("target", m.BotID), slog.Any("error", addErr))
		}
	}

	// Seed heartbeat config for the manager bot.
	heartbeatPrompt := strings.TrimSpace(req.HeartbeatPrompt)
	if heartbeatPrompt == "" {
		heartbeatPrompt = buildDefaultManagerHeartbeatPrompt(req.Name)
	}
	h.seedManagerHeartbeat(ctx, managerBot.ID, heartbeatPrompt)

	return c.JSON(http.StatusCreated, toTeamResponse(team))
}

// ListTeams godoc
// @Summary List teams
// @Description List all teams owned by the current user
// @Tags teams
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} ErrorResponse
// @Router /teams [get]
func (h *TeamsHandler) ListTeams(c echo.Context) error {
	userID, err := RequireChannelIdentityID(c)
	if err != nil {
		return err
	}
	ownerUUID, err := db.ParseUUID(userID)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	rows, err := h.queries.ListTeamsByOwner(c.Request().Context(), ownerUUID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	items := make([]TeamResponse, 0, len(rows))
	for _, r := range rows {
		items = append(items, toTeamResponse(r))
	}
	return c.JSON(http.StatusOK, map[string]any{"items": items})
}

// GetTeam godoc
// @Summary Get a team
// @Description Get a team by ID
// @Tags teams
// @Param team_id path string true "Team ID"
// @Success 200 {object} TeamResponse
// @Failure 400 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /teams/{team_id} [get]
func (h *TeamsHandler) GetTeam(c echo.Context) error {
	userID, err := RequireChannelIdentityID(c)
	if err != nil {
		return err
	}
	team, err := h.requireTeamOwner(c, userID)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, toTeamResponse(team))
}

// DeleteTeam godoc
// @Summary Delete a team
// @Description Delete a team. The manager bot becomes independent; member bots are unaffected.
// @Tags teams
// @Param team_id path string true "Team ID"
// @Success 204 "No Content"
// @Failure 400 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /teams/{team_id} [delete]
func (h *TeamsHandler) DeleteTeam(c echo.Context) error {
	userID, err := RequireChannelIdentityID(c)
	if err != nil {
		return err
	}
	team, err := h.requireTeamOwner(c, userID)
	if err != nil {
		return err
	}
	if err := h.queries.DeleteTeam(c.Request().Context(), team.ID); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.NoContent(http.StatusNoContent)
}

// AddTeamMember godoc
// @Summary Add a directed call edge to a team
// @Description Allow source_bot to call target_bot within this team
// @Tags teams
// @Param team_id path string true "Team ID"
// @Param payload body AddMemberRequest true "Member edge payload"
// @Success 201 {object} TeamMemberResponse
// @Failure 400 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Router /teams/{team_id}/members [post]
func (h *TeamsHandler) AddTeamMember(c echo.Context) error {
	userID, err := RequireChannelIdentityID(c)
	if err != nil {
		return err
	}
	team, err := h.requireTeamOwner(c, userID)
	if err != nil {
		return err
	}

	var req AddMemberRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	if strings.TrimSpace(req.SourceBotID) == "" || strings.TrimSpace(req.TargetBotID) == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "source_bot_id and target_bot_id are required")
	}

	ctx := c.Request().Context()

	// Both bots must belong to this user.
	for _, bid := range []string{req.SourceBotID, req.TargetBotID} {
		bot, err := h.botService.Get(ctx, bid)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("bot %s not found", bid))
		}
		if bot.OwnerUserID != userID {
			return echo.NewHTTPError(http.StatusForbidden, fmt.Sprintf("bot %s does not belong to you", bid))
		}
	}

	srcUUID, err := db.ParseUUID(req.SourceBotID)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	dstUUID, err := db.ParseUUID(req.TargetBotID)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	row, err := h.queries.AddTeamMember(ctx, sqlc.AddTeamMemberParams{
		TeamID:          team.ID,
		SourceBotID:     srcUUID,
		TargetBotID:     dstUUID,
		RoleDescription: strings.TrimSpace(req.RoleDescription),
	})
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusCreated, toTeamMemberResponse(row, ""))
}

// RemoveTeamMember godoc
// @Summary Remove a call edge from a team
// @Tags teams
// @Param team_id path string true "Team ID"
// @Param member_id path string true "Member edge ID"
// @Success 204 "No Content"
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /teams/{team_id}/members/{member_id} [delete]
func (h *TeamsHandler) RemoveTeamMember(c echo.Context) error {
	userID, err := RequireChannelIdentityID(c)
	if err != nil {
		return err
	}
	if _, err := h.requireTeamOwner(c, userID); err != nil {
		return err
	}
	memberID := strings.TrimSpace(c.Param("member_id"))
	if memberID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "member_id is required")
	}
	memberUUID, err := db.ParseUUID(memberID)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	if err := h.queries.RemoveTeamMember(c.Request().Context(), memberUUID); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.NoContent(http.StatusNoContent)
}

// ListTeamMembers godoc
// @Summary List team call edges
// @Tags teams
// @Param team_id path string true "Team ID"
// @Success 200 {object} map[string]interface{}
// @Failure 403 {object} ErrorResponse
// @Router /teams/{team_id}/members [get]
func (h *TeamsHandler) ListTeamMembers(c echo.Context) error {
	userID, err := RequireChannelIdentityID(c)
	if err != nil {
		return err
	}
	team, err := h.requireTeamOwner(c, userID)
	if err != nil {
		return err
	}
	rows, err := h.queries.ListTeamMembersByTeam(c.Request().Context(), team.ID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	items := make([]TeamMemberResponse, 0, len(rows))
	for _, r := range rows {
		items = append(items, TeamMemberResponse{
			ID:                db.UUIDToString(r.ID),
			TeamID:            db.UUIDToString(r.TeamID),
			SourceBotID:       db.UUIDToString(r.SourceBotID),
			TargetBotID:       db.UUIDToString(r.TargetBotID),
			RoleDescription:   r.RoleDescription,
			TargetDisplayName: db.TextToString(r.TargetDisplayName),
			CreatedAt:         db.TimeFromPg(r.CreatedAt),
		})
	}
	return c.JSON(http.StatusOK, map[string]any{"items": items})
}

// GetBotTeamContext godoc
// @Summary Preview team context for a bot (debug)
// @Tags teams
// @Param bot_id path string true "Bot ID"
// @Success 200 {object} map[string]interface{}
// @Failure 403 {object} ErrorResponse
// @Router /bots/{bot_id}/team-context [get]
func (h *TeamsHandler) GetBotTeamContext(c echo.Context) error {
	userID, err := RequireChannelIdentityID(c)
	if err != nil {
		return err
	}
	botID := strings.TrimSpace(c.Param("bot_id"))
	if botID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "bot_id is required")
	}
	bot, err := h.botService.Get(c.Request().Context(), botID)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "bot not found")
	}
	if bot.OwnerUserID != userID {
		isAdmin, _ := h.accountService.IsAdmin(c.Request().Context(), userID)
		if !isAdmin {
			return echo.NewHTTPError(http.StatusForbidden, "access denied")
		}
	}
	botUUID, err := db.ParseUUID(botID)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	rows, err := h.queries.ListAllTeamContextForBot(c.Request().Context(), botUUID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	teamContent := BuildTeamContent(rows, botID)
	return c.JSON(http.StatusOK, map[string]any{
		"bot_id":       botID,
		"team_content": teamContent,
	})
}

// ListBotCallLogs godoc
// @Summary List call logs for a bot
// @Tags teams
// @Param bot_id path string true "Bot ID"
// @Success 200 {object} map[string]interface{}
// @Router /bots/{bot_id}/call-logs [get]
func (h *TeamsHandler) ListBotCallLogs(c echo.Context) error {
	userID, err := RequireChannelIdentityID(c)
	if err != nil {
		return err
	}
	botID := strings.TrimSpace(c.Param("bot_id"))
	bot, err := h.botService.Get(c.Request().Context(), botID)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "bot not found")
	}
	if bot.OwnerUserID != userID {
		isAdmin, _ := h.accountService.IsAdmin(c.Request().Context(), userID)
		if !isAdmin {
			return echo.NewHTTPError(http.StatusForbidden, "access denied")
		}
	}
	botUUID, err := db.ParseUUID(botID)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	rows, dbErr := h.queries.ListBotCallLogs(c.Request().Context(), sqlc.ListBotCallLogsParams{
		CallerBotID: botUUID,
		Limit:       50,
	})
	if dbErr != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, dbErr.Error())
	}
	type callLogResp struct {
		ID          string  `json:"id"`
		CallerBotID string  `json:"caller_bot_id"`
		TargetBotID string  `json:"target_bot_id"`
		Message     string  `json:"message"`
		Result      string  `json:"result"`
		Status      string  `json:"status"`
		CallDepth   int32   `json:"call_depth"`
		CreatedAt   string  `json:"created_at"`
		CompletedAt *string `json:"completed_at,omitempty"`
	}
	items := make([]callLogResp, 0, len(rows))
	for _, r := range rows {
		item := callLogResp{
			ID:          db.UUIDToString(r.ID),
			CallerBotID: db.UUIDToString(r.CallerBotID),
			TargetBotID: db.UUIDToString(r.TargetBotID),
			Message:     r.Message,
			Result:      db.TextToString(r.Result),
			Status:      r.Status,
			CallDepth:   r.CallDepth,
			CreatedAt:   db.TimeFromPg(r.CreatedAt).Format(time.RFC3339),
		}
		if r.CompletedAt.Valid {
			s := r.CompletedAt.Time.Format(time.RFC3339)
			item.CompletedAt = &s
		}
		items = append(items, item)
	}
	return c.JSON(http.StatusOK, map[string]any{"items": items})
}

// ------------------------------------------------------------------
// BuildTeamContent is exported so resolver.go can call it directly.
// It builds the Markdown team context section that is injected into
// the agent system prompt at runtime.
// ------------------------------------------------------------------

// BuildTeamContent converts ListAllTeamContextForBot rows into a Markdown
// string suitable for injection into the agent system prompt.
func BuildTeamContent(rows []sqlc.ListAllTeamContextForBotRow, currentBotID string) string {
	if len(rows) == 0 {
		return ""
	}

	// Group rows by team.
	type teamEntry struct {
		name       string
		managerID  string
		managerName string
		members    []struct {
			targetID   string
			targetName string
			role       string
		}
	}
	teamMap := make(map[string]*teamEntry)
	teamOrder := []string{}

	for _, row := range rows {
		tid := db.UUIDToString(row.TeamID)
		if _, ok := teamMap[tid]; !ok {
			teamMap[tid] = &teamEntry{
				name:        row.TeamName,
				managerID:   db.UUIDToString(row.ManagerBotID),
				managerName: db.TextToString(row.ManagerDisplayName),
			}
			teamOrder = append(teamOrder, tid)
		}
		teamMap[tid].members = append(teamMap[tid].members, struct {
			targetID   string
			targetName string
			role       string
		}{
			targetID:   db.UUIDToString(row.TargetBotID),
			targetName: db.TextToString(row.TargetDisplayName),
			role:       row.RoleDescription,
		})
	}

	var sb strings.Builder
	for _, tid := range teamOrder {
		entry := teamMap[tid]
		sb.WriteString(fmt.Sprintf("### 团队：%s\n\n", entry.name))
		if entry.managerID != "" {
			managerLabel := entry.managerName
			if managerLabel == "" {
				managerLabel = entry.managerID
			}
			if entry.managerID == currentBotID {
				managerLabel += "（你）"
			}
			sb.WriteString(fmt.Sprintf("- **大总管**: %s\n", managerLabel))
		}
		if len(entry.members) > 0 {
			sb.WriteString("- **可调用成员**（使用 `call_agent` 工具）:\n")
			for _, m := range entry.members {
				label := m.targetName
				if label == "" {
					label = m.targetID
				}
				role := m.role
				if role != "" {
					sb.WriteString(fmt.Sprintf("  - %s（%s）— bot_id: `%s`\n", label, role, m.targetID))
				} else {
					sb.WriteString(fmt.Sprintf("  - %s — bot_id: `%s`\n", label, m.targetID))
				}
			}
		}
		sb.WriteString("\n")
	}
	return strings.TrimRight(sb.String(), "\n")
}

// ------------------------------------------------------------------
// Helpers
// ------------------------------------------------------------------

func (h *TeamsHandler) requireTeamOwner(c echo.Context, userID string) (sqlc.BotTeam, error) {
	teamID := strings.TrimSpace(c.Param("team_id"))
	if teamID == "" {
		return sqlc.BotTeam{}, echo.NewHTTPError(http.StatusBadRequest, "team_id is required")
	}
	teamUUID, err := db.ParseUUID(teamID)
	if err != nil {
		return sqlc.BotTeam{}, echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	team, err := h.queries.GetTeamByID(c.Request().Context(), teamUUID)
	if err != nil {
		if isNotFound(err) {
			return sqlc.BotTeam{}, echo.NewHTTPError(http.StatusNotFound, "team not found")
		}
		return sqlc.BotTeam{}, echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	if db.UUIDToString(team.OwnerUserID) != userID {
		isAdmin, _ := h.accountService.IsAdmin(c.Request().Context(), userID)
		if !isAdmin {
			return sqlc.BotTeam{}, echo.NewHTTPError(http.StatusForbidden, "access denied")
		}
	}
	return team, nil
}

func (h *TeamsHandler) seedManagerHeartbeat(ctx context.Context, botID, prompt string) {
	botUUID, err := db.ParseUUID(botID)
	if err != nil {
		return
	}
	eventTriggers, _ := json.Marshal([]string{})
	if _, err := h.queries.CreateHeartbeatConfig(ctx, sqlc.CreateHeartbeatConfigParams{
		BotID:           botUUID,
		Enabled:         true,
		IntervalSeconds: 3600,
		Prompt:          prompt,
		EventTriggers:   eventTriggers,
	}); err != nil {
		h.logger.Warn("failed to seed manager heartbeat", slog.String("bot_id", botID), slog.Any("error", err))
	}
}

func appendTeamRoutingTable(task, teamName string, members []TeamMemberInput) string {
	var sb strings.Builder
	sb.WriteString(task)
	if task != "" && !strings.HasSuffix(strings.TrimSpace(task), "\n") {
		sb.WriteString("\n")
	}
	sb.WriteString(fmt.Sprintf("\n## 团队成员（%s）\n\n", teamName))
	sb.WriteString("使用 `call_agent` 工具将任务委派给以下专属成员：\n\n")
	for _, m := range members {
		role := strings.TrimSpace(m.RoleDescription)
		if role != "" {
			sb.WriteString(fmt.Sprintf("- **bot_id**: `%s`  — %s\n", m.BotID, role))
		} else {
			sb.WriteString(fmt.Sprintf("- **bot_id**: `%s`\n", m.BotID))
		}
	}
	sb.WriteString("\n委派原则：明确任务描述、预期交付物和截止时间；等待结果后再汇总给用户。\n")
	return sb.String()
}

func buildDefaultManagerHeartbeatPrompt(teamName string) string {
	return fmt.Sprintf(
		"你是「%s」团队的大总管。请检查团队成员的任务状态（通过 call_agent），汇总进展，处理阻碍，并向用户报告团队整体状态。",
		teamName,
	)
}

func toTeamResponse(t sqlc.BotTeam) TeamResponse {
	return TeamResponse{
		ID:           db.UUIDToString(t.ID),
		OwnerUserID:  db.UUIDToString(t.OwnerUserID),
		Name:         t.Name,
		ManagerBotID: db.UUIDToString(t.ManagerBotID),
		CreatedAt:    db.TimeFromPg(t.CreatedAt),
		UpdatedAt:    db.TimeFromPg(t.UpdatedAt),
	}
}

func toTeamMemberResponse(r sqlc.BotTeamMember, targetDisplayName string) TeamMemberResponse {
	return TeamMemberResponse{
		ID:                db.UUIDToString(r.ID),
		TeamID:            db.UUIDToString(r.TeamID),
		SourceBotID:       db.UUIDToString(r.SourceBotID),
		TargetBotID:       db.UUIDToString(r.TargetBotID),
		RoleDescription:   r.RoleDescription,
		TargetDisplayName: targetDisplayName,
		CreatedAt:         db.TimeFromPg(r.CreatedAt),
	}
}

func isNotFound(err error) bool {
	return err != nil && (err == pgx.ErrNoRows || strings.Contains(err.Error(), "no rows"))
}

