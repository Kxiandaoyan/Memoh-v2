# 概念指南

> 返回 [文档首页](./README.md) · [项目首页](../README.md)

---

## 模型类型：Chat vs Embedding

Memoh 使用两种类型的 AI 模型：

| | Chat 模型 | Embedding 模型 |
|---|---|---|
| **用途** | 理解指令并生成回复（对话的"大脑"） | 将文本转为数字向量（记忆搜索的"索引器"） |
| **类比** | 像一个能思考和说话的人 | 像一个图书馆编目员，给每本书贴标签 |
| **输入** | 对话上下文 + 用户消息 | 任意文本片段 |
| **输出** | 自然语言回复 | 固定长度的浮点数数组（向量） |
| **在 Memoh 中** | Bot 的主对话模型、摘要模型 | 记忆存储和检索时的向量化 |

没有 Embedding 模型，Bot 就没有长期记忆召回能力。

## 常见模型速查表

### Chat 模型

| 提供商 | 模型名称 | 特点 |
|---|---|---|
| **OpenAI** | `gpt-4o` | 多模态旗舰，支持图片输入 |
| **OpenAI** | `gpt-4o-mini` | 性价比之选，速度快 |
| **OpenAI** | `o3-mini` | 推理增强型 |
| **Anthropic** | `claude-sonnet-4-20250514` | 代码和长文本强项 |
| **Anthropic** | `claude-3-5-haiku-20241022` | 快速轻量 |
| **Google** | `gemini-2.0-flash` | 速度极快，上下文窗口大 |
| **DeepSeek** | `deepseek-chat` | 中文优秀，性价比高 |
| **Qwen** | `qwen-plus` | 阿里千问，中文强项 |
| **本地部署** | `qwen3-8b` / `llama-3-8b` 等 | 通过 Ollama / vLLM 本地运行，零 API 费用 |

### Embedding 模型

| 提供商 | 模型名称 | 维度 | 特点 |
|---|---|---|---|
| **OpenAI** | `text-embedding-3-small` | 1536 | 性价比最高，推荐首选 |
| **OpenAI** | `text-embedding-3-large` | 3072 | 精度最高 |
| **阿里/Dashscope** | `text-embedding-v3` | 1024 | 中文优化 |
| **Qwen** | [`qwen3-embedding-8b`](https://openrouter.ai/qwen/qwen3-embedding-8b) | 可变 | **推荐** — 多语言 + 长文本 + 代码，32K 上下文 |
| **本地部署** | `nomic-embed-text` | 768 | Ollama 可直接运行 |

> **推荐首选：[Qwen3 Embedding 8B](https://openrouter.ai/qwen/qwen3-embedding-8b)** — 多语言、长文本、代码检索表现领先，32K 上下文，通过 OpenRouter 使用仅 $0.01/百万 token。

> **维度（Dimensions）** 是 Embedding 模型输出向量的长度。创建 Embedding 模型时需要填写正确的维度值。

## 人设体系

Bot 的"个性"由三个维度定义，每个维度对应一份 Markdown 内容：

| 维度 | 文件 | 作用 |
|------|------|------|
| **Identity** | `IDENTITY.md` | 定义 Bot 是谁 — 名字、角色、背景、核心哲学 |
| **Soul** | `SOUL.md` | 定义 Bot 怎么行为 — 核心信条、行为原则、边界、沟通风格 |
| **Task** | `TASK.md` | 定义 Bot 做什么 — 具体工作流、检查清单、输出格式 |

**优先级规则：数据库优先，文件兜底。**

| 来源 | 管理方式 | 优先级 |
|------|----------|--------|
| **人设栏目**（数据库） | Web UI 直接编辑 | 高 — 有值则使用 |
| **容器文件**（.md 文件） | 文件栏目编辑或 Bot 自行修改 | 低 — 数据库为空时回退 |

开启"自我进化"后，Bot 可在对话中自行修改容器文件，逐渐发展个性。`TOOLS.md` 始终从容器文件读取。

## Provider 配置示例

| 场景 | Base URL | 说明 |
|------|----------|------|
| OpenAI 官方 | `https://api.openai.com/v1` | 需要 API Key |
| Azure OpenAI | `https://{name}.openai.azure.com/openai` | 企业方案 |
| 本地 Ollama | `http://host.docker.internal:11434/v1` | 免费，无需 Key |
| 本地 vLLM | `http://192.168.x.x:8000/v1` | 局域网 GPU 服务器 |
| 第三方代理 | `https://api.openrouter.ai/v1` | 多模型聚合 |

> 本地模型（Ollama / vLLM）可同时用于 Chat 和 Embedding，零 API 费用。
