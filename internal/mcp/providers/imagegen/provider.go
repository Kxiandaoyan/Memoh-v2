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

func callOpenAICompatibleImageAPI(ctx context.Context, req generateRequest) ([]byte, error) {
	baseURL := strings.TrimRight(req.baseURL, "/")
	endpoint := baseURL + "/chat/completions"

	payload := map[string]any{
		"model": req.modelName,
		"messages": []map[string]any{
			{"role": "user", "content": "Generate an image: " + req.prompt},
		},
		// Required for Gemini image generation models via OpenRouter / OpenAI-compat APIs.
		// Maps to Gemini's responseModalities: ["TEXT", "IMAGE"].
		"modalities": []string{"text", "image"},
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
		preview := truncate(string(respBody), 500)
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, preview)
	}

	slog.Info("imagegen: raw API response", slog.String("body_preview", truncate(string(respBody), 500)))

	return extractImageFromResponse(respBody, req.modelName)
}

// extractImageFromResponse walks through the OpenAI-compatible response
// trying multiple known multimodal content layouts used by providers.
func extractImageFromResponse(respBody []byte, modelName string) ([]byte, error) {
	var raw map[string]any
	if err := json.Unmarshal(respBody, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse response JSON: %w", err)
	}

	// Check top-level error.
	if errObj, ok := raw["error"]; ok && errObj != nil {
		if errMap, ok := errObj.(map[string]any); ok {
			return nil, fmt.Errorf("API error: %v", errMap["message"])
		}
	}

	choices, ok := raw["choices"].([]any)
	if !ok || len(choices) == 0 {
		return nil, fmt.Errorf("API returned no choices (response: %s)", truncate(string(respBody), 300))
	}

	choice, ok := choices[0].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("unexpected choice format")
	}
	msg, ok := choice["message"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("unexpected message format")
	}

	content := msg["content"]

	// --- Case 1: content is an array of parts (multimodal response) ---
	if parts, ok := content.([]any); ok {
		for _, partRaw := range parts {
			part, ok := partRaw.(map[string]any)
			if !ok {
				continue
			}
			if imgData := extractImageFromPart(part); imgData != nil {
				return imgData, nil
			}
		}
		return nil, fmt.Errorf("multimodal response had %d parts but none contained image data", len(parts))
	}

	// --- Case 2: content is a plain string ---
	if text, ok := content.(string); ok {
		// Maybe it's a data URI directly.
		if strings.HasPrefix(text, "data:image/") {
			return decodeBase64Data(text)
		}
		// Maybe it's a URL to an image.
		if (strings.HasPrefix(text, "http://") || strings.HasPrefix(text, "https://")) && looksLikeImageURL(text) {
			return fetchImageURL(text)
		}
		return nil, fmt.Errorf("response was text only, no image (model: %s, preview: %s)", modelName, truncate(text, 200))
	}

	return nil, fmt.Errorf("unexpected content type in response (model: %s)", modelName)
}

// extractImageFromPart tries to find image data in a single content part.
// Supports multiple provider formats:
//   - {"type":"image_url","image_url":{"url":"data:image/png;base64,..."}}   (OpenAI format)
//   - {"type":"image","source":{"type":"base64","data":"..."}}                (Anthropic format)
//   - {"type":"inline_data","inline_data":{"mime_type":"image/png","data":"..."}} (Gemini-like)
//   - Part with "b64_json" or "data" fields directly
func extractImageFromPart(part map[string]any) []byte {
	partType, _ := part["type"].(string)

	// OpenAI format: image_url with data URI
	if partType == "image_url" {
		if imgURL, ok := part["image_url"].(map[string]any); ok {
			if url, ok := imgURL["url"].(string); ok {
				if data, err := resolveImageString(url); err == nil {
					return data
				}
			}
		}
	}

	// Anthropic format: source.data
	if partType == "image" {
		if source, ok := part["source"].(map[string]any); ok {
			if b64, ok := source["data"].(string); ok {
				if data, err := base64.StdEncoding.DecodeString(b64); err == nil {
					return data
				}
			}
		}
	}

	// Gemini-like: inline_data
	if partType == "inline_data" {
		if inline, ok := part["inline_data"].(map[string]any); ok {
			if b64, ok := inline["data"].(string); ok {
				if data, err := base64.StdEncoding.DecodeString(b64); err == nil {
					return data
				}
			}
		}
	}

	// Direct fields: b64_json, data, image_data
	for _, key := range []string{"b64_json", "data", "image_data"} {
		if b64, ok := part[key].(string); ok && len(b64) > 100 {
			if data, err := base64.StdEncoding.DecodeString(b64); err == nil {
				return data
			}
			if data, err := base64.RawStdEncoding.DecodeString(b64); err == nil {
				return data
			}
		}
	}

	return nil
}

// resolveImageString handles data URIs, https URLs, and raw base64.
func resolveImageString(s string) ([]byte, error) {
	if strings.HasPrefix(s, "data:") {
		return decodeBase64Data(s)
	}
	if strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://") {
		return fetchImageURL(s)
	}
	// Possibly raw base64 without prefix.
	if len(s) > 200 {
		if data, err := base64.StdEncoding.DecodeString(s); err == nil {
			return data, nil
		}
	}
	return nil, fmt.Errorf("cannot resolve image string: %s", truncate(s, 60))
}

// decodeBase64Data extracts raw bytes from "data:image/png;base64,..." strings.
func decodeBase64Data(uri string) ([]byte, error) {
	comma := strings.Index(uri, ",")
	if comma < 0 {
		return nil, fmt.Errorf("malformed data URI")
	}
	encoded := uri[comma+1:]
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		decoded, err = base64.RawStdEncoding.DecodeString(encoded)
		if err != nil {
			return nil, fmt.Errorf("failed to decode base64: %w", err)
		}
	}
	return decoded, nil
}

// fetchImageURL downloads an image from an HTTPS URL.
func fetchImageURL(url string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("image download failed: HTTP %d", resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, 20*1024*1024)) // 20 MB cap
	if err != nil {
		return nil, err
	}
	if len(data) < 100 {
		return nil, fmt.Errorf("downloaded data too small (%d bytes), likely not an image", len(data))
	}
	return data, nil
}

func looksLikeImageURL(url string) bool {
	lower := strings.ToLower(url)
	for _, ext := range []string{".png", ".jpg", ".jpeg", ".webp", ".gif"} {
		if strings.Contains(lower, ext) {
			return true
		}
	}
	return strings.Contains(lower, "image") || strings.Contains(lower, "img")
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
