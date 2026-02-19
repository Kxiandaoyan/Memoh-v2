# Known Limitations

> Back to [Documentation](./README.md) · [Project Home](../README_EN.md)

---

An objective assessment of current shortcomings. Some have workarounds, others require future iteration.

## Embedding Provider Compatibility

| Issue | Only OpenAI-compatible and DashScope embedding providers are fully implemented. Other providers (Bedrock, Cohere, etc.) return "provider not implemented" errors |
|-------|------|
| **Impact** | Users with non-OpenAI-format embedding APIs cannot use the memory system |
| **Workaround** | Use OpenRouter or similar OpenAI-compatible aggregation services, or deploy a local Embedding model via Ollama |

## Channel Adapter Coverage

| Issue | Only 4 adapters implemented: Telegram, Feishu, Web, CLI. Discord, Slack, WhatsApp are not implemented |
|-------|------|
| **Impact** | Users on Discord / Slack platforms cannot connect directly |
| **Note** | This is an intentional trade-off — the project targets single-user personal assistants; Telegram + Feishu covers the primary use cases |

## Channel Binding Error Messages

| Issue | Telegram and Feishu adapters return vague "binding is incomplete" errors without specifying which field is missing |
|-------|------|
| **Impact** | Users have difficulty troubleshooting configuration issues |

## No Evolution Auto-Rollback

| Issue | Self-evolution can modify IDENTITY.md / SOUL.md / TOOLS.md, but there's no one-click rollback if evolution degrades behavior |
|-------|------|
| **Workaround** | Use container snapshot functionality to manually restore to a historical state |
| **Planned** | Evolution diff tracking and one-click revert |

## Evolution Quality Depends on Model Capability

| Issue | Self-evolution quality depends heavily on the underlying LLM model's reflection and self-assessment capabilities |
|-------|------|
| **Impact** | Weaker models may produce low-quality evolution changes or fail to accurately identify conversation friction points |
| **Recommendation** | Use Claude Sonnet, GPT-4o, or equivalent-capability models for evolution |

## OpenViking Documentation Gap

| Issue | The OpenViking feature toggle exists, but lacks user documentation explaining how it works, when to use it, and how it relates to the standard memory system |
|-------|------|
| **Impact** | Users are unsure whether to enable this feature |

## Platform Support

| Platform | Status |
|----------|--------|
| **Linux** | Fully supported, recommended for production |
| **macOS** | Requires Lima for containerd (`mise run lima-up`) |
| **Windows** | No native containerd support; requires WSL2 or Docker Desktop |

## SDK Type Sync

| Issue | The template system and evolution log API additions have not yet been regenerated into the frontend TypeScript SDK via `mise run swagger-generate && mise run sdk-generate` |
|-------|------|
| **Impact** | Frontend temporarily uses `as any` type casts and raw `client.get()` calls as workarounds |
