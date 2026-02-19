# 增强日志系统与单轮导出

> 状态：✅ 全部完成（3/3 tasks）

## 目标

用户需要把单轮对话的完整日志分享出来用于调试。这需要两件事：(1) 日志要捕获足够的上下文；(2) 需要一个导出机制。

---

## Part 1：日志数据增强

在 `internal/conversation/flow/resolver.go` 的关键步骤中丰富 `data` 字段：

| 日志步骤 | 新增字段 |
|---------|---------|
| `user_message_received` | `platform`、`identity_id`、完整 query（移除 500 字符截断） |
| `prompt_built` | `provider`、`timezone`、`language`、`system_prompt_length`、`context_window` |
| `llm_request_sent` | `provider` |
| `llm_response_received` | `response_preview`（前 300 字符）、`finish_reason` |
| `stream_completed` | `model`、`duration_ms` |
| `response_sent` | `response_preview`（前 300 字符） |
| `tool_call_started/completed` | 截断上限从 500 提升到 2000 字符 |
| `memory_extract_completed` | `extracted_preview`（前 500 字符） |
| `memory_searched` | `result_count` |
| `openviking_session` | `message_count` |

---

## Part 2：导出 API 端点

新增端点：`GET /logs/trace/:traceId/export`

返回结构化 JSON 诊断报告：

```json
{
  "version": "1.0",
  "exported_at": "2026-02-17T12:00:00Z",
  "trace_id": "...",
  "bot_id": "...",
  "channel": "telegram",
  "time_range": { "start": "...", "end": "..." },
  "total_duration_ms": 1234,
  "summary": {
    "user_query": "...",
    "assistant_response": "...",
    "model": "deepseek-chat",
    "provider": "openai-compatible",
    "token_usage": { "prompt": 100, "completion": 50, "total": 150 },
    "steps_count": 12,
    "errors": [],
    "warnings": []
  },
  "steps": [ ... ]
}
```

`summary` 由服务端扫描 trace 步骤自动计算。

---

## Part 3：前端导出 UI

在 `packages/web/src/pages/logs/index.vue` 的每个 trace 组头部添加「导出」按钮（剪贴板图标）：

1. 点击 → 调用 `GET /logs/trace/:traceId/export`
2. 将 JSON 复制到剪贴板（`navigator.clipboard.writeText()`）
3. 显示 Toast 提示「已复制到剪贴板」

用户可直接粘贴给 AI 进行调试分析。

---

## 修改文件清单

| 文件 | 修改内容 |
|------|---------|
| `internal/conversation/flow/resolver.go` | 丰富约 12 个日志步骤的 data 字段 |
| `internal/processlog/service.go` | 新增 `ExportTrace` 方法 + 报告结构体 |
| `internal/handlers/processlog.go` | 新增 `ExportTrace` handler + 路由注册 |
| `packages/web/src/lib/api-logs.ts` | 新增 `exportTrace()` API 函数 + 类型定义 |
| `packages/web/src/pages/logs/index.vue` | 在 trace 头部添加导出按钮 |
