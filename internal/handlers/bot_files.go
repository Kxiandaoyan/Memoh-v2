package handlers

import (
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
)

// allowedExtensions restricts file access to safe text-based extensions.
var allowedExtensions = map[string]bool{
	".md":   true,
	".txt":  true,
	".json": true,
	".yaml": true,
	".yml":  true,
	".toml": true,
	".conf": true,
}

// BotFileEntry represents a single file in the bot data directory.
type BotFileEntry struct {
	Name      string `json:"name"`
	Size      int64  `json:"size"`
	UpdatedAt string `json:"updated_at"`
}

// BotFileContent represents a file's content.
type BotFileContent struct {
	Name    string `json:"name"`
	Content string `json:"content"`
	Size    int64  `json:"size"`
}

// BotFileWriteRequest is the request body for updating a file.
type BotFileWriteRequest struct {
	Content string `json:"content"`
}

// ListBotFiles godoc
// @Summary List text files in bot data directory
// @Description Returns a list of text/markdown files in the bot's data directory (non-recursive, top-level only)
// @Tags bot-files
// @Produce json
// @Param bot_id path string true "Bot ID"
// @Success 200 {object} map[string][]BotFileEntry
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /bots/{bot_id}/files [get]
func (h *ContainerdHandler) ListBotFiles(c echo.Context) error {
	botID, err := h.requireBotAccess(c)
	if err != nil {
		return err
	}
	dataDir, err := h.ensureBotDataRoot(botID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	entries, err := os.ReadDir(dataDir)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	var files []BotFileEntry
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if !allowedExtensions[ext] {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		files = append(files, BotFileEntry{
			Name:      entry.Name(),
			Size:      info.Size(),
			UpdatedAt: info.ModTime().UTC().Format(time.RFC3339),
		})
	}
	sort.Slice(files, func(i, j int) bool {
		return files[i].Name < files[j].Name
	})
	return c.JSON(http.StatusOK, map[string]any{"files": files})
}

// ReadBotFile godoc
// @Summary Read a text file from bot data directory
// @Description Returns the content of the specified text/markdown file
// @Tags bot-files
// @Produce json
// @Param bot_id path string true "Bot ID"
// @Param filename path string true "File name"
// @Success 200 {object} BotFileContent
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /bots/{bot_id}/files/{filename} [get]
func (h *ContainerdHandler) ReadBotFile(c echo.Context) error {
	botID, err := h.requireBotAccess(c)
	if err != nil {
		return err
	}
	filename := c.Param("filename")
	if err := validateFilename(filename); err != nil {
		return err
	}

	dataDir, err := h.ensureBotDataRoot(botID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	filePath := filepath.Join(dataDir, filename)
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return echo.NewHTTPError(http.StatusNotFound, "file not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, BotFileContent{
		Name:    filename,
		Content: string(data),
		Size:    int64(len(data)),
	})
}

// WriteBotFile godoc
// @Summary Write/update a text file in bot data directory
// @Description Creates or updates the specified text/markdown file with the given content
// @Tags bot-files
// @Accept json
// @Produce json
// @Param bot_id path string true "Bot ID"
// @Param filename path string true "File name"
// @Param payload body BotFileWriteRequest true "File content"
// @Success 200 {object} BotFileContent
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /bots/{bot_id}/files/{filename} [put]
func (h *ContainerdHandler) WriteBotFile(c echo.Context) error {
	botID, err := h.requireBotAccess(c)
	if err != nil {
		return err
	}
	filename := c.Param("filename")
	if err := validateFilename(filename); err != nil {
		return err
	}

	var req BotFileWriteRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	// Guard against unreasonably large payloads (1 MB).
	if len(req.Content) > 1<<20 {
		return echo.NewHTTPError(http.StatusBadRequest, "content too large (max 1MB)")
	}

	dataDir, err := h.ensureBotDataRoot(botID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	filePath := filepath.Join(dataDir, filename)
	if err := os.WriteFile(filePath, []byte(req.Content), 0o644); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, BotFileContent{
		Name:    filename,
		Content: req.Content,
		Size:    int64(len(req.Content)),
	})
}

// DeleteBotFile godoc
// @Summary Delete a text file from bot data directory
// @Description Removes the specified text/markdown file from the bot's data directory
// @Tags bot-files
// @Produce json
// @Param bot_id path string true "Bot ID"
// @Param filename path string true "File name"
// @Success 204
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /bots/{bot_id}/files/{filename} [delete]
func (h *ContainerdHandler) DeleteBotFile(c echo.Context) error {
	botID, err := h.requireBotAccess(c)
	if err != nil {
		return err
	}
	filename := c.Param("filename")
	if err := validateFilename(filename); err != nil {
		return err
	}

	dataDir, err := h.ensureBotDataRoot(botID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	filePath := filepath.Join(dataDir, filename)
	if err := os.Remove(filePath); err != nil {
		if os.IsNotExist(err) {
			return echo.NewHTTPError(http.StatusNotFound, "file not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.NoContent(http.StatusNoContent)
}

// DownloadBotFile serves any file from the bot's data directory as a direct download.
func (h *ContainerdHandler) DownloadBotFile(c echo.Context) error {
	botID, err := h.requireBotAccess(c)
	if err != nil {
		return err
	}
	reqPath := strings.TrimPrefix(c.Param("*"), "/")
	if reqPath == "" || strings.Contains(reqPath, "..") {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid file path")
	}
	dataDir, err := h.ensureBotDataRoot(botID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	filePath := filepath.Clean(filepath.Join(dataDir, reqPath))
	if !strings.HasPrefix(filePath, dataDir) {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid file path")
	}
	info, statErr := os.Stat(filePath)
	if statErr != nil || info.IsDir() {
		return echo.NewHTTPError(http.StatusNotFound, "file not found")
	}
	c.Response().Header().Set("Content-Disposition",
		"attachment; filename=\""+filepath.Base(filePath)+"\"")
	return c.File(filePath)
}

// validateFilename checks that the filename is safe (no path traversal, allowed extension).
func validateFilename(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "filename is required")
	}
	if strings.ContainsAny(name, "/\\") || name == "." || name == ".." || strings.HasPrefix(name, ".") {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid filename")
	}
	ext := strings.ToLower(filepath.Ext(name))
	if !allowedExtensions[ext] {
		return echo.NewHTTPError(http.StatusBadRequest, "unsupported file type; allowed: .md, .txt, .json, .yaml, .yml, .toml, .conf")
	}
	return nil
}

// PreviewBotFile serves a file inline for browser rendering.
// GET /bots/:bot_id/files/preview/*
func (h *ContainerdHandler) PreviewBotFile(c echo.Context) error {
	botID := strings.TrimSpace(c.Param("bot_id"))
	if botID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "bot_id required")
	}
	reqPath := c.Param("*")
	if reqPath == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "file path required")
	}
	reqPath = strings.TrimPrefix(reqPath, "/")
	if strings.Contains(reqPath, "..") {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid path")
	}
	dataDir, err := h.ensureBotDataRoot(botID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	filePath := filepath.Clean(filepath.Join(dataDir, reqPath))
	if !strings.HasPrefix(filePath, dataDir) {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid path")
	}
	info, statErr := os.Stat(filePath)
	if statErr != nil || info.IsDir() {
		return echo.NewHTTPError(http.StatusNotFound, "file not found")
	}
	return c.File(filePath)
}

// systemFiles lists files/dirs that should NOT be cleaned from /data.
var systemFiles = map[string]bool{
	".skills":     true,
	"IDENTITY.md": true,
	"SOUL.md":     true,
	"TOOLS.md":    true,
	"mcp":         true,
}

// CleanBotFiles removes non-system files from the bot's /data directory.
// POST /bots/:bot_id/files/clean
func (h *ContainerdHandler) CleanBotFiles(c echo.Context) error {
	botID := strings.TrimSpace(c.Param("bot_id"))
	if botID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "bot_id required")
	}
	dataDir, err := h.ensureBotDataRoot(botID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	entries, err := os.ReadDir(dataDir)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	var deletedCount int
	var freedBytes int64
	for _, entry := range entries {
		name := entry.Name()
		if systemFiles[name] || strings.HasPrefix(name, ".") {
			continue
		}
		full := filepath.Join(dataDir, name)
		info, _ := entry.Info()
		var size int64
		if info != nil {
			size = info.Size()
		}
		if err := os.RemoveAll(full); err == nil {
			deletedCount++
			freedBytes += size
		}
	}
	return c.JSON(http.StatusOK, map[string]any{
		"deleted_count": deletedCount,
		"freed_bytes":   freedBytes,
	})
}

// cleanBotDataDir removes non-system files from a bot's data directory.
// Called automatically during container setup/rebuild.
func (h *ContainerdHandler) cleanBotDataDir(dataDir string) {
	entries, err := os.ReadDir(dataDir)
	if err != nil {
		return
	}
	for _, entry := range entries {
		name := entry.Name()
		if systemFiles[name] || strings.HasPrefix(name, ".") {
			continue
		}
		_ = os.RemoveAll(filepath.Join(dataDir, name))
	}
}
