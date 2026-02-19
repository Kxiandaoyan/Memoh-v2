<div align="center">

# Memoh-v2

**容器化 · 结构化长记忆 · 自我进化 AI Agent 系统**

[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Go](https://img.shields.io/badge/Go-1.25-00ADD8?logo=go)](https://go.dev)
[![Vue](https://img.shields.io/badge/Vue-3-4FC08D?logo=vuedotjs)](https://vuejs.org)
[![Docker](https://img.shields.io/badge/Docker-Compose-2496ED?logo=docker)](https://docs.docker.com/compose/)

每个 Bot 拥有独立容器、独立记忆、独立技能、自我进化能力 —— 你的私人 AI 智能体平台。

[English](./README_EN.md) · [功能详解](./doc/features.md) · [安装与升级](./doc/installation.md) · [使用教程](./doc/README.md) · [界面截图](./doc/screenshots.md)

</div>

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

> 详细安装、升级、卸载、数据迁移指南请查看 **[安装与升级](./doc/installation.md)**。

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

---

## 界面预览

<p align="center">
  <img src="./doc/1.png" width="100%" />
</p>

<p align="center">
  <a href="./doc/screenshots.md">👉 点击查看更多截图</a>
</p>

---

## 核心功能

> 每项功能的完整说明请查看 **[功能详解](./doc/features.md)**。

- **Bot 管理与模板** — 13 套思维模型模板（含 10 位真实思想家），两步创建专业化 Bot
- **对话与流式推送** — SSE 实时流式 + 同步两种模式，自动上下文管理与记忆召回
- **三层记忆系统** — 向量语义搜索 + BM25 关键词 + LLM 智能提取，对话后自动入库
- **独立容器沙箱** — 每个 Bot 拥有 containerd 隔离容器，支持文件、命令、浏览器、快照回滚
- **多平台频道接入** — Telegram / 飞书 / Web 聊天 / CLI，跨平台身份统一
- **MCP 工具系统** — 15 个内置工具 + 任意外部 MCP 服务器，支持 Stdio 和 Remote 传输
- **心跳与定时任务** — 定时 + 事件双模式触发，Bot 主动行动而非被动应答
- **自我进化** — 三阶段有机进化周期（反思/实验/审查），进化日志完整追踪
- **子智能体与技能** — 自动调度子智能体，ClawHub 技能市场一键安装
- **OpenViking 分层上下文** — L0/L1/L2 三层结构化记忆，大幅减少 Token 消耗
- **Token 用量与诊断** — Dashboard 曲线图 + 多 Bot 对比，一键服务健康检查
- **跨 Bot 协作** — `/shared` 共享工作区，文件驱动的简单协调机制

---

## 文档导航

| 文档 | 说明 |
|------|------|
| **[功能详解](./doc/features.md)** | 12 项核心功能的完整介绍 |
| **[概念指南](./doc/concepts.md)** | 模型类型、人设体系、Provider 配置示例 |
| **[安装与升级](./doc/installation.md)** | 一键安装 / 升级 / 卸载 / 数据迁移 |
| **[已知局限性](./doc/known-limitations.md)** | 当前不足与变通方案 |
| **[与 OpenClaw 对比](./doc/comparison.md)** | 42 项全面对比 |
| **[使用教程](./doc/README.md)** | 18 篇操作教程（快速上手到高级技巧） |
| **[界面截图](./doc/screenshots.md)** | 更多界面展示 |
| **[项目完成进度](./doc/FEATURE_AUDIT.md)** | 74 项功能审计 |
| **[项目提示词](./doc/PROMPTS_INVENTORY.md)** | 全部提示词清单 |

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
