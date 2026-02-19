# 超越 OpenClaw 三梯队改造计划

> 状态：✅ 全部完成（9/9 tasks）

## 现状盘点（基于代码实测）

当前已有，无需重做：

- `loadMemoryContextMessage` 已有 `extractKeywords` + `rrfMerge` 二次搜索扩展
- `softTrimToolResults` + `pruneMessagesByTokenBudget` 工具结果截断已完整
- `sendWithConfig` 已有指数退避重试（`(i+1) * RetryBackoffMs`）
- SubagentRegistry 已有级联终止 (`abort` 递归 kill children)
- `agent/src/prompts/system.ts` 已有 static/dynamic 分离（prefix cache 友好）

真正缺失的（代码验证）：

- `stopWhen: stepCountIs(Infinity)` — 无任何工具循环检测
- 记忆搜索结果无 MMR 重排序、无时间衰减
- 无 embedding 缓存，每次写记忆都调 embedding API
- SubagentRegistry 是内存 `Map`，重启即丢失
- 心跳无活跃时段检查，凌晨也会触发

---

## 第一梯队：记忆质量（ROI 最高）

### T1-1 MMR 重排序 ✅

**文件：** `internal/memory/service.go`

向 `SearchRequest` 添加字段：

```go
UseMMR    bool    `json:"use_mmr,omitempty"`
MMRLambda float64 `json:"mmr_lambda,omitempty"` // default 0.7
```

新增函数 `mmrRerank(items []MemoryItem, lambda float64, topK int) []MemoryItem`：

- 修改 Qdrant Query 请求加 `WithVectors: true`，获取真实向量
- 用余弦相似度计算结果间相似性（OpenClaw 只用文本 Jaccard，我们用真实 embedding）
- 迭代选取：`MMR = λ × relevance − (1-λ) × max_cosine_to_selected`

**超越 OpenClaw 的点：** 基于 Qdrant 返回的真实 float32 向量做余弦相似度，而非文本 Jaccard；Qdrant 向量维度更高、语义更准确。

---

### T1-2 时间衰减评分 ✅

**文件：** `internal/memory/service.go`

新增函数 `applyTemporalDecay(items []MemoryItem, halfLifeDays float64) []MemoryItem`：

```go
ageInDays := time.Since(createdAt).Hours() / 24
multiplier := math.Exp(-math.Log(2) / halfLifeDays * ageInDays)
item.Score *= multiplier
```

Evergreen 处理：`Metadata["source"]` 为 `identity` / `soul` / `tools` 的条目跳过衰减。

**超越 OpenClaw 的点：** 衰减参数（半衰期天数）可配置；OpenClaw 的半衰期是代码写死的 30 天。

---

### T1-3 Embedding Cache ✅

**新文件：** `internal/memory/embedding_cache.go`  
**新迁移：** `db/migrations/0024_embedding_cache.up.sql`

```sql
CREATE TABLE IF NOT EXISTS embedding_cache (
  id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  provider    TEXT NOT NULL,
  model       TEXT NOT NULL,
  hash        TEXT NOT NULL,  -- SHA-256 of input text
  embedding   JSONB NOT NULL,
  dims        INT NOT NULL,
  updated_at  BIGINT NOT NULL,
  UNIQUE (provider, model, hash)
);
```

LRU 淘汰：超过 50,000 条时删除最旧 10%（按 `updated_at ASC`）。

**超越 OpenClaw 的点：** 存入 PostgreSQL，多实例共享缓存；OpenClaw 是 SQLite 单进程。

---

## 第二梯队：系统可靠性

### T2-1 工具调用循环检测 ✅

**新文件：** `agent/src/tools/loop-detection.ts`

三个检测器：

- `repeatedNoProgress`：同工具 + 同参数 + 同结果哈希，连续 ≥ 8 次 → 终止
- `pingPong`：A→B→A→B 交替且结果无变化，连续 ≥ 6 对 → 终止
- `globalCircuitBreaker`：窗口内相同调用 ≥ 25 次 → 终止

**超越 OpenClaw 的点：** 循环检测结果在 Web UI 流程日志中可见；OpenClaw 仅在 CLI 输出错误。

---

### T2-2 子智能体运行记录持久化 ✅

**新迁移：** `db/migrations/0025_subagent_runs.up.sql`  
**新 Go 文件：** `internal/handlers/subagent_runs.go`

- `POST /subagent-runs` — 注册新运行
- `PATCH /subagent-runs/:runId` — 更新状态
- `GET /subagent-runs?botId=&status=` — 查询

**超越 OpenClaw 的点：** PostgreSQL 存储可在 Web UI 展示运行历史；OpenClaw 仅落磁盘文件无 UI。

---

### T2-3 心跳活跃时段 ✅

**迁移：** `db/migrations/0026_heartbeat_active_hours.up.sql`

```sql
ALTER TABLE heartbeat_configs
  ADD COLUMN IF NOT EXISTS active_hours_start SMALLINT DEFAULT 0,
  ADD COLUMN IF NOT EXISTS active_hours_end   SMALLINT DEFAULT 23,
  ADD COLUMN IF NOT EXISTS active_days        SMALLINT[] DEFAULT '{0,1,2,3,4,5,6}';
```

**超越 OpenClaw 的点：** 使用 bot 自己的时区设置而非系统时区；活跃星期可精选（如只工作日）；全部通过 Web UI 配置。

---

## 第三梯队：工程质量

### T3-1 系统提示词 full/minimal 双模式 ✅

**文件：** `agent/src/prompts/system.ts`

- `triggerSchedule` → `minimal`（节省约 400-600 token）
- `askAsSubagent` → `minimal`
- `ask` / `stream` → `full`

**超越 OpenClaw 的点：** 基于调用上下文动态切换，与 Prefix Cache 配合，minimal 模式缓存命中率更高。

---

### T3-2 Telegram Markdown → HTML ✅

Telegram 发送时自动将内部 Markdown 转为 HTML 格式，支持粗体、斜体、代码块、超链接、文件路径保护。

---

### T3-3 群组消息防抖队列 ✅

**新文件：** `internal/message/debounce.go`

群组收到消息时进入 3 秒等待窗口，窗口内多条消息合并为一个 `\n---\n` 分隔的请求再触发 Agent。DM 直接透传。

---

## 执行顺序

建议执行顺序：T1-3 → T1-1 → T1-2 → T2-1 → T2-2 → T2-3 → T3-1 → T3-2 → T3-3

每完成一组运行 `mise run sqlc-generate && mise run db-up` 应用迁移，`go build ./...` 验证编译。
