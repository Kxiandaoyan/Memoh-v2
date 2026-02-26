package handlers

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/Kxiandaoyan/Memoh-v2/internal/config"
)

// SharedFileEntry represents a file or directory in the shared workspace.
type SharedFileEntry struct {
	Name      string `json:"name"`
	Size      int64  `json:"size"`
	IsDir     bool   `json:"is_dir"`
	UpdatedAt string `json:"updated_at"`
}

// SharedFilesHandler serves the cross-bot shared workspace directory.
type SharedFilesHandler struct {
	dataRoot string
}

// NewSharedFilesHandler creates a handler for the shared workspace.
func NewSharedFilesHandler(cfg config.MCPConfig) *SharedFilesHandler {
	root := strings.TrimSpace(cfg.DataRoot)
	if root == "" {
		root = config.DefaultDataRoot
	}
	return &SharedFilesHandler{dataRoot: root}
}

// Register mounts routes on the Echo instance.
func (h *SharedFilesHandler) Register(e *echo.Echo) {
	g := e.Group("/shared/files")
	g.GET("", h.List)
	g.GET("/download/*", h.Download)
	g.GET("/preview/*", h.Preview)
	g.GET("/*", h.Read)
	g.PUT("/*", h.Write)
	g.DELETE("/*", h.Delete)
	g.POST("/upload", h.Upload)
	g.POST("/rename", h.Rename)
	g.POST("/mkdir", h.Mkdir)
	g.POST("/move", h.Move)
	g.POST("/copy", h.Copy)
	g.POST("/batch-delete", h.BatchDelete)
}

func (h *SharedFilesHandler) sharedDir() (string, error) {
	abs, err := filepath.Abs(h.dataRoot)
	if err != nil {
		return "", err
	}
	dir := filepath.Join(abs, "shared")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return dir, nil
}

// validateSharedPath ensures the resolved path stays within the shared root
// and the file extension is allowed.
func validateSharedPath(root, relPath string) (string, error) {
	relPath = strings.TrimSpace(relPath)
	relPath = filepath.Clean(relPath)
	if relPath == "" || relPath == "." {
		return root, nil
	}
	full := filepath.Join(root, relPath)
	// Ensure the resolved path is strictly within root (add separator to
	// prevent "/data/shared" matching "/data/shared-evil/…").
	rootPrefix := root + string(filepath.Separator)
	if !strings.HasPrefix(full, rootPrefix) && full != root {
		return "", echo.NewHTTPError(http.StatusBadRequest, "invalid path")
	}
	return full, nil
}

// List returns files and directories at the given path (query param ?path=subdir).
func (h *SharedFilesHandler) List(c echo.Context) error {
	root, err := h.sharedDir()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	subPath := c.QueryParam("path")
	dir, err := validateSharedPath(root, subPath)
	if err != nil {
		return err
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return c.JSON(http.StatusOK, map[string]any{"files": []SharedFileEntry{}, "path": subPath})
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	var files []SharedFileEntry
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		info, infoErr := entry.Info()
		if infoErr != nil {
			continue
		}
		if entry.IsDir() {
			files = append(files, SharedFileEntry{
				Name:      entry.Name(),
				IsDir:     true,
				UpdatedAt: info.ModTime().UTC().Format(time.RFC3339),
			})
			continue
		}
		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if !allowedExtensions[ext] {
			continue
		}
		files = append(files, SharedFileEntry{
			Name:      entry.Name(),
			Size:      info.Size(),
			UpdatedAt: info.ModTime().UTC().Format(time.RFC3339),
		})
	}
	sort.Slice(files, func(i, j int) bool {
		if files[i].IsDir != files[j].IsDir {
			return files[i].IsDir
		}
		return files[i].Name < files[j].Name
	})
	return c.JSON(http.StatusOK, map[string]any{"files": files, "path": subPath})
}

// Read returns the content of a file in the shared workspace.
func (h *SharedFilesHandler) Read(c echo.Context) error {
	root, err := h.sharedDir()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	relPath := c.Param("*")
	full, err := validateSharedPath(root, relPath)
	if err != nil {
		return err
	}
	ext := strings.ToLower(filepath.Ext(full))
	if !allowedExtensions[ext] {
		return echo.NewHTTPError(http.StatusBadRequest, "unsupported file type")
	}
	data, err := os.ReadFile(full)
	if err != nil {
		if os.IsNotExist(err) {
			return echo.NewHTTPError(http.StatusNotFound, "file not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, BotFileContent{
		Name:    filepath.Base(full),
		Content: string(data),
		Size:    int64(len(data)),
	})
}

// Write creates or updates a file in the shared workspace.
func (h *SharedFilesHandler) Write(c echo.Context) error {
	root, err := h.sharedDir()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	relPath := c.Param("*")
	full, err := validateSharedPath(root, relPath)
	if err != nil {
		return err
	}
	ext := strings.ToLower(filepath.Ext(full))
	if !allowedExtensions[ext] {
		return echo.NewHTTPError(http.StatusBadRequest, "unsupported file type")
	}

	var req BotFileWriteRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if len(req.Content) > 1<<20 {
		return echo.NewHTTPError(http.StatusBadRequest, "content too large (max 1MB)")
	}

	if dir := filepath.Dir(full); dir != root {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
	}

	if err := os.WriteFile(full, []byte(req.Content), 0o644); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, BotFileContent{
		Name:    filepath.Base(full),
		Content: req.Content,
		Size:    int64(len(req.Content)),
	})
}

// Download serves a file from the shared workspace as a direct download.
func (h *SharedFilesHandler) Download(c echo.Context) error {
	root, err := h.sharedDir()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	relPath := c.Param("*")
	full, err := validateSharedPath(root, relPath)
	if err != nil {
		return err
	}
	info, statErr := os.Stat(full)
	if statErr != nil || info.IsDir() {
		return echo.NewHTTPError(http.StatusNotFound, "file not found")
	}
	c.Response().Header().Set("Content-Disposition",
		"attachment; filename=\""+filepath.Base(full)+"\"")
	return c.File(full)
}

// Delete removes a file from the shared workspace.
func (h *SharedFilesHandler) Delete(c echo.Context) error {
	root, err := h.sharedDir()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	relPath := c.Param("*")
	full, err := validateSharedPath(root, relPath)
	if err != nil {
		return err
	}
	if full == root {
		return echo.NewHTTPError(http.StatusBadRequest, "cannot delete root")
	}
	if err := os.Remove(full); err != nil {
		if os.IsNotExist(err) {
			return echo.NewHTTPError(http.StatusNotFound, "file not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.NoContent(http.StatusNoContent)
}

// Upload handles multipart file upload. POST /shared/files/upload
func (h *SharedFilesHandler) Upload(c echo.Context) error {
	root, err := h.sharedDir()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	file, err := c.FormFile("file")
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "file field required")
	}
	if file.Size > 50<<20 {
		return echo.NewHTTPError(http.StatusBadRequest, "file too large (max 50MB)")
	}
	src, err := file.Open()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	defer src.Close()

	uploadsDir := filepath.Join(root, "uploads")
	if err := os.MkdirAll(uploadsDir, 0o755); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	safeName := filepath.Base(file.Filename)
	destName := fmt.Sprintf("%s_%s", uuid.New().String()[:8], safeName)
	destPath := filepath.Join(uploadsDir, destName)

	dst, err := os.Create(destPath)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	defer dst.Close()
	if _, err := io.Copy(dst, src); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, map[string]any{
		"name": safeName,
		"size": file.Size,
		"mime": file.Header.Get("Content-Type"),
		"path": "/shared/uploads/" + destName,
	})
}

// Preview serves a file inline for browser rendering. GET /shared/files/preview/*
func (h *SharedFilesHandler) Preview(c echo.Context) error {
	root, err := h.sharedDir()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	relPath := c.Param("*")
	full, err := validateSharedPath(root, relPath)
	if err != nil {
		return err
	}
	info, statErr := os.Stat(full)
	if statErr != nil || info.IsDir() {
		return echo.NewHTTPError(http.StatusNotFound, "file not found")
	}
	return c.File(full)
}

// Rename renames a file or directory. POST /shared/files/rename
func (h *SharedFilesHandler) Rename(c echo.Context) error {
	root, err := h.sharedDir()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	var req struct {
		From string `json:"from"`
		To   string `json:"to"`
	}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}
	fromFull, err := validateSharedPath(root, req.From)
	if err != nil {
		return err
	}
	toFull, err := validateSharedPath(root, req.To)
	if err != nil {
		return err
	}
	if err := os.Rename(fromFull, toFull); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

// Mkdir creates a directory. POST /shared/files/mkdir
func (h *SharedFilesHandler) Mkdir(c echo.Context) error {
	root, err := h.sharedDir()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	var req struct {
		Path string `json:"path"`
	}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}
	full, err := validateSharedPath(root, req.Path)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(full, 0o755); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

// Move moves a file or directory. POST /shared/files/move
func (h *SharedFilesHandler) Move(c echo.Context) error {
	root, err := h.sharedDir()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	var req struct {
		From string `json:"from"`
		To   string `json:"to"`
	}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}
	fromFull, err := validateSharedPath(root, req.From)
	if err != nil {
		return err
	}
	toFull, err := validateSharedPath(root, req.To)
	if err != nil {
		return err
	}
	if dir := filepath.Dir(toFull); dir != root {
		os.MkdirAll(dir, 0o755)
	}
	if err := os.Rename(fromFull, toFull); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

// Copy copies a file. POST /shared/files/copy
func (h *SharedFilesHandler) Copy(c echo.Context) error {
	root, err := h.sharedDir()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	var req struct {
		From string `json:"from"`
		To   string `json:"to"`
	}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}
	fromFull, err := validateSharedPath(root, req.From)
	if err != nil {
		return err
	}
	toFull, err := validateSharedPath(root, req.To)
	if err != nil {
		return err
	}
	data, err := os.ReadFile(fromFull)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "source not found")
	}
	if dir := filepath.Dir(toFull); dir != root {
		os.MkdirAll(dir, 0o755)
	}
	if err := os.WriteFile(toFull, data, 0o644); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

// BatchDelete removes multiple files/dirs. POST /shared/files/batch-delete
func (h *SharedFilesHandler) BatchDelete(c echo.Context) error {
	root, err := h.sharedDir()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	var req struct {
		Paths []string `json:"paths"`
	}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}
	deleted := 0
	for _, p := range req.Paths {
		full, vErr := validateSharedPath(root, p)
		if vErr != nil || full == root {
			continue
		}
		if err := os.RemoveAll(full); err == nil {
			deleted++
		}
	}
	return c.JSON(http.StatusOK, map[string]int{"deleted": deleted})
}
