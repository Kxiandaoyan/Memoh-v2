# Concepts Guide

> Back to [Documentation](./README.md) · [Project Home](../README_EN.md)

---

## Model Types: Chat vs Embedding

Memoh uses two types of AI models:

| | Chat Model | Embedding Model |
|---|---|---|
| **Purpose** | Understand instructions and generate replies (the "brain") | Convert text to number vectors (the "indexer" for memory search) |
| **Analogy** | A person who can think and speak | A librarian who catalogs books with tags |
| **Input** | Conversation context + user message | Any text snippet |
| **Output** | Natural language response | Fixed-length array of floats (a vector) |
| **In Memoh** | Bot's main chat model, summarization model | Vectorizing memories for storage and retrieval |

Without an Embedding model, your bot has no long-term memory recall.

## Common Models Cheat Sheet

### Chat Models

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

### Embedding Models

| Provider | Model Name | Dimensions | Notes |
|---|---|---|---|
| **OpenAI** | `text-embedding-3-small` | 1536 | Best value, recommended default |
| **OpenAI** | `text-embedding-3-large` | 3072 | Highest precision |
| **Alibaba/Dashscope** | `text-embedding-v3` | 1024 | Optimized for Chinese |
| **Qwen** | [`qwen3-embedding-8b`](https://openrouter.ai/qwen/qwen3-embedding-8b) | Variable | **Recommended** — multilingual + long text + code, 32K context |
| **Local** | `nomic-embed-text` | 768 | Runs directly in Ollama |

> **Top Pick: [Qwen3 Embedding 8B](https://openrouter.ai/qwen/qwen3-embedding-8b)** — Multilingual, long-text, and code retrieval leader. 32K context. Available via OpenRouter at just $0.01/M tokens.

> **Dimensions** is the length of the vector output. You must enter the correct value when creating an Embedding model.

## Persona System

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

## Provider Configuration Examples

| Scenario | Base URL | Notes |
|----------|----------|-------|
| OpenAI Official | `https://api.openai.com/v1` | Requires API Key |
| Azure OpenAI | `https://{name}.openai.azure.com/openai` | Enterprise option |
| Local Ollama | `http://host.docker.internal:11434/v1` | Free, no Key needed |
| Local vLLM | `http://192.168.x.x:8000/v1` | LAN GPU server |
| Third-party proxy | `https://api.openrouter.ai/v1` | Multi-model aggregator |

> Local models (Ollama / vLLM) work for both Chat and Embedding — zero API cost.
