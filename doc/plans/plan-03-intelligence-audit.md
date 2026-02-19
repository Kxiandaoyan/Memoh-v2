# Memoh-v2 vs OpenClaw 对标审计：智能化 / 节约 Token / 记忆力

> 状态：✅ P0+P1+P2 全部完成（14/14 tasks）

---

## A. 智能化 (Intelligence)

### A.1 系统提示词缓存 ✅

**问题：** Agent Gateway 每次请求都通过 MCP 读取容器内的 `IDENTITY.md`, `SOUL.md`, `TOOLS.md`，即使文件未变化。

**修复：** 在 `agent/src/agent.ts` 的 `generateSystemPrompt()` 中添加内存缓存层，以 `botId + file mtime` 为 key，命中则跳过 MCP read 调用。

- 预估收益：每轮对话减少 3 次 MCP 工具调用，降低延迟 200-500ms

---

### A.3 动态工具结果截断 ✅

**问题：** 工具返回结果超过 50KB 时直接截断，LLM 看到的是不完整的内容。

**修复：**
- 动态截断：按 `contextWindow * 0.3 * 4` 计算最大字符数
- 保留头尾：`headChars=1500 + tailChars=1500`
- 加提示后缀：`"[Content truncated — original was N chars]"`

---

### A.6 上下文溢出自动压缩 ✅

**问题：** Token 超出预算时仅丢弃最旧消息，不做任何摘要或压缩。

**修复：**
- 在 `pruneMessagesByTokenBudget` 触发时，对被裁剪的消息同步生成摘要
- 失败时降级：截断大工具结果 → 重试 → 最终丢弃最旧消息

---

## B. 节约 Token (Token Efficiency)

### B.1 Token 估算改进 ✅

**问题：** 用 `len(json) * 10 / 25` 估算，对 CJK 内容误差可达 50%+。

**修复：**
- CJK 内容用 1.5 chars/token，Latin 用 3.5 chars/token
- 加安全余量：`estimatedTokens *= 1.2`

---

### B.2 系统提示词计入 Token 预算 ✅

**问题：** `pruneMessagesByTokenBudget` 只计算消息列表的 token，不包含系统提示词。

**修复：** 预算改为：`budget = (contextWindow - systemPromptTokens) * 0.6`

---

### B.3 分块裁剪 + 保护最近 N 轮 ✅

**问题：** 从最旧消息开始丢弃，不考虑重要性。

**修复：**
- 分块裁剪：将消息分为 2 个 chunk，优先丢弃旧半段
- 自适应 chunk：如果平均消息 > 10% context window，减小 chunk 比例
- 保护最近 3 轮 assistant 回复

---

### B.4 工具结果软裁剪 ✅

**问题：** 工具结果进入 context 时无限制，大工具结果挤占历史消息空间。

**修复：** 单条工具结果上限：`contextWindow * 0.3 * 4` 字符，超限时保留头尾各 1500 字符。

---

### B.5 主动摘要策略 ✅

**问题：** 摘要仅在消息被裁剪时异步创建，只有一个层级。

**修复：**
- 主动触发：当消息数超过 `historyLimit * 0.8` 时，预生成摘要
- 多级摘要：旧摘要 + 新消息 → 合并为更新的摘要

---

### B.6 记忆独立预算 ✅

**问题：** 记忆内容与历史消息共享同一 token 预算，高记忆使用会挤出历史消息。

**修复：** 为记忆分配独立预算：`memoryBudget = contextWindow * 0.05`，动态调整注入数量。

---

## C. 记忆力 (Memory)

### C.1 记忆相关性阈值过滤 ✅

**问题：** 搜索后直接取 top-8，不管分数多低。低相关性记忆浪费 token 并可能误导 LLM。

**修复：** 添加最低分数阈值（BM25 score > 0.1 或 cosine > 0.3），过滤掉低质量匹配。

---

### C.2 查询扩展 ✅

**问题：** 使用原始 query 做单次搜索，对简短或口语化查询召回率低。

**修复：**
- 提取关键词（去停用词，支持中英文分词）
- 同时搜索原始查询 + 关键词查询
- 合并结果，RRF 融合排序

---

### C.3 MMR 多样性排序 ✅

**问题：** 搜索结果按分数排序，可能返回多条高度相似的记忆，浪费 token。

**修复：** 添加 MMR 去重，Lambda=0.7（平衡相关性和多样性）。

---

### C.4 时间衰减 ✅

**问题：** 旧记忆和新记忆同等权重，无时间衰减。

**修复：** 指数衰减 `multiplier = exp(-ln2/30days * age)`，半衰期 30 天。

---

### C.7 Embedding 缓存 ✅

**问题：** 每次搜索和存储都调用 embedding API，无缓存。

**修复：** PostgreSQL 持久化缓存，Key: `provider + model + text_hash`，LRU 淘汰策略。

---

## 优先级排序（已全部完成）

| 优先级 | 项目 | 状态 |
|--------|------|------|
| P0 | B.1 Token 估算、C.1 相关性阈值、B.4 工具软裁剪、C.4 时间衰减 | ✅ |
| P1 | A.1 提示词缓存、B.2 提示词计入预算、B.3 分块裁剪、C.2 查询扩展、B.6 记忆独立预算 | ✅ |
| P2 | A.3 动态截断、A.6 溢出压缩、C.3 MMR、C.7 Embedding 缓存、B.5 主动摘要 | ✅ |
