# Solo Company Skill

一人公司的 AI 团队。根据任务快速组建临时 AI Agent 团队协作，让你像拥有完整团队一样独立运作。

## 功能特性

- 智能组建：自动从 10 位专业 Agent 中选择最合适的成员
- 团队协作：支持多个 Agent 并行工作，完成复杂任务
- 角色齐全：覆盖 CEO、CTO、产品、设计、开发、QA、营销、运营、销售等全流程

## 可用 Agent

| Agent    | 职能                               | 思维模型       |
| -------- | ---------------------------------- | -------------- |
| CEO      | 战略决策、商业模式、PR/FAQ、优先级 | Jeff Bezos     |
| CTO      | 技术架构、技术选型、系统设计       | Werner Vogels  |
| 产品设计 | 产品定义、用户体验、可用性         | Donald Norman  |
| UI 设计  | 视觉设计、设计系统、配色排版       | Alberto Duarte |
| 交互设计 | 用户流程、Persona、交互模式        | Alan Cooper    |
| 全栈开发 | 代码实现、技术方案、开发           | DHH            |
| QA       | 测试策略、质量把控、Bug 分析       | J.S. Bach      |
| 营销     | 定位、品牌、获客、内容             | Seth Godin     |
| 运营     | 用户运营、增长、社区、PMF          | Paul Graham    |
| 销售     | 定价、销售漏斗、转化               | Gary Ross      |

## 安装步骤

### 1. 复制 Skill 文件

将整个仓库克隆或下载后，将 `SKILL.md` 复制到你的项目 `.claude/skills/` 目录下：

```bash
# 假设你的项目在 ~/my-project
cp SKILL.md ~/my-project/.claude/skills/solo-company.md
```

### 2. 复制 Agent 文件

将 `agents/` 目录下的所有 `.md` 文件复制到你的项目 `.claude/agents/` 目录下：

```bash
cp agents/*.md ~/my-project/.claude/agents/
```

### 3. 启用 Agent Teams（必需）

在你的项目 `.claude/settings.json` 中添加：

```json
{
  "env": {
    "CLAUDE_CODE_EXPERIMENTAL_AGENT_TEAMS": "1"
  }
}
```

> 或直接使用项目根目录下的 `config/settings.json` 作为参考。

## 使用方法

在 Claude Code 中运行：

```
/solo-company [任务描述]
```

## 使用示例

### 产品设计场景
```
/solo-company 我要做一个面向独立开发者的项目管理工具，帮我从定位到设计给出完整方案
```

### 技术实现场景
```
/solo-company 我们要开发一个用户认证系统，包括登录、注册、密码重置功能
```

### 营销策略场景
```
/solo-company 帮我制定一个 SaaS 产品的上线营销策略
```

### 商业决策场景
```
/solo-company 我们应该按订阅收费还是一次性买断？帮我分析
```

## 工作原理

1. **任务分析**：Skill 分析任务性质，选择 2-5 个最相关的 Agent
2. **团队组建**：创建临时 Agent Team，为每个成员分配具体任务
3. **并行协作**：各 Agent 独立工作，产出存放于 `docs/<role>/` 目录
4. **汇总输出**：Team lead 协调并汇总各成员结论

## 目录结构

```
├── SKILL.md               # 核心 skill 文件
├── agents/                # 10 个专业 agent
│   ├── ceo-bezos.md
│   ├── cto-vogels.md
│   └── ...
├── config/
│   └── settings.json      # Agent Teams 配置参考
├── README.md              # 本文档
└── LICENSE                # MIT License
```

## 注意事项

- 所有沟通使用中文，技术术语保留英文
- 团队是临时的，任务完成后即解散
- 创始人是最终决策者，Agent 提供建议但不替代决策
- 需要启用 `CLAUDE_CODE_EXPERIMENTAL_AGENT_TEAMS` 功能

## License

MIT
