<!-- version: 2.0.0 - Added 15 browser/webread MCP tools (browser_*, actionbook_*, web_read) -->

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

## Browser Automation — MCP Tools

You have first-class MCP tools for browser automation. These are **native MCP tools**, not CLI wrappers — use them directly without `exec`.

### Core Workflow

```
navigate → snapshot → interact → extract → persist
```

1. **Navigate**: Open a URL with `browser_navigate`
2. **Snapshot**: Get interactive elements with `browser_snapshot` (returns @eN references)
3. **Interact**: Use `browser_click`, `browser_fill`, etc. with @eN refs
4. **Extract**: Get data with `browser_get_text`, `browser_screenshot`
5. **Persist**: Save session state with `browser_state_save` for reuse

### Available Tools

#### browser_navigate

Navigate to a URL.

**Parameters**:
- `url` (string, required): The URL to navigate to

**Example**:
```json
{
  "url": "https://github.com/login"
}
```

**Returns**: Success message with final URL

---

#### browser_snapshot

Get a snapshot of the current page, including interactive elements with @eN references.

**Parameters**: None

**Example**:
```json
{}
```

**Returns**:
```
URL: https://github.com/login
Title: Sign in to GitHub

Interactive elements:
[@e1] input#login_field (Username or email)
[@e2] input#password (Password)
[@e3] input[type=submit] (Sign in)
[@e4] a[href="/password_reset"] (Forgot password?)
```

**Usage**: Always call `browser_snapshot` after navigation to get fresh element references.

---

#### browser_click

Click an element by its @eN reference.

**Parameters**:
- `selector` (string, required): Element reference from snapshot (e.g., `"@e1"`)

**Example**:
```json
{
  "selector": "@e3"
}
```

**Returns**: Success message

---

#### browser_fill

Fill an input field (clears existing content first, then types new text).

**Parameters**:
- `selector` (string, required): Element reference from snapshot (e.g., `"@e1"`)
- `value` (string, required): Text to fill

**Example**:
```json
{
  "selector": "@e1",
  "value": "octocat@github.com"
}
```

**Returns**: Success message

---

#### browser_get_text

Extract text content from an element or the entire page.

**Parameters**:
- `selector` (string, optional): Element reference from snapshot (e.g., `"@e5"`). If omitted, returns all page text.

**Example (specific element)**:
```json
{
  "selector": "@e5"
}
```

**Example (entire page)**:
```json
{}
```

**Returns**: Text content of the element or page

---

#### browser_screenshot

Take a screenshot of the current page.

**Parameters**:
- `path` (string, optional): File path to save screenshot (if omitted, returns base64)
- `full_page` (boolean, optional): Capture full page (default: false)

**Example (full page to file)**:
```json
{
  "path": "/shared/screenshots/login-page.png",
  "full_page": true
}
```

**Example (viewport as base64)**:
```json
{}
```

**Returns**: File path or base64-encoded PNG data

---

#### browser_get_url

Get the current URL of the page.

**Parameters**: None

**Example**:
```json
{}
```

**Returns**: Current URL as string

---

#### browser_state_save

Save the current browser state (cookies, localStorage, sessionStorage) to a file for later reuse.

**Parameters**:
- `path` (string, required): File path to save state (e.g., `/shared/browser-states/github-auth.json`)

**Example**:
```json
{
  "path": "/shared/browser-states/github-auth.json"
}
```

**Returns**: Success message with file path

**Use case**: Save login sessions to avoid re-authenticating on every task.

---

#### browser_state_load

Load previously saved browser state (cookies, localStorage, sessionStorage) from a file.

**Parameters**:
- `path` (string, required): File path to load state from

**Example**:
```json
{
  "path": "/shared/browser-states/github-auth.json"
}
```

**Returns**: Success message

**Workflow**:
1. Login once manually with `browser_navigate` + `browser_fill` + `browser_click`
2. Save state with `browser_state_save`
3. On subsequent tasks, load state with `browser_state_load` before navigating

---

#### browser_close

Close the browser instance.

**Parameters**: None

**Example**:
```json
{}
```

**Returns**: Success message

**Note**: Browser resources are automatically cleaned up, but explicit closing is recommended for long-running tasks.

---

#### browser_scroll

Scroll to a specific element or by x/y coordinates.

**Parameters**:
- `selector` (string, optional): Element reference to scroll to (e.g., `"@e5"`)
- `x` (number, optional): Horizontal scroll offset in pixels
- `y` (number, optional): Vertical scroll offset in pixels

**Example (scroll to element)**:
```json
{
  "selector": "@e5"
}
```

**Example (scroll by coordinates)**:
```json
{
  "x": 0,
  "y": 1000
}
```

**Returns**: Success message

---

#### browser_wait

Wait for an element to appear on the page or wait for a fixed duration.

**Parameters**:
- `selector` (string, optional): Element reference or CSS selector to wait for
- `timeout` (number, optional): Maximum wait time in milliseconds (default: 30000)

**Example (wait for element)**:
```json
{
  "selector": "@e10",
  "timeout": 5000
}
```

**Example (wait for duration)**:
```json
{
  "timeout": 3000
}
```

**Returns**: Success message when element appears, or timeout error

---

#### actionbook_search

Search the Actionbook registry for pre-computed website operation manuals.

**Parameters**:
- `query` (string, required): Search query (e.g., `"github create repo"`, `"twitter post tweet"`)

**Example**:
```json
{
  "query": "github create repository"
}
```

**Returns**: List of matching actionbooks with IDs and descriptions

**Use case**: Before automating a known website, search for existing actionbooks to avoid manual exploration.

---

#### actionbook_get

Retrieve a specific actionbook by its ID.

**Parameters**:
- `id` (string, required): Actionbook ID from search results

**Example**:
```json
{
  "id": "github-create-repo"
}
```

**Returns**: Full actionbook content with step-by-step instructions and element selectors

**Workflow**:
1. Search with `actionbook_search`
2. Get the manual with `actionbook_get`
3. Follow the instructions with `browser_navigate`, `browser_click`, `browser_fill`, etc.

---

### Migration Note

**Before (CLI wrapper)**:
```json
{
  "tool": "exec",
  "parameters": {
    "command": "agent-browser open https://example.com"
  }
}
```

**After (native MCP tool)**:
```json
{
  "tool": "browser_navigate",
  "parameters": {
    "url": "https://example.com",
    "wait_for": "networkidle"
  }
}
```

No more `exec` needed — these are first-class MCP tools with proper parameter validation, error handling, and return values.

---

## Smart Web Reading — web_read

The `web_read` tool provides intelligent web content extraction with automatic fallback strategies.

### Overview

Instead of manually choosing between markdown headers, actionbooks, or browser automation, `web_read` tries multiple strategies in priority order:

1. **Markdown Header** — Fast, lightweight (tries `Accept: text/markdown`)
2. **Web Search** — For search queries instead of direct URLs
3. **Actionbook** — Pre-computed manuals for known websites
4. **Browser Automation** — Full browser rendering for complex pages

### Parameters

- `url` (string, required): URL to read or search query
- `force_strategy` (string, optional): Force a specific strategy
  - `"markdown"` — Only try markdown header
  - `"search"` — Treat as search query
  - `"browser"` — Only use browser automation
  - Omit for automatic fallback (recommended)
- `include_metadata` (boolean, optional): Include page metadata (title, description, etc.) in response (default: false)

### Example Request

**Automatic fallback**:
```json
{
  "url": "https://docs.python.org/3/library/asyncio.html"
}
```

**Force browser**:
```json
{
  "url": "https://example.com/dynamic-content",
  "force_strategy": "browser",
  "include_metadata": true
}
```

**Search query**:
```json
{
  "url": "latest React hooks documentation",
  "force_strategy": "search"
}
```

### Example Response

```
Title: asyncio — Asynchronous I/O
URL: https://docs.python.org/3/library/asyncio.html

# asyncio — Asynchronous I/O

asyncio is a library to write concurrent code using the async/await syntax.

## Coroutines and Tasks

Coroutines declared with async/await syntax...

[... full markdown content ...]

---
Strategy used: markdown
```

### When to Use

- **Reading documentation**: Use `web_read` — it will try markdown first, fall back to browser if needed
- **Complex interactions** (login, form submission, multi-step flows): Use `browser_navigate` + `browser_click` + `browser_fill` directly
- **Known websites**: `web_read` will automatically check actionbooks
- **Search queries**: Pass a search query with `force_strategy: "search"`

### Comparison: web_read vs. browser_* tools

| Use Case | Recommended Tool |
|----------|------------------|
| Read an article or documentation | `web_read` (automatic fallback) |
| Login to a website | `browser_navigate` + `browser_fill` + `browser_click` |
| Extract data from a dynamic page | `web_read` (will use browser if needed) |
| Multi-step workflow (create repo, post tweet) | `actionbook_search` + `browser_*` tools |
| Save login session for reuse | `browser_state_save` after manual login |
| Quick content preview | `web_read` with `force_strategy: "markdown"` |

---

## Integration Notes

### MCP Tool Architecture

All browser automation and web reading tools are **first-class MCP tools**:

- No `exec` wrapper needed
- Proper JSON parameter validation
- Structured error responses
- Return values are machine-readable

### Tool Categories

| Category | Tools | Use Case |
|----------|-------|----------|
| **Browser Navigation** | `browser_navigate`, `browser_get_url`, `browser_close` | Page navigation |
| **Browser Inspection** | `browser_snapshot`, `browser_screenshot` | Page analysis |
| **Browser Interaction** | `browser_click`, `browser_fill`, `browser_scroll`, `browser_wait` | User actions |
| **Browser Extraction** | `browser_get_text` | Data extraction |
| **Browser Persistence** | `browser_state_save`, `browser_state_load` | Session management |
| **Actionbook** | `actionbook_search`, `actionbook_get` | Pre-computed manuals |
| **Smart Reading** | `web_read` | Automatic content extraction |

### Total Tools Added

**15 new MCP tools**:
- 11 browser automation tools (`browser_*`)
- 2 actionbook tools (`actionbook_*`)
- 1 smart reading tool (`web_read`)
- 1 deprecated CLI (`agent-browser` — still available but prefer MCP tools)

---
