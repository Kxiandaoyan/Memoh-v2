package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	ctr "github.com/Kxiandaoyan/Memoh-v2/internal/containerd"
	"github.com/Kxiandaoyan/Memoh-v2/internal/mcp"
	"github.com/labstack/echo/v4"
	"gopkg.in/yaml.v3"
)

type SkillItem struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Content     string         `json:"content"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

type SkillsResponse struct {
	Skills []SkillItem `json:"skills"`
}

type SkillsUpsertRequest struct {
	Skills []SkillItem `json:"skills"`
}

type SkillsDeleteRequest struct {
	Names []string `json:"names"`
}

type skillsOpResponse struct {
	OK bool `json:"ok"`
}

// ListSkills godoc
// @Summary List skills from data directory
// @Tags containerd
// @Param bot_id path string true "Bot ID"
// @Success 200 {object} SkillsResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /bots/{bot_id}/container/skills [get]
func (h *ContainerdHandler) ListSkills(c echo.Context) error {
	botID, err := h.requireBotAccess(c)
	if err != nil {
		return err
	}
	skillsDir, err := h.ensureSkillsDirHost(botID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	h.logger.Info("ListSkills: resolved skills dir",
		slog.String("bot_id", botID),
		slog.String("skills_dir", skillsDir))

	entries, err := listSkillEntries(skillsDir)
	if err != nil {
		h.logger.Warn("ListSkills: failed to list entries",
			slog.String("bot_id", botID),
			slog.Any("error", err))
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	entryNames := make([]string, 0, len(entries))
	for _, e := range entries {
		entryNames = append(entryNames, e.Path)
	}
	h.logger.Info("ListSkills: found entries",
		slog.String("bot_id", botID),
		slog.Int("count", len(entries)),
		slog.Any("entries", entryNames))

	skills := make([]SkillItem, 0, len(entries))
	for _, entry := range entries {
		skillPath, name := skillPathForEntry(entry)
		if skillPath == "" {
			h.logger.Warn("ListSkills: skillPathForEntry returned empty",
				slog.String("bot_id", botID),
				slog.String("entry_path", entry.Path),
				slog.Bool("is_dir", entry.IsDir))
			continue
		}
		raw, err := h.readSkillFile(skillsDir, skillPath)
		if err != nil {
			h.logger.Warn("ListSkills: readSkillFile failed",
				slog.String("bot_id", botID),
				slog.String("skill_path", skillPath),
				slog.Any("error", err))
			continue
		}
		parsed := parseSkillFile(raw, name)
		skills = append(skills, SkillItem{
			Name:        parsed.Name,
			Description: parsed.Description,
			Content:     parsed.Content,
			Metadata:    parsed.Metadata,
		})
	}

	h.logger.Info("ListSkills: returning skills",
		slog.String("bot_id", botID),
		slog.Int("total", len(skills)))
	return c.JSON(http.StatusOK, SkillsResponse{Skills: skills})
}

// UpsertSkills godoc
// @Summary Upload skills into data directory
// @Tags containerd
// @Param bot_id path string true "Bot ID"
// @Param payload body SkillsUpsertRequest true "Skills payload"
// @Success 200 {object} skillsOpResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /bots/{bot_id}/container/skills [post]
func (h *ContainerdHandler) UpsertSkills(c echo.Context) error {
	botID, err := h.requireBotAccess(c)
	if err != nil {
		return err
	}
	var req SkillsUpsertRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	if len(req.Skills) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "skills is required")
	}

	skillsDir, err := h.ensureSkillsDirHost(botID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	for _, skill := range req.Skills {
		name := strings.TrimSpace(skill.Name)
		if !isValidSkillName(name) {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid skill name")
		}
		content := strings.TrimSpace(skill.Content)
		if content == "" {
			content = buildSkillContent(name, strings.TrimSpace(skill.Description))
		}
		dirPath := filepath.Join(skillsDir, name)
		if err := os.MkdirAll(dirPath, 0o755); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
		filePath := filepath.Join(dirPath, "SKILL.md")
		if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
	}

	return c.JSON(http.StatusOK, skillsOpResponse{OK: true})
}

// DeleteSkills godoc
// @Summary Delete skills from data directory
// @Tags containerd
// @Param bot_id path string true "Bot ID"
// @Param payload body SkillsDeleteRequest true "Delete skills payload"
// @Success 200 {object} skillsOpResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /bots/{bot_id}/container/skills [delete]
func (h *ContainerdHandler) DeleteSkills(c echo.Context) error {
	botID, err := h.requireBotAccess(c)
	if err != nil {
		return err
	}
	var req SkillsDeleteRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	if len(req.Names) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "names is required")
	}

	skillsDir, err := h.ensureSkillsDirHost(botID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	for _, name := range req.Names {
		skillName := strings.TrimSpace(name)
		if !isValidSkillName(skillName) {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid skill name")
		}
		deletePath := filepath.Join(skillsDir, skillName)
		if err := os.RemoveAll(deletePath); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
	}

	return c.JSON(http.StatusOK, skillsOpResponse{OK: true})
}

// LoadSkills loads all skills from the container for the given bot.
// This implements chat.SkillLoader.
func (h *ContainerdHandler) LoadSkills(ctx context.Context, botID string) ([]SkillItem, error) {
	skillsDir, err := h.ensureSkillsDirHost(botID)
	if err != nil {
		return nil, err
	}

	entries, err := listSkillEntries(skillsDir)
	if err != nil {
		return nil, err
	}

	skills := make([]SkillItem, 0, len(entries))
	for _, entry := range entries {
		skillPath, name := skillPathForEntry(entry)
		if skillPath == "" {
			continue
		}
		raw, err := h.readSkillFile(skillsDir, skillPath)
		if err != nil {
			h.logger.Warn("LoadSkills: readSkillFile failed",
				slog.String("bot_id", botID),
				slog.String("skill_path", skillPath),
				slog.Any("error", err))
			continue
		}
		parsed := parseSkillFile(raw, name)
		skills = append(skills, SkillItem{
			Name:        parsed.Name,
			Description: parsed.Description,
			Content:     parsed.Content,
			Metadata:    parsed.Metadata,
		})
	}
	return skills, nil
}

func (h *ContainerdHandler) ensureSkillsDirHost(botID string) (string, error) {
	root, err := h.ensureBotDataRoot(botID)
	if err != nil {
		return "", err
	}
	skillsDir := filepath.Join(root, ".skills")
	if err := os.MkdirAll(skillsDir, 0o755); err != nil {
		return "", err
	}
	return skillsDir, nil
}

func (h *ContainerdHandler) readSkillFile(skillsDir, filePath string) (string, error) {
	safeRel := strings.TrimPrefix(strings.TrimPrefix(filePath, ".skills/"), "./.skills/")
	if safeRel == "" {
		return "", os.ErrInvalid
	}
	target := filepath.Join(skillsDir, filepath.FromSlash(safeRel))
	data, err := os.ReadFile(target)
	if err == nil {
		return string(data), nil
	}
	if !os.IsNotExist(err) {
		return "", err
	}

	// Primary file not found — try alternative filenames in the same directory.
	dir := filepath.Dir(target)
	for _, candidate := range skillFileCandidates {
		alt := filepath.Join(dir, candidate)
		if alt == target {
			continue
		}
		data, err := os.ReadFile(alt)
		if err == nil {
			return string(data), nil
		}
	}

	// Last resort: read the first .md file in the directory.
	dirEntries, readErr := os.ReadDir(dir)
	if readErr != nil {
		return "", err
	}
	for _, de := range dirEntries {
		if de.IsDir() {
			continue
		}
		if strings.HasSuffix(strings.ToLower(de.Name()), ".md") {
			data, readErr := os.ReadFile(filepath.Join(dir, de.Name()))
			if readErr == nil {
				return string(data), nil
			}
		}
	}

	return "", err
}

func listSkillEntries(skillsDir string) ([]skillEntry, error) {
	dirEntries, err := os.ReadDir(skillsDir)
	if err != nil {
		return nil, err
	}
	entries := make([]skillEntry, 0, len(dirEntries))
	for _, entry := range dirEntries {
		name := entry.Name()
		if name == "" {
			continue
		}
		if entry.IsDir() {
			entries = append(entries, skillEntry{
				Path:  path.Join(".skills", name),
				IsDir: true,
			})
			continue
		}
		if name == "SKILL.md" {
			entries = append(entries, skillEntry{
				Path:  path.Join(".skills", name),
				IsDir: false,
			})
		}
	}
	return entries, nil
}

type skillEntry struct {
	Path  string
	IsDir bool
}

func skillNameFromPath(rel string) string {
	if rel == "" || rel == "SKILL.md" {
		return "default"
	}
	parent := path.Dir(rel)
	if parent == "." {
		return "default"
	}
	return path.Base(parent)
}

func skillPathForEntry(entry skillEntry) (string, string) {
	rel := strings.TrimPrefix(entry.Path, ".skills/")
	if rel == entry.Path {
		rel = strings.TrimPrefix(entry.Path, "./.skills/")
	}
	if entry.IsDir {
		name := path.Base(rel)
		if name == "." || name == "" {
			return "", ""
		}
		return path.Join(".skills", name, "SKILL.md"), name
	}
	if path.Base(rel) == "SKILL.md" {
		return path.Join(".skills", "SKILL.md"), skillNameFromPath(rel)
	}
	return "", ""
}

// skillFileCandidates lists filenames to try when looking for a skill definition,
// ordered by priority. This handles cases where bots create skills with
// non-standard filenames via conversation.
var skillFileCandidates = []string{
	"SKILL.md",
	"skill.md",
	"Skill.md",
	"README.md",
	"readme.md",
	"index.md",
}

// parsedSkill holds the result of parsing a SKILL.md file with YAML frontmatter.
type parsedSkill struct {
	Name        string
	Description string
	Content     string         // body after frontmatter
	Metadata    map[string]any // "metadata" key from frontmatter
}

// parseSkillFile parses a SKILL.md file with YAML frontmatter delimited by "---".
// Format:
//
//	---
//	name: your-skill-name
//	description: Brief description
//	metadata:
//	  key: value
//	---
//	# Body content ...
func parseSkillFile(raw string, fallbackName string) parsedSkill {
	result := parsedSkill{Name: fallbackName}

	trimmed := strings.TrimSpace(raw)
	if !strings.HasPrefix(trimmed, "---") {
		result.Content = trimmed
		return result
	}

	// Find closing "---".
	rest := trimmed[3:]
	rest = strings.TrimLeft(rest, " \t")
	if len(rest) > 0 && rest[0] == '\n' {
		rest = rest[1:]
	} else if len(rest) > 1 && rest[0] == '\r' && rest[1] == '\n' {
		rest = rest[2:]
	}
	closingIdx := strings.Index(rest, "\n---")
	if closingIdx < 0 {
		return result
	}

	frontmatterRaw := rest[:closingIdx]
	body := rest[closingIdx+4:]
	body = strings.TrimLeft(body, "\r\n")
	result.Content = body

	var fm struct {
		Name        string         `yaml:"name"`
		Description string         `yaml:"description"`
		Metadata    map[string]any `yaml:"metadata"`
	}
	if err := yaml.Unmarshal([]byte(frontmatterRaw), &fm); err != nil {
		return result
	}

	if strings.TrimSpace(fm.Name) != "" {
		result.Name = strings.TrimSpace(fm.Name)
	}
	result.Description = strings.TrimSpace(fm.Description)
	result.Metadata = fm.Metadata

	return result
}

func buildSkillContent(name, description string) string {
	if description == "" {
		description = name
	}
	return "---\nname: " + name + "\ndescription: " + description + "\n---\n\n# " + name + "\n\n" + description
}

func isValidSkillName(name string) bool {
	if name == "" {
		return false
	}
	if strings.Contains(name, "..") {
		return false
	}
	if strings.Contains(name, "/") || strings.Contains(name, "\\") {
		return false
	}
	return true
}

// --- ClawHub marketplace proxy APIs ---

type clawHubSearchRequest struct {
	Query string `json:"query"`
}

type clawHubInstallRequest struct {
	Slug string `json:"slug"`
}

type clawHubSearchResult struct {
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description"`
	Author      string `json:"author"`
	Version     string `json:"version,omitempty"`
}

// ClawHubSearch godoc
// @Summary Search ClawHub skill marketplace
// @Tags containerd
// @Param bot_id path string true "Bot ID"
// @Param payload body clawHubSearchRequest true "Search query"
// @Success 200 {object} map[string]any
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /bots/{bot_id}/container/clawhub/search [post]
func (h *ContainerdHandler) ClawHubSearch(c echo.Context) error {
	botID, err := h.requireBotAccess(c)
	if err != nil {
		return err
	}

	var req clawHubSearchRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	query := strings.TrimSpace(req.Query)
	if query == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "query is required")
	}

	containerID := mcp.ContainerPrefix + botID

	var stdout, stderr bytes.Buffer
	result, err := h.service.ExecTask(c.Request().Context(), containerID, ctr.ExecTaskRequest{
		Args:   []string{"clawhub", "search", query, "--json"},
		Stdout: &stdout,
		Stderr: &stderr,
	})
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "exec failed: "+err.Error())
	}
	if result.ExitCode != 0 {
		errMsg := strings.TrimSpace(stderr.String())
		if errMsg == "" {
			errMsg = strings.TrimSpace(stdout.String())
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "clawhub search failed: "+errMsg)
	}

	// Try to parse as JSON array; if it fails, return raw output
	var results []clawHubSearchResult
	if err := json.Unmarshal(stdout.Bytes(), &results); err != nil {
		return c.JSON(http.StatusOK, map[string]any{
			"results": []any{},
			"raw":     strings.TrimSpace(stdout.String()),
		})
	}
	return c.JSON(http.StatusOK, map[string]any{"results": results})
}

// ClawHubInstall godoc
// @Summary Install a skill from ClawHub
// @Tags containerd
// @Param bot_id path string true "Bot ID"
// @Param payload body clawHubInstallRequest true "Skill slug"
// @Success 200 {object} map[string]any
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /bots/{bot_id}/container/clawhub/install [post]
func (h *ContainerdHandler) ClawHubInstall(c echo.Context) error {
	botID, err := h.requireBotAccess(c)
	if err != nil {
		return err
	}

	var req clawHubInstallRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	slug := strings.TrimSpace(req.Slug)
	if slug == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "slug is required")
	}
	if strings.ContainsAny(slug, ";|&$`") {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid slug")
	}

	containerID := mcp.ContainerPrefix + botID

	var stdout, stderr bytes.Buffer
	result, err := h.service.ExecTask(c.Request().Context(), containerID, ctr.ExecTaskRequest{
		Args:   []string{"clawhub", "install", slug, "--dir", "/data/.skills"},
		Stdout: &stdout,
		Stderr: &stderr,
	})
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "exec failed: "+err.Error())
	}
	if result.ExitCode != 0 {
		errMsg := strings.TrimSpace(stderr.String())
		if errMsg == "" {
			errMsg = strings.TrimSpace(stdout.String())
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "clawhub install failed: "+errMsg)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"ok":      true,
		"message": strings.TrimSpace(stdout.String()),
	})
}
