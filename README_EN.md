<div align="center">

# Memoh-v2

**Containerized · Structured Long-Memory · Self-Evolving AI Agent System**

[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Go](https://img.shields.io/badge/Go-1.25-00ADD8?logo=go)](https://go.dev)
[![Vue](https://img.shields.io/badge/Vue-3-4FC08D?logo=vuedotjs)](https://vuejs.org)
[![Docker](https://img.shields.io/badge/Docker-Compose-2496ED?logo=docker)](https://docs.docker.com/compose/)

Each bot gets its own container, memory, skills, and self-evolution — your personal AI agent platform.

[中文](./README.md) · [Quick Start](#quick-start) · [Feature Guide](#feature-guide) · [Installation & Upgrade](#installation--upgrade)

</div>

---

## Table of Contents

- [Quick Start](#quick-start)
- [Architecture Overview](#architecture-overview)
- [Feature Guide](#feature-guide)
  - [Bot Management & Templates](#1-bot-management--templates)
  - [Conversation & Streaming](#2-conversation--streaming)
  - [Memory System](#3-memory-system)
  - [Container System](#4-container-system)
  - [Channel Adapters](#5-channel-adapters)
  - [MCP Tool System](#6-mcp-tool-system)
  - [Heartbeat & Scheduled Tasks](#7-heartbeat--scheduled-tasks)
  - [Self-Evolution System](#8-self-evolution-system)
  - [Subagents & Skills](#9-subagents--skills)
  - [OpenViking Tiered Context](#10-openviking-tiered-context-database)
  - [Token Usage & Diagnostics](#11-token-usage--diagnostics)
  - [Cross-Bot Collaboration](#12-cross-bot-collaboration)
- [Concepts Guide](#concepts-guide)
- [Known Limitations](#known-limitations)
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

After installation, configure in this order:

```
1. Settings -> Provider    Add API provider, enter API Key and Base URL
       |
2. Provider -> Models      Add models (chat or embedding type)
       |
3. New Bot                 Select a template or start blank, set name and type
       |
4. Bot -> Settings         Choose Chat model, Embedding model, language, etc.
       |
5. Bot -> Channels         Connect Telegram / Lark messaging platforms (optional)
```

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
| **Server** (Go) | REST API, auth, database, container management, conversation routing, memory retrieval |
| **Agent Gateway** (Bun) | AI inference, system prompt assembly, tool execution, streaming, subagent dispatch |
| **Web** (Vue 3) | Management UI: bots, models, channels, skills, files, evolution, heartbeat visualization |
| **PostgreSQL** | Relational data (users, bots, messages, configs, evolution logs) |
| **Qdrant** | Vector database (memory semantic search) |
| **Containerd** | Container runtime (one isolated container per bot) |

**Data flow:** User message → Channel adapter → Server (auth, memory retrieval, payload assembly) → Agent Gateway (LLM inference, tool calls) → Server (response persistence, memory extraction, token tracking) → User

---

## Feature Guide

### 1. Bot Management & Templates

**Bot Management** is the system's core. Each bot is an independent AI agent entity with:

- Independent identity definition (Identity / Soul / Task — three-layer persona)
- Independent container sandbox (file system, command execution, network access)
- Independent memory space (vector database partition isolation)
- Independent channel configurations (Telegram / Lark / etc.)
- Membership and permissions (Owner / Admin / Member roles)
- Lifecycle management (creating → ready → deleting)
- Runtime health checks (container init, data paths, task status)

**Bot Templates** are new in v2, providing 10 pre-built deep persona templates for two-step professional bot creation:

| Step | Action |
|------|--------|
| **Step 1** | Select a preset persona from the template grid (or choose "Blank Bot" to start from scratch) |
| **Step 2** | Fill in bot name, type, and other basics, then submit |

The system automatically applies the template's Identity, Soul, and Task content to the bot's persona configuration.

**13 Built-in Templates:**

| Template | Mental Model | Category | Focus |
|----------|-------------|----------|-------|
| CEO Strategist | Jeff Bezos | Business | Strategic decisions, business models, prioritization |
| CTO Architect | Werner Vogels | Development | Tech architecture, selection decisions, reliability |
| Full Stack Dev | DHH | Development | Code implementation, tech approach, code review |
| Interaction Design | Alan Cooper | Design | User flows, Persona-driven design, interaction patterns |
| Marketing Strategy | Seth Godin | Business | Positioning, differentiation, growth strategy |
| Growth Operations | Paul Graham | Business | Cold start, user retention, community ops |
| Product Design | Don Norman | Design | Product definition, usability, cognitive design |
| Quality Assurance | James Bach | Development | Test strategy, risk assessment, quality control |
| Sales Strategy | Aaron Ross | Business | Pricing strategy, sales funnels, conversion optimization |
| UI Design | Matias Duarte | Design | Visual design, design systems, typography & color |
| Research Analyst | — | Productivity | Deep research, multi-source verification, structured output |
| Daily Secretary | — | Productivity | Task management, scheduling, commitment tracking |
| Knowledge Curator | — | Productivity | Knowledge capture, organization, second brain |

10 templates are ported from the [Solo-Company-Skill](https://github.com/anthropics/anthropic-cookbook) project's real thought-leader mental models, each containing core beliefs, decision frameworks, solo developer tips, and communication styles. The remaining 3 are general-purpose productivity roles.

Each template contains three Markdown files (`identity.md`, `soul.md`, `task.md`), written in Chinese, extracting the essence for maximum information density and minimum token consumption.

### 2. Conversation & Streaming

Two conversation modes are supported:

| Mode | Description | Use Case |
|------|-------------|----------|
| **SSE Streaming** | Server pushes tokens incrementally, frontend renders in real-time | Web chat UI, scenarios needing instant feedback |
| **Synchronous** | Waits for complete response, returns all at once | API calls, channel messages (Telegram/Lark) |

**Context management:**

- Last 24 hours of conversation auto-loaded as short-term memory
- Token-budget pruning + LLM summarization when context exceeds limits
- Relevant long-term memories recalled via semantic search
- Key information auto-extracted into long-term memory after each exchange

### 3. Memory System

The memory system is Memoh's core differentiator, using a three-layer hybrid architecture:

| Layer | Technology | Purpose |
|-------|-----------|---------|
| **Vector Semantic Search** | Qdrant | Embeds text into vectors via Embedding model, retrieves by semantic similarity |
| **BM25 Keyword Index** | Built-in | Traditional keyword matching, supplements semantic search for exact lookups |
| **LLM Smart Extraction** | Chat model | Auto-extracts key facts after each conversation turn, filters noise |

**Memory lifecycle:**

```
Conversation → LLM extracts key info → Embedding vectorization → Stored in Qdrant
                                                                      ↓
New conversation → Semantic search + BM25 matching → Recall memories → Inject into context
                                                                      ↓
Periodic maintenance → LLM memory compaction → Merge similar, remove noise → Lean memory store
```

**Memory management UI** provides: manual memory creation, semantic search, batch deletion, memory compaction (three intensity levels: light/medium/heavy), usage statistics.

### 4. Container System

Each bot has its own containerd container, providing a fully isolated execution environment:

| Capability | Description |
|-----------|-------------|
| **File I/O** | Bot can create, read, modify, and delete files in its data directory |
| **Command Execution** | Bot can execute arbitrary shell commands inside the container |
| **Network Access** | Bot can access external APIs and websites |
| **Browser Automation** | Built-in Chromium + xvfb for headless browser operations |
| **Snapshots & Rollback** | Create snapshots at any time, restore to historical state |
| **Skill Installation** | One-click install community skills via ClawHub CLI |
| **Shared Directory** | `/shared` directory shared across all bot containers |

**Pre-installed capabilities:**
- **agent-browser** — Browser automation framework
- **Actionbook** — Pre-compiled website operation manuals for common sites
- **ClawHub CLI** — Skill marketplace command-line tool
- **OpenViking** — Tiered context database (requires activation)

### 5. Channel Adapters

Bots connect to external messaging platforms through channel adapters. Four adapters are currently implemented:

| Adapter | Platform | Description |
|---------|----------|-------------|
| **Telegram** | Telegram | Via Bot Token, supports private and group chats |
| **Feishu** | Feishu / Lark | Via Feishu app credentials |
| **Web** | Built-in web chat | Chat directly in the management UI, no external platform needed |
| **Local CLI** | Command line | For development and debugging, local CLI chat |

Each bot can have multiple channel configurations simultaneously. Channel adapters handle: message format conversion, identity resolution, routing, response delivery.

Users can link Telegram / Feishu accounts to system accounts via Bind Codes for cross-platform identity unification.

### 6. MCP Tool System

Bot capabilities are extended through the MCP (Model Context Protocol) tool system, divided into two categories:

**Built-in Tools (15)** — Every bot has these automatically:

| Category | Tools | Function |
|----------|-------|----------|
| File Ops | `read` / `write` / `list` / `edit` | Container file I/O, directory listing, text replacement |
| Execution | `exec` | Execute shell commands inside the container |
| Messaging | `send` / `react` | Send messages to channels, add emoji reactions |
| User Lookup | `lookupChannelUser` | Look up users or groups by platform ID |
| Memory | `searchMemory` | Search memories relevant to current conversation |
| Web Search | `webSearch` | Search the web via configured search provider |
| Scheduling | `listSchedule` / `getSchedule` / `createSchedule` / `updateSchedule` / `deleteSchedule` | Full scheduled task management |

**External MCP Servers** — Add per-bot via the management UI:

| Transport | Description | Example |
|-----------|-------------|---------|
| **Stdio** | Launches process inside the container | `npx @modelcontextprotocol/server-filesystem` |
| **Remote** | Connects to remote HTTP/SSE service | `https://mcp.example.com/sse` |

Supports bulk import of standard `mcpServers` JSON configuration. The Tool Gateway proxies tool calls from the Agent Gateway to MCP servers running inside containers.

### 7. Heartbeat & Scheduled Tasks

**Heartbeat** transforms bots from passive responders to proactive actors:

- New bots **automatically** get a default heartbeat (hourly maintenance check)
- Supports **time-based triggers** (interval in seconds) and **event triggers** (task completed, message received, etc.)
- When triggered, sends a prompt to the bot; the bot decides its own actions
- Add multiple heartbeats via the management UI, each with independent intervals, prompts, and triggers
- The self-evolution system uses a dedicated heartbeat (marked `[evolution-reflection]`, default 24-hour interval)

**Scheduled Tasks** let bots execute periodic work via cron expressions:

- Bots can **autonomously create** scheduled tasks during conversation (via built-in schedule tools)
- Also manageable through the management UI or API
- Each task has a cron expression, command prompt, max execution count, and enable/disable toggle
- Management UI displays all task statuses and execution counts

### 8. Self-Evolution System

Self-evolution is Memoh's core differentiating capability. Bots can learn from conversations and automatically improve their persona and behavior.

**Core philosophy:** Evolution is organic — driven by real conversations, not forced schedules. If recent conversations went smoothly with no friction, no evolution is needed. The system only triggers change when there is genuine material to learn from.

**Three-phase evolution cycle:**

| Phase | Name | Action |
|-------|------|--------|
| **Phase 1** | Reflect | Review recent conversations for friction (wrong answers, user frustration), delight (satisfied user), patterns (recurring topics), and gaps (missing knowledge). If no meaningful signals are found, report "no evolution needed" and stop |
| **Phase 2** | Experiment | For each actionable insight: log in EXPERIMENTS.md (trigger, observation, action, expected outcome), then make a small, reversible change in the appropriate file (IDENTITY.md / SOUL.md / TOOLS.md) |
| **Phase 3** | Review | Self-healing maintenance: check scheduled tasks, distill daily notes into long-term memory, verify coordination files, report anomalies to user |

**Evolution log tracking:**

When an evolution heartbeat fires, the system automatically creates an `evolution_logs` record (status: running). After the Agent Gateway returns the result, the system automatically updates the status:
- `completed` — Changes were made
- `skipped` — No evolution needed (recent conversations were smooth)
- `failed` — An error occurred

The management UI's evolution page shows:
- Toggle control and manual trigger button
- Experiments timeline (parsed from EXPERIMENTS.md)
- Evolution history timeline (from evolution_logs with status badges and expandable agent response details)
- Persona file viewer (IDENTITY.md / SOUL.md / TOOLS.md / EXPERIMENTS.md / NOTES.md)

### 9. Subagents & Skills

**Subagents** are specialized workers that bots can delegate tasks to:

- Agents **automatically create and dispatch** subagents during conversations (no manual setup needed)
- Pre-register templates (name, description, skills) in the management UI — agents prioritize registered definitions
- Two dispatch modes: **spawn** (async background) and **query** (sync wait for result)
- Each subagent has its own conversation context and tool permissions

**Skills** are Markdown files stored in the container that extend bot capabilities in specific domains:

- Each skill contains a name, description, and detailed instructions
- The agent auto-loads relevant skills based on conversation context
- Create manually via the management UI or install from ClawHub (community skill marketplace)
- Bots can also install new skills during conversations

### 10. OpenViking Tiered Context Database

[OpenViking](https://github.com/volcengine/OpenViking) is a tiered context database integrated into Memoh, giving bots structured long-term memory beyond flat vector retrieval.

**Why OpenViking?**

Regular vector memory is "flat" — every memory is stored and retrieved equally. But human memory is layered: core knowledge (always needed) vs. fine details (occasionally referenced). OpenViking mirrors this structure:

| Tier | Name | Description | Analogy |
|------|------|-------------|---------|
| **L0** | Summary Layer | Highly compressed overview | A book's table of contents |
| **L1** | Knowledge Layer | Structured knowledge and key facts | A book's chapter summaries |
| **L2** | Detail Layer | Complete original content and details | A book's full text |

During conversation, the bot first loads L0 summaries (minimal tokens) for the big picture, then loads specific L1/L2 sections on demand, dramatically reducing token consumption.

**How to enable:** Toggle "Enable OpenViking Context Database" in bot settings. The system auto-generates `ov.conf` with pre-filled API info.

### 11. Token Usage & Diagnostics

**Token Usage Tracking:**

- Every LLM call (chat, heartbeat, scheduled task, memory operations) auto-records token consumption
- Management UI provides a Dashboard: daily usage charts + multi-bot comparison
- View data for the last 7 / 30 / 90 days

**System Diagnostics:**

One-click health check for all dependent services (PostgreSQL, Qdrant, Agent Gateway, Containerd) for quick issue identification.

### 12. Cross-Bot Collaboration

**Shared Workspace:** All bot containers auto-mount a `/shared` directory pointing to the same host path.

When a bot is created, the system auto-creates a dedicated output folder at `/shared/{bot_name}/`. File storage convention:

| Path | Scope | Purpose |
|------|-------|---------|
| `/data/` | Private | System files (IDENTITY.md, SOUL.md, TOOLS.md, etc.) |
| `/shared/{bot_name}/` | Shared | Bot's output documents (reports, analysis, drafts) |
| `/shared/{other_bot}/` | Read-only | Other bots' output (readable, not writable) |

Bots coordinate through files:

```
Agent A writes report to /shared/AgentA/ → Agent B reads → Agent B writes draft to /shared/AgentB/
```

Coordination is file-based — simple, reliable, no API needed. The management UI provides file browsing and editing for the shared workspace.

---

## Concepts Guide

### Model Types: Chat vs Embedding

Memoh uses two types of AI models:

| | Chat Model | Embedding Model |
|---|---|---|
| **Purpose** | Understand instructions and generate replies (the "brain") | Convert text to number vectors (the "indexer" for memory search) |
| **Analogy** | A person who can think and speak | A librarian who catalogs books with tags |
| **Input** | Conversation context + user message | Any text snippet |
| **Output** | Natural language response | Fixed-length array of floats (a vector) |
| **In Memoh** | Bot's main chat model, summarization model | Vectorizing memories for storage and retrieval |

Without an Embedding model, your bot has no long-term memory recall.

### Common Models Cheat Sheet

#### Chat Models

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

#### Embedding Models

| Provider | Model Name | Dimensions | Notes |
|---|---|---|---|
| **OpenAI** | `text-embedding-3-small` | 1536 | Best value, recommended default |
| **OpenAI** | `text-embedding-3-large` | 3072 | Highest precision |
| **Alibaba/Dashscope** | `text-embedding-v3` | 1024 | Optimized for Chinese |
| **Qwen** | [`qwen3-embedding-8b`](https://openrouter.ai/qwen/qwen3-embedding-8b) | Variable | **Recommended** — multilingual + long text + code, 32K context |
| **Local** | `nomic-embed-text` | 768 | Runs directly in Ollama |

> **Top Pick: [Qwen3 Embedding 8B](https://openrouter.ai/qwen/qwen3-embedding-8b)** — Multilingual, long-text, and code retrieval leader. 32K context. Available via OpenRouter at just $0.01/M tokens.

> **Dimensions** is the length of the vector output. You must enter the correct value when creating an Embedding model.

### Persona System

A bot's "personality" is defined across three dimensions, each corresponding to a Markdown file:

| Dimension | File | Purpose |
|-----------|------|---------|
| **Identity** | `IDENTITY.md` | Who the bot is — name, role, background, core philosophy |
| **Soul** | `SOUL.md` | How the bot behaves — core beliefs, principles, boundaries, communication style |
| **Task** | `TASK.md` | What the bot does — specific workflows, checklists, output formats |

**Priority rule: Database first, files as fallback.**

| Source | Management | Priority |
|--------|-----------|----------|
| **Persona Tab** (database) | Edit directly in Web UI | High — used if values exist |
| **Container Files** (.md files) | Edit via Files tab or bot self-modification | Low — fallback when database is empty |

With self-evolution enabled, bots can modify their own container files during conversations, gradually developing personality. `TOOLS.md` is always read from container files.

### Provider Configuration Examples

| Scenario | Base URL | Notes |
|----------|----------|-------|
| OpenAI Official | `https://api.openai.com/v1` | Requires API Key |
| Azure OpenAI | `https://{name}.openai.azure.com/openai` | Enterprise option |
| Local Ollama | `http://host.docker.internal:11434/v1` | Free, no Key needed |
| Local vLLM | `http://192.168.x.x:8000/v1` | LAN GPU server |
| Third-party proxy | `https://api.openrouter.ai/v1` | Multi-model aggregator |

> Local models (Ollama / vLLM) work for both Chat and Embedding — zero API cost.

---

## Known Limitations

An objective assessment of current shortcomings. Some have workarounds, others require future iteration.

### Embedding Provider Compatibility

| Issue | Only OpenAI-compatible and DashScope embedding providers are fully implemented. Other providers (Bedrock, Cohere, etc.) return "provider not implemented" errors |
|-------|------|
| **Impact** | Users with non-OpenAI-format embedding APIs cannot use the memory system |
| **Workaround** | Use OpenRouter or similar OpenAI-compatible aggregation services, or deploy a local Embedding model via Ollama |

### Channel Adapter Coverage

| Issue | Only 4 adapters implemented: Telegram, Feishu, Web, CLI. Discord, Slack, WhatsApp are not implemented |
|-------|------|
| **Impact** | Users on Discord / Slack platforms cannot connect directly |
| **Note** | This is an intentional trade-off — the project targets single-user personal assistants; Telegram + Feishu covers the primary use cases |

### Channel Binding Error Messages

| Issue | Telegram and Feishu adapters return vague "binding is incomplete" errors without specifying which field is missing |
|-------|------|
| **Impact** | Users have difficulty troubleshooting configuration issues |

### No Evolution Auto-Rollback

| Issue | Self-evolution can modify IDENTITY.md / SOUL.md / TOOLS.md, but there's no one-click rollback if evolution degrades behavior |
|-------|------|
| **Workaround** | Use container snapshot functionality to manually restore to a historical state |
| **Planned** | Evolution diff tracking and one-click revert |

### Evolution Quality Depends on Model Capability

| Issue | Self-evolution quality depends heavily on the underlying LLM model's reflection and self-assessment capabilities |
|-------|------|
| **Impact** | Weaker models may produce low-quality evolution changes or fail to accurately identify conversation friction points |
| **Recommendation** | Use Claude Sonnet, GPT-4o, or equivalent-capability models for evolution |

### OpenViking Documentation Gap

| Issue | The OpenViking feature toggle exists, but lacks user documentation explaining how it works, when to use it, and how it relates to the standard memory system |
|-------|------|
| **Impact** | Users are unsure whether to enable this feature |

### Platform Support

| Platform | Status |
|----------|--------|
| **Linux** | Fully supported, recommended for production |
| **macOS** | Requires Lima for containerd (`mise run lima-up`) |
| **Windows** | No native containerd support; requires WSL2 or Docker Desktop |

### SDK Type Sync

| Issue | The template system and evolution log API additions have not yet been regenerated into the frontend TypeScript SDK via `mise run swagger-generate && mise run sdk-generate` |
|-------|------|
| **Impact** | Frontend temporarily uses `as any` type casts and raw `client.get()` calls as workarounds |

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
| 13 | Tool Execution | MCP protocol (sandboxed in container) | Pi Runtime built-in (Browser/Canvas/Nodes) | **O** |
| 14 | MCP Protocol | Native, connects to any MCP Server | Limited + ACP protocol | **M** |
| 15 | Browser Automation | Chromium + agent-browser + Actionbook + xvfb | Built-in Browser + agent-browser + Actionbook | **=** |
| 16 | Smart Web Strategy | Markdown Header → Actionbook → curl 3-tier fallback | Standard fetching | **M** |
| 17 | Skill Marketplace | ClawHub + OPC Skills | ClawHub + OPC Skills | **=** |
| 18 | Short-term Memory | Last 24h auto-loaded | Current session only | **M** |
| 19 | Long-term Memory | Qdrant vector semantic + BM25 keyword, auto-indexed per turn | SQLite-vec vector + memoryFlush | **M** |
| 20 | Context Compression | Token-budget pruning + LLM auto-summarization | /compact manual compression | **M** |
| 21 | Tiered Context | OpenViking (L0/L1/L2), toggleable per bot | None | **M** |
| 22 | Self-Evolution | Three-phase organic cycle (Reflect/Experiment/Review) + evolution log tracking | MEMORY.md manual iteration | **M** |
| 23 | Bot Templates | 13 mental-model templates (10 real thought-leaders), 2-step creation | None | **M** |
| 24 | Daily Notes | Template + heartbeat auto-distillation to long-term memory | memory/date.md manual logging | **M** |
| 25 | Cross-Agent Coordination | /shared auto-mounted + file coordination | sessions tools + file coordination | **=** |
| 26 | Scheduled Tasks | Cron + visual management UI | Cron scheduling (CLI config) | **M** |
| 27 | Heartbeat | Periodic + event-driven dual mode | Periodic heartbeat | **M** |
| 28 | Self-Healing | Auto-detect stale tasks + force re-run + report to user | HEARTBEAT.md manual config | **M** |
| 29 | Management UI | Full Web UI (10+ modules) | Control UI + CLI + TUI combo | **M** |
| 30 | Multi-User | Native multi-member + role permissions (admin/member) | Single-user | **M** |
| 31 | Platform Coverage | Telegram, Lark, Web chat, CLI | Telegram, Discord, WhatsApp, Slack, Teams, Signal, iMessage, etc. 12+ | **O** |
| 32 | Token Usage | Per-response display + Dashboard charts + multi-bot comparison | /usage command query | **M** |
| 33 | Bot File Management | Web UI online view/edit | Local filesystem + Git auto-init | **M** |
| 34 | Auth Security | JWT + multi-user permission system | Gateway Token + Pairing Code | **M** |
| 35 | Snapshots / Rollback | containerd snapshots + version rollback | Git version control | **M** |
| 36 | Search Engines | Configurable multiple engines (Brave / SerpAPI) | Brave Search only | **M** |
| 37 | Frontend i18n | Full Chinese + English i18n | English primary, partial Chinese docs | **M** |
| 38 | Voice / TTS | None | Voice Wake + Talk Mode + ElevenLabs TTS | **O** |
| 39 | Visual Canvas | None | Canvas + A2UI interactive workspace | **O** |
| 40 | Companion Apps | None | macOS + iOS + Android native apps | **O** |
| 41 | Webhook / Email | None | Webhook + Gmail Pub/Sub | **O** |
| 42 | Model Failover | Fallback model auto-failover (sync + stream) | Automatic model failover | **=** |

**Summary: Memoh-v2 wins 27 · OpenClaw wins 8 · Tied 7**

---

## Installation & Upgrade

### One-Click Install

```bash
curl -fsSL https://raw.githubusercontent.com/Kxiandaoyan/Memoh-v2/main/scripts/install.sh | sh
```

The install script: detects Docker → detects previous installation (optional cleanup) → clones repo → generates config.toml → builds and starts all services.

Supports interactive configuration for workspace, data directory, admin password, etc. Add `-y` for silent mode.

### Upgrade (No Data Loss)

```bash
curl -fsSL https://raw.githubusercontent.com/Kxiandaoyan/Memoh-v2/main/scripts/upgrade.sh | sh
```

The script auto-locates the Memoh project directory. Or run directly:

```bash
cd ~/memoh/Memoh-v2 && ./scripts/upgrade.sh
```

Upgrade flow: auto-backup database → `git pull` → rebuild Docker images → run database migrations → health check.

**All data (PostgreSQL, Qdrant, bot files) is stored in Docker named volumes and host directories. Upgrades never lose data.**

| Flag | Description |
|------|-------------|
| `--no-backup` | Skip pre-upgrade database backup |
| `--no-pull` | Skip git pull (if code was updated manually) |
| `-y` | Silent mode, skip all confirmation prompts |

### Uninstall

```bash
curl -fsSL https://raw.githubusercontent.com/Kxiandaoyan/Memoh-v2/main/scripts/uninstall.sh | sh
```

| Flag | Description |
|------|-------------|
| `--keep-data` | Keep Docker volumes (database, vector DB, bot data preserved) |
| `--keep-images` | Keep built Docker images |
| `-y` | Silent mode |

A final database backup is auto-created in `backups/` before uninstalling.

### Database Management

```bash
./scripts/db-up.sh      # Run database migrations (incremental)
./scripts/db-drop.sh     # Rollback all tables (dangerous, requires confirmation)
```

### Data Migration

```bash
# Backup on old server
docker compose exec -T postgres pg_dump -U memoh memoh | gzip > memoh-backup.sql.gz

# Restore on new server
gunzip -c memoh-backup.sql.gz | docker compose exec -T postgres psql -U memoh memoh
```

Bot file data is in the host `data/bots/` directory — simply copy it over.

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
| Server (Backend) | Go + Echo + Uber FX + pgx/v5 + sqlc | 8080 |
| Agent Gateway | Bun + Elysia + Vercel AI SDK | 8081 |
| Web (Frontend) | Vue 3 + Vite + Tailwind CSS + Pinia | 8082 |

| Dependency | Version | Purpose |
|-----------|---------|---------|
| PostgreSQL | 18 | Relational data storage |
| Qdrant | latest | Vector database |
| Containerd | v2 | Container runtime |

---

## Acknowledgments

This project is a secondary development based on [Memoh](https://github.com/memohai/Memoh). Thanks to the original authors for their excellent work.
