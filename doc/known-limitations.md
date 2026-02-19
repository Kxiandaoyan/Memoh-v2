# 已知局限性

> 返回 [文档首页](./README.md) · [项目首页](../README.md)

---

以下是对系统当前不足的客观评估。这些问题已知，部分有解决方案，部分需要后续迭代。

## Embedding 提供商兼容性

| 问题 | 只有 OpenAI 兼容和 DashScope 的 Embedding 提供商完整实现，其他提供商（Bedrock、Cohere 等）会返回"provider not implemented"错误 |
|------|------|
| **影响** | 使用非 OpenAI 格式 Embedding API 的用户无法使用记忆系统 |
| **变通** | 使用 OpenRouter 等兼容 OpenAI 格式的聚合服务，或使用本地 Ollama 部署 Embedding 模型 |

## 频道适配器覆盖

| 问题 | 目前仅实现 Telegram、飞书、Web、CLI 四种适配器。Discord、Slack、WhatsApp 等平台未实现 |
|------|------|
| **影响** | 使用 Discord / Slack 等平台的用户无法直接接入 |
| **说明** | 这是有意的取舍 —— 项目定位为单用户个人助手，Telegram + 飞书覆盖了目标用户的主要场景 |

## 频道绑定错误提示

| 问题 | Telegram 和飞书适配器在配置不完整时返回"binding is incomplete"，未指明具体缺少哪个字段 |
|------|------|
| **影响** | 用户难以自行排查配置问题 |

## 进化系统无自动回滚

| 问题 | 自我进化可以修改 Bot 的 IDENTITY.md / SOUL.md / TOOLS.md，但如果进化方向错误导致行为退化，没有一键回滚机制 |
|------|------|
| **变通** | 使用容器快照功能手动恢复到历史状态 |
| **改进方向** | 后续计划增加进化 diff 追踪和一键回退 |

## 进化质量依赖模型能力

| 问题 | 自我进化的质量高度依赖底层 LLM 模型的反思和自我评估能力 |
|------|------|
| **影响** | 较弱的模型可能产生低质量的进化改变，或无法准确识别对话中的摩擦点 |
| **建议** | 进化功能推荐使用 Claude Sonnet、GPT-4o 或同等能力以上的模型 |

## OpenViking 用户文档不足

| 问题 | OpenViking 功能开关存在，但缺乏用户文档说明其工作原理、适用场景、与标准记忆系统的关系 |
|------|------|
| **影响** | 用户不确定是否该开启这个功能 |

## 平台支持限制

| 平台 | 状态 |
|------|------|
| **Linux** | 完全支持，推荐生产部署 |
| **macOS** | 需要通过 Lima 运行 containerd（`mise run lima-up`） |
| **Windows** | 无原生 containerd 支持，需要 WSL2 或 Docker Desktop |

## SDK 类型同步

| 问题 | 新增的模板系统和进化日志 API 尚未通过 `mise run swagger-generate && mise run sdk-generate` 重新生成前端 SDK 类型定义 |
|------|------|
| **影响** | 前端暂时使用 `as any` 类型断言和原始 `client.get()` 调用作为临时方案 |
