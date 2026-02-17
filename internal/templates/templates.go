package templates

import (
	"embed"
	"fmt"
	"sync"
)

//go:embed */identity.md */soul.md */task.md
var templateFS embed.FS

// Template represents a predefined bot persona template.
type Template struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Icon        string `json:"icon"`
	Category    string `json:"category"`
	Identity    string `json:"identity"`
	Soul        string `json:"soul"`
	Task        string `json:"task"`
}

// ListTemplatesResponse wraps a list of templates.
type ListTemplatesResponse struct {
	Items []Template `json:"items"`
}

// templateMeta holds the static metadata for each template.
// Content (identity/soul/task) is loaded from embedded .md files.
type templateMeta struct {
	ID          string
	Name        string
	Description string
	Icon        string
	Category    string
}

var allMetas = []templateMeta{
	{
		ID:          "research-analyst",
		Name:        "Research Analyst",
		Description: "Deep web researcher with critical analysis skills. Finds, verifies, and synthesizes information from multiple sources.",
		Icon:        "magnifying-glass",
		Category:    "productivity",
	},
	{
		ID:          "code-architect",
		Name:        "Code Architect",
		Description: "Pragmatic full-stack developer. Builds clean, maintainable software with a bias toward simplicity over complexity.",
		Icon:        "code",
		Category:    "development",
	},
	{
		ID:          "writing-editor",
		Name:        "Writing Editor",
		Description: "Meticulous editor and writing coach. Helps draft, revise, and polish written content with clarity and precision.",
		Icon:        "pen-nib",
		Category:    "creative",
	},
	{
		ID:          "daily-secretary",
		Name:        "Daily Secretary",
		Description: "Efficient personal assistant. Manages tasks, tracks commitments, and keeps your daily operations running smoothly.",
		Icon:        "calendar-check",
		Category:    "productivity",
	},
	{
		ID:          "data-wrangler",
		Name:        "Data Wrangler",
		Description: "Analytical data partner. Turns messy data into clear insights and helps make data-driven decisions.",
		Icon:        "chart-bar",
		Category:    "development",
	},
	{
		ID:          "knowledge-curator",
		Name:        "Knowledge Curator",
		Description: "Personal knowledge architect. Captures, organizes, connects, and retrieves information — your second brain.",
		Icon:        "book-open",
		Category:    "productivity",
	},
	{
		ID:          "language-tutor",
		Name:        "Language Tutor",
		Description: "Patient language learning partner. Helps with vocabulary, grammar, and conversation practice at your pace.",
		Icon:        "language",
		Category:    "education",
	},
	{
		ID:          "creative-muse",
		Name:        "Creative Muse",
		Description: "Brainstorming and ideation partner. Generates ideas, challenges assumptions, and pushes past conventional thinking.",
		Icon:        "lightbulb",
		Category:    "creative",
	},
	{
		ID:          "ops-monitor",
		Name:        "Ops Monitor",
		Description: "Systems reliability partner. Helps monitor, troubleshoot, and improve infrastructure with a reliability-first mindset.",
		Icon:        "server",
		Category:    "development",
	},
	{
		ID:          "life-strategist",
		Name:        "Life Strategist",
		Description: "Personal growth partner. Helps with goal-setting, decision-making, and structured reflection for long-term improvement.",
		Icon:        "compass",
		Category:    "personal",
	},
}

var (
	registry     []Template
	registryOnce sync.Once
	registryErr  error
)

func loadRegistry() {
	registry = make([]Template, 0, len(allMetas))
	for _, m := range allMetas {
		identity, err := templateFS.ReadFile(m.ID + "/identity.md")
		if err != nil {
			registryErr = fmt.Errorf("load template %s/identity.md: %w", m.ID, err)
			return
		}
		soul, err := templateFS.ReadFile(m.ID + "/soul.md")
		if err != nil {
			registryErr = fmt.Errorf("load template %s/soul.md: %w", m.ID, err)
			return
		}
		task, err := templateFS.ReadFile(m.ID + "/task.md")
		if err != nil {
			registryErr = fmt.Errorf("load template %s/task.md: %w", m.ID, err)
			return
		}
		registry = append(registry, Template{
			ID:          m.ID,
			Name:        m.Name,
			Description: m.Description,
			Icon:        m.Icon,
			Category:    m.Category,
			Identity:    string(identity),
			Soul:        string(soul),
			Task:        string(task),
		})
	}
}

// List returns all available templates.
func List() ([]Template, error) {
	registryOnce.Do(loadRegistry)
	if registryErr != nil {
		return nil, registryErr
	}
	return registry, nil
}

// Get returns a template by ID, or an error if not found.
func Get(id string) (Template, error) {
	registryOnce.Do(loadRegistry)
	if registryErr != nil {
		return Template{}, registryErr
	}
	for _, t := range registry {
		if t.ID == id {
			return t, nil
		}
	}
	return Template{}, fmt.Errorf("template not found: %s", id)
}
