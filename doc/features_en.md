# Feature Guide

> Back to [Documentation](./README.md) · [Project Home](../README_EN.md)

---

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
