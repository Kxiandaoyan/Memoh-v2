<div align="center">

# Memoh-v2

**多成员 · 结构化长记忆 · 容器化 AI Agent 系统平台**

[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Go](https://img.shields.io/badge/Go-1.23-00ADD8?logo=go)](https://go.dev)
[![Vue](https://img.shields.io/badge/Vue-3-4FC08D?logo=vuedotjs)](https://vuejs.org)
[![Docker](https://img.shields.io/badge/Docker-Compose-2496ED?logo=docker)](https://docs.docker.com/compose/)

每个 Bot 拥有独立容器、独立记忆、独立技能 —— 像雇佣真正的员工一样部署 AI Agent。

[English](./README_EN.md) · [快速开始](#快速开始) · [概念指南](#概念指南) · [安装与升级](#安装与升级)

</div>

---

## 目录

- [快速开始](#快速开始)
- [核心特性](#核心特性)
- [架构总览](#架构总览)
- [概念指南](#概念指南)
  - [模型类型：Chat vs Embedding](#模型类型chat-vs-embedding)
  - [常见模型速查表](#常见模型速查表)
  - [配置步骤](#配置步骤)
  - [人设与文件的关系](#人设与文件的关系)
  - [心跳与子智能体](#心跳与子智能体)
- [与 OpenClaw 全面对比](#与-openclaw-全面对比42-项)
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

---

## 核心特性

### 基础能力

- **多 Bot 管理** — 创建多个 Bot，人与 Bot、Bot 与 Bot 可私聊或群聊协作
- **容器隔离** — 每个 Bot 独立 containerd 容器，可自由执行命令、编辑文件、访问网络
- **记忆工程** — 对话存入 PostgreSQL + Qdrant 向量数据库，语义搜索召回记忆
- **多平台接入** — Telegram、Discord、飞书（Lark）等
- **可视化配置** — Provider、Model、Memory、Channel、MCP、Skills 全图形界面
- **定时任务** — Cron 表达式调度

### v2 增强

| 能力 | 说明 |
|---|---|
| **子 Agent 自主能力** | spawn/kill/steer 子 Agent，独立工具权限 |
| **浏览器控制** | 内置 Chromium + agent-browser + xvfb，容器内可自动操作网页 |
| **心跳自愈引擎** | 定时 + 事件驱动心跳，新建 Bot 自动配置，自动检测并补跑失败任务 |
| **上下文智能压缩** | Token 预算裁剪 + LLM 摘要，长对话不丢关键信息 |
| **OpenViking 集成** | 分层上下文数据库（L0/L1/L2），每 Bot 可独立开关 |
| **双技能市场** | ClawHub + OPC Skills，容器内一条命令安装 |
| **Actionbook** | 预编译网站操作手册，高效浏览器自动化 |
| **智能网页抓取** | Markdown Header → Actionbook → 普通 curl 三级策略 |
| **自我进化循环** | EXPERIMENTS.md 实验记录 + SOUL.md 自省机制 |
| **Daily Notes** | 日志模板 + 心跳蒸馏为长期记忆 |
| **跨 Bot 共享工作区** | `/shared` 目录挂载到所有容器，文件即协调 |
| **Token 用量统计** | 每次回复显示消耗，Dashboard 曲线图对比各 Bot |
| **模型故障切换** | 配置备用模型，主模型失败自动 Failover |
| **系统诊断** | 一键检测 PostgreSQL、Qdrant、Gateway、Containerd 健康状态 |
| **完整管理 UI** | Files、Skills、Subagents、Heartbeat、History 全部可视化 |

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
| **Server** | REST API、认证、数据库、容器管理 |
| **Agent Gateway** | AI 对话路由、工具调用、流式推送 |
| **Web** | 管理界面：Bot、模型、频道、技能、文件等可视化配置 |
| **PostgreSQL** | 关系数据存储（用户、Bot、消息、配置） |
| **Qdrant** | 向量数据库（记忆语义搜索） |
| **Containerd** | 容器运行时（每个 Bot 一个隔离容器） |

---

## 概念指南

### 模型类型：Chat vs Embedding

Memoh 使用两种类型的 AI 模型，理解它们的区别是配置系统的基础：

| | Chat 模型 | Embedding 模型 |
|---|---|---|
| **用途** | 理解指令并生成回复（对话的"大脑"） | 将文本转为数字向量（记忆搜索的"索引器"） |
| **类比** | 像一个能思考和说话的人 | 像一个图书馆的编目员，给每本书贴标签 |
| **输入** | 对话上下文 + 用户消息 | 任意文本片段 |
| **输出** | 自然语言回复 | 固定长度的浮点数数组（向量） |
| **在 Memoh 中** | Bot 的主对话模型、摘要模型 | 记忆存储和检索时的向量化 |

**为什么需要 Embedding 模型？**

当 Bot 和你对话时，它需要从大量历史记忆中找到相关内容。Embedding 模型把"今天天气真好"和"天气晴朗适合出门"映射到相似的向量空间，这样即使措辞不同，也能通过语义相似度找到相关记忆。没有 Embedding 模型，Bot 就没有长期记忆召回能力。

### 常见模型速查表

#### Chat 模型（对话/推理）

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

#### Embedding 模型（向量化）

| 提供商 | 模型名称 | 维度 | 特点 |
|---|---|---|---|
| **OpenAI** | `text-embedding-3-small` | 1536 | 性价比最高，推荐首选 |
| **OpenAI** | `text-embedding-3-large` | 3072 | 精度最高 |
| **OpenAI** | `text-embedding-ada-002` | 1536 | 旧版经典 |
| **阿里/Dashscope** | `text-embedding-v3` | 1024 | 中文优化 |
| **Cohere** | `embed-multilingual-v3.0` | 1024 | 多语言强项 |
| **本地部署** | `bge-large-zh-v1.5` | 1024 | 中文本地模型（HuggingFace） |
| **本地部署** | `nomic-embed-text` | 768 | Ollama 可直接运行 |

> **维度（Dimensions）** 是 Embedding 模型输出向量的长度。创建 Embedding 模型时需要填写正确的维度值，否则向量存储会出错。

### 配置步骤

安装完成后，按以下顺序配置：

```
1. 设置 → Provider      添加 API 服务商（OpenAI / 自定义等），填入 API Key 和 Base URL
       ↓
2. Provider → 模型       在 Provider 下添加模型，选择类型（chat 或 embedding）
       ↓
3. Bot → 设置           选择 Chat 模型、Embedding 模型、语言等
       ↓
4. Bot → 人设           定义 Bot 的身份、灵魂、任务（可选，也可通过文件定义）
       ↓
5. Bot → 频道           连接 Telegram / Discord / 飞书等消息平台
```

**Provider 配置示例：**

| 场景 | Base URL | 说明 |
|---|---|---|
| OpenAI 官方 | `https://api.openai.com/v1` | 需要 API Key |
| Azure OpenAI | `https://{name}.openai.azure.com/openai` | 企业方案 |
| 本地 Ollama | `http://host.docker.internal:11434/v1` | 免费，无需 Key |
| 本地 vLLM | `http://192.168.x.x:8000/v1` | 局域网 GPU 服务器 |
| 第三方代理 | `https://api.openrouter.ai/v1` | 多模型聚合 |

> 本地模型（Ollama / vLLM）可同时用于 Chat 和 Embedding，**零 API 费用**。

### 人设与文件的关系

Bot 的"个性"由两个来源定义，**数据库优先，文件兜底**：

| 来源 | 管理方式 | 优先级 |
|---|---|---|
| **人设栏目**（数据库） | Web UI 直接编辑身份/灵魂/任务 | 高 — 有值则使用 |
| **容器文件**（IDENTITY.md / SOUL.md） | Web UI 文件栏目或 Bot 自己修改 | 低 — 数据库为空时回退 |

- 如果你在"人设"栏目填写了内容，Agent 会使用数据库中的值
- 如果"人设"栏目为空，Agent 会自动读取容器中的 `.md` 文件
- 开启"自我进化"后，Bot 可以在对话中自行修改容器文件，逐渐发展个性
- `TOOLS.md` 始终从容器文件读取（定义 Bot 的工具和能力）

### 心跳与子智能体

**心跳（Heartbeat）**

心跳让 Bot 不只是被动应答，而是能主动行动：

- 新建 Bot 自动创建默认心跳（每小时一次）
- 支持定时触发（间隔秒数）和事件触发（任务完成、收到消息等）
- 心跳触发时，系统向 Bot 发送一条提示词（如"检查待办任务"），Bot 自主执行
- 可在 Web UI 中修改间隔、提示词、触发条件，或添加多个心跳

**子智能体（Subagents）**

子智能体是 Bot 可以委派任务的专业化工作者：

- Agent 在对话中**自动创建和调度**子智能体（无需手动干预）
- 你也可以在 Web UI 中预注册模板（名称、描述、技能），Agent 会优先使用
- 支持 spawn（后台异步）和 query（同步等待结果）两种调度模式
- 每个子智能体有独立的对话上下文和工具权限

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
| 19 | 长期记忆 | Qdrant 向量语义搜索，每轮自动入库 | SQLite-vec 向量搜索 + memoryFlush | **M** |
| 20 | 上下文压缩 | Token 预算裁剪 + LLM 自动摘要 | /compact 手动压缩 | **M** |
| 21 | 分层上下文 | OpenViking（L0/L1/L2），每 Bot 可独立开关 | 无 | **M** |
| 22 | 自我进化机制 | EXPERIMENTS.md 实验追踪 + SOUL.md 自省循环 | MEMORY.md 手动迭代 | **M** |
| 23 | Daily Notes | 模板 + 心跳自动蒸馏为长期记忆 | memory/日期.md 手动记录 | **M** |
| 24 | 跨 Agent 协调 | /shared 自动挂载 + 文件协调 | sessions 工具 + 文件协调 | **=** |
| 25 | 定时任务 | Cron + 可视化管理 UI | Cron 调度（CLI 配置） | **M** |
| 26 | 心跳机制 | 定时 + 事件驱动双模式 | 定时心跳 | **M** |
| 27 | 自愈能力 | 自动检测过期任务并补跑 + 异常上报用户 | HEARTBEAT.md 手动配置自愈逻辑 | **M** |
| 28 | 管理界面 | 完整 Web UI（10+ 模块） | Control UI + CLI + TUI 三合一 | **M** |
| 29 | 多用户支持 | 原生多成员 + 角色权限（admin/member） | 单用户 | **M** |
| 30 | 平台覆盖 | Telegram、Discord、飞书、Web 聊天 | Telegram、Discord、WhatsApp、Slack、Teams、Signal、iMessage 等 12+ | **O** |
| 31 | Token 用量统计 | 每条回复显示消耗 + Dashboard 曲线图 + 多 Bot 对比 | /usage 命令查询 | **M** |
| 32 | Bot 文件管理 | Web UI 在线查看/编辑模板文件 | 本地文件系统 + Git 自动初始化 | **M** |
| 33 | 认证安全 | JWT + 多用户权限体系 | Gateway Token + Pairing Code | **M** |
| 34 | 容器快照/回滚 | containerd 快照 + 版本回滚 | Git 版本控制 | **M** |
| 35 | 搜索引擎集成 | 可配置多搜索引擎 | Brave Search 单一引擎 | **M** |
| 36 | 前端国际化 | 中文 + 英文完整 i18n | 英文为主，部分中文文档 | **M** |
| 37 | 语音 / TTS | 无 | Voice Wake + Talk Mode + ElevenLabs TTS | **O** |
| 38 | 可视化画布 | 无 | Canvas + A2UI 交互式画布 | **O** |
| 39 | Companion Apps | 无 | macOS + iOS + Android 原生应用 | **O** |
| 40 | Webhook / 邮件集成 | 无 | Webhook + Gmail Pub/Sub | **O** |
| 41 | 模型故障切换 | 备用模型自动 Failover（sync + stream） | Model Failover 自动切换 | **=** |
| 42 | 诊断工具 | 系统诊断面板（PG/Qdrant/Gateway/Containerd/磁盘） | openclaw doctor 安全审计 + 诊断 | **=** |

**汇总：Memoh-v2 胜 26 项 · OpenClaw 胜 8 项 · 持平 8 项**

---

## 安装与升级

### 一键安装

```bash
curl -fsSL https://raw.githubusercontent.com/Kxiandaoyan/Memoh-v2/main/scripts/install.sh | sh
```

安装脚本会自动：检测 Docker → 检测旧版本（可选清理）→ 克隆代码 → 生成 config.toml → 构建并启动所有服务。

支持交互式配置工作目录、数据目录、管理员密码等；加 `-y` 跳过交互。

### 升级（不丢数据）

```bash
curl -fsSL https://raw.githubusercontent.com/Kxiandaoyan/Memoh-v2/main/scripts/upgrade.sh | sh
```

脚本会自动查找 Memoh 项目目录（当前目录、`./Memoh-v2/`、`~/memoh/Memoh-v2/`），无需手动 `cd`。

也可以在项目目录下直接执行：

```bash
cd ~/memoh/Memoh-v2 && ./scripts/upgrade.sh
```

升级流程：自动备份数据库 → `git pull` 拉取最新代码 → 重建 Docker 镜像 → 执行数据库迁移 → 健康检查。

所有数据（PostgreSQL、Qdrant、Bot 文件）存储在 Docker named volumes 和宿主机目录中，**升级不会丢失任何数据**。

| 参数 | 说明 |
|------|------|
| `--no-backup` | 跳过升级前数据库备份 |
| `--no-pull` | 跳过 git pull（已手动更新代码时） |
| `-y` | 静默模式，跳过所有确认提示 |

> 传参示例：`curl -fsSL ... | sh -s -- --no-backup -y`

### 卸载

```bash
curl -fsSL https://raw.githubusercontent.com/Kxiandaoyan/Memoh-v2/main/scripts/uninstall.sh | sh
```

或在项目目录下直接执行：

```bash
cd ~/memoh/Memoh-v2 && ./scripts/uninstall.sh
```

默认卸载会删除容器、镜像和数据卷。可加参数保留数据：

| 参数 | 说明 |
|------|------|
| `--keep-data` | 保留 Docker volumes（数据库、向量库、Bot 数据不删） |
| `--keep-images` | 保留已构建的 Docker 镜像 |
| `-y` | 静默模式 |

> 传参示例：`curl -fsSL ... | sh -s -- --keep-data`

卸载前会自动创建数据库最终备份到 `backups/` 目录。

### 数据库管理

```bash
./scripts/db-up.sh      # 执行数据库迁移（增量，跳过已应用的）
./scripts/db-drop.sh     # 回滚所有表（⚠️ 危险操作，需确认）
```

### 数据迁移到新服务器

1. 在旧服务器备份：

```bash
docker compose exec -T postgres pg_dump -U memoh memoh | gzip > memoh-backup.sql.gz
```

2. 拷贝到新服务器，安装 Memoh-v2

3. 启动服务后导入：

```bash
gunzip -c memoh-backup.sql.gz | docker compose exec -T postgres psql -U memoh memoh
```

Bot 文件数据（TOOLS.md、ov.conf 等）在宿主机 `data/bots/` 目录下，直接拷贝即可。

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
| Server（后端） | Go + Echo + FX | 8080 |
| Agent Gateway | Bun + Elysia | 8081 |
| Web（前端） | Vue 3 + Vite + Tailwind | 8082 |

依赖：PostgreSQL、Qdrant、Containerd

---

## 致谢

本项目基于 [Memoh](https://github.com/memohai/Memoh) 二次开发，感谢原作者的优秀工作。
