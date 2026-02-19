package schedule

import (
	"context"
	"log/slog"
	"strings"

	mcpgw "github.com/Kxiandaoyan/Memoh-v2/internal/mcp"
	sched "github.com/Kxiandaoyan/Memoh-v2/internal/schedule"
)

const (
	toolScheduleList   = "list_schedule"
	toolScheduleGet    = "get_schedule"
	toolScheduleCreate = "create_schedule"
	toolScheduleUpdate = "update_schedule"
	toolScheduleDelete = "delete_schedule"
)

type Scheduler interface {
	List(ctx context.Context, botID string) ([]sched.Schedule, error)
	Get(ctx context.Context, id string) (sched.Schedule, error)
	Create(ctx context.Context, botID string, req sched.CreateRequest) (sched.Schedule, error)
	Update(ctx context.Context, id string, req sched.UpdateRequest) (sched.Schedule, error)
	Delete(ctx context.Context, id string) error
}

type Executor struct {
	service Scheduler
	logger  *slog.Logger
}

func NewExecutor(log *slog.Logger, service Scheduler) *Executor {
	if log == nil {
		log = slog.Default()
	}
	return &Executor{
		service: service,
		logger:  log.With(slog.String("provider", "schedule_tool")),
	}
}

func (p *Executor) ListTools(ctx context.Context, session mcpgw.ToolSessionContext) ([]mcpgw.ToolDescriptor, error) {
	if p.service == nil {
		return []mcpgw.ToolDescriptor{}, nil
	}
	return []mcpgw.ToolDescriptor{
		{
			Name:        toolScheduleList,
			Description: "List all scheduled tasks for the current bot. Returns an array of schedule objects.",
			InputSchema: emptyObjectSchema(),
		},
		{
			Name:        toolScheduleGet,
			Description: "Get details of a specific scheduled task by its ID.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"id": map[string]any{"type": "string", "description": "The UUID of the schedule to retrieve"},
				},
				"required": []string{"id"},
			},
		},
		{
			Name:        toolScheduleCreate,
			Description: "Create a new scheduled task. The task will fire at the specified cron pattern and execute the command as a bot prompt. Returns the created schedule with its ID.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"name": map[string]any{
						"type":        "string",
						"description": "A short unique name for the schedule (e.g. 'daily_report', 'water_reminder')",
					},
					"description": map[string]any{
						"type":        "string",
						"description": "Human-readable description of what this schedule does",
					},
					"pattern": map[string]any{
						"type":        "string",
						"description": "Cron expression: 'minute hour day-of-month month day-of-week' (e.g. '30 9 * * *' for daily at 9:30, '0 */2 * * *' for every 2 hours). Supports optional seconds prefix.",
					},
					"command": map[string]any{
						"type":        "string",
						"description": "The prompt/instruction text that will be sent to the bot when the schedule fires. This is NOT a shell command — it's a natural language instruction (e.g. 'Send a water reminder to the user').",
					},
					"max_calls": map[string]any{
						"type":        "integer",
						"description": "Maximum number of times this schedule will fire. Omit or set to null for unlimited.",
					},
					"enabled": map[string]any{
						"type":        "boolean",
						"description": "Whether the schedule is enabled. Defaults to true if omitted.",
					},
				},
				"required": []string{"name", "description", "pattern", "command"},
			},
		},
		{
			Name:        toolScheduleUpdate,
			Description: "Update an existing scheduled task. Only provided fields will be changed.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"id": map[string]any{
						"type":        "string",
						"description": "The UUID of the schedule to update",
					},
					"name": map[string]any{
						"type":        "string",
						"description": "New name for the schedule",
					},
					"description": map[string]any{
						"type":        "string",
						"description": "New description",
					},
					"pattern": map[string]any{
						"type":        "string",
						"description": "New cron expression",
					},
					"max_calls": map[string]any{
						"type":        "integer",
						"description": "New max calls limit, or null for unlimited",
					},
					"enabled": map[string]any{
						"type":        "boolean",
						"description": "Enable or disable the schedule",
					},
					"command": map[string]any{
						"type":        "string",
						"description": "New prompt/instruction text",
					},
				},
				"required": []string{"id"},
			},
		},
		{
			Name:        toolScheduleDelete,
			Description: "Delete a scheduled task permanently by its ID.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"id": map[string]any{"type": "string", "description": "The UUID of the schedule to delete"},
				},
				"required": []string{"id"},
			},
		},
	}, nil
}

func (p *Executor) CallTool(ctx context.Context, session mcpgw.ToolSessionContext, toolName string, arguments map[string]any) (map[string]any, error) {
	if p.service == nil {
		return mcpgw.BuildToolErrorResult("schedule service not available"), nil
	}
	botID := strings.TrimSpace(session.BotID)
	if botID == "" {
		return mcpgw.BuildToolErrorResult("bot_id is required"), nil
	}

	switch toolName {
	case toolScheduleList:
		items, err := p.service.List(ctx, botID)
		if err != nil {
			p.logger.Warn("schedule list failed", slog.String("bot_id", botID), slog.Any("error", err))
			return mcpgw.BuildToolErrorResult(err.Error()), nil
		}
		p.logger.Info("schedule list", slog.String("bot_id", botID), slog.Int("count", len(items)))
		return mcpgw.BuildToolSuccessResult(map[string]any{
			"items": items,
		}), nil
	case toolScheduleGet:
		id := mcpgw.StringArg(arguments, "id")
		if id == "" {
			return mcpgw.BuildToolErrorResult("id is required"), nil
		}
		item, err := p.service.Get(ctx, id)
		if err != nil {
			return mcpgw.BuildToolErrorResult(err.Error()), nil
		}
		if item.BotID != botID {
			return mcpgw.BuildToolErrorResult("bot mismatch"), nil
		}
		return mcpgw.BuildToolSuccessResult(item), nil
	case toolScheduleCreate:
		name := mcpgw.StringArg(arguments, "name")
		description := mcpgw.StringArg(arguments, "description")
		pattern := mcpgw.StringArg(arguments, "pattern")
		command := mcpgw.StringArg(arguments, "command")
		p.logger.Info("schedule create attempt",
			slog.String("bot_id", botID),
			slog.String("name", name),
			slog.String("pattern", pattern),
			slog.String("command", command),
			slog.String("platform", session.CurrentPlatform),
			slog.String("reply_target", session.ReplyTarget),
		)
		if name == "" || description == "" || pattern == "" || command == "" {
			p.logger.Warn("schedule create missing required fields",
				slog.String("bot_id", botID),
				slog.Bool("has_name", name != ""),
				slog.Bool("has_description", description != ""),
				slog.Bool("has_pattern", pattern != ""),
				slog.Bool("has_command", command != ""),
			)
			return mcpgw.BuildToolErrorResult("name, description, pattern, command are required"), nil
		}

		req := sched.CreateRequest{
			Name:        name,
			Description: description,
			Pattern:     pattern,
			Command:     command,
			Platform:    strings.TrimSpace(session.CurrentPlatform),
			ReplyTarget: strings.TrimSpace(session.ReplyTarget),
		}
		maxCalls, err := parseNullableIntArg(arguments, "max_calls")
		if err != nil {
			p.logger.Warn("schedule create bad max_calls", slog.String("bot_id", botID), slog.Any("error", err))
			return mcpgw.BuildToolErrorResult(err.Error()), nil
		}
		req.MaxCalls = maxCalls
		if enabled, ok, err := mcpgw.BoolArg(arguments, "enabled"); err != nil {
			return mcpgw.BuildToolErrorResult(err.Error()), nil
		} else if ok {
			req.Enabled = &enabled
		}
		item, err := p.service.Create(ctx, botID, req)
		if err != nil {
			p.logger.Error("schedule create failed",
				slog.String("bot_id", botID),
				slog.String("name", name),
				slog.String("pattern", pattern),
				slog.Any("error", err),
			)
			return mcpgw.BuildToolErrorResult(err.Error()), nil
		}
		p.logger.Info("schedule created successfully",
			slog.String("bot_id", botID),
			slog.String("schedule_id", item.ID),
			slog.String("name", item.Name),
			slog.String("pattern", item.Pattern),
		)
		return mcpgw.BuildToolSuccessResult(item), nil
	case toolScheduleUpdate:
		id := mcpgw.StringArg(arguments, "id")
		if id == "" {
			return mcpgw.BuildToolErrorResult("id is required"), nil
		}
		existing, err := p.service.Get(ctx, id)
		if err != nil {
			return mcpgw.BuildToolErrorResult(err.Error()), nil
		}
		if existing.BotID != botID {
			return mcpgw.BuildToolErrorResult("bot mismatch"), nil
		}
		req := sched.UpdateRequest{}
		maxCalls, err := parseNullableIntArg(arguments, "max_calls")
		if err != nil {
			return mcpgw.BuildToolErrorResult(err.Error()), nil
		}
		req.MaxCalls = maxCalls
		if value := mcpgw.StringArg(arguments, "name"); value != "" {
			req.Name = &value
		}
		if value := mcpgw.StringArg(arguments, "description"); value != "" {
			req.Description = &value
		}
		if value := mcpgw.StringArg(arguments, "pattern"); value != "" {
			req.Pattern = &value
		}
		if value := mcpgw.StringArg(arguments, "command"); value != "" {
			req.Command = &value
		}
		if enabled, ok, err := mcpgw.BoolArg(arguments, "enabled"); err != nil {
			return mcpgw.BuildToolErrorResult(err.Error()), nil
		} else if ok {
			req.Enabled = &enabled
		}
		item, err := p.service.Update(ctx, id, req)
		if err != nil {
			return mcpgw.BuildToolErrorResult(err.Error()), nil
		}
		return mcpgw.BuildToolSuccessResult(item), nil
	case toolScheduleDelete:
		id := mcpgw.StringArg(arguments, "id")
		if id == "" {
			return mcpgw.BuildToolErrorResult("id is required"), nil
		}
		item, err := p.service.Get(ctx, id)
		if err != nil {
			return mcpgw.BuildToolErrorResult(err.Error()), nil
		}
		if item.BotID != botID {
			return mcpgw.BuildToolErrorResult("bot mismatch"), nil
		}
		if err := p.service.Delete(ctx, id); err != nil {
			return mcpgw.BuildToolErrorResult(err.Error()), nil
		}
		return mcpgw.BuildToolSuccessResult(map[string]any{"success": true}), nil
	default:
		return nil, mcpgw.ErrToolNotFound
	}
}

func parseNullableIntArg(arguments map[string]any, key string) (sched.NullableInt, error) {
	req := sched.NullableInt{}
	if arguments == nil {
		return req, nil
	}
	raw, exists := arguments[key]
	if !exists {
		return req, nil
	}
	req.Set = true
	if raw == nil {
		req.Value = nil
		return req, nil
	}
	value, _, err := mcpgw.IntArg(arguments, key)
	if err != nil {
		return sched.NullableInt{}, err
	}
	req.Value = &value
	return req, nil
}

func emptyObjectSchema() map[string]any {
	return map[string]any{
		"type":       "object",
		"properties": map[string]any{},
	}
}
