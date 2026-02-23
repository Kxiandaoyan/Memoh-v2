package message

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"

	"github.com/Kxiandaoyan/Memoh-v2/internal/channel"
	mcpgw "github.com/Kxiandaoyan/Memoh-v2/internal/mcp"
)

const (
	toolSend  = "send"
	toolReact = "react"
)

// Sender sends outbound messages through channel manager.
type Sender interface {
	Send(ctx context.Context, botID string, channelType channel.ChannelType, req channel.SendRequest) error
}

// Reactor adds or removes emoji reactions through channel manager.
type Reactor interface {
	React(ctx context.Context, botID string, channelType channel.ChannelType, req channel.ReactRequest) error
}

// ChannelTypeResolver parses platform name to channel type.
type ChannelTypeResolver interface {
	ParseChannelType(raw string) (channel.ChannelType, error)
}

// ContainerFileReader reads binary file content from a bot container.
type ContainerFileReader interface {
	ReadFileBytes(ctx context.Context, botID, filePath string) ([]byte, error)
}

// Executor exposes send and react as MCP tools.
type Executor struct {
	sender     Sender
	reactor    Reactor
	resolver   ChannelTypeResolver
	fileReader ContainerFileReader
	logger     *slog.Logger
}

// NewExecutor creates a message tool executor.
// reactor and fileReader may be nil.
func NewExecutor(log *slog.Logger, sender Sender, reactor Reactor, resolver ChannelTypeResolver, fileReader ContainerFileReader) *Executor {
	if log == nil {
		log = slog.Default()
	}
	return &Executor{
		sender:     sender,
		reactor:    reactor,
		resolver:   resolver,
		fileReader: fileReader,
		logger:     log.With(slog.String("provider", "message_tool")),
	}
}

func (p *Executor) ListTools(ctx context.Context, session mcpgw.ToolSessionContext) ([]mcpgw.ToolDescriptor, error) {
	var tools []mcpgw.ToolDescriptor
	if p.sender != nil && p.resolver != nil {
		tools = append(tools, mcpgw.ToolDescriptor{
			Name:        toolSend,
			Description: "Send a message to a channel or session. Supports text, structured messages, attachments, and replies.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"bot_id": map[string]any{
						"type":        "string",
						"description": "Bot ID, optional and defaults to current bot",
					},
					"platform": map[string]any{
						"type":        "string",
						"description": "Channel platform name",
					},
					"target": map[string]any{
						"type":        "string",
						"description": "Channel target (chat/group/thread ID)",
					},
					"channel_identity_id": map[string]any{
						"type":        "string",
						"description": "Target identity ID when direct target is absent",
					},
					"to_user_id": map[string]any{
						"type":        "string",
						"description": "Alias for channel_identity_id",
					},
					"text": map[string]any{
						"type":        "string",
						"description": "Message text shortcut when message object is omitted",
					},
					"reply_to": map[string]any{
						"type":        "string",
						"description": "Message ID to reply to. The reply will reference this message on the platform.",
					},
					"message": map[string]any{
						"type":        "object",
						"description": "Structured message payload with text/parts/attachments",
					},
					"attachments": map[string]any{
						"type":        "array",
						"description": "File attachments: container paths (/data/...), HTTP URLs, or objects {path, url, type, name}",
						"items":       map[string]any{},
					},
				},
				"required": []string{},
			},
		})
	}
	if p.reactor != nil && p.resolver != nil {
		tools = append(tools, mcpgw.ToolDescriptor{
			Name:        toolReact,
			Description: "Add or remove an emoji reaction on a channel message",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"bot_id": map[string]any{
						"type":        "string",
						"description": "Bot ID, optional and defaults to current bot",
					},
					"platform": map[string]any{
						"type":        "string",
						"description": "Channel platform name. Defaults to current session platform.",
					},
					"target": map[string]any{
						"type":        "string",
						"description": "Channel target (chat/group ID). Defaults to current session reply target.",
					},
					"message_id": map[string]any{
						"type":        "string",
						"description": "The message ID to react to",
					},
					"emoji": map[string]any{
						"type":        "string",
						"description": "Emoji to react with (e.g. 👍, ❤️). Required when adding a reaction.",
					},
					"remove": map[string]any{
						"type":        "boolean",
						"description": "If true, remove the reaction instead of adding it. Default false.",
					},
				},
				"required": []string{"message_id"},
			},
		})
	}
	return tools, nil
}

func (p *Executor) CallTool(ctx context.Context, session mcpgw.ToolSessionContext, toolName string, arguments map[string]any) (map[string]any, error) {
	switch toolName {
	case toolSend:
		return p.callSend(ctx, session, arguments)
	case toolReact:
		return p.callReact(ctx, session, arguments)
	default:
		return nil, mcpgw.ErrToolNotFound
	}
}

// --- send ---

func (p *Executor) callSend(ctx context.Context, session mcpgw.ToolSessionContext, arguments map[string]any) (map[string]any, error) {
	if p.sender == nil || p.resolver == nil {
		return mcpgw.BuildToolErrorResult("message service not available"), nil
	}

	botID, err := p.resolveBotID(arguments, session)
	if err != nil {
		return mcpgw.BuildToolErrorResult(err.Error()), nil
	}
	channelType, err := p.resolvePlatform(arguments, session)
	if err != nil {
		return mcpgw.BuildToolErrorResult(err.Error()), nil
	}

	messageText := mcpgw.FirstStringArg(arguments, "text")
	outboundMessage, parseErr := parseOutboundMessage(arguments, messageText)
	if parseErr != nil {
		if rawAtt, ok := arguments["attachments"]; !ok || rawAtt == nil {
			return mcpgw.BuildToolErrorResult(parseErr.Error()), nil
		}
		outboundMessage = channel.Message{Text: strings.TrimSpace(messageText)}
	}

	// Resolve top-level attachments parameter.
	if rawAtt, ok := arguments["attachments"]; ok && rawAtt != nil {
		if arr, ok := rawAtt.([]any); ok && len(arr) > 0 {
			resolved := p.resolveAttachments(ctx, botID, arr)
			outboundMessage.Attachments = append(outboundMessage.Attachments, resolved...)
		}
	}

	if outboundMessage.IsEmpty() && len(outboundMessage.Attachments) == 0 {
		return mcpgw.BuildToolErrorResult("message or attachments required"), nil
	}

	// Attach reply reference if reply_to is provided.
	if replyTo := mcpgw.FirstStringArg(arguments, "reply_to"); replyTo != "" {
		outboundMessage.Reply = &channel.ReplyRef{MessageID: replyTo}
	}

	target := mcpgw.FirstStringArg(arguments, "target")
	if target == "" {
		target = strings.TrimSpace(session.ReplyTarget)
	}
	channelIdentityID := mcpgw.FirstStringArg(arguments, "channel_identity_id", "to_user_id")
	if target == "" && channelIdentityID == "" {
		return mcpgw.BuildToolErrorResult("target or channel_identity_id is required"), nil
	}

	sendReq := channel.SendRequest{
		Target:            target,
		ChannelIdentityID: channelIdentityID,
		Message:           outboundMessage,
	}
	if err := p.sender.Send(ctx, botID, channelType, sendReq); err != nil {
		p.logger.Warn("send failed", slog.Any("error", err), slog.String("bot_id", botID), slog.String("platform", string(channelType)))
		return mcpgw.BuildToolErrorResult(err.Error()), nil
	}

	payload := map[string]any{
		"ok":                  true,
		"bot_id":              botID,
		"platform":            channelType.String(),
		"target":              target,
		"channel_identity_id": channelIdentityID,
		"instruction":         "Message delivered successfully. You have completed your response. Please STOP now and do not call any more tools.",
	}
	return mcpgw.BuildToolSuccessResult(payload), nil
}

// --- react ---

func (p *Executor) callReact(ctx context.Context, session mcpgw.ToolSessionContext, arguments map[string]any) (map[string]any, error) {
	if p.reactor == nil || p.resolver == nil {
		return mcpgw.BuildToolErrorResult("reaction service not available"), nil
	}

	botID, err := p.resolveBotID(arguments, session)
	if err != nil {
		return mcpgw.BuildToolErrorResult(err.Error()), nil
	}
	channelType, err := p.resolvePlatform(arguments, session)
	if err != nil {
		return mcpgw.BuildToolErrorResult(err.Error()), nil
	}

	target := mcpgw.FirstStringArg(arguments, "target")
	if target == "" {
		target = strings.TrimSpace(session.ReplyTarget)
	}
	if target == "" {
		return mcpgw.BuildToolErrorResult("target is required"), nil
	}

	messageID := mcpgw.FirstStringArg(arguments, "message_id")
	if messageID == "" {
		return mcpgw.BuildToolErrorResult("message_id is required"), nil
	}

	emoji := mcpgw.FirstStringArg(arguments, "emoji")
	remove, _, _ := mcpgw.BoolArg(arguments, "remove")

	reactReq := channel.ReactRequest{
		Target:    target,
		MessageID: messageID,
		Emoji:     emoji,
		Remove:    remove,
	}
	if err := p.reactor.React(ctx, botID, channelType, reactReq); err != nil {
		p.logger.Warn("react failed", slog.Any("error", err), slog.String("bot_id", botID), slog.String("platform", string(channelType)))
		return mcpgw.BuildToolErrorResult(err.Error()), nil
	}

	action := "added"
	if remove {
		action = "removed"
	}
	payload := map[string]any{
		"ok":         true,
		"bot_id":     botID,
		"platform":   channelType.String(),
		"target":     target,
		"message_id": messageID,
		"emoji":      emoji,
		"action":     action,
	}
	return mcpgw.BuildToolSuccessResult(payload), nil
}

// --- shared helpers ---

func (p *Executor) resolveBotID(arguments map[string]any, session mcpgw.ToolSessionContext) (string, error) {
	botID := mcpgw.FirstStringArg(arguments, "bot_id")
	if botID == "" {
		botID = strings.TrimSpace(session.BotID)
	}
	if botID == "" {
		return "", fmt.Errorf("bot_id is required")
	}
	if strings.TrimSpace(session.BotID) != "" && botID != strings.TrimSpace(session.BotID) {
		return "", fmt.Errorf("bot_id mismatch")
	}
	return botID, nil
}

func (p *Executor) resolvePlatform(arguments map[string]any, session mcpgw.ToolSessionContext) (channel.ChannelType, error) {
	platform := mcpgw.FirstStringArg(arguments, "platform")
	if platform == "" {
		platform = strings.TrimSpace(session.CurrentPlatform)
	}
	if platform == "" {
		return "", fmt.Errorf("platform is required")
	}
	return p.resolver.ParseChannelType(platform)
}

func parseOutboundMessage(arguments map[string]any, fallbackText string) (channel.Message, error) {
	var msg channel.Message
	if raw, ok := arguments["message"]; ok && raw != nil {
		switch value := raw.(type) {
		case string:
			msg.Text = strings.TrimSpace(value)
		case map[string]any:
			data, err := json.Marshal(value)
			if err != nil {
				return channel.Message{}, err
			}
			if err := json.Unmarshal(data, &msg); err != nil {
				return channel.Message{}, err
			}
		default:
			return channel.Message{}, fmt.Errorf("message must be object or string")
		}
	}
	if msg.IsEmpty() && strings.TrimSpace(fallbackText) != "" {
		msg.Text = strings.TrimSpace(fallbackText)
	}
	if msg.IsEmpty() {
		return channel.Message{}, fmt.Errorf("message is required")
	}
	return msg, nil
}

// --- attachment resolution ---

func (p *Executor) resolveAttachments(ctx context.Context, botID string, arr []any) []channel.Attachment {
	var out []channel.Attachment
	for _, item := range arr {
		var att channel.Attachment
		switch v := item.(type) {
		case string:
			att = p.resolveAttachmentRef(ctx, botID, v, "")
		case map[string]any:
			ref := stringVal(v, "path", "url")
			name := stringVal(v, "name")
			att = p.resolveAttachmentRef(ctx, botID, ref, name)
			if t := stringVal(v, "type"); t != "" {
				att.Type = channel.AttachmentType(t)
			}
		default:
			continue
		}
		out = append(out, att)
	}
	return out
}

func (p *Executor) resolveAttachmentRef(ctx context.Context, botID, ref, name string) channel.Attachment {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return channel.Attachment{Type: channel.AttachmentFile, Name: name}
	}

	// HTTP/HTTPS URL — pass through.
	if strings.HasPrefix(ref, "http://") || strings.HasPrefix(ref, "https://") {
		att := channel.Attachment{URL: ref, Name: name}
		att.Type = inferAttachmentTypeFromExt(ref)
		return att
	}

	// Data URL — decode inline.
	if strings.HasPrefix(ref, "data:") {
		mime, data, err := decodeDataURL(ref)
		if err != nil {
			return channel.Attachment{Type: channel.AttachmentFile, Name: name}
		}
		return channel.Attachment{
			Type: inferAttachmentTypeFromMime(mime),
			Data: data, Mime: mime, Name: name,
			Size: int64(len(data)),
		}
	}

	// Container file path.
	if p.fileReader != nil && strings.HasPrefix(ref, "/") {
		cleanPath := filepath.Clean(ref)
		if !strings.HasPrefix(cleanPath, "/data/") {
			return channel.Attachment{Type: channel.AttachmentFile, Name: name}
		}
		data, err := p.fileReader.ReadFileBytes(ctx, botID, cleanPath)
		if err != nil {
			p.logger.Warn("read container file failed", slog.String("path", cleanPath), slog.Any("error", err))
			return channel.Attachment{Type: channel.AttachmentFile, URL: cleanPath, Name: name}
		}
		if name == "" {
			name = filepath.Base(cleanPath)
		}
		mime := inferMimeFromExt(cleanPath)
		return channel.Attachment{
			Type: inferAttachmentTypeFromExt(cleanPath),
			Data: data, Mime: mime, Name: name,
			Size: int64(len(data)),
		}
	}

	// Unknown — pass through as URL.
	return channel.Attachment{Type: channel.AttachmentFile, URL: ref, Name: name}
}

func stringVal(m map[string]any, keys ...string) string {
	for _, k := range keys {
		if v, ok := m[k].(string); ok && v != "" {
			return v
		}
	}
	return ""
}

var extToMime = map[string]string{
	".jpg": "image/jpeg", ".jpeg": "image/jpeg", ".png": "image/png",
	".gif": "image/gif", ".webp": "image/webp", ".svg": "image/svg+xml",
	".mp3": "audio/mpeg", ".ogg": "audio/ogg", ".wav": "audio/wav",
	".mp4": "video/mp4", ".webm": "video/webm", ".mov": "video/quicktime",
	".pdf": "application/pdf", ".zip": "application/zip",
}

func inferMimeFromExt(path string) string {
	if i := strings.IndexAny(path, "?#"); i >= 0 {
		path = path[:i]
	}
	ext := strings.ToLower(filepath.Ext(path))
	if m, ok := extToMime[ext]; ok {
		return m
	}
	return "application/octet-stream"
}

func inferAttachmentTypeFromExt(path string) channel.AttachmentType {
	return inferAttachmentTypeFromMime(inferMimeFromExt(path))
}

func inferAttachmentTypeFromMime(mime string) channel.AttachmentType {
	switch {
	case strings.HasPrefix(mime, "image/gif"):
		return channel.AttachmentGIF
	case strings.HasPrefix(mime, "image/"):
		return channel.AttachmentImage
	case strings.HasPrefix(mime, "audio/"):
		return channel.AttachmentAudio
	case strings.HasPrefix(mime, "video/"):
		return channel.AttachmentVideo
	default:
		return channel.AttachmentFile
	}
}

func decodeDataURL(dataURL string) (mime string, data []byte, err error) {
	// data:[<mediatype>][;base64],<data>
	rest := strings.TrimPrefix(dataURL, "data:")
	commaIdx := strings.Index(rest, ",")
	if commaIdx < 0 {
		return "", nil, fmt.Errorf("invalid data URL")
	}
	meta, payload := rest[:commaIdx], rest[commaIdx+1:]
	if !strings.HasSuffix(meta, ";base64") {
		return "", nil, fmt.Errorf("only base64 data URLs supported")
	}
	mime = strings.TrimSuffix(meta, ";base64")
	if mime == "" {
		mime = "application/octet-stream"
	}
	payload = strings.TrimRight(payload, "=")
	data, err = base64.RawStdEncoding.DecodeString(payload)
	return
}

// --- ContainerFileReader adapter ---

type execRunnerFileReader struct {
	runner interface {
		ExecWithCapture(ctx context.Context, req mcpgw.ExecRequest) (*mcpgw.ExecWithCaptureResult, error)
	}
	workDir string
}

// NewContainerFileReader wraps an ExecRunner (MCP Manager) as a ContainerFileReader.
func NewContainerFileReader(runner interface {
	ExecWithCapture(ctx context.Context, req mcpgw.ExecRequest) (*mcpgw.ExecWithCaptureResult, error)
}, workDir string) ContainerFileReader {
	return &execRunnerFileReader{runner: runner, workDir: workDir}
}

func (r *execRunnerFileReader) ReadFileBytes(ctx context.Context, botID, filePath string) ([]byte, error) {
	res, err := r.runner.ExecWithCapture(ctx, mcpgw.ExecRequest{
		BotID:   botID,
		Command: []string{"base64", filePath},
		WorkDir: r.workDir,
	})
	if err != nil {
		return nil, err
	}
	if res.ExitCode != 0 {
		return nil, fmt.Errorf("base64 %s: exit %d: %s", filePath, res.ExitCode, res.Stderr)
	}
	return base64.StdEncoding.DecodeString(strings.TrimSpace(res.Stdout))
}
