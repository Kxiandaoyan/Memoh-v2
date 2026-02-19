package handlers

import (
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/labstack/echo/v4"
)

var mediaExtToMIME = map[string]string{
	".png":  "image/png",
	".jpg":  "image/jpeg",
	".jpeg": "image/jpeg",
	".gif":  "image/gif",
	".webp": "image/webp",
	".svg":  "image/svg+xml",
	".mp4":  "video/mp4",
	".webm": "video/webm",
	".mp3":  "audio/mpeg",
	".wav":  "audio/wav",
	".ogg":  "audio/ogg",
	".pdf":  "application/pdf",
}

func (h *ContainerdHandler) ReadBotMedia(c echo.Context) error {
	botID := strings.TrimSpace(c.Param("bot_id"))
	if botID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "bot_id is required"})
	}
	filename := strings.TrimSpace(c.Param("filename"))
	if filename == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "filename is required"})
	}

	if strings.Contains(filename, "..") || strings.ContainsAny(filename, `/\`) {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid filename"})
	}

	root, err := h.ensureBotDataRoot(botID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "data root error"})
	}
	filePath := filepath.Join(root, "media", filename)

	clean := filepath.Clean(filePath)
	if !strings.HasPrefix(clean, filepath.Join(root, "media")) {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid path"})
	}

	info, err := os.Stat(clean)
	if err != nil || info.IsDir() {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "file not found"})
	}

	return c.File(clean)
}

func (h *ContainerdHandler) ListBotMedia(c echo.Context) error {
	botID := strings.TrimSpace(c.Param("bot_id"))
	if botID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "bot_id is required"})
	}

	root, err := h.ensureBotDataRoot(botID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "data root error"})
	}
	mediaDir := filepath.Join(root, "media")

	entries, err := os.ReadDir(mediaDir)
	if err != nil {
		if os.IsNotExist(err) {
			return c.JSON(http.StatusOK, map[string]any{"files": []any{}})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to list media"})
	}

	type mediaEntry struct {
		Name      string `json:"name"`
		Size      int64  `json:"size"`
		Mime      string `json:"mime"`
		UpdatedAt string `json:"updated_at"`
	}
	files := make([]mediaEntry, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		files = append(files, mediaEntry{
			Name:      entry.Name(),
			Size:      info.Size(),
			Mime:      resolveMediaContentType(entry.Name()),
			UpdatedAt: info.ModTime().UTC().Format("2006-01-02T15:04:05Z"),
		})
	}
	return c.JSON(http.StatusOK, map[string]any{"files": files})
}

func resolveMediaContentType(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	if ct, ok := mediaExtToMIME[ext]; ok {
		return ct
	}
	if ct := mime.TypeByExtension(ext); ct != "" {
		return ct
	}
	return "application/octet-stream"
}
