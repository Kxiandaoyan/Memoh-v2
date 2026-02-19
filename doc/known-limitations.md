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

~~此项已解决~~（见下方"已解决"列表）

## 进化系统无自动回滚

~~此项已解决~~（见下方"已解决"列表）

## 进化质量依赖模型能力

| 问题 | 自我进化的质量高度依赖底层 LLM 模型的反思和自我评估能力 |
|------|------|
| **影响** | 较弱的模型可能产生低质量的进化改变，或无法准确识别对话中的摩擦点 |
| **建议** | 进化功能推荐使用 Claude Sonnet、GPT-4o 或同等能力以上的模型 |

## OpenViking 用户文档不足

~~此项已解决~~（见下方"已解决"列表）

## 平台支持限制

| 平台 | 状态 |
|------|------|
| **Linux** | 完全支持，推荐生产部署 |
| **macOS** | 需要通过 Lima 运行 containerd（`mise run lima-up`） |
| **Windows** | 无原生 containerd 支持，需要 WSL2 或 Docker Desktop |

## BM25 统计数据不持久化

~~此项已解决~~（见下方"已解决"列表）

## 记忆提取无 Embedding 去重

~~此项已解决~~（见下方"已解决"列表）

## 工具循环检测基于内存状态

| 问题 | `agent/src/tools/loop-detection.ts` 的检测历史存在进程内存中，Agent Gateway 重启后历史清零 |
|------|------|
| **影响** | 极端情况下进程重启后同一会话可能重置循环计数 |
| **说明** | 实际影响极小，重启本身已打断了循环 |

## 群组消息防抖时间窗口固定

~~此项已解决~~（见下方"已解决"列表）

## 子智能体历史 UI 未实现

~~此项已解决~~（见下方"已解决"列表）

---

## 已解决的历史局限性

以下问题曾列为已知局限性，已在后续版本中解决：

| 问题 | 解决版本 | 说明 |
|------|---------|------|
| SDK 类型同步 —— 前端临时使用 `as any` | 2026-02-18 | 已通过 swagger-generate + sdk-generate 重新生成完整类型定义 |
| 无工具调用循环检测 | 2026-02-19 | 新增 `loop-detection.ts`，三重检测器防止 LLM 陷入重复调用 |
| 记忆搜索无多样性排序 | 2026-02-19 | 实现基于真实 embedding 向量的 MMR 重排序 |
| 无记忆时间衰减 | 2026-02-19 | 实现指数衰减（半衰期 30 天），evergreen 条目跳过衰减 |
| 无 Embedding 缓存 | 2026-02-19 | PostgreSQL 持久化缓存，多实例共享，LRU 淘汰 |
| 心跳无活跃时段控制 | 2026-02-19 | 新增 `active_hours_start/end/days`，支持 per-bot 时区感知 |
| 定时任务只有 send 命令能通知用户 | 2026-02-19 | Prompt 强制 send 指令 + Go 侧工具结果兜底 |
| 频道绑定错误提示不明确 | 2026-02-19 | Telegram/Feishu `resolveTarget` 错误信息明确列出缺少的字段 |
| OpenViking 用户文档不足 | 2026-02-19 | 补充「何时开启」决策树、适用场景与标准记忆关系说明 |
| 群组消息防抖窗口固定 | 2026-02-19 | 新增 `SubmitWithWindow`，从 bot `metadata.group_debounce_ms` 读取；Bot 设置页可配置 |
| 子智能体历史 UI 未实现 | 2026-02-19 | `bot-subagents.vue` 新增可折叠「运行历史」面板 |
| BM25 统计不持久化 | 2026-02-19 | 迁移 0027 + `SetPool`/`periodicSave`，30 秒异步刷新 DocCount+AvgDocLen |
| 记忆提取无 Embedding 去重 | 2026-02-19 | `applyAdd` 前加 0.92 余弦阈值守卫，命中缓存时开销极小 |
| 进化系统无自动回滚 | 2026-02-19 | 迁移 0028 快照 + `RollbackEvolution` 端点 + bot-evolution.vue 回退按钮 |
| 容器人设文件重建后为空 | 2026-02-19 | agent.ts 检测到容器文件空但 DB 有值时异步写回 IDENTITY/SOUL.md |
