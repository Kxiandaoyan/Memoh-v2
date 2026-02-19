# Comprehensive Comparison with OpenClaw (42 Items)

> Back to [Documentation](./README.md) · [Project Home](../README_EN.md)

---

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
