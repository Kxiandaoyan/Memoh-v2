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

## Skill Marketplaces — ClawHub & OPC Skills

You have access to two skill registries:

1. **[ClawHub](https://clawhub.ai)** — Thousands of ready-to-use skills. Use the `clawhub` CLI (pre-installed).
2. **[OPC Skills](https://github.com/ReScienceLab/opc-skills)** — Curated skills for solopreneurs (SEO, Reddit, Twitter, domain hunting, logo/banner creation). Use `npx skills add`.

### OPC Skills (quick install)

```bash
# Install a specific OPC skill
npx skills add ReScienceLab/opc-skills --skill reddit
npx skills add ReScienceLab/opc-skills --skill seo-geo
npx skills add ReScienceLab/opc-skills --skill domain-hunter

# Install multiple at once
npx skills add ReScienceLab/opc-skills --skill reddit --skill twitter --skill domain-hunter
```

> Available OPC skills: `seo-geo`, `requesthunt`, `domain-hunter`, `logo-creator`, `banner-creator`, `nanobanana`, `reddit`, `twitter`, `producthunt`

### ClawHub (search + install)

### Search for skills

```bash
clawhub search "your query"
```

### Install a skill

```bash
clawhub install <skill-slug> --dir /data/.skills
```

> **Important**: Always use `--dir /data/.skills` so the skill is installed to the correct directory and immediately available via `use_skill`.

### Update skills

```bash
clawhub update <skill-slug> --dir /data/.skills
clawhub update --all --dir /data/.skills
```

### List installed skills

```bash
clawhub list --dir /data/.skills
```

### Workflow

When the user asks for a capability you don't have, follow this process:

1. Search ClawHub: `clawhub search "relevant keywords"`
2. Pick the best matching skill from the results
3. Install it: `clawhub install <slug> --dir /data/.skills`
4. Activate it: call `use_skill` with the installed skill name
5. Follow the skill's instructions to complete the task

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

> **Note**: OpenViking is only available when enabled in your bot's Persona settings.

### Python API (via `exec`)

```bash
# Initialize a data directory for this bot
python3 -c "
import openviking as ov
client = ov.SyncOpenViking(path='/app/openviking-data')
client.initialize()
print('OpenViking initialized')
client.close()
"
```

### Core Operations

```bash
# Add a resource (URL, file, or directory)
python3 -c "
import openviking as ov
client = ov.SyncOpenViking(path='/app/openviking-data')
client.initialize()
result = client.add_resource(path='https://example.com/doc.md')
print(result)
client.close()
"

# List directory structure
python3 -c "
import openviking as ov
client = ov.SyncOpenViking(path='/app/openviking-data')
client.initialize()
print(client.ls('viking://resources/'))
client.close()
"

# Semantic search
python3 -c "
import openviking as ov
client = ov.SyncOpenViking(path='/app/openviking-data')
client.initialize()
client.wait_processed()
results = client.find('your search query', target_uri='viking://resources/')
for r in results.resources:
    print(f'{r.uri} (score: {r.score:.4f})')
client.close()
"

# Read content by URI
python3 -c "
import openviking as ov
client = ov.SyncOpenViking(path='/app/openviking-data')
client.initialize()
content = client.read('viking://resources/...')
print(content)
client.close()
"

# Get tiered summaries
python3 -c "
import openviking as ov
client = ov.SyncOpenViking(path='/app/openviking-data')
client.initialize()
print('Abstract:', client.abstract('viking://resources/...'))
print('Overview:', client.overview('viking://resources/...'))
client.close()
"
```

### Key Concepts

- **viking://resources/** — External resources (docs, repos, web pages)
- **viking://user/memories/** — User preferences, habits
- **viking://agent/skills/** — Agent capabilities
- **viking://agent/memories/** — Agent task experience
- **L0 (Abstract)** — One-sentence summary (~100 tokens)
- **L1 (Overview)** — Core info for planning (~2k tokens)
- **L2 (Details)** — Full original data (load on demand)
- Use `find()` for semantic search, `ls()` + `glob()` for filesystem navigation.
- Always call `wait_processed()` after adding resources before searching.

---

## Shared Workspace — Cross-Agent Coordination

A shared directory is mounted at `/shared` in every bot container. Use it for file-based coordination between agents.

### Directory structure convention

```
/shared/
├── intel/                  # Research agent writes here
│   └── daily-research.md
├── drafts/                 # Content agents write drafts here
│   └── tweets-2026-02-17.md
└── logs/                   # Coordination logs
    └── handoff-log.md
```

### Usage rules

1. **One writer per file.** Name files with your role prefix so others know who owns it.
2. **Include timestamps.** Add `Last updated: YYYY-MM-DD HH:MM` at the top of shared files.
3. **Read before writing.** Check what others have produced before starting your task.
4. **Don't delete others' files.** Only modify files you own.

### Example workflow

```bash
# Research agent writes intel
echo "## $(date +%F) Research\n- Finding 1\n- Finding 2" > /shared/intel/daily-research.md

# Content agent reads intel and writes drafts
cat /shared/intel/daily-research.md
echo "Draft based on today's intel..." > /shared/drafts/content-$(date +%F).md
```

> **Tip:** The filesystem _is_ the coordination layer. No APIs, no message queues. Just files. Simple and reliable.

---
