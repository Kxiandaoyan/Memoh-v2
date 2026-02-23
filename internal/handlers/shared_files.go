package handlers

import (
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

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
	g.GET("/*", h.Read)
	g.PUT("/*", h.Write)
	g.DELETE("/*", h.Delete)
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
