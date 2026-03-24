---
name: web-search
description: "Real-time web search using a Playwright-controlled browser. Search websites, extract page content, and verify current facts beyond the knowledge cutoff."
---

# Web Search Skill

## When to Use

- Current information beyond the knowledge cutoff (January 2025)
- Latest framework/library documentation
- Fact verification and real-time data
- Troubleshooting specific error messages
- Recent community discussions, news, and comparisons

## Environment Variables

- `SKILLS_ROOT` - Path to the skills directory (automatically set by the system)

## Basic Usage

```bash
bash "$SKILLS_ROOT/web-search/scripts/search.sh" "search query" [max_results]
```

For non-ASCII queries (Chinese/Japanese/etc.), use UTF-8 file input:

```bash
cat > /tmp/web-query.txt <<'TXT'
苹果 Siri AI 2026 发布计划
TXT
bash "$SKILLS_ROOT/web-search/scripts/search.sh" @/tmp/web-query.txt 10
```

## Workflow

1. **Search** for the topic:
   ```bash
   bash "$SKILLS_ROOT/web-search/scripts/search.sh" "Next.js 14 features" 5
   ```
2. **Validate** results - check that URLs and snippets are relevant before synthesizing
3. **Synthesize** - parse Markdown output, summarize key findings, cite sources
4. **Follow up** if needed:
   ```bash
   bash "$SKILLS_ROOT/web-search/scripts/search.sh" "Next.js Server Actions tutorial" 3
   ```

## Output Format

Results are returned as Markdown:

```markdown
# Search Results: TypeScript 5.0 new features

**Query:** TypeScript 5.0 new features
**Results:** 5
**Time:** 834ms

---

## TypeScript 5.0 Release Notes

**URL:** [https://www.typescriptlang.org/docs/...]

TypeScript 5.0 introduces decorators, const type parameters...
```

## Best Practices

- **Be specific** - include version numbers, dates, or specific aspects in queries
- **Limit results** - use 3-5 for quick lookups, 10 for comprehensive research
- **Synthesize** - don't dump raw results; extract key information and summarize
- **Cite sources** - tell the user which sources you are drawing from
- **Verify** - cross-check important claims across multiple results

## Examples

### Latest Documentation

```bash
bash "$SKILLS_ROOT/web-search/scripts/search.sh" "React 19 new features and breaking changes" 5
```
Parse results, find official docs, summarize key features.

### Troubleshooting an Error

```bash
bash "$SKILLS_ROOT/web-search/scripts/search.sh" "TypeScript Cannot find module error solution" 5
```
Extract solutions from Stack Overflow and GitHub issues, provide a step-by-step fix.

### Current Events

```bash
bash "$SKILLS_ROOT/web-search/scripts/search.sh" "AI news January 2026" 10
```
Synthesize news from multiple sources into a concise summary.

## Error Handling

| Issue | Symptom | Solution |
|-------|---------|----------|
| Server down | `Connection refused` | Run `bash "$SKILLS_ROOT/web-search/scripts/start-server.sh"` |
| Browser missing | `Chrome not found` | Install Chrome or Chromium |
| Port conflict | `Address already in use` | Stop conflicting process on port 8923 |
| Stale connection | `Connection not found` | Remove `$SKILLS_ROOT/web-search/.connection` cache file |
| No results | `Found 0 results` | Broaden the query or check internet connection |

## Limitations

- No CAPTCHA handling - user must solve manually if triggered
- Cannot access authenticated or paywalled pages
- Extracts titles and snippets only, not full page content
- Optimized for English and Chinese results

## Additional Resources

- Full documentation: `$SKILLS_ROOT/web-search/README.md`
- Usage examples: `$SKILLS_ROOT/web-search/examples/basic-search.md`
