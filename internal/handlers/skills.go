package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	ctr "github.com/Kxiandaoyan/Memoh-v2/internal/containerd"
	"github.com/Kxiandaoyan/Memoh-v2/internal/mcp"
	"github.com/Kxiandaoyan/Memoh-v2/internal/skills"
	"github.com/labstack/echo/v4"
	"gopkg.in/yaml.v3"
)

type SkillItem struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Content     string         `json:"content"`
	Metadata    map[string]any `json:"metadata,omitempty"`
	Order       int            `json:"order,omitempty"`
	Enabled     bool           `json:"enabled,omitempty"`
	Version     string         `json:"version,omitempty"`
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

type SkillToggleRequest struct {
	Enabled bool `json:"enabled"`
}

type SkillOrderRequest struct {
	Skills []SkillOrderItem `json:"skills"`
}

type SkillOrderItem struct {
	Name  string `json:"name"`
	Order int    `json:"order"`
}

// loadBotSkillConfig loads the skill configuration for a specific bot.
func (h *ContainerdHandler) loadBotSkillConfig(botID string) *skills.SkillConfig {
	skillsDir, err := h.ensureSkillsDirHost(botID)
	if err != nil {
		return skills.LoadDefaultSkillConfig()
	}
	return skills.LoadSkillConfig(filepath.Join(skillsDir, "skills.config.json"))
}

// saveBotSkillConfig saves the skill configuration to the bot's data directory.
func (h *ContainerdHandler) saveBotSkillConfig(config *skills.SkillConfig, botID string) error {
	skillsDir, err := h.ensureSkillsDirHost(botID)
	if err != nil {
		return err
	}
	return skills.SaveSkillConfigTo(config, filepath.Join(skillsDir, "skills.config.json"))
}

// getDefaultSkillsDir returns the path to the default skills directory.
func (h *ContainerdHandler) getDefaultSkillsDir() string {
	return filepath.Join("internal", "skills", "defaults")
}

// mergedSkillEntry represents a skill entry with source tracking.
type mergedSkillEntry struct {
	entry     skillEntry
	isFromBot bool
}

// mergeSkillEntries merges bot-specific and default skill entries.
func mergeSkillEntries(botEntries, defaultEntries []skillEntry) []mergedSkillEntry {
	botMap := make(map[string]struct{})
	for _, entry := range botEntries {
		_, name := skillPathForEntry(entry)
		if name != "" {
			botMap[name] = struct{}{}
		}
	}
	merged := make([]mergedSkillEntry, 0, len(botEntries)+len(defaultEntries))
	for _, entry := range botEntries {
		merged = append(merged, mergedSkillEntry{entry: entry, isFromBot: true})
	}
	for _, entry := range defaultEntries {
		_, name := skillPathForEntry(entry)
		if name == "" {
			continue
		}
		if _, exists := botMap[name]; !exists {
			merged = append(merged, mergedSkillEntry{entry: entry, isFromBot: false})
		}
	}
	return merged
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

	botConfig := h.loadBotSkillConfig(botID)
	botSkillsDir, err := h.ensureSkillsDirHost(botID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	defaultsDir := h.getDefaultSkillsDir()

	botEntries, err := listSkillEntries(botSkillsDir)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	var defaultEntries []skillEntry
	if defaultsDir != "" {
		defaultEntries, _ = listSkillEntries(defaultsDir)
	}

	mergedEntries := mergeSkillEntries(botEntries, defaultEntries)

	result := make([]SkillItem, 0, len(mergedEntries))
	for _, entryInfo := range mergedEntries {
		skillPath, name := skillPathForEntry(entryInfo.entry)
		if skillPath == "" {
			continue
		}
		var raw string
		if entryInfo.isFromBot {
			raw, err = h.readSkillFile(botSkillsDir, skillPath)
		} else {
			raw, err = h.readSkillFile(defaultsDir, skillPath)
		}
		if err != nil {
			continue
		}
		parsed := parseSkillFile(raw, name)
		configEntry := botConfig.GetSkillEntry(parsed.Name, parsed.Metadata)
		result = append(result, SkillItem{
			Name:        parsed.Name,
			Description: parsed.Description,
			Content:     parsed.Content,
			Metadata:    parsed.Metadata,
			Order:       configEntry.Order,
			Enabled:     configEntry.Enabled,
			Version:     parsed.Version,
		})
	}

	sort.Slice(result, func(i, j int) bool { return result[i].Order < result[j].Order })
	return c.JSON(http.StatusOK, SkillsResponse{Skills: result})
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
	h.logger.Info("UpsertSkills: starting", slog.String("bot_id", botID))
	var req SkillsUpsertRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	if len(req.Skills) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "skills is required")
	}

	skillsDir, err := h.ensureSkillsDirHost(botID)
	if err != nil {
		h.logger.Warn("UpsertSkills: ensure skills dir failed", slog.String("bot_id", botID), slog.Any("error", err))
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	for _, skill := range req.Skills {
		name := strings.TrimSpace(skill.Name)
		if !isValidSkillName(name) {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid skill name")
		}
		content := strings.TrimSpace(skill.Content)
		if content == "" {
			content = buildSkillContent(name, strings.TrimSpace(skill.Description), skill.Version)
		}
		dirPath := filepath.Join(skillsDir, name)
		if err := os.MkdirAll(dirPath, 0o755); err != nil {
			h.logger.Warn("UpsertSkills: mkdir failed", slog.String("skill", skill.Name), slog.Any("error", err))
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
		filePath := filepath.Join(dirPath, "SKILL.md")
		if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
			h.logger.Warn("UpsertSkills: write file failed", slog.String("skill", skill.Name), slog.Any("error", err))
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
	}

	h.logger.Info("UpsertSkills: completed", slog.String("bot_id", botID), slog.Int("count", len(req.Skills)))
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
	h.logger.Info("DeleteSkills: starting", slog.String("bot_id", botID))
	var req SkillsDeleteRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	if len(req.Names) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "names is required")
	}

	skillsDir, err := h.ensureSkillsDirHost(botID)
	if err != nil {
		h.logger.Warn("DeleteSkills: ensure skills dir failed", slog.String("bot_id", botID), slog.Any("error", err))
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	for _, name := range req.Names {
		skillName := strings.TrimSpace(name)
		if !isValidSkillName(skillName) {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid skill name")
		}
		deletePath := filepath.Join(skillsDir, skillName)
		if err := os.RemoveAll(deletePath); err != nil {
			h.logger.Warn("DeleteSkills: remove failed", slog.String("skill", name), slog.Any("error", err))
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
	}

	h.logger.Info("DeleteSkills: completed", slog.String("bot_id", botID), slog.Int("count", len(req.Names)))
	return c.JSON(http.StatusOK, skillsOpResponse{OK: true})
}

// LoadSkills loads all skills from the container for the given bot.
// This implements chat.SkillLoader.
func (h *ContainerdHandler) LoadSkills(ctx context.Context, botID string) ([]SkillItem, error) {
	botConfig := h.loadBotSkillConfig(botID)
	skillsDir, err := h.ensureSkillsDirHost(botID)
	if err != nil {
		return nil, err
	}

	entries, err := listSkillEntries(skillsDir)
	if err != nil {
		return nil, err
	}

	result := make([]SkillItem, 0, len(entries))
	for _, entry := range entries {
		skillPath, name := skillPathForEntry(entry)
		if skillPath == "" {
			continue
		}
		raw, err := h.readSkillFile(skillsDir, skillPath)
		if err != nil {
			continue
		}
		parsed := parseSkillFile(raw, name)
		configEntry := botConfig.GetSkillEntry(parsed.Name, parsed.Metadata)
		if !configEntry.Enabled {
			continue
		}
		result = append(result, SkillItem{
			Name:        parsed.Name,
			Description: parsed.Description,
			Content:     parsed.Content,
			Metadata:    parsed.Metadata,
			Order:       configEntry.Order,
			Enabled:     configEntry.Enabled,
			Version:     parsed.Version,
		})
	}

	sort.Slice(result, func(i, j int) bool { return result[i].Order < result[j].Order })
	return result, nil
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
	Version     string
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
	result := parsedSkill{Name: fallbackName, Version: "1.0.0"}

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
		Version     string         `yaml:"version"`
		Metadata    map[string]any `yaml:"metadata"`
	}
	if err := yaml.Unmarshal([]byte(frontmatterRaw), &fm); err != nil {
		return result
	}

	if strings.TrimSpace(fm.Name) != "" {
		result.Name = strings.TrimSpace(fm.Name)
	}
	result.Description = strings.TrimSpace(fm.Description)
	if strings.TrimSpace(fm.Version) != "" {
		result.Version = strings.TrimSpace(fm.Version)
	}
	result.Metadata = fm.Metadata

	return result
}

func buildSkillContent(name, description, version string) string {
	if description == "" {
		description = name
	}
	if version == "" {
		version = "1.0.0"
	}
	return fmt.Sprintf("---\nname: %s\ndescription: %s\nversion: %s\n---\n\n# %s\n\n%s", name, description, version, name, description)
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

// ToggleSkill godoc
// @Summary Toggle a skill's enabled state
// @Tags containerd
// @Param bot_id path string true "Bot ID"
// @Param name path string true "Skill name"
// @Param payload body SkillToggleRequest true "Toggle request"
// @Success 200 {object} skillsOpResponse
// @Router /bots/{bot_id}/container/skills/{name} [patch]
func (h *ContainerdHandler) ToggleSkill(c echo.Context) error {
	botID, err := h.requireBotAccess(c)
	if err != nil {
		return err
	}

	skillName := strings.TrimSpace(c.Param("name"))
	if !isValidSkillName(skillName) {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid skill name")
	}

	var req SkillToggleRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	h.logger.Info("ToggleSkill", slog.String("bot_id", botID), slog.String("skill", skillName), slog.Bool("enabled", req.Enabled))

	skillsDir, err := h.ensureSkillsDirHost(botID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	if _, err := os.Stat(filepath.Join(skillsDir, skillName)); os.IsNotExist(err) {
		if _, err := os.Stat(filepath.Join(h.getDefaultSkillsDir(), skillName)); os.IsNotExist(err) {
			return echo.NewHTTPError(http.StatusNotFound, "skill not found")
		}
	}

	botConfig := h.loadBotSkillConfig(botID)
	if err := botConfig.UpdateSkillEnabled(skillName, req.Enabled); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	if err := h.saveBotSkillConfig(botConfig, botID); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, skillsOpResponse{OK: true})
}

// UpdateSkillOrder godoc
// @Summary Update the order of skills
// @Tags containerd
// @Param bot_id path string true "Bot ID"
// @Param payload body SkillOrderRequest true "Order update request"
// @Success 200 {object} skillsOpResponse
// @Router /bots/{bot_id}/container/skills/order [put]
func (h *ContainerdHandler) UpdateSkillOrder(c echo.Context) error {
	botID, err := h.requireBotAccess(c)
	if err != nil {
		return err
	}

	var req SkillOrderRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	if len(req.Skills) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "skills array is required")
	}

	skillsDir, err := h.ensureSkillsDirHost(botID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	for _, item := range req.Skills {
		if !isValidSkillName(item.Name) {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid skill name: "+item.Name)
		}
		if item.Order < 0 {
			return echo.NewHTTPError(http.StatusBadRequest, "order must be non-negative")
		}
		if _, err := os.Stat(filepath.Join(skillsDir, item.Name)); os.IsNotExist(err) {
			if _, err := os.Stat(filepath.Join(h.getDefaultSkillsDir(), item.Name)); os.IsNotExist(err) {
				return echo.NewHTTPError(http.StatusNotFound, "skill not found: "+item.Name)
			}
		}
	}

	botConfig := h.loadBotSkillConfig(botID)
	for _, item := range req.Skills {
		if err := botConfig.UpdateSkillOrder(item.Name, item.Order); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
	}

	if err := h.saveBotSkillConfig(botConfig, botID); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, skillsOpResponse{OK: true})
}

// SyncDefaultSkills godoc
// @Summary Sync default skills to bot directory
// @Tags containerd
// @Param bot_id path string true "Bot ID"
// @Param force query bool false "Force overwrite existing skills"
// @Success 200 {object} map[string]any
// @Router /bots/{bot_id}/container/skills/sync [post]
func (h *ContainerdHandler) SyncDefaultSkills(c echo.Context) error {
	botID, err := h.requireBotAccess(c)
	if err != nil {
		return err
	}

	force := c.QueryParam("force") == "true"

	botSkillsDir, err := h.ensureSkillsDirHost(botID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	defaultsDir := h.getDefaultSkillsDir()
	if defaultsDir == "" {
		return echo.NewHTTPError(http.StatusInternalServerError, "defaults directory not found")
	}

	count, err := skills.SyncDefaultSkills(botSkillsDir, defaultsDir, force)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, map[string]any{"synced": true, "count": count})
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
	h.logger.Info("ClawHubSearch: starting")

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
		h.logger.Warn("ClawHubSearch: exec failed", slog.Any("error", err))
		return echo.NewHTTPError(http.StatusInternalServerError, "exec failed: "+err.Error())
	}
	if result.ExitCode != 0 {
		h.logger.Warn("ClawHubSearch: non-zero exit", slog.Uint64("exit_code", uint64(result.ExitCode)))
		errMsg := strings.TrimSpace(stderr.String())
		if errMsg == "" {
			errMsg = strings.TrimSpace(stdout.String())
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "clawhub search failed: "+errMsg)
	}

	// Try to parse as JSON array; if it fails, return raw output
	var results []clawHubSearchResult
	if err := json.Unmarshal(stdout.Bytes(), &results); err != nil {
		h.logger.Warn("ClawHubSearch: parse failed", slog.Any("error", err))
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
	h.logger.Info("ClawHubInstall: starting", slog.String("slug", slug))

	containerID := mcp.ContainerPrefix + botID

	var stdout, stderr bytes.Buffer
	result, err := h.service.ExecTask(c.Request().Context(), containerID, ctr.ExecTaskRequest{
		Args:   []string{"clawhub", "install", slug, "--dir", "/data/.skills"},
		Stdout: &stdout,
		Stderr: &stderr,
	})
	if err != nil {
		h.logger.Warn("ClawHubInstall: exec failed", slog.Any("error", err))
		return echo.NewHTTPError(http.StatusInternalServerError, "exec failed: "+err.Error())
	}
	if result.ExitCode != 0 {
		h.logger.Warn("ClawHubInstall: non-zero exit", slog.Uint64("exit_code", uint64(result.ExitCode)))
		errMsg := strings.TrimSpace(stderr.String())
		if errMsg == "" {
			errMsg = strings.TrimSpace(stdout.String())
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "clawhub install failed: "+errMsg)
	}

	h.logger.Info("ClawHubInstall: completed", slog.String("slug", slug))
	return c.JSON(http.StatusOK, map[string]any{
		"ok":      true,
		"message": strings.TrimSpace(stdout.String()),
	})
}
