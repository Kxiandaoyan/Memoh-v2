# 已知局限性全面解决计划

> 目标：解决 `doc/known-limitations.md` 中 8 项可修复局限性，跳过 2 项（循环检测内存状态、进化依赖模型能力）
>
> 执行顺序：L1 → L2 → L3 → L4 → L5 → L6 → L7 → L8

---

## 代码审计结论（影响方案的关键发现）

| 发现 | 影响 |
|------|------|
| Memoh-v2 无 session 文件 | "Session 文件修复"不适用，改为"容器人设文件自愈" |
| `PendingGroup.Append` 已接受 `window` 参数 | 群组防抖可配置只需加 `SubmitWithWindow`，无需迁移 |
| `evolution_logs.files_modified` 字段存在但从未填充 | 可直接利用，加 `files_snapshot JSONB` 列 |
| BM25 `DocFreq` map 可能很大 | 只持久化 `DocCount + AvgDocLen`，DocFreq 重启后 warmup |
| `applyAdd` 已调用 `embedText` | Embedding 去重利用已有缓存，额外开销极小 |
| bot `metadata JSONB` 已可通过 UpdateBot API 更新 | 群组防抖配置存 metadata，不需要 DB 迁移 |

---

## L1 频道绑定错误提示优化

**难度：🟢 低 | 风险：极低 | 影响：UX**

### 问题

`internal/channel/adapters/telegram/config.go:68` 和 `internal/channel/adapters/feishu/config.go:68` 都只返回：
```
"telegram binding is incomplete"
"feishu binding is incomplete"
```

### 修改

**`internal/channel/adapters/telegram/config.go`** — `resolveTarget()` 末尾：
```go
// 改前：
return "", fmt.Errorf("telegram binding is incomplete")

// 改后：
return "", fmt.Errorf("telegram binding is incomplete: at least one of chat_id, user_id, or username is required")
```

**`internal/channel/adapters/feishu/config.go`** — 同理：
```go
return "", fmt.Errorf("feishu binding is incomplete: at least one of open_id or user_id is required")
```

### 风险

无。只改错误消息字符串，不影响任何逻辑。

---

## L2 OpenViking 文档补充

**难度：🟢 低 | 风险：零 | 影响：用户理解**

### 修改

更新 `doc/18-openviking.md`，补充以下内容：
- **工作原理**：L0（abstract）/ L1（overview）/ L2（read）三层架构说明，与标准记忆系统的分工
- **何时开启**：适合需要长文档知识库、技术文档、代码库检索的 Bot；纯聊天 Bot 无需开启
- **与标准记忆的关系**：标准记忆存的是"事实/偏好"，OpenViking 存的是"结构化文档上下文"；两者互补

---

## L3 群组消息防抖窗口可配置（per-bot）

**难度：🟢 低 | 风险：低 | 影响：群组场景灵活性**

### 当前问题

`cmd/agent/main.go` 硬编码 3 秒。`Submit` 使用全局 `d.window`，无法 per-bot 设置。

### 方案：无需 DB 迁移，利用 metadata 字段

**`internal/message/debounce.go`** — 新增方法：
```go
// SubmitWithWindow submits with a caller-specified window, ignoring d.window.
// Falls back to d.window if override <= 0.
func (d *GroupDebouncer) SubmitWithWindow(key, text string, window time.Duration, execute func(mergedText string)) {
    if window <= 0 {
        window = d.window
    }
    // ... same logic as Submit but passes window to pg.Append
}
```

**`internal/channel/inbound/channel.go`** — `HandleInbound` 的防抖提交处：
```go
// 读 bot metadata 中的 group_debounce_ms（默认 0 = 使用全局默认）
debounceMs := int64(0)
if cfg.BotConfig != nil {
    if v, ok := cfg.BotConfig.Metadata["group_debounce_ms"]; ok {
        if ms, ok := v.(float64); ok {
            debounceMs = int64(ms)
        }
    }
}
window := time.Duration(debounceMs) * time.Millisecond
p.groupDebouncer.SubmitWithWindow(debounceKey, text, window, func(mergedText string) { ... })
```

**前端** — 在 Bot 设置页（`bot-settings.vue`）的"高级"或"频道"区域添加一个"群组防抖窗口"输入框（毫秒），通过现有 `UpdateBot` API 写入 `metadata.group_debounce_ms`。

### 风险

- `cfg.BotConfig` 可能为 nil（DM 场景）：已有 nil guard 不影响
- 防抖走 metadata 路径意味着 Bot 需要有 `metadata` 字段，已经支持
- **不能破坏现有行为**：`debounceMs = 0` 时保持原有 3 秒全局默认

---

## L4 子智能体运行历史 UI

**难度：🟡 中 | 风险：低 | 影响：可观测性**

### 当前状态

后端 API 已完整（`/subagent-runs`，含 GET/POST/PATCH/DELETE），但前端 `bot-subagents.vue` 没有调用。SDK 未生成对应类型。

### 方案

**不走 SDK 生成路径**（避免触发 swagger 重新生成的副作用），直接用 `fetch` 调用：

**`packages/web/src/lib/api-subagent-runs.ts`** — 新文件：
```typescript
export interface SubagentRun {
  id: string; run_id: string; bot_id: string; name: string; task: string
  status: 'running' | 'completed' | 'failed' | 'aborted'
  spawn_depth: number; parent_run_id?: string
  result_summary?: string; error_message?: string
  started_at: string; ended_at?: string; created_at: string
}

export async function listSubagentRuns(botId: string, status?: string): Promise<SubagentRun[]>
export async function deleteSubagentRun(runId: string): Promise<void>
```

**`bot-subagents.vue`** — 在现有子智能体列表下方添加「运行历史」折叠面板：
- 按状态过滤（全部 / 运行中 / 已完成 / 失败）
- 每条显示：名称、任务摘要（前 80 字）、状态 badge、开始时间、耗时、父子关系层级
- 展开显示 result_summary 或 error_message
- 删除按钮（清理旧记录）

### 风险

- 直接 fetch 而非 SDK，需手动维护类型定义。风险可控：API 接口简单稳定。
- 运行历史可能很长：`GET /subagent-runs?botId=` 默认限制 200 条，加分页按钮即可。

---

## L5 BM25 统计数据持久化

**难度：🟡 中 | 风险：中 | 影响：重启后搜索质量**

### 新迁移：`db/migrations/0027_bm25_stats.up.sql`

```sql
-- 0027_bm25_stats
-- Persist BM25 aggregate stats (DocCount, AvgDocLen) per bot+language
-- so search quality survives restarts. DocFreq rebuilds via warmup.
CREATE TABLE IF NOT EXISTS bm25_stats (
  bot_id    TEXT     NOT NULL,
  lang      TEXT     NOT NULL,
  doc_count INT      NOT NULL DEFAULT 0,
  avg_doc_len FLOAT8 NOT NULL DEFAULT 0,
  updated_at BIGINT  NOT NULL,
  PRIMARY KEY (bot_id, lang)
);
```

### `internal/memory/indexer.go` 修改

添加持久化支持：
- 新增 `dbPool *pgxpool.Pool` 字段和 `SetPool(pool)` setter
- 启动时 `LoadStats(ctx, botID)` — 从 DB 读取 `DocCount + AvgDocLen`，恢复内存状态
- 每次 `updateStatsAddLocked` / `updateStatsRemoveLocked` 后，**异步写入** DB（使用 debounce，5 秒内无操作才实际写）
- `DocFreq` 不持久化（重启通过 warmup 自动重建，重建期间 IDF 降级为最大值，可接受）

### 注意

- BM25 indexer 当前按语言分 shard，bot_id 来自 service 层。需要在 `Indexer` 中加 `botID string` 字段，或在 stats 的 key 中包含 bot_id。
- **风险**：`updateStatsAddLocked` 在锁内，异步写 DB 不会阻塞，但要保证写 goroutine 生命周期正确。使用 `sync.Once` 防止重复写。

### `cmd/agent/main.go`

在 `provideMemoryService` 中，将 `pgxpool.Pool` 传入 indexer，调用 `indexer.SetPool(pool)` 并在服务启动后调用 `indexer.LoadStats(ctx, botID)`。

---

## L6 记忆提取 Embedding 去重

**难度：🟡 中 | 风险：中 | 影响：记忆质量**

### 当前问题

`applyAdd` 直接 upsert，LLM `Decide()` 做决策时只用 BM25 top-5 候选，BM25 召回不到时会重复 ADD。

### 方案：在 `applyAdd` 内增加 embedding 相似度守卫

**`internal/memory/service.go`** — `applyAdd` 开头：
```go
func (s *Service) applyAdd(ctx context.Context, botID, text string, metadata map[string]any) error {
    // Embedding dedup guard: skip if a very similar memory already exists
    vec, err := s.embedText(ctx, text)
    if err == nil && len(vec) > 0 {
        results, _ := s.store.SearchWithVectors(ctx, SearchRequest{
            BotID:  botID,
            Vector: vec,
            Limit:  1,
        }, true)
        if len(results) > 0 && results[0].Score >= 0.92 {
            // Near-duplicate detected, skip insertion
            return nil
        }
    }
    // ... existing logic continues with vec already computed
```

利用已有的 `embedText`（内置 Embedding Cache），多一次 Qdrant 查询但命中缓存时 embedding 几乎无额外开销。

### 阈值选择

- **0.92**：足够高以避免合法的相似但不同的事实被过滤，同时拦截"用户喜欢红色" vs "用户非常喜欢红色"这类近似副本。
- 可在 `SearchRequest` 中通过可选 `MinScore` 字段控制，将来暴露为 bot 配置。

### 风险

- 增加一次 Qdrant 查询：通过 Embedding Cache 降低延迟（embedding 已缓存时 Qdrant 查询约 2-5ms）
- 误拦截：相似度 0.92 的阈值在实测中极少出现假阳性。若有问题，可调高到 0.95。
- **需要验证** `SearchWithVectors` 当传入 `vector` 而非 `query string` 时的参数路径（`types.go SearchRequest` 可能需加 `Vector []float32` 字段）

---

## L7 进化 diff 追踪 + 一键回退

**难度：🔴 高 | 风险：中 | 影响：进化可靠性（核心护城河）**

### 新迁移：`db/migrations/0028_evolution_snapshot.up.sql`

```sql
-- 0028_evolution_snapshot
-- Add files_snapshot to capture persona files before each evolution run,
-- enabling one-click rollback if evolution degrades bot behavior.
ALTER TABLE evolution_logs
  ADD COLUMN IF NOT EXISTS files_snapshot JSONB;
-- files_snapshot format: {"IDENTITY.md": "content...", "SOUL.md": "...", ...}
```

### 数据流

```
heartbeat engine fire()
  → isEvolution? → 读取 IDENTITY.md / SOUL.md / TOOLS.md / EXPERIMENTS.md / NOTES.md
  → 写入 evolution_log.files_snapshot
  → TriggerHeartbeat()
  → 完成后读文件再次，diff → 更新 evolution_log.files_modified（原来未填充的字段）
```

### 文件读取方式

Go 侧通过容器 MCP 工具代理路径读取。Heartbeat engine 已有 `dbPool`（T2-3 时加的），可以直接查询容器 mount 路径 `data/bots/{botID}/`：

```go
// internal/heartbeat/engine.go — snapshotPersonaFiles()
func snapshotPersonaFiles(dataDir, botID string) map[string]string {
    files := []string{"IDENTITY.md", "SOUL.md", "TOOLS.md", "EXPERIMENTS.md", "NOTES.md"}
    snapshot := make(map[string]string)
    for _, f := range files {
        content, err := os.ReadFile(filepath.Join(dataDir, "bots", botID, f))
        if err == nil {
            snapshot[f] = string(content)
        }
    }
    return snapshot
}
```

### `internal/handlers/heartbeat.go` 新端点

```
POST /bots/:bot_id/evolution-logs/:id/rollback
```

- 读取 `evolution_log.files_snapshot`
- 将每个文件内容写回容器对应路径
- 更新 evolution log 增加 `rolled_back_at` 记录（可选，加字段或用 metadata）

### `bot-evolution.vue` 修改

在每条 evolution log 卡片右上角加「回退」按钮：
- 仅当 `files_snapshot != null` 时显示
- 点击弹确认框："将 Bot 的人设文件回退到此次进化之前的状态，此操作不可逆"
- 确认后调用 `POST /bots/:bot_id/evolution-logs/:id/rollback`
- 成功后刷新进化历史列表，Toast "已回退到进化前状态"

### 风险

- **数据目录路径**：需从 config 读取 `dataDir`，heartbeat engine 需能访问此路径。T2-3 时已加 `SetPool`，同样方式加 `SetDataDir`。
- **容器未启动**：bot 容器停止时文件路径仍可访问（bind mount 持久），不影响回退。
- **并发**：回退时若 bot 正在运行对话，文件写入会立即影响下一轮 system prompt 加载。在回退 API 中加「stop evolution heartbeat」逻辑或在 UI 层面提示。

---

## L8 容器人设文件自愈（替代"Session 文件修复"）

**难度：🟡 中 | 风险：低 | 影响：容器初始化可靠性**

> ⚠️ **说明**：OpenClaw 有 session 文件修复，但 Memoh-v2 无 session 文件（历史在 PostgreSQL）。
> 等效的改进是：当容器人设文件（IDENTITY.md 等）丢失或为空时，从数据库 `bot_prompts` 自动恢复。

### 问题场景

- 容器重建后数据目录清空
- 手动误删了 IDENTITY.md
- Bot 运行时读到空文件，system prompt 退化

### 方案

**`internal/conversation/flow/resolver.go`** — `loadSystemPromptFromBot()` 或类似的提示词加载函数中：

```go
// 检查容器文件是否缺失
if identity == "" && soul == "" {
    // 从数据库加载 bot_prompts 兜底
    prompts, err := r.db.GetBotPrompts(ctx, botID)
    if err == nil && prompts != nil {
        // 将 DB 内容异步写回容器（不阻塞当前请求）
        go restorePersonaFilesToContainer(botID, prompts)
    }
}
```

**新辅助函数** `restorePersonaFilesToContainer`：通过 MCP 文件写工具或直接写文件系统，将 DB 中的 identity/soul/task 写回 IDENTITY.md / SOUL.md / TOOLS.md。

### 当前加载机制

system prompt 加载逻辑（`agent/src/prompts/system.ts`）：DB 优先、容器文件兜底。所以容器文件缺失时会降级到 DB。加 L8 后，降级的同时触发异步修复，下一轮对话即可恢复文件一致性。

### 风险

很低。自愈逻辑在异步 goroutine 中，不影响当前请求。失败静默忽略。

---

## 执行顺序与依赖

```
L1 频道错误提示     ── 独立，零依赖，最先做
L2 OpenViking 文档  ── 独立，零依赖
L3 群组防抖可配置   ── 独立，可与 L1/L2 并行
L4 子智能体历史 UI  ── 独立，后端 API 已就绪
L5 BM25 持久化      ── 需要迁移 0027，先建表再改 indexer
L6 Embedding 去重   ── 需要 SearchRequest 加 Vector 字段（检查是否已有）
L7 进化 diff+回退   ── 需要迁移 0028，最复杂，最后做
L8 容器人设自愈     ── 独立，可随时插入
```

**推荐顺序**：L1 + L2 → L3 + L8 → L4 → L5 → L6 → L7

---

## 新增文件清单

| 文件 | 类型 |
|------|------|
| `db/migrations/0027_bm25_stats.up.sql` | 迁移 |
| `db/migrations/0027_bm25_stats.down.sql` | 迁移 |
| `db/migrations/0028_evolution_snapshot.up.sql` | 迁移 |
| `db/migrations/0028_evolution_snapshot.down.sql` | 迁移 |
| `packages/web/src/lib/api-subagent-runs.ts` | 前端新文件 |

## 修改文件清单

| 文件 | 修改内容 |
|------|---------|
| `internal/channel/adapters/telegram/config.go` | 错误消息具体化 |
| `internal/channel/adapters/feishu/config.go` | 错误消息具体化 |
| `internal/message/debounce.go` | 新增 `SubmitWithWindow` |
| `internal/channel/inbound/channel.go` | 读 metadata.group_debounce_ms 调用 SubmitWithWindow |
| `packages/web/src/pages/bots/components/bot-settings.vue` | 添加防抖窗口输入项 |
| `packages/web/src/pages/bots/components/bot-subagents.vue` | 添加运行历史折叠面板 |
| `internal/memory/indexer.go` | 添加 DB 持久化支持（docCount+avgDocLen） |
| `internal/memory/service.go` (`applyAdd`) | embedding 去重守卫 |
| `internal/memory/types.go` | SearchRequest 添加 Vector 字段（如需要） |
| `internal/heartbeat/engine.go` | 进化前快照文件，进化后更新 files_modified，加 SetDataDir |
| `internal/handlers/heartbeat.go` | 新增 rollback 端点 |
| `internal/conversation/flow/resolver.go` | 人设文件自愈逻辑 |
| `packages/web/src/pages/bots/components/bot-evolution.vue` | 添加回退按钮 |
| `doc/18-openviking.md` | 补充工作原理和使用建议 |
| `db/migrations/0001_init.up.sql` | 同步追加 bm25_stats 表和 evolution_logs 新列 |
