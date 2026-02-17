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
	// --- Solo-Company ports (10) ---
	{
		ID:          "ceo-bezos",
		Name:        "CEO 战略顾问",
		Description: "Jeff Bezos 思维模型。评估新产品/功能想法、商业模式和定价方向、重大战略选择、资源分配和优先级排序。",
		Icon:        "crown",
		Category:    "business",
	},
	{
		ID:          "cto-vogels",
		Name:        "CTO 架构师",
		Description: "Werner Vogels 思维模型。技术架构设计、技术选型决策、系统性能和可靠性评估、技术债务评估。",
		Icon:        "cpu",
		Category:    "development",
	},
	{
		ID:          "fullstack-dhh",
		Name:        "全栈开发",
		Description: "DHH 思维模型。写代码和实现功能、技术实现方案选择、代码审查和重构、开发工具和流程优化。",
		Icon:        "code",
		Category:    "development",
	},
	{
		ID:          "interaction-cooper",
		Name:        "交互设计",
		Description: "Alan Cooper 思维模型。设计用户流程和导航、定义目标用户画像、选择交互模式、从用户角度排序功能优先级。",
		Icon:        "cursor-click",
		Category:    "design",
	},
	{
		ID:          "marketing-godin",
		Name:        "营销策略",
		Description: "Seth Godin 思维模型。产品定位和差异化、制定营销策略、内容方向和传播计划、品牌建设。",
		Icon:        "megaphone",
		Category:    "business",
	},
	{
		ID:          "operations-pg",
		Name:        "运营增长",
		Description: "Paul Graham 思维模型。冷启动和早期用户获取、用户留存和活跃度提升、社区运营策略、运营数据分析。",
		Icon:        "rocket",
		Category:    "business",
	},
	{
		ID:          "product-norman",
		Name:        "产品设计",
		Description: "Don Norman 思维模型。定义产品功能和体验、评估设计方案的可用性、分析用户困惑或流失、规划可用性测试。",
		Icon:        "cube",
		Category:    "design",
	},
	{
		ID:          "qa-bach",
		Name:        "质量保证",
		Description: "James Bach 思维模型。制定测试策略、发布前质量检查、Bug 分析和分类、质量风险评估。",
		Icon:        "shield-check",
		Category:    "development",
	},
	{
		ID:          "sales-ross",
		Name:        "销售策略",
		Description: "Aaron Ross 思维模型。定价策略、销售模式选择、转化率优化、客户获取成本分析。",
		Icon:        "currency-dollar",
		Category:    "business",
	},
	{
		ID:          "ui-duarte",
		Name:        "UI 设计",
		Description: "Matías Duarte 思维模型。设计页面布局和视觉风格、建立或更新设计系统、配色和排版决策、动效和过渡设计。",
		Icon:        "paint-brush",
		Category:    "design",
	},
	// --- Original templates (3) ---
	{
		ID:          "research-analyst",
		Name:        "研究分析师",
		Description: "深度网络调研，多源验证，信息综合与结构化输出。",
		Icon:        "magnifying-glass",
		Category:    "productivity",
	},
	{
		ID:          "daily-secretary",
		Name:        "日程秘书",
		Description: "任务管理、日程追踪、承诺跟进，让日常运营井然有序。",
		Icon:        "calendar-check",
		Category:    "productivity",
	},
	{
		ID:          "knowledge-curator",
		Name:        "知识管理师",
		Description: "知识捕获、结构化组织、关联连接和按需检索——你的第二大脑。",
		Icon:        "book-open",
		Category:    "productivity",
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
