<div align="center">

# Memoh-v2

**容器化 · 结构化长记忆 · 自我进化 AI Agent 系统**

[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Go](https://img.shields.io/badge/Go-1.25-00ADD8?logo=go)](https://go.dev)
[![Vue](https://img.shields.io/badge/Vue-3-4FC08D?logo=vuedotjs)](https://vuejs.org)
[![Docker](https://img.shields.io/badge/Docker-Compose-2496ED?logo=docker)](https://docs.docker.com/compose/)

每个 Bot 拥有独立容器、独立记忆、独立技能、自我进化能力 —— 你的私人 AI 智能体平台。

[English](./README_EN.md) · [快速开始](#快速开始) · [功能详解](#功能详解) · [安装与升级](#安装与升级)

</div>

---

## 目录

- [快速开始](#快速开始)
- [架构总览](#架构总览)
- [功能详解](#功能详解)
  - [Bot 管理与模板](#1-bot-管理与模板)
  - [对话与流式推送](#2-对话与流式推送)
  - [记忆系统](#3-记忆系统)
  - [容器系统](#4-容器系统)
  - [频道接入](#5-频道接入)
  - [MCP 工具系统](#6-mcp-工具系统)
  - [心跳与定时任务](#7-心跳与定时任务)
  - [自我进化系统](#8-自我进化系统)
  - [子智能体与技能](#9-子智能体与技能)
  - [OpenViking 分层上下文](#10-openviking-分层上下文数据库)
  - [Token 用量与诊断](#11-token-用量与系统诊断)
  - [跨 Bot 协作](#12-跨-bot-协作)
- [概念指南](#概念指南)
- [已知局限性](#已知局限性)
- [与 OpenClaw 对比](#与-openclaw-全面对比42-项)
- [安装与升级](#安装与升级)
- [技术栈](#技术栈)
- [致谢](#致谢)

---

## 快速开始

**需要 Docker：**

```bash
curl -fsSL https://raw.githubusercontent.com/Kxiandaoyan/Memoh-v2/main/scripts/install.sh | sh
```

> 静默安装（跳过交互）：`curl -fsSL ... | sh -s -- -y`

或手动：

```bash
git clone --depth 1 https://github.com/Kxiandaoyan/Memoh-v2.git
cd Memoh-v2
docker compose up -d
```

访问 **http://localhost:8082**，默认登录：`admin` / `admin123`

安装完成后，按以下顺序配置：

```
1. 设置 → Provider      添加 API 服务商，填入 API Key 和 Base URL
       ↓
2. Provider → 模型       在 Provider 下添加模型（chat 或 embedding 类型）
       ↓
3. 新建 Bot              选择模板或空白创建，设置名称和类型
       ↓
4. Bot → 设置           选择 Chat 模型、Embedding 模型、语言等
       ↓
5. Bot → 频道           连接 Telegram / 飞书等消息平台（可选）
```

---

## 架构总览

```
                    ┌──────────────┐
                    │   Web UI     │ :8082
                    │  Vue 3       │
                    └──────┬───────┘
                           │
              ┌────────────┼────────────┐
              │            │            │
      ┌───────▼──────┐ ┌──▼──────────┐ │
      │   Server     │ │ Agent       │ │
      │   Go + Echo  │ │ Gateway     │ │
      │   :8080      │ │ Bun + Elysia│ │
      └──┬────┬──────┘ │ :8081       │ │
         │    │         └──┬──────────┘ │
         │    │            │            │
    ┌────▼┐ ┌─▼─────┐  ┌──▼──────────┐ │
    │ PG  │ │Qdrant │  │ Containerd  │◄┘
    │     │ │       │  │ (per-bot    │
    │     │ │       │  │  containers)│
    └─────┘ └───────┘  └─────────────┘
```

| 服务 | 职责 |
|---|---|
| **Server** (Go) | REST API、认证、数据库、容器管理、对话路由、记忆检索 |
| **Agent Gateway** (Bun) | AI 推理、系统提示词组装、工具执行、流式推送、子智能体调度 |
| **Web** (Vue 3) | 管理界面：Bot、模型、频道、技能、文件、进化、心跳等可视化配置 |
| **PostgreSQL** | 关系数据存储（用户、Bot、消息、配置、进化日志） |
| **Qdrant** | 向量数据库（记忆语义搜索） |
| **Containerd** | 容器运行时（每个 Bot 一个隔离容器） |

**数据流：** 用户消息 → 频道适配器 → Server（认证、记忆检索、载荷组装）→ Agent Gateway（LLM 推理、工具调用）→ Server（响应持久化、记忆提取、Token 统计）→ 用户

---

## 功能详解

### 1. Bot 管理与模板

**Bot 管理** 是系统的核心。每个 Bot 是一个独立的 AI 智能体实体，拥有：

- 独立的身份定义（Identity / Soul / Task 三层人设）
- 独立的容器沙箱（文件系统、命令执行、网络访问）
- 独立的记忆空间（向量数据库分区隔离）
- 独立的频道配置（Telegram / 飞书等）
- 成员权限管理（Owner / Admin / Member 三级角色）
- 生命周期管理（creating → ready → deleting）
- 运行时健康检查（容器初始化、数据路径、任务状态）

**Bot 模板** 是 v2 新增的功能，提供 10 套预设深度人格模板，让创建专业化 Bot 只需两步：

| 步骤 | 操作 |
|------|------|
| **第一步** | 在模板网格中选择一个预设人格（或选"空白 Bot"从零开始） |
| **第二步** | 填写 Bot 名称、类型等基础信息，提交创建 |

系统自动将模板的 Identity（身份）、Soul（灵魂）、Task（任务）内容写入 Bot 的人设配置。

**13 套内置模板：**

| 模板 | 思维模型 | 分类 | 定位 |
|------|----------|------|------|
| CEO 战略顾问 | Jeff Bezos | 商业 | 战略决策、商业模式、优先级排序 |
| CTO 架构师 | Werner Vogels | 开发 | 技术架构、选型决策、可靠性评估 |
| 全栈开发 | DHH | 开发 | 代码实现、技术方案、代码审查 |
| 交互设计 | Alan Cooper | 设计 | 用户流程、Persona 驱动、交互模式 |
| 营销策略 | Seth Godin | 商业 | 产品定位、差异化、增长策略 |
| 运营增长 | Paul Graham | 商业 | 冷启动、用户留存、社区运营 |
| 产品设计 | Don Norman | 设计 | 产品定义、可用性、认知设计 |
| 质量保证 | James Bach | 开发 | 测试策略、风险评估、质量把控 |
| 销售策略 | Aaron Ross | 商业 | 定价策略、销售漏斗、转化优化 |
| UI 设计 | Matías Duarte | 设计 | 视觉设计、设计系统、配色排版 |
| 研究分析师 | — | 效率 | 深度调研、多源验证、结构化输出 |
| 日程秘书 | — | 效率 | 任务管理、日程追踪、承诺跟进 |
| 知识管理师 | — | 效率 | 知识捕获、组织连接、第二大脑 |

其中 10 套来自 [Solo-Company-Skill](https://github.com/anthropics/anthropic-cookbook) 项目的真实思想家思维模型，每个角色包含核心信条、决策框架、独立开发者建议和沟通风格。另外 3 套为通用效率工具角色。

每套模板包含三个 Markdown 文件（`identity.md`、`soul.md`、`task.md`），用中文编写，提取精髓以最大化信息密度、最小化 Token 消耗。

### 2. 对话与流式推送

系统支持两种对话模式：

| 模式 | 说明 | 使用场景 |
|------|------|----------|
| **流式推送 (SSE)** | 服务端逐 token 推送，前端实时渲染 | Web 聊天界面、需要即时反馈的场景 |
| **同步响应** | 等待完整回复后一次性返回 | API 调用、频道消息（Telegram/飞书） |

**对话上下文管理：**

- 最近 24 小时的对话自动加载为短期记忆
- 超出 Token 预算时自动裁剪 + LLM 摘要压缩
- 相关长期记忆通过语义搜索召回
- 对话结束后自动提取关键信息存入长期记忆

### 3. 记忆系统

记忆系统是 Memoh 的核心竞争力，采用三层混合架构：

| 层级 | 技术 | 作用 |
|------|------|------|
| **向量语义搜索** | Qdrant | 通过 Embedding 模型将文本转为向量，按语义相似度检索相关记忆 |
| **BM25 关键词索引** | 内置 | 传统关键词匹配，弥补语义搜索的精确查找不足 |
| **LLM 智能提取** | Chat 模型 | 每轮对话后自动提取关键事实，过滤噪声 |

**记忆生命周期：**

```
对话 → LLM 提取关键信息 → Embedding 向量化 → 存入 Qdrant
                                                    ↓
新对话 → 语义搜索 + BM25 匹配 → 召回相关记忆 → 注入上下文
                                                    ↓
定期维护 → LLM 记忆压缩 → 合并相似项、删除噪声 → 精简记忆库
```

**记忆管理界面** 提供：手动创建记忆、语义搜索、批量删除、记忆压缩（三档力度：轻度/中度/重度）、用量统计。

### 4. 容器系统

每个 Bot 拥有独立的 containerd 容器，提供完全隔离的运行环境：

| 能力 | 说明 |
|------|------|
| **文件读写** | Bot 可在自己的数据目录中创建、读取、修改、删除文件 |
| **命令执行** | Bot 可在容器内执行任意 Shell 命令 |
| **网络访问** | Bot 可访问外部 API 和网站 |
| **浏览器自动化** | 内置 Chromium + xvfb，支持无头浏览器操作 |
| **快照与回滚** | 任意时刻创建快照，可恢复到历史状态 |
| **技能安装** | 通过 ClawHub CLI 一键安装社区技能 |
| **共享目录** | `/shared` 目录跨 Bot 共享 |

**容器内预装能力：**
- **agent-browser** — 浏览器自动化框架
- **Actionbook** — 预编译的网站操作手册，加速常见网站的自动化操作
- **ClawHub CLI** — 技能市场命令行工具
- **OpenViking** — 分层上下文数据库（需开启）

### 5. 频道接入

Bot 通过频道适配器连接外部消息平台。目前实现了 4 种适配器：

| 适配器 | 平台 | 说明 |
|--------|------|------|
| **Telegram** | Telegram | 通过 Bot Token 接入，支持私聊和群聊 |
| **飞书 (Feishu)** | 飞书 / Lark | 通过飞书应用凭证接入 |
| **Web** | 内置 Web 聊天 | 直接在管理界面中对话，无需外部平台 |
| **Local CLI** | 命令行 | 开发调试用，本地命令行直接对话 |

每个 Bot 可同时配置多个频道。频道适配器负责：消息格式转换、身份解析、路由分发、响应投递。

用户可通过 Bind Code（绑定码）将 Telegram / 飞书账号关联到系统用户，实现跨平台身份统一。

### 6. MCP 工具系统

Bot 的能力通过 MCP（Model Context Protocol）工具系统扩展。分为两类：

**内置工具（15 个）** — 每个 Bot 自动拥有，无需配置：

| 类别 | 工具 | 功能 |
|------|------|------|
| 文件操作 | `read` / `write` / `list` / `edit` | 容器内文件读写、目录列表、文本替换 |
| 命令执行 | `exec` | 在容器内执行 Shell 命令 |
| 消息发送 | `send` / `react` | 向频道发消息、添加表情回应 |
| 用户查找 | `lookupChannelUser` | 按平台 ID 查找用户或群组 |
| 记忆搜索 | `searchMemory` | 搜索与当前对话相关的记忆 |
| 网页搜索 | `webSearch` | 通过搜索提供方搜索网页 |
| 定时任务 | `listSchedule` / `getSchedule` / `createSchedule` / `updateSchedule` / `deleteSchedule` | 完整的定时任务管理 |

**外部 MCP 服务器** — 可在管理界面中为每个 Bot 单独添加：

| 传输方式 | 说明 | 示例 |
|----------|------|------|
| **Stdio** | 在容器内启动进程 | `npx @modelcontextprotocol/server-filesystem` |
| **Remote** | 连接远程 HTTP/SSE 服务 | `https://mcp.example.com/sse` |

支持批量导入标准 `mcpServers` JSON 配置。工具网关（Tool Gateway）负责将 Agent Gateway 的工具调用代理到容器内的 MCP 服务器。

### 7. 心跳与定时任务

**心跳 (Heartbeat)** 让 Bot 从被动应答转为主动行动：

- 新建 Bot **自动创建**默认心跳（每小时一次维护检查）
- 支持 **定时触发**（间隔秒数）和 **事件触发**（定时任务完成、收到消息等）
- 触发时向 Bot 发送一条提示词，Bot 自主决定行动
- 可在管理界面添加多个心跳，每个有独立的间隔、提示词和触发条件
- 自我进化系统使用专用心跳（标记为 `[evolution-reflection]`，默认 24 小时间隔）

**定时任务 (Schedule)** 让 Bot 按 Cron 表达式执行周期性工作：

- Bot 可在对话中**自主创建**定时任务（通过内置 schedule 工具）
- 也可通过管理界面或 API 手动创建
- 每个任务有 Cron 表达式、命令提示词、最大执行次数、启用/禁用控制
- 管理界面展示所有任务的状态和执行次数

### 8. 自我进化系统

自我进化是 Memoh 的核心差异化能力。Bot 可以通过对话学习，自动改进自己的人设和行为。

**核心理念：** 进化是有机的 —— 源于真实对话，而非强制调度。如果近期对话顺畅无摩擦，不需要进化。系统只在有实质性学习材料时才触发改变。

**三阶段进化周期：**

| 阶段 | 名称 | 动作 |
|------|------|------|
| **Phase 1** | 反思 (Reflect) | 回顾近期对话，寻找摩擦点（错误回答、用户不满）、亮点（用户满意）、模式（反复出现的话题）、空白（缺失的知识）。如果没有发现有意义的信号，直接报告"无需进化"并结束 |
| **Phase 2** | 实验 (Experiment) | 对每个可行洞察：在 EXPERIMENTS.md 中记录（触发原因、观察、行动、预期效果），然后在对应文件中做小幅、可逆的修改（IDENTITY.md / SOUL.md / TOOLS.md） |
| **Phase 3** | 审查 (Review) | 自愈维护：检查定时任务是否正常运行、蒸馏日常笔记为长期记忆、检查协调文件、上报异常 |

**进化日志追踪：**

每次进化心跳触发时，系统自动创建一条 `evolution_logs` 记录（状态：运行中）。Agent 执行完毕后，系统根据返回内容自动更新状态：
- `completed` — 做了改变
- `skipped` — 无需进化（近期对话顺畅）
- `failed` — 执行出错

管理界面的进化页面展示：
- 开关控制和手动触发按钮
- 实验时间线（解析自 EXPERIMENTS.md）
- 进化历史时间线（来自 evolution_logs，含状态徽章和可展开的 Agent 回复详情）
- 人设文件查看器（IDENTITY.md / SOUL.md / TOOLS.md / EXPERIMENTS.md / NOTES.md）

### 9. 子智能体与技能

**子智能体 (Subagents)** 是 Bot 可以委派任务的专业化工作者：

- Agent 在对话中**自动创建和调度**子智能体（无需手动干预）
- 支持在管理界面预注册模板（名称、描述、技能列表），Agent 会优先使用已注册的定义
- 两种调度模式：**spawn**（后台异步执行）和 **query**（同步等待结果）
- 每个子智能体有独立的对话上下文和工具权限

**技能 (Skills)** 是存储在容器中的 Markdown 文件，为 Bot 提供特定领域的能力扩展：

- 每个技能包含名称、描述和详细指令
- Agent 根据对话上下文自动加载相关技能
- 可通过管理界面手动创建，也可通过 ClawHub（社区技能市场）一键安装
- Bot 自身也可以在对话中安装新技能

### 10. OpenViking 分层上下文数据库

[OpenViking](https://github.com/volcengine/OpenViking) 是集成在 Memoh 中的分层上下文数据库，为 Bot 提供超越平面向量检索的结构化长期记忆。

**为什么需要 OpenViking？**

普通向量记忆是"平面"的 —— 每段记忆平等地存储和检索。但人类记忆是分层的：核心知识（总是需要的）和详细细节（偶尔查阅的）。OpenViking 模仿了这种结构：

| 层级 | 名称 | 说明 | 类比 |
|------|------|------|------|
| **L0** | 摘要层 | 高度压缩的概要信息 | 一本书的目录 |
| **L1** | 知识层 | 结构化知识和关键事实 | 一本书的章节总结 |
| **L2** | 细节层 | 完整的原始内容和细节 | 一本书的完整正文 |

对话时先加载 L0 摘要（极少 Token）了解全貌，再按需加载 L1/L2 的特定部分，大幅减少 Token 消耗。

**启用方式：** 在 Bot 设置页面开启"启用 OpenViking 上下文数据库"开关，系统自动生成 `ov.conf` 并预填 API 信息。

### 11. Token 用量与系统诊断

**Token 用量统计：**

- 每次 LLM 调用（对话、心跳、定时任务、记忆操作）自动记录 Token 消耗
- 管理界面提供 Dashboard：每日用量曲线图 + 多 Bot 横向对比
- 支持查看近 7 天 / 30 天 / 90 天数据

**系统诊断：**

一键检测所有依赖服务的健康状态（PostgreSQL、Qdrant、Agent Gateway、Containerd），快速定位问题。

### 12. 跨 Bot 协作

**共享工作区：** 所有 Bot 容器自动挂载 `/shared` 目录，指向同一宿主机路径。

创建 Bot 时，系统自动在 `/shared/{bot_name}/` 下创建该 Bot 的专属输出文件夹。文件存放约定：

| 路径 | 范围 | 用途 |
|------|------|------|
| `/data/` | 私有 | 系统文件（IDENTITY.md、SOUL.md、TOOLS.md 等） |
| `/shared/{bot_name}/` | 共享 | Bot 的产出文件（报告、分析、草稿等） |
| `/shared/{other_bot}/` | 只读 | 其他 Bot 的产出（可读不可写） |

Bot 之间通过文件进行协调：

```
Agent A 写入报告到 /shared/AgentA/ → Agent B 读取 → Agent B 写入草稿到 /shared/AgentB/
```

协调的本质是文件系统 —— 简单、可靠、无需 API。管理界面提供共享工作区的文件浏览和编辑功能。

---

## 概念指南

### 模型类型：Chat vs Embedding

Memoh 使用两种类型的 AI 模型：

| | Chat 模型 | Embedding 模型 |
|---|---|---|
| **用途** | 理解指令并生成回复（对话的"大脑"） | 将文本转为数字向量（记忆搜索的"索引器"） |
| **类比** | 像一个能思考和说话的人 | 像一个图书馆编目员，给每本书贴标签 |
| **输入** | 对话上下文 + 用户消息 | 任意文本片段 |
| **输出** | 自然语言回复 | 固定长度的浮点数数组（向量） |
| **在 Memoh 中** | Bot 的主对话模型、摘要模型 | 记忆存储和检索时的向量化 |

没有 Embedding 模型，Bot 就没有长期记忆召回能力。

### 常见模型速查表

#### Chat 模型

| 提供商 | 模型名称 | 特点 |
|---|---|---|
| **OpenAI** | `gpt-4o` | 多模态旗舰，支持图片输入 |
| **OpenAI** | `gpt-4o-mini` | 性价比之选，速度快 |
| **OpenAI** | `o3-mini` | 推理增强型 |
| **Anthropic** | `claude-sonnet-4-20250514` | 代码和长文本强项 |
| **Anthropic** | `claude-3-5-haiku-20241022` | 快速轻量 |
| **Google** | `gemini-2.0-flash` | 速度极快，上下文窗口大 |
| **DeepSeek** | `deepseek-chat` | 中文优秀，性价比高 |
| **Qwen** | `qwen-plus` | 阿里千问，中文强项 |
| **本地部署** | `qwen3-8b` / `llama-3-8b` 等 | 通过 Ollama / vLLM 本地运行，零 API 费用 |

#### Embedding 模型

| 提供商 | 模型名称 | 维度 | 特点 |
|---|---|---|---|
| **OpenAI** | `text-embedding-3-small` | 1536 | 性价比最高，推荐首选 |
| **OpenAI** | `text-embedding-3-large` | 3072 | 精度最高 |
| **阿里/Dashscope** | `text-embedding-v3` | 1024 | 中文优化 |
| **Qwen** | [`qwen3-embedding-8b`](https://openrouter.ai/qwen/qwen3-embedding-8b) | 可变 | **推荐** — 多语言 + 长文本 + 代码，32K 上下文 |
| **本地部署** | `nomic-embed-text` | 768 | Ollama 可直接运行 |

> **推荐首选：[Qwen3 Embedding 8B](https://openrouter.ai/qwen/qwen3-embedding-8b)** — 多语言、长文本、代码检索表现领先，32K 上下文，通过 OpenRouter 使用仅 $0.01/百万 token。

> **维度（Dimensions）** 是 Embedding 模型输出向量的长度。创建 Embedding 模型时需要填写正确的维度值。

### 人设体系

Bot 的"个性"由三个维度定义，每个维度对应一份 Markdown 内容：

| 维度 | 文件 | 作用 |
|------|------|------|
| **Identity** | `IDENTITY.md` | 定义 Bot 是谁 — 名字、角色、背景、核心哲学 |
| **Soul** | `SOUL.md` | 定义 Bot 怎么行为 — 核心信条、行为原则、边界、沟通风格 |
| **Task** | `TASK.md` | 定义 Bot 做什么 — 具体工作流、检查清单、输出格式 |

**优先级规则：数据库优先，文件兜底。**

| 来源 | 管理方式 | 优先级 |
|------|----------|--------|
| **人设栏目**（数据库） | Web UI 直接编辑 | 高 — 有值则使用 |
| **容器文件**（.md 文件） | 文件栏目编辑或 Bot 自行修改 | 低 — 数据库为空时回退 |

开启"自我进化"后，Bot 可在对话中自行修改容器文件，逐渐发展个性。`TOOLS.md` 始终从容器文件读取。

### Provider 配置示例

| 场景 | Base URL | 说明 |
|------|----------|------|
| OpenAI 官方 | `https://api.openai.com/v1` | 需要 API Key |
| Azure OpenAI | `https://{name}.openai.azure.com/openai` | 企业方案 |
| 本地 Ollama | `http://host.docker.internal:11434/v1` | 免费，无需 Key |
| 本地 vLLM | `http://192.168.x.x:8000/v1` | 局域网 GPU 服务器 |
| 第三方代理 | `https://api.openrouter.ai/v1` | 多模型聚合 |

> 本地模型（Ollama / vLLM）可同时用于 Chat 和 Embedding，零 API 费用。

---

## 已知局限性

以下是对系统当前不足的客观评估。这些问题已知，部分有解决方案，部分需要后续迭代。

### Embedding 提供商兼容性

| 问题 | 只有 OpenAI 兼容和 DashScope 的 Embedding 提供商完整实现，其他提供商（Bedrock、Cohere 等）会返回"provider not implemented"错误 |
|------|------|
| **影响** | 使用非 OpenAI 格式 Embedding API 的用户无法使用记忆系统 |
| **变通** | 使用 OpenRouter 等兼容 OpenAI 格式的聚合服务，或使用本地 Ollama 部署 Embedding 模型 |

### 频道适配器覆盖

| 问题 | 目前仅实现 Telegram、飞书、Web、CLI 四种适配器。Discord、Slack、WhatsApp 等平台未实现 |
|------|------|
| **影响** | 使用 Discord / Slack 等平台的用户无法直接接入 |
| **说明** | 这是有意的取舍 —— 项目定位为单用户个人助手，Telegram + 飞书覆盖了目标用户的主要场景 |

### 频道绑定错误提示

| 问题 | Telegram 和飞书适配器在配置不完整时返回"binding is incomplete"，未指明具体缺少哪个字段 |
|------|------|
| **影响** | 用户难以自行排查配置问题 |

### 进化系统无自动回滚

| 问题 | 自我进化可以修改 Bot 的 IDENTITY.md / SOUL.md / TOOLS.md，但如果进化方向错误导致行为退化，没有一键回滚机制 |
|------|------|
| **变通** | 使用容器快照功能手动恢复到历史状态 |
| **改进方向** | 后续计划增加进化 diff 追踪和一键回退 |

### 进化质量依赖模型能力

| 问题 | 自我进化的质量高度依赖底层 LLM 模型的反思和自我评估能力 |
|------|------|
| **影响** | 较弱的模型可能产生低质量的进化改变，或无法准确识别对话中的摩擦点 |
| **建议** | 进化功能推荐使用 Claude Sonnet、GPT-4o 或同等能力以上的模型 |

### OpenViking 用户文档不足

| 问题 | OpenViking 功能开关存在，但缺乏用户文档说明其工作原理、适用场景、与标准记忆系统的关系 |
|------|------|
| **影响** | 用户不确定是否该开启这个功能 |

### 平台支持限制

| 平台 | 状态 |
|------|------|
| **Linux** | 完全支持，推荐生产部署 |
| **macOS** | 需要通过 Lima 运行 containerd（`mise run lima-up`） |
| **Windows** | 无原生 containerd 支持，需要 WSL2 或 Docker Desktop |

### SDK 类型同步

| 问题 | 新增的模板系统和进化日志 API 尚未通过 `mise run swagger-generate && mise run sdk-generate` 重新生成前端 SDK 类型定义 |
|------|------|
| **影响** | 前端暂时使用 `as any` 类型断言和原始 `client.get()` 调用作为临时方案 |

---

## 与 OpenClaw 全面对比（42 项）

> 结果列：**M** = Memoh-v2 胜 · **O** = OpenClaw 胜 · **=** = 持平

| # | 维度 | Memoh-v2 | OpenClaw | 结果 |
|---|---|---|---|:---:|
| 1 | 后端语言 | Go（高并发、编译型） | Node.js（单线程、解释型） | **M** |
| 2 | 架构模式 | 三服务分离（Server / Gateway / Web） | 单体应用 | **M** |
| 3 | 通信协议 | SSE 单向流式推送 | WebSocket 全双工 | **O** |
| 4 | 容器隔离 | containerd 独立容器/Bot，完全隔离 | 共享运行时（可选 Docker 沙盒） | **M** |
| 5 | 结构化数据库 | PostgreSQL | SQLite | **M** |
| 6 | 向量数据库 | Qdrant（独立服务） | SQLite-vec（嵌入式） | **M** |
| 7 | 水平扩展 | 服务可独立部署与扩展 | 单机运行 | **M** |
| 8 | 资源占用 | 需 Docker + PostgreSQL + Qdrant | 轻量单进程，几十 MB 内存 | **O** |
| 9 | 部署方式 | Docker Compose 一键部署 | npm install -g + CLI 启动 | **=** |
| 10 | 远程访问 | 天然支持（Docker 部署到任意服务器） | 需 Tailscale / SSH 隧道 | **M** |
| 11 | Agent 定义体系 | SOUL + IDENTITY + TOOLS + EXPERIMENTS + NOTES | SOUL + IDENTITY + TOOLS + AGENTS + HEARTBEAT + BOOTSTRAP + USER | **=** |
| 12 | 子 Agent 管理 | spawn/kill/steer + 独立工具权限 + 注册表 | spawn/kill/steer + 深度限制 + 子数上限 | **=** |
| 13 | 工具执行框架 | MCP 协议（容器内沙盒执行） | Pi Runtime 内置工具（Browser/Canvas/Nodes） | **O** |
| 14 | MCP 协议支持 | 原生支持，可连接任意 MCP Server | 有限支持 + ACP 协议 | **M** |
| 15 | 浏览器自动化 | Chromium + agent-browser + Actionbook + xvfb | 内置 Browser + agent-browser + Actionbook | **=** |
| 16 | 智能网页策略 | Markdown Header → Actionbook → curl 三级降级 | 标准抓取 | **M** |
| 17 | 技能市场 | ClawHub + OPC Skills | ClawHub + OPC Skills | **=** |
| 18 | 短期记忆 | 最近 24h 对话自动加载 | 当前 session 对话 | **M** |
| 19 | 长期记忆 | Qdrant 向量语义搜索 + BM25 关键词匹配，每轮自动入库 | SQLite-vec 向量搜索 + memoryFlush | **M** |
| 20 | 上下文压缩 | Token 预算裁剪 + LLM 自动摘要 | /compact 手动压缩 | **M** |
| 21 | 分层上下文 | OpenViking（L0/L1/L2），每 Bot 可独立开关 | 无 | **M** |
| 22 | 自我进化机制 | 三阶段有机进化（反思/实验/审查）+ 进化日志追踪 | MEMORY.md 手动迭代 | **M** |
| 23 | Bot 模板 | 13 套思维模型模板（含 10 套真实思想家），2 步创建流程 | 无 | **M** |
| 24 | Daily Notes | 日志模板 + 心跳自动蒸馏为长期记忆 | memory/日期.md 手动记录 | **M** |
| 25 | 跨 Agent 协调 | /shared 自动挂载 + 文件协调 | sessions 工具 + 文件协调 | **=** |
| 26 | 定时任务 | Cron + 可视化管理 UI | Cron 调度（CLI 配置） | **M** |
| 27 | 心跳机制 | 定时 + 事件驱动双模式 | 定时心跳 | **M** |
| 28 | 自愈能力 | 自动检测过期任务并补跑 + 异常上报用户 | HEARTBEAT.md 手动配置自愈逻辑 | **M** |
| 29 | 管理界面 | 完整 Web UI（10+ 模块） | Control UI + CLI + TUI 三合一 | **M** |
| 30 | 多用户支持 | 原生多成员 + 角色权限（admin/member） | 单用户 | **M** |
| 31 | 平台覆盖 | Telegram、飞书、Web 聊天、CLI | Telegram、Discord、WhatsApp、Slack、Teams、Signal、iMessage 等 12+ | **O** |
| 32 | Token 用量统计 | 每条回复显示消耗 + Dashboard 曲线图 + 多 Bot 对比 | /usage 命令查询 | **M** |
| 33 | Bot 文件管理 | Web UI 在线查看/编辑模板文件 | 本地文件系统 + Git 自动初始化 | **M** |
| 34 | 认证安全 | JWT + 多用户权限体系 | Gateway Token + Pairing Code | **M** |
| 35 | 容器快照/回滚 | containerd 快照 + 版本回滚 | Git 版本控制 | **M** |
| 36 | 搜索引擎集成 | 可配置多搜索引擎（Brave / SerpAPI） | Brave Search 单一引擎 | **M** |
| 37 | 前端国际化 | 中文 + 英文完整 i18n | 英文为主，部分中文文档 | **M** |
| 38 | 语音 / TTS | 无 | Voice Wake + Talk Mode + ElevenLabs TTS | **O** |
| 39 | 可视化画布 | 无 | Canvas + A2UI 交互式画布 | **O** |
| 40 | Companion Apps | 无 | macOS + iOS + Android 原生应用 | **O** |
| 41 | Webhook / 邮件集成 | 无 | Webhook + Gmail Pub/Sub | **O** |
| 42 | 模型故障切换 | 备用模型自动 Failover（sync + stream） | Model Failover 自动切换 | **=** |

**汇总：Memoh-v2 胜 27 项 · OpenClaw 胜 8 项 · 持平 7 项**

---

## 安装与升级

### 一键安装

```bash
curl -fsSL https://raw.githubusercontent.com/Kxiandaoyan/Memoh-v2/main/scripts/install.sh | sh
```

安装脚本自动：检测 Docker → 检测旧版本（可选清理）→ 克隆代码 → 生成 config.toml → 构建并启动所有服务。

支持交互式配置工作目录、数据目录、管理员密码等；加 `-y` 跳过交互。

### 升级（不丢数据）

```bash
curl -fsSL https://raw.githubusercontent.com/Kxiandaoyan/Memoh-v2/main/scripts/upgrade.sh | sh
```

脚本自动查找 Memoh 项目目录，无需手动 `cd`。也可在项目目录下直接执行：

```bash
cd ~/memoh/Memoh-v2 && ./scripts/upgrade.sh
```

升级流程：自动备份数据库 → `git pull` → 重建 Docker 镜像 → 数据库迁移 → 健康检查。

**所有数据（PostgreSQL、Qdrant、Bot 文件）存储在 Docker named volumes 和宿主机目录中，升级不会丢失任何数据。**

| 参数 | 说明 |
|------|------|
| `--no-backup` | 跳过升级前数据库备份 |
| `--no-pull` | 跳过 git pull（已手动更新代码时） |
| `-y` | 静默模式，跳过所有确认提示 |

### 卸载

```bash
curl -fsSL https://raw.githubusercontent.com/Kxiandaoyan/Memoh-v2/main/scripts/uninstall.sh | sh
```

| 参数 | 说明 |
|------|------|
| `--keep-data` | 保留 Docker volumes（数据库、向量库、Bot 数据不删） |
| `--keep-images` | 保留已构建的 Docker 镜像 |
| `-y` | 静默模式 |

卸载前自动创建数据库最终备份到 `backups/` 目录。

### 数据库管理

```bash
./scripts/db-up.sh      # 执行数据库迁移（增量）
./scripts/db-drop.sh     # 回滚所有表（危险操作，需确认）
```

### 数据迁移

```bash
# 旧服务器备份
docker compose exec -T postgres pg_dump -U memoh memoh | gzip > memoh-backup.sql.gz

# 新服务器恢复
gunzip -c memoh-backup.sql.gz | docker compose exec -T postgres psql -U memoh memoh
```

Bot 文件数据在宿主机 `data/bots/` 目录下，直接拷贝即可。

### 脚本一览

| 脚本 | 用途 |
|------|------|
| `scripts/install.sh` | 一键安装（全新部署） |
| `scripts/upgrade.sh` | 一键升级（保留数据） |
| `scripts/uninstall.sh` | 卸载（可选保留数据） |
| `scripts/db-up.sh` | 数据库迁移 |
| `scripts/db-drop.sh` | 数据库回滚 |
| `scripts/compile-mcp.sh` | 编译 MCP 二进制并热更新到容器 |

---

## 技术栈

| 服务 | 技术 | 端口 |
|---|---|---|
| Server（后端） | Go + Echo + Uber FX + pgx/v5 + sqlc | 8080 |
| Agent Gateway | Bun + Elysia + Vercel AI SDK | 8081 |
| Web（前端） | Vue 3 + Vite + Tailwind CSS + Pinia | 8082 |

| 依赖 | 版本 | 用途 |
|------|------|------|
| PostgreSQL | 18 | 关系数据存储 |
| Qdrant | latest | 向量数据库 |
| Containerd | v2 | 容器运行时 |

---

## 致谢

本项目基于 [Memoh](https://github.com/memohai/Memoh) 二次开发，感谢原作者的优秀工作。
