---
name: skill-creator
description: "Guide for creating effective skills. Use when users want to create a new skill, write a SKILL.md, define trigger conditions, write YAML frontmatter, or update an existing skill file that extends Claude's capabilities with specialized knowledge, workflows, or tool integrations."
---

# Skill Creator

This skill provides guidance for creating effective skills.

## Core Principles

### Concise is Key

The context window is a public good. Skills share the context window with everything else Claude needs: system prompt, conversation history, other Skills' metadata, and the actual user request.

**Default assumption: Claude is already very smart.** Only add context Claude doesn't already have. Challenge each piece of information: "Does Claude really need this explanation?" and "Does this paragraph justify its token cost?"

Prefer concise examples over verbose explanations.

### Set Appropriate Degrees of Freedom

Match the level of specificity to the task's fragility and variability:

**High freedom (text-based instructions)**: Use when multiple approaches are valid, decisions depend on context, or heuristics guide the approach.

**Medium freedom (pseudocode or scripts with parameters)**: Use when a preferred pattern exists, some variation is acceptable, or configuration affects behavior.

**Low freedom (specific scripts, few parameters)**: Use when operations are fragile and error-prone, consistency is critical, or a specific sequence must be followed.

### Anatomy of a Skill

Every skill consists of a required SKILL.md file and optional bundled resources:

```
skill-name/
├── SKILL.md (required)
│   ├── YAML frontmatter: name (required), description (required)
│   └── Markdown instructions
└── Bundled Resources (optional)
    ├── scripts/     - Executable code (Python/Bash/etc.)
    ├── references/  - Documentation loaded into context as needed
    └── assets/      - Files used in output (templates, icons, etc.)
```

#### SKILL.md Frontmatter

- **name**: Unique identifier (lowercase, hyphens for spaces)
- **description**: What the skill does AND when to use it — this is the primary trigger

#### Body

- Keep under 500 lines
- Use imperative/infinitive form
- Move detailed content to `references/` files and link from SKILL.md

**Do NOT include**: README.md, CHANGELOG.md, or other auxiliary documentation. Skills are for AI agents, not users.

### Progressive Disclosure

Three-level loading system:

1. **Metadata** (name + description) — always in context (~100 words)
2. **SKILL.md body** — loaded when skill triggers (<5k words)
3. **Bundled resources** — loaded by Claude as needed

Keep SKILL.md under 500 lines. When splitting content to references, clearly describe when to read them.

**Key principle:** Keep core workflow in SKILL.md; move variant-specific details to reference files.

Example structure for multi-domain skills:

```
bigquery-skill/
├── SKILL.md (overview + navigation)
└── references/
    ├── finance.md
    ├── sales.md
    └── product.md
```

Claude only loads the relevant reference file when needed.

## Creating a Skill

### 1. Understand concrete examples

Ask: "What would a user say to trigger this skill?" Collect 3-5 real examples.

### 2. Plan reusable contents

For each example, identify what scripts, references, or assets would avoid repeated work.

| Resource Type | When to Use                     | Example                               |
| ------------- | ------------------------------- | ------------------------------------- |
| `scripts/`    | Code rewritten repeatedly       | `rotate_pdf.py` for PDF rotation      |
| `assets/`     | Same boilerplate each time      | HTML/React starter for webapp builder |
| `references/` | Documentation needed repeatedly | Database schemas for BigQuery skill   |

### 3. Create SKILL.md

Here is a complete, minimal working example:

```markdown
---
name: deploy-checker
description: "Verify deployment readiness by running pre-deploy checks. Use when users ask to check if a service is ready to deploy, run pre-deploy validation, or verify release criteria."
---

# Deploy Checker

Run these checks before any production deployment:

## Pre-deploy Checklist

1. Run the test suite: `npm test`
2. Verify no uncommitted changes: `git status --porcelain`
3. Confirm the target branch is up to date with main: `git fetch origin && git diff HEAD..origin/main --stat`
4. Check for known vulnerability advisories: `npm audit --production`

## Output Format

Report results as a table:

| Check | Status | Details |
|-------|--------|---------|
| Tests | Pass/Fail | Number of tests run |
| Clean tree | Pass/Fail | List of uncommitted files |
| Branch sync | Pass/Fail | Commits behind main |
| Vulnerabilities | Pass/Fail | Count by severity |

If any check fails, do not proceed with deployment. List the failures and recommend fixes.
```

**Writing guidelines:** Always use imperative/infinitive form.

#### Frontmatter

Write the YAML frontmatter with `name` and `description`:

- `name`: The skill name
- `description`: Primary trigger mechanism. Must include what the skill does AND when to use it (body only loads after triggering).
  - Example: "Document creation and editing with tracked changes. Use for: creating .docx files, modifying content, working with tracked changes."

#### Body

Write instructions for using the skill and its bundled resources. Keep it concise and focused on what Claude needs to know.

### 4. Add bundled resources (optional)

- `scripts/` — deterministic code to avoid rewriting
- `references/` — detailed docs, schemas, API specs
- `assets/` — templates, fonts, boilerplate files

**Avoid duplication**: Information lives in SKILL.md OR references, not both.

### 5. Iterate

Test on real tasks. Improve based on actual performance gaps.

**Iteration workflow:**

1. Use the skill on real tasks
2. Notice struggles or inefficiencies
3. Identify how SKILL.md or bundled resources should be updated
4. Implement changes and test again

## What NOT to Include

- README.md, INSTALLATION.md, CHANGELOG.md
- Explanations of things Claude already knows
- Verbose examples when concise ones suffice
