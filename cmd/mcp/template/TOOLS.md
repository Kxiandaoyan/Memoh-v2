Skills define _how_ tools work. This file is for _your_ specifics — the stuff that's unique to your setup.

## What Goes Here

Things like:

- SSH hosts and aliases
- Anything environment-specific

## Examples

```markdown
### SSH

- home-server → 192.168.1.100, user: admin
```

## Why Separate?

Skills are shared. Your setup is yours. Keeping them apart means you can update skills without losing your notes, and share skills without leaking your infrastructure.

---

## File Storage Convention

You have two storage locations:

| Path | Scope | Purpose |
|------|-------|---------|
| `/data/` | Private — only you | System files (IDENTITY.md, SOUL.md, TOOLS.md, EXPERIMENTS.md, NOTES.md), skills, config |
| `/shared/` | Shared — all bots can read and write | Output documents, reports, analysis, cross-bot coordination files |

**Rules:**
- **System files** (identity, soul, tools, experiments, notes) stay in `/data/`. Never move them.
- **Output documents** (reports, analysis, drafts, exported data) go to `/shared/`.
- Use descriptive filenames to avoid conflicts (e.g. `research-daily.md`, `content-drafts.md`).

---

## Skill Discovery & Self-Creation

You can autonomously discover, import, and create skills using two dedicated tools:

### discover_skills — Search for skills

Search multiple sources at once for reusable skills:

| Source | What it searches |
|--------|-----------------|
| `clawhub` | ClawHub marketplace (thousands of community skills) |
| `web` | Internet search for public SKILL.md files |
| `shared` | Skills created by other bots in `/shared/.skills/` |
| `all` | All sources in parallel (default) |

Usage: call `discover_skills` with a `query` and optional `source`.

### fork_skill — Import a skill

Fetch a skill from any source and save it to your `/data/.skills/` directory:

| Parameter | Required | Description |
|-----------|----------|-------------|
| `source` | Yes | `clawhub`, `web`, or `shared` |
| `save_as` | Yes | Name for the new skill directory |
| `slug` | clawhub only | ClawHub skill slug |
| `url` | web only | URL to fetch SKILL.md content |
| `skill_name` | shared only | Skill name in `/shared/.skills/` |

The tool returns the original skill content so you can review and adapt it.

### Workflow

When the user asks for a capability you don't have:

1. **Discover**: call `discover_skills` with relevant keywords
2. **Import**: call `fork_skill` to save the best match into your skills
3. **Adapt**: use `write` to modify `/data/.skills/<name>/SKILL.md` to fit your role and context
4. **Activate**: call `use_skill` with the new skill name
5. **Share** (optional): copy your adapted skill to `/shared/.skills/<name>/` so other bots can find it

### Sharing skills between bots

Any skill you place under `/shared/.skills/<name>/SKILL.md` becomes discoverable by other bots via `discover_skills` with `source=shared`. Use this for cross-bot collaboration.

---

## Skill Marketplaces — ClawHub & OPC Skills (CLI)

You also have direct CLI access to skill registries:

1. **[ClawHub](https://clawhub.ai)** — Thousands of ready-to-use skills. Use the `clawhub` CLI (pre-installed).
2. **[OPC Skills](https://github.com/ReScienceLab/opc-skills)** — Curated skills for solopreneurs (SEO, Reddit, Twitter, domain hunting, logo/banner creation). Use `npx skills add`.

### OPC Skills (quick install)

```bash
npx skills add ReScienceLab/opc-skills --skill reddit
npx skills add ReScienceLab/opc-skills --skill seo-geo
npx skills add ReScienceLab/opc-skills --skill domain-hunter
```

> Available OPC skills: `seo-geo`, `requesthunt`, `domain-hunter`, `logo-creator`, `banner-creator`, `nanobanana`, `reddit`, `twitter`, `producthunt`

### ClawHub CLI

```bash
clawhub search "your query"
clawhub install <skill-slug> --dir /data/.skills
clawhub update <skill-slug> --dir /data/.skills
clawhub update --all --dir /data/.skills
clawhub list --dir /data/.skills
```

> **Important**: Always use `--dir /data/.skills` so the skill is immediately available via `use_skill`.

---

## Browser Automation — agent-browser

You have `agent-browser` pre-installed, a headless Chromium CLI for web automation. Use the `exec` tool to run these commands.

### Core workflow

```bash
agent-browser open <url>          # Navigate to a page
agent-browser snapshot -i         # Get interactive elements with refs (@e1, @e2, ...)
agent-browser click @e1           # Click an element by ref
agent-browser fill @e2 "text"     # Fill an input field
agent-browser close               # Close the browser
```

### Common commands

```bash
# Navigation
agent-browser open <url>
agent-browser back / forward / reload

# Page analysis
agent-browser snapshot -i              # Interactive elements only (recommended)
agent-browser screenshot               # Take screenshot
agent-browser get text @e1             # Get element text
agent-browser get url                  # Get current URL

# Interactions (use @refs from snapshot)
agent-browser click @e1
agent-browser fill @e2 "text"          # Clear + type
agent-browser type @e2 "text"          # Append text
agent-browser select @e1 "value"       # Select dropdown
agent-browser scroll down 500
agent-browser press Enter

# Wait
agent-browser wait @e1                 # Wait for element
agent-browser wait --text "Success"    # Wait for text
agent-browser wait --load networkidle  # Wait for network idle

# Tabs
agent-browser tab new [url]            # New tab
agent-browser tab list                 # List tabs

# State
agent-browser state save auth.json     # Save session (cookies, storage)
agent-browser state load auth.json     # Restore session
```

### Tips

- Always `snapshot -i` after navigation to get fresh element refs.
- Use `--json` flag for machine-readable output when parsing results.
- Save login state with `state save` to avoid re-authenticating.
- The browser runs headless inside the container; no display needed.

---

## Actionbook — Pre-computed Website Manuals

[Actionbook](https://actionbook.ai) provides pre-computed "operation manuals" for popular websites. Instead of exploring a page from scratch each time, load the relevant actionbook to know exactly which elements to interact with.

### When to use

- **Known websites** (GitHub, Google, Twitter, YouTube, etc.): Use actionbook first — it's faster and more reliable.
- **Unknown websites**: Fall back to `agent-browser` with `snapshot -i` for exploration.

### Commands

```bash
# Search for an actionbook
actionbook search "github create repo"

# List available actionbooks
actionbook list

# Get a specific actionbook
actionbook get <actionbook-id>

# Example: load the actionbook, then follow its step-by-step instructions
actionbook get github-create-repo
```

### Workflow

1. Before automating a known website, search actionbook: `actionbook search "site action"`
2. If a matching manual exists, follow its steps with `agent-browser`
3. If no manual exists, use `agent-browser snapshot -i` to explore the page manually
4. Consider contributing new actionbooks for frequently used workflows

---

## Smart Web Fetching — Priority Strategy

When you need to read web content, follow this priority order:

### 1. Markdown Header (fastest, cheapest)

Many websites (especially Cloudflare-fronted, documentation sites) support returning Markdown directly:

```bash
curl -s -H "Accept: text/markdown" "https://example.com/docs/page" -o page.md
```

If the response is valid Markdown, you're done — no browser needed.

### 2. Actionbook + agent-browser (for interactions)

If you need to click, fill forms, or navigate through a site:

```bash
actionbook search "site name action"     # Check for pre-built manual
agent-browser open <url>                  # Open the page
agent-browser snapshot -i                 # Explore elements
```

### 3. Plain curl (fallback)

```bash
curl -s "https://example.com/api/data"   # API / JSON endpoints
curl -s "https://example.com/page" | head -200  # Quick HTML peek
```

### Rules of Thumb

- **Reading docs / articles** → Try `Accept: text/markdown` header first
- **Interacting with web apps** → Actionbook + agent-browser
- **APIs / data endpoints** → Plain curl
- **Unknown site, just need text** → Try markdown header, fall back to agent-browser
- Always prefer the lightest method that gets the job done

---

## OpenViking — Context Database (if enabled)

OpenViking is a context database that manages memories, resources, and skills via a filesystem paradigm (`viking://` URIs). It provides tiered context loading (L0 abstract → L1 overview → L2 full) and semantic search.

> **Note**: OpenViking is only available when enabled in your bot's Persona settings. When enabled, dedicated `ov_*` tools appear automatically.

### Configuration

When OpenViking is enabled, an `ov.conf` file is **auto-generated** at `/data/ov.conf`,
pre-populated from your system's configured Models & Providers (embedding model → `embedding.dense`, multimodal chat model → `vlm`).
In most cases, no manual editing is needed. The data directory at `/app/openviking-data` is auto-initialized on first use.

### Native Tools

When OpenViking is enabled, these tools are available directly (no need to write Python scripts):

| Tool | Description |
|------|-------------|
| `ov_initialize` | Initialize the OpenViking data directory (auto-called on first use) |
| `ov_find` | Quick semantic search — returns matching URIs with scores |
| `ov_search` | Advanced retrieval with intent analysis and hierarchical search |
| `ov_read` | Read full content (L2) from a viking:// URI |
| `ov_abstract` | Get L0 abstract (~100 tokens, one-sentence summary) |
| `ov_overview` | Get L1 overview (~2k tokens, structure and key points) |
| `ov_ls` | List directory contents under a viking:// URI |
| `ov_tree` | Get a tree view of directory structure |
| `ov_add_resource` | Add a resource (URL, file, or directory) to be indexed |
| `ov_rm` | Remove a resource by its viking:// URI |
| `ov_session_commit` | Commit conversation messages, extracting long-term memories |

### Typical Workflow

1. **Add resources**: Use `ov_add_resource` to ingest documents, repos, or web pages
2. **Browse**: Use `ov_ls` or `ov_tree` to explore the `viking://` filesystem
3. **Search**: Use `ov_find` for quick semantic search or `ov_search` for deep retrieval
4. **Read**: Use `ov_abstract` for quick summaries, `ov_overview` for detailed summaries, or `ov_read` for full content
5. **Session management**: Conversations are automatically committed for memory extraction after each chat

### Key Concepts

- **viking://resources/** — External resources (docs, repos, web pages)
- **viking://user/memories/** — User preferences, habits
- **viking://agent/skills/** — Agent capabilities
- **viking://agent/memories/** — Agent task experience
- **L0 (Abstract)** — One-sentence summary (~100 tokens)
- **L1 (Overview)** — Core info for planning (~2k tokens)
- **L2 (Details)** — Full original data (load on demand)
- Use `ov_find` for semantic search, `ov_ls` for filesystem navigation
- Resources are automatically processed after adding — `ov_find` will return results once processing completes

### Advanced: Direct Python API

For operations not covered by the native tools, you can still use the Python API via `exec`:

```bash
python3 -c "
import openviking as ov
client = ov.SyncOpenViking(path='/app/openviking-data', config_file='/data/ov.conf')
client.initialize()
# ... your custom operations ...
client.close()
"
```

---

## Shared Workspace — Cross-Agent Coordination

A shared directory is mounted at `/shared` in every bot container. All bots can read and write freely. Use it for file-based coordination.

### Usage rules

1. **Use descriptive filenames.** e.g. `daily-research.md`, `content-drafts.md`, `handoff-log.md`.
2. **Include timestamps.** Add `Last updated: YYYY-MM-DD HH:MM` at the top of shared files.
3. **Read before writing.** Check what others have produced before starting your task.
4. **Organize with subdirectories if needed.** e.g. `/shared/intel/`, `/shared/drafts/`.

> **Tip:** The filesystem _is_ the coordination layer. No APIs, no message queues. Just files. Simple and reliable.

---
