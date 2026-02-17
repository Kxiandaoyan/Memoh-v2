<div align="center">

# Memoh-v2

**Multi-Member · Structured Long-Memory · Containerized AI Agent System**

[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Go](https://img.shields.io/badge/Go-1.23-00ADD8?logo=go)](https://go.dev)
[![Vue](https://img.shields.io/badge/Vue-3-4FC08D?logo=vuedotjs)](https://vuejs.org)
[![Docker](https://img.shields.io/badge/Docker-Compose-2496ED?logo=docker)](https://docs.docker.com/compose/)

Each bot gets its own container, memory, and skills — deploy AI agents like hiring real employees.

[中文](./README.md) · [Quick Start](#quick-start) · [Concepts Guide](#concepts-guide) · [Installation & Upgrade](#installation--upgrade)

</div>

---

## Table of Contents

- [Quick Start](#quick-start)
- [Core Features](#core-features)
- [Architecture Overview](#architecture-overview)
- [Concepts Guide](#concepts-guide)
  - [Model Types: Chat vs Embedding](#model-types-chat-vs-embedding)
  - [Common Models Cheat Sheet](#common-models-cheat-sheet)
  - [Configuration Steps](#configuration-steps)
  - [Persona vs Files](#persona-vs-files)
  - [Heartbeat & Subagents](#heartbeat--subagents)
- [Comparison with OpenClaw](#comprehensive-comparison-with-openclaw-42-items)
- [Installation & Upgrade](#installation--upgrade)
- [Tech Stack](#tech-stack)
- [Acknowledgments](#acknowledgments)

---

## Quick Start

**Requires Docker:**

```bash
curl -fsSL https://raw.githubusercontent.com/Kxiandaoyan/Memoh-v2/main/scripts/install.sh | sh
```

> Silent install (skip prompts): `curl -fsSL ... | sh -s -- -y`

Or manually:

```bash
git clone --depth 1 https://github.com/Kxiandaoyan/Memoh-v2.git
cd Memoh-v2
docker compose up -d
```

Visit **http://localhost:8082**. Default login: `admin` / `admin123`

---

## Core Features

### Base Capabilities

- **Multi-Bot Management** — Create multiple bots; humans and bots can chat privately or collaborate in groups
- **Container Isolation** — Each bot runs in its own containerd container with full command, file, and network access
- **Memory Engineering** — Conversations stored in PostgreSQL + Qdrant vector DB with semantic search recall
- **Multi-Platform** — Telegram, Discord, Lark (Feishu), and more
- **Visual Configuration** — GUI for Provider, Model, Memory, Channel, MCP, and Skills
- **Scheduled Tasks** — Cron-based task scheduling

### v2 Enhancements

| Feature | Description |
|---|---|
| **Sub-Agent Autonomy** | Spawn/kill/steer sub-agents with independent tool permissions |
| **Browser Control** | Built-in Chromium + agent-browser + xvfb for in-container web automation |
| **Heartbeat Self-Healing** | Periodic + event-driven heartbeat, auto-configured for new bots |
| **Smart Context Compression** | Token-budget pruning + LLM summarization for long conversations |
| **OpenViking Integration** | Tiered context database (L0/L1/L2), toggleable per bot |
| **Dual Skill Marketplaces** | ClawHub + OPC Skills, one-command install inside containers |
| **Actionbook** | Pre-compiled website operation manuals for efficient browser automation |
| **Smart Web Fetching** | Markdown Header -> Actionbook -> plain curl, three-tier strategy |
| **Self-Evolution Loop** | EXPERIMENTS.md tracking + SOUL.md self-reflection mechanism |
| **Daily Notes** | Log template + heartbeat distillation into long-term memory |
| **Cross-Bot Shared Workspace** | `/shared` directory mounted in all containers — files as coordination |
| **Token Usage Tracking** | Per-response token display + Dashboard with comparison charts |
| **Model Failover** | Configure fallback model, auto-switch on primary model failure |
| **System Diagnostics** | One-click health check for PostgreSQL, Qdrant, Gateway, Containerd |
| **Full Management UI** | Files, Skills, Subagents, Heartbeat, History — all visualized |

---

## Architecture Overview

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

| Service | Responsibility |
|---|---|
| **Server** | REST API, authentication, database, container management |
| **Agent Gateway** | AI chat routing, tool execution, streaming |
| **Web** | Management UI: bots, models, channels, skills, files |
| **PostgreSQL** | Relational data storage (users, bots, messages, configs) |
| **Qdrant** | Vector database (memory semantic search) |
| **Containerd** | Container runtime (one isolated container per bot) |

---

## Concepts Guide

### Model Types: Chat vs Embedding

Memoh uses two types of AI models. Understanding the difference is key to configuring the system:

| | Chat Model | Embedding Model |
|---|---|---|
| **Purpose** | Understand instructions and generate replies (the "brain") | Convert text to number vectors (the "indexer" for memory search) |
| **Analogy** | A person who can think and speak | A librarian who catalogs books with tags |
| **Input** | Conversation context + user message | Any text snippet |
| **Output** | Natural language response | Fixed-length array of floats (a vector) |
| **In Memoh** | Bot's main chat model, summarization model | Vectorizing memories for storage and retrieval |

**Why do you need an Embedding model?**

When your bot chats with you, it needs to find relevant content from a large history of memories. An Embedding model maps "the weather is great today" and "it's sunny and perfect for going out" to similar positions in vector space. Even with different wording, semantic similarity finds related memories. Without an Embedding model, your bot has no long-term memory recall.

### Common Models Cheat Sheet

#### Chat Models (Conversation / Reasoning)

| Provider | Model Name | Notes |
|---|---|---|
| **OpenAI** | `gpt-4o` | Multimodal flagship, supports image input |
| **OpenAI** | `gpt-4o-mini` | Best value, fast |
| **OpenAI** | `o3-mini` | Enhanced reasoning |
| **Anthropic** | `claude-sonnet-4-20250514` | Strong at code and long text |
| **Anthropic** | `claude-3-5-haiku-20241022` | Fast and lightweight |
| **Google** | `gemini-2.0-flash` | Extremely fast, large context window |
| **DeepSeek** | `deepseek-chat` | Great for Chinese, high value |
| **Qwen** | `qwen-plus` | Alibaba, strong Chinese support |
| **Local** | `qwen3-8b` / `llama-3-8b` etc. | Run via Ollama / vLLM, zero API cost |

#### Embedding Models (Vectorization)

| Provider | Model Name | Dimensions | Notes |
|---|---|---|---|
| **OpenAI** | `text-embedding-3-small` | 1536 | Best value, recommended default |
| **OpenAI** | `text-embedding-3-large` | 3072 | Highest precision |
| **OpenAI** | `text-embedding-ada-002` | 1536 | Legacy classic |
| **Alibaba/Dashscope** | `text-embedding-v3` | 1024 | Optimized for Chinese |
| **Cohere** | `embed-multilingual-v3.0` | 1024 | Strong multilingual support |
| **Local** | `bge-large-zh-v1.5` | 1024 | Chinese local model (HuggingFace) |
| **Local** | `nomic-embed-text` | 768 | Runs directly in Ollama |

> **Dimensions** is the length of the vector output by the Embedding model. You must enter the correct dimension value when creating an Embedding model, or vector storage will fail.

### Configuration Steps

After installation, configure in this order:

```
1. Settings -> Provider    Add API provider (OpenAI / custom), enter API Key and Base URL
       |
2. Provider -> Models      Add models under the provider, select type (chat or embedding)
       |
3. Bot -> Settings         Choose Chat model, Embedding model, language, etc.
       |
4. Bot -> Persona          Define bot identity, soul, task (optional, can also use files)
       |
5. Bot -> Channels         Connect Telegram / Discord / Lark messaging platforms
```

**Provider Configuration Examples:**

| Scenario | Base URL | Notes |
|---|---|---|
| OpenAI Official | `https://api.openai.com/v1` | Requires API Key |
| Azure OpenAI | `https://{name}.openai.azure.com/openai` | Enterprise option |
| Local Ollama | `http://host.docker.internal:11434/v1` | Free, no Key needed |
| Local vLLM | `http://192.168.x.x:8000/v1` | LAN GPU server |
| Third-party proxy | `https://api.openrouter.ai/v1` | Multi-model aggregator |

> Local models (Ollama / vLLM) work for both Chat and Embedding — **zero API cost**.

### Persona vs Files

A bot's "personality" comes from two sources, **database takes priority, files as fallback**:

| Source | Management | Priority |
|---|---|---|
| **Persona Tab** (database) | Edit identity/soul/task directly in Web UI | High — used if values exist |
| **Container Files** (IDENTITY.md / SOUL.md) | Edit via Files tab or let bot modify itself | Low — fallback when database is empty |

- If you fill in the Persona tab, the agent uses database values
- If the Persona tab is empty, the agent auto-reads container `.md` files
- With "Self-Evolution" enabled, the bot can modify its own container files over time
- `TOOLS.md` is always read from container files (defines bot tools and capabilities)

### Heartbeat & Subagents

**Heartbeat**

Heartbeats let bots act proactively instead of only responding:

- New bots automatically get a default heartbeat (hourly)
- Supports time-based triggers (interval in seconds) and event triggers (task completed, message received, etc.)
- When triggered, the system sends a prompt to the bot (e.g., "check pending tasks"), and the bot acts autonomously
- Customize interval, prompt, and triggers in the Web UI, or add multiple heartbeats

**Subagents**

Subagents are specialized workers that bots can delegate tasks to:

- Agents **automatically create and dispatch** subagents during conversations (no manual setup needed)
- You can also pre-register templates (name, description, skills) in the Web UI — agents prioritize registered definitions
- Supports spawn (async background) and query (sync wait for result) dispatch modes
- Each subagent has its own conversation context and tool permissions

---

## Comprehensive Comparison with OpenClaw (42 Items)

> Result column: **M** = Memoh-v2 wins · **O** = OpenClaw wins · **=** = Tied

| # | Dimension | Memoh-v2 | OpenClaw | Result |
|---|---|---|---|:---:|
| 1 | Backend Language | Go (high concurrency, compiled) | Node.js (single-threaded, interpreted) | **M** |
| 2 | Architecture | Three-service split (Server / Gateway / Web) | Monolithic application | **M** |
| 3 | Communication | SSE unidirectional streaming | WebSocket full-duplex | **O** |
| 4 | Container Isolation | containerd isolated container per bot | Shared runtime (optional Docker sandbox) | **M** |
| 5 | Structured Database | PostgreSQL | SQLite | **M** |
| 6 | Vector Database | Qdrant (standalone service) | SQLite-vec (embedded) | **M** |
| 7 | Horizontal Scaling | Services deploy and scale independently | Single-machine only | **M** |
| 8 | Resource Usage | Requires Docker + PostgreSQL + Qdrant | Lightweight single process, ~tens of MB RAM | **O** |
| 9 | Deployment | Docker Compose one-click | npm install -g + CLI start | **=** |
| 10 | Remote Access | Native (Docker deploys to any server) | Requires Tailscale / SSH tunnel | **M** |
| 11 | Agent Definition | SOUL + IDENTITY + TOOLS + EXPERIMENTS + NOTES | SOUL + IDENTITY + TOOLS + AGENTS + HEARTBEAT + BOOTSTRAP + USER | **=** |
| 12 | Sub-Agent Management | spawn/kill/steer + independent tool perms + registry | spawn/kill/steer + depth limit + max children | **=** |
| 13 | Tool Execution Framework | MCP protocol (sandboxed in container) | Pi Runtime built-in (Browser/Canvas/Nodes) | **O** |
| 14 | MCP Protocol Support | Native, connects to any MCP Server | Limited + ACP protocol | **M** |
| 15 | Browser Automation | Chromium + agent-browser + Actionbook + xvfb | Built-in Browser + agent-browser + Actionbook | **=** |
| 16 | Smart Web Strategy | Markdown Header -> Actionbook -> curl 3-tier fallback | Standard fetching | **M** |
| 17 | Skill Marketplace | ClawHub + OPC Skills | ClawHub + OPC Skills | **=** |
| 18 | Short-term Memory | Last 24h auto-loaded | Current session only | **M** |
| 19 | Long-term Memory | Qdrant vector semantic search, auto-indexed per turn | SQLite-vec vector search + memoryFlush | **M** |
| 20 | Context Compression | Token-budget pruning + LLM auto-summarization | /compact manual compression | **M** |
| 21 | Tiered Context | OpenViking (L0/L1/L2), toggleable per bot | None | **M** |
| 22 | Self-Evolution | EXPERIMENTS.md tracking + SOUL.md self-reflection loop | MEMORY.md manual iteration | **M** |
| 23 | Daily Notes | Template + heartbeat auto-distillation to long-term memory | memory/date.md manual logging | **M** |
| 24 | Cross-Agent Coordination | /shared auto-mounted + file coordination | sessions tools + file coordination | **=** |
| 25 | Scheduled Tasks | Cron + visual management UI | Cron scheduling (CLI config) | **M** |
| 26 | Heartbeat | Periodic + event-driven dual mode | Periodic heartbeat | **M** |
| 27 | Self-Healing | Auto-detect stale tasks + force re-run + report to user | HEARTBEAT.md manual self-healing config | **M** |
| 28 | Management UI | Full Web UI (10+ modules) | Control UI + CLI + TUI triple combo | **M** |
| 29 | Multi-User | Native multi-member + role permissions (admin/member) | Single-user | **M** |
| 30 | Platform Coverage | Telegram, Discord, Lark, Web chat | Telegram, Discord, WhatsApp, Slack, Teams, Signal, iMessage, etc. 12+ | **O** |
| 31 | Token Usage Tracking | Per-response display + Dashboard charts + multi-bot comparison | /usage command query | **M** |
| 32 | Bot File Management | Web UI online view/edit template files | Local filesystem + Git auto-init | **M** |
| 33 | Auth Security | JWT + multi-user permission system | Gateway Token + Pairing Code | **M** |
| 34 | Snapshots / Rollback | containerd snapshots + version rollback | Git version control | **M** |
| 35 | Search Engine Integration | Configurable multiple search engines | Brave Search only | **M** |
| 36 | Frontend i18n | Full Chinese + English i18n | English primary, partial Chinese docs | **M** |
| 37 | Voice / TTS | None | Voice Wake + Talk Mode + ElevenLabs TTS | **O** |
| 38 | Visual Canvas | None | Canvas + A2UI interactive workspace | **O** |
| 39 | Companion Apps | None | macOS + iOS + Android native apps | **O** |
| 40 | Webhook / Email Integration | None | Webhook + Gmail Pub/Sub | **O** |
| 41 | Model Failover | Fallback model auto-failover (sync + stream) | Automatic model failover switching | **=** |
| 42 | Diagnostics | System diagnostics panel (PG/Qdrant/Gateway/Containerd/Disk) | openclaw doctor security audit + diagnostics | **=** |

**Summary: Memoh-v2 wins 26 · OpenClaw wins 8 · Tied 8**

---

## Installation & Upgrade

### One-Click Install

```bash
curl -fsSL https://raw.githubusercontent.com/Kxiandaoyan/Memoh-v2/main/scripts/install.sh | sh
```

The install script will: detect Docker -> detect previous installation (optional cleanup) -> clone repo -> generate config.toml -> build and start all services.

Supports interactive configuration for workspace, data directory, admin password, etc. Add `-y` for silent mode.

### Upgrade (No Data Loss)

```bash
curl -fsSL https://raw.githubusercontent.com/Kxiandaoyan/Memoh-v2/main/scripts/upgrade.sh | sh
```

The script automatically locates the Memoh project directory (current dir, `./Memoh-v2/`, `~/memoh/Memoh-v2/`) — no manual `cd` required.

Or run directly from the project directory:

```bash
cd ~/memoh/Memoh-v2 && ./scripts/upgrade.sh
```

Upgrade flow: auto-backup database -> `git pull` latest code -> rebuild Docker images -> run database migrations -> health check.

All data (PostgreSQL, Qdrant, bot files) is stored in Docker named volumes and host directories. **Upgrades never lose data.**

| Flag | Description |
|------|-------------|
| `--no-backup` | Skip pre-upgrade database backup |
| `--no-pull` | Skip git pull (if code was updated manually) |
| `-y` | Silent mode, skip all confirmation prompts |

> Passing flags: `curl -fsSL ... | sh -s -- --no-backup -y`

### Uninstall

```bash
curl -fsSL https://raw.githubusercontent.com/Kxiandaoyan/Memoh-v2/main/scripts/uninstall.sh | sh
```

Or run directly from the project directory:

```bash
cd ~/memoh/Memoh-v2 && ./scripts/uninstall.sh
```

By default, uninstall removes containers, images, and data volumes. Add flags to preserve data:

| Flag | Description |
|------|-------------|
| `--keep-data` | Keep Docker volumes (database, vector DB, bot data preserved) |
| `--keep-images` | Keep built Docker images |
| `-y` | Silent mode |

> Passing flags: `curl -fsSL ... | sh -s -- --keep-data`

A final database backup is automatically created in `backups/` before uninstalling.

### Database Management

```bash
./scripts/db-up.sh      # Run database migrations (incremental, skips already applied)
./scripts/db-drop.sh     # Rollback all tables (dangerous, requires confirmation)
```

### Migrate to a New Server

1. Backup on the old server:

```bash
docker compose exec -T postgres pg_dump -U memoh memoh | gzip > memoh-backup.sql.gz
```

2. Copy to the new server and install Memoh-v2

3. After starting services, import:

```bash
gunzip -c memoh-backup.sql.gz | docker compose exec -T postgres psql -U memoh memoh
```

Bot files (TOOLS.md, ov.conf, etc.) are in the host `data/bots/` directory — simply copy them over.

### Script Reference

| Script | Purpose |
|--------|---------|
| `scripts/install.sh` | One-click install (fresh deployment) |
| `scripts/upgrade.sh` | One-click upgrade (data preserved) |
| `scripts/uninstall.sh` | Uninstall (optional data retention) |
| `scripts/db-up.sh` | Database migration |
| `scripts/db-drop.sh` | Database rollback |
| `scripts/compile-mcp.sh` | Compile MCP binary and hot-reload into container |

---

## Tech Stack

| Service | Stack | Port |
|---|---|---|
| Server (Backend) | Go + Echo + FX | 8080 |
| Agent Gateway | Bun + Elysia | 8081 |
| Web (Frontend) | Vue 3 + Vite + Tailwind | 8082 |

Dependencies: PostgreSQL, Qdrant, Containerd

---

## Acknowledgments

This project is a secondary development based on [Memoh](https://github.com/memohai/Memoh). Thanks to the original authors for their excellent work.
