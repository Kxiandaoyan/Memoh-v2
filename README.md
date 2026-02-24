<div align="center">

<h1>🧠 Memoh-v2</h1>

**真隔离 · 真记忆 · 真进化 · 真协作 —— 不妥协的 AI Agent 平台**

<p>
  <a href="./LICENSE"><img src="https://img.shields.io/badge/license-AGPL--v3-blue.svg" alt="License" /></a>
  <a href="https://go.dev"><img src="https://img.shields.io/badge/Go-1.25-00ADD8?logo=go&logoColor=white" alt="Go" /></a>
  <a href="https://vuejs.org"><img src="https://img.shields.io/badge/Vue-3-4FC08D?logo=vuedotjs&logoColor=white" alt="Vue 3" /></a>
  <a href="https://docs.docker.com/compose/"><img src="https://img.shields.io/badge/Docker-Compose-2496ED?logo=docker&logoColor=white" alt="Docker" /></a>
  <a href="https://github.com/Kxiandaoyan/Memoh-v2/stargazers"><img src="https://img.shields.io/github/stars/Kxiandaoyan/Memoh-v2?style=social" alt="Stars" /></a>
</p>

<p>
  <a href="#-快速开始">快速开始</a> ·
  <a href="#-核心亮点">核心亮点</a> ·
  <a href="#-21-项内置技能">技能系统</a> ·
  <a href="#-团队协作多-bot-编排">团队协作</a> ·
  <a href="#-多平台频道接入">多平台接入</a> ·
  <a href="./README_EN.md">English</a>
</p>

<p>
  <a href="./doc/installation.md">🚀 安装与升级</a> ·
  <a href="./doc/features.md">📖 功能详解</a> ·
  <a href="./doc/concepts.md">💡 概念指南</a> ·
  <a href="./doc/README.md">📚 使用教程</a>
</p>
<p>
  <a href="./doc/screenshots.md">🖼️ 界面截图</a> ·
  <a href="./doc/comparison.md">⚖️ 与 OpenClaw 对比</a> ·
  <a href="./doc/known-limitations.md">⚠️ 已知局限性</a>
</p>
<p>
  <a href="./doc/FEATURE_AUDIT.md">📊 功能审计</a> ·
  <a href="./doc/PROMPTS_INVENTORY.md">📝 提示词清单</a> ·
  <a href="./doc/README.md#开发计划归档">🗂️ 开发计划归档</a>
</p>

<br/>

每个 Bot 一个 containerd 容器，不是共享运行时；<br/>
Qdrant + BM25 + LLM 三层记忆提取，不是 SQLite 向量搜索；<br/>
Bot 自己反思、实验、审查，持续进化，不是手动编辑记忆文件；<br/>
组建 AI 团队，大总管调度成员协作，不是单打独斗；<br/>
对接 Telegram / 飞书 / 个人微信 / Discord，一个 Bot 服务全平台。

</div>

---

<p align="center">
  <img src="./doc/1.png" width="100%" />
</p>

<p align="center">
  <a href="./doc/screenshots.md">👉 查看更多截图</a>
</p>

---

## 🚀 快速开始

**一键安装（需要 Docker）：**

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

**首次配置流程：**

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

**升级：**

```bash
./scripts/upgrade.sh
```

> 自动备份数据库 → 拉取最新代码 → 重建服务 → 运行迁移 → 同步技能到所有 Bot

> 📖 详细安装、升级、卸载、数据迁移指南：[安装与升级](./doc/installation.md)

---

## ✨ 核心亮点

### 🔒 真隔离 — 每个 Bot 一个容器

不是共享进程，不是 Docker-in-Docker，是真正的 **containerd 容器隔离**。每个 Bot 拥有独立的文件系统、Shell、网络和浏览器环境。支持快照回滚，一键恢复到任意历史状态。

### 🧠 真记忆 — 三层记忆提取

| 层级 | 技术 | 作用 |
|------|------|------|
| 语义搜索 | Qdrant 向量数据库 | 理解"意思相近"的内容 |
| 关键词索引 | BM25 | 精确匹配专有名词和术语 |
| 智能提取 | LLM | 每次对话后自动提炼关键信息入库 |

24 小时短期记忆自动加载，长期记忆语义召回，Token 预算自动裁剪并用 LLM 摘要补偿。

**更多记忆特性：**
- 🧩 **事实自动提取** — 对话后自动提炼用户偏好、个人信息、计划意图等关键事实
- 🔧 **解决方案记忆** — 自动识别对话中的「问题→解决方案」对，下次遇到类似问题直接召回
- 🤝 **集体记忆** — 同一 Bot 的所有对话共享知识库，A 用户教会的知识，B 用户也能受益
- ⏳ **时间衰减** — 记忆按半衰期（30天）自动降权，越新的记忆权重越高
- 🗜️ **记忆压缩** — 当记忆条目过多时，LLM 自动合并冗余、去除矛盾，保持精简
- 🌍 **30+ 语言** — BM25 索引支持中日韩英法德等 30+ 语言的分词和检索

### 🧬 真进化 — 三阶段自我进化

Bot 不是静态的 prompt，而是会成长的智能体：

```
反思 (Reflection) → 实验 (Experiment) → 审查 (Review) → 循环
```

完整进化日志追踪，每一步改变都有据可查。

### 👥 真协作 — 多 Bot 团队编排

一键组建 AI 团队，自动生成"大总管"Bot 统一调度：

- 大总管通过 `call_agent` 工具调用团队成员
- 成员回复自动带 **【Bot名称】** 前缀，来源清晰
- 团队可直接对接 Telegram / 飞书等平台
- 支持心跳巡检，大总管定期检查团队任务状态

---

## 💰 零成本基础设施

不依赖任何付费第三方 API，核心能力全部自托管：

| 能力 | 方案 | 费用 |
|------|------|:----:|
| **联网搜索** | SearXNG 元搜索引擎（Docker 内置） | **免费** |
| **向量数据库** | Qdrant（Docker 内置） | **免费** |
| **关键词索引** | BM25 内置引擎（无需 Elasticsearch） | **免费** |
| **容器隔离** | Containerd（Docker 内置） | **免费** |
| **本地模型** | Ollama 支持（可选） | **免费** |

> 💡 搜索不花钱！SearXNG 聚合 Google、Bing、DuckDuckGo 等多个搜索引擎，无需任何 API Key。

---

## 🔌 10 种 AI 模型提供方

不绑定任何单一厂商，自由选择最适合的模型：

| 提供方 | 说明 |
|--------|------|
| OpenAI | GPT-4o、GPT-4、GPT-3.5 等 |
| Anthropic | Claude 3.5、Claude 3 等 |
| Google | Gemini 系列 |
| Azure OpenAI | 企业级 Azure 部署 |
| AWS Bedrock | AWS 托管模型 |
| Mistral AI | Mistral 系列 |
| xAI | Grok 系列 |
| Ollama | 🏠 **本地模型，零 API 费用** |
| 阿里云 DashScope | 通义千问系列 |
| OpenAI 兼容 | 任何兼容 OpenAI 格式的第三方 API |

> 每个 Bot 可独立配置对话模型、记忆模型、嵌入模型和视觉模型（VLM），灵活组合。

---

## 🛠 21 项内置技能

每个新建 Bot 自动获得全部技能，开箱即用。Bot 在对话中通过 `use_skill` 工具按需激活。

| 技能 | 用途 | 默认 |
|------|------|:----:|
| **web-search** | 联网搜索（SearXNG） | ✅ |
| **mcp-builder** | 构建自定义 MCP 服务器 | ✅ |
| **skill-creator** | 创建新技能 | ✅ |
| **webapp-testing** | Web 应用测试 | ✅ |
| **scheduled-task** | 定时任务调度 | ✅ |
| **web-artifacts-builder** | 构建 Web 制品 | ✅ |
| **develop-web-game** | 开发 Web 游戏 | ✅ |
| **playwright** | 浏览器自动化 | ✅ |
| **create-plan** | 任务规划 | ✅ |
| **canvas-design** | Canvas 画布设计 | ✅ |
| **frontend-design** | 前端页面设计 | ✅ |
| **frontend-slides** | HTML 演示文稿（12 种预设风格） | ✅ |
| **local-tools** | 本地系统工具 | ✅ |
| **weather** | 天气查询 | ✅ |
| **imap-smtp-email** | 邮件收发（IMAP/SMTP） | ✅ |
| **docx** | Word 文档生成 | ✅ |
| **xlsx** | Excel 表格生成 | ✅ |
| **pptx** | PowerPoint 生成 | ✅ |
| **pdf** | PDF 文档处理 | ✅ |
| **x-tweet-fetcher** | X/Twitter 推文抓取 + 中文平台内容获取 | ✅ |
| **remotion** | 视频渲染 | ⬚ |

> 技能可在 Bot 设置页单独开关。支持通过 ClawHub 一键安装社区技能。

## 👥 团队协作（多 Bot 编排）

```
用户消息 → 大总管 Bot → call_agent(成员A) → 【成员A】结果
                      → call_agent(成员B) → 【成员B】结果
                      → 汇总回复 → Telegram / 飞书 / Web
```

- 在 Bots 页面一键"组建团队"，选择成员 Bot 并描述角色
- 系统自动生成大总管 Bot，注入团队路由表和调度指令
- 团队分组展示，可直接点击"设置平台"对接外部频道
- 调用记录完整审计（caller → target、消息、结果、状态）
- 调用深度限制（最大 3 层），防止无限递归

---

## 🌐 多平台频道接入

一个 Bot 同时服务多个平台，消息跨平台同步：

| 平台 | 私聊 | 群组 | @提及触发 | 被动同步 |
|------|:----:|:----:|:---------:|:--------:|
| **Telegram** | ✅ | ✅ | ✅ | ✅ |
| **飞书** | ✅ | ✅ | ✅ | ✅ |
| 🟢 **个人微信** | ✅ | ✅ | — | — |
| **Discord** | ✅ | ✅ | ✅ | ✅ |
| **Web 聊天** | ✅ | — | — | — |
| **CLI** | ✅ | — | — | — |

> 🟢 **个人微信**：通过 Webhook + Poll 模式接入，支持私聊和群聊，异步处理不阻塞，消息自动路由到对应 Bot。

- **Bind Code** 跨平台身份统一 — 用户在不同平台绑定同一身份
- **群组被动同步** — 群内非 @消息 自动存入对话历史，Bot 拥有完整上下文
- **跨平台广播** — Bot 回复自动同步到其他已绑定平台（同平台内不串台）
- **处理状态通知** — 飞书等平台显示"正在思考…"状态反馈

---

## 🔌 MCP 工具系统

15 个内置工具 + 无限扩展：

- **内置工具**：文件读写、Web 搜索、日程管理、浏览器自动化、Canvas 设计、文档生成等
- **外部 MCP 服务器**：支持 Stdio（容器内进程）、HTTP、SSE 三种传输方式
- **批量导入**：通过 `mcpServers` JSON 配置一键导入多个 MCP 服务器
- **MCP Builder 技能**：Bot 可以自己构建新的 MCP 服务器

---

## 🤖 13 套思维模型模板

基于真实思想家的决策框架，两步创建专业化 Bot：

| 模板 | 角色 | 思维模型 |
|------|------|----------|
| Jeff Bezos | CEO | 逆向工作法、Day 1 思维 |
| Werner Vogels | CTO | 分布式系统、可靠性工程 |
| DHH | 全栈技术主管 | 简洁优先、反过度工程 |
| Paul Graham | 运营总监 | 冷启动、用户增长 |
| Seth Godin | 营销总监 | 紫牛理论、部落营销 |
| Alan Cooper | 交互设计总监 | 目标导向设计、Persona |
| Aaron Ross | 销售总监 | 可预测收入、销售漏斗 |
| James Bach | QA 总监 | 探索性测试、上下文驱动 |
| Don Norman | 产品设计总监 | 情感化设计、可用性 |
| Matías Duarte | UI 设计总监 | Material Design、动效 |
| 通用助手 | 多面手 | 通用对话 |
| 团队大总管 | 团队协调 | 任务分配、成员调度 |
| 空白模板 | 自定义 | 完全自定义 |

每个模板包含三层身份定义：**Identity**（我是谁）、**Soul**（我怎么思考）、**Task**（我做什么）。

---

## 🏗 架构总览

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
    │ 18  │ │       │  │ (per-bot    │
    │     │ │       │  │  containers)│
    └─────┘ └───────┘  └─────────────┘
```

| 服务 | 技术 | 职责 |
|------|------|------|
| **Server** | Go + Echo + pgx/v5 + sqlc | REST API、认证、数据库、容器管理、对话路由、记忆检索 |
| **Agent Gateway** | Bun + Elysia + Vercel AI SDK | AI 推理、提示词组装、工具执行、流式推送、子智能体调度 |
| **Web** | Vue 3 + Vite + Tailwind CSS | 管理界面：Bot、模型、频道、技能、文件、进化、心跳 |
| **PostgreSQL 18** | — | 关系数据（用户、Bot、消息、配置、进化日志） |
| **Qdrant** | — | 向量数据库（记忆语义搜索） |
| **Containerd** | — | 容器运行时（每个 Bot 一个隔离容器） |
| **SearXNG** | — | 元搜索引擎（Web 搜索技能后端） |

---

## 🔧 技术栈

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
| SearXNG | latest | 元搜索引擎 |

---

## 🎯 核心功能概览

> 每项功能的完整说明请查看顶部导航 → **功能详解**。

- **Bot 管理与模板** — 13 套思维模型模板（含 10 位真实思想家），两步创建专业化 Bot
- **对话与流式推送** — SSE 实时流式 + 同步两种模式，自动上下文管理与记忆召回
- **三层记忆系统** — 向量语义搜索 + BM25 关键词 + LLM 智能提取，对话后自动入库
- **独立容器沙箱** — 每个 Bot 拥有 containerd 隔离容器，支持文件、命令、浏览器、快照回滚
- **多平台频道接入** — Telegram / 飞书 / 个人微信 / Discord / Web / CLI，跨平台身份统一
- **MCP 工具系统** — 15 个内置工具 + 任意外部 MCP 服务器，支持 Stdio 和 Remote 传输
- **零成本搜索** — SearXNG 自托管元搜索引擎，聚合多引擎结果，无需任何 API Key
- **10 种模型提供方** — OpenAI / Claude / Gemini / Ollama 本地模型等，不绑定任何厂商
- **心跳与定时任务** — 定时 + 事件双模式触发，Bot 主动行动而非被动应答
- **自我进化** — 三阶段有机进化周期（反思/实验/审查），进化日志完整追踪
- **子智能体与技能** — 自动调度子智能体，ClawHub 技能市场一键安装
- **团队协作** — 一键组建 AI 团队，大总管调度成员，直接对接外部平台
- **OpenViking 分层上下文** — L0/L1/L2 三层结构化记忆，大幅减少 Token 消耗
- **Token 用量与诊断** — Dashboard 曲线图 + 多 Bot 对比，一键服务健康检查
- **跨 Bot 协作** — `/shared` 共享工作区，文件驱动的简单协调机制

---

## 📦 更多特性

- **SSE 实时流式推送** — 打字机效果，支持同步和流式两种模式
- **心跳与定时任务** — 定时 + 事件双模式触发，Bot 主动行动而非被动应答
- **OpenViking 分层上下文** — L0/L1/L2 三层结构化记忆，大幅减少 Token 消耗
- **子智能体调度** — 自动拆解复杂任务，分配给子智能体并行处理
- **容器快照回滚** — 一键保存/恢复 Bot 容器状态，安全实验不怕搞坏
- **`/shared` 共享工作区** — 跨 Bot 文件共享，简单高效的协作机制
- **Token 用量仪表盘** — 曲线图 + 多 Bot 对比，一键服务健康检查
- **成员权限管理** — Owner / Admin / Member 三级角色控制
- **重复回复检测** — Jaccard 相似度自动检测重复回答，避免 Bot 陷入复读循环
- **解决方案记忆** — 自动提取对话中的「问题→方案」对，形成可复用的知识库
- **完整文档体系** — 18 篇详细教程，覆盖从安装到高级用法

---

## 📖 文档导航

| 文档 | 内容 |
|------|------|
| [安装与升级](./doc/installation.md) | Docker 部署、升级、卸载、数据迁移 |
| [功能详解](./doc/features.md) | 所有功能的完整说明 |
| [概念指南](./doc/concepts.md) | 核心概念与设计理念 |
| [Bot 管理](./doc/02-bot-management.md) | 创建、模板、生命周期 |
| [对话系统](./doc/04-chat.md) | 聊天界面与上下文管理 |
| [频道接入](./doc/05-channel-integration.md) | Telegram、飞书配置 |
| [记忆系统](./doc/06-memory-system.md) | 三层记忆架构详解 |
| [MCP 与技能](./doc/08-mcp-skills.md) | 工具系统与技能管理 |
| [进化与心跳](./doc/09-evolution-heartbeat.md) | 自我进化与定时任务 |
| [界面截图](./doc/screenshots.md) | UI 截图集 |

---

## 🤝 致谢

本项目基于 [Memoh](https://github.com/memohai/Memoh) 二次开发，感谢原作者的优秀工作。

---

## 📄 License

[GNU AGPL v3](./LICENSE)

---

## ⭐ Star History

<a href="https://www.star-history.com/#Kxiandaoyan/Memoh-v2&Date">
 <picture>
   <source media="(prefers-color-scheme: dark)" srcset="https://api.star-history.com/svg?repos=Kxiandaoyan/Memoh-v2&type=Date&theme=dark" />
   <source media="(prefers-color-scheme: light)" srcset="https://api.star-history.com/svg?repos=Kxiandaoyan/Memoh-v2&type=Date" />
   <img alt="Star History Chart" src="https://api.star-history.com/svg?repos=Kxiandaoyan/Memoh-v2&type=Date" />
 </picture>
</a>
