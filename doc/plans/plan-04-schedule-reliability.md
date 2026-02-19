# 定时任务通知可靠性修复方案

> 状态：✅ 全部完成（2/2 tasks）

---

## 当前状况诊断

**工具调用执行层面 — 没有问题**

Vercel AI SDK 的 `generateText` 会自动执行所有 MCP 工具调用，所以：
- `exec 'script.sh'` → 脚本会真实运行 ✅
- "查今日天气" → `web_search` 会真实请求 ✅
- "搜索记忆" → `search_memory` 会真实执行 ✅

**问题只在用户通知层面**

触发链路：

```
CronPool 触发
  → runSchedule (schedule/service.go)
  → executeTrigger (resolver.go)
  → Agent Gateway triggerSchedule()
  → LLM + 工具执行（均正常）
  → resp.Messages 返回 Go
  → LLM 是否调用了 send 工具？
      ├── 是 → 消息到达 Telegram ✅
      ├── 否，但有文字 → Go fallback 发送文字 ✅
      └── 否，只有工具调用无文字 → 消息丢失 ❌
```

**三种场景对应三种结果：**

| 命令类型 | 工具执行 | 用户收到消息 |
|---------|--------|------------|
| `send '喝水提醒'` | N/A | ✅ |
| "查天气发给我"（LLM 生成文字） | ✅ 正常 | fallback 保证 ✅ |
| `exec 'backup.sh'`（LLM 只有工具调用，无文字总结） | ✅ 正常 | ❌ 用户不知道执行了 |

---

## 修复方案

### 修改 1：完善通用 schedule prompt ✅

**文件：** `agent/src/prompts/schedule.ts`

对非 `send 'xxx'` 的通用命令分支，在末尾追加强制通知指令：

```
MANDATORY FOLLOW-UP: After completing the task above, you MUST call the `send` tool
to deliver a brief result summary to the user. Use the current session platform and target.
Do NOT skip this step even if the task produced no output — report what happened.
```

### 修改 2：强化 Go 侧兜底 ✅

**文件：** `internal/conversation/flow/resolver.go`

新增辅助函数 `extractLastToolResultSummary`：扫描最后一条 `role:"tool"` 消息，从工具结果的 JSON 中提取可读文字（优先 `result` 字段，截断到合理长度）。

在 `executeTrigger` 的 fallback 块中新增 else 分支，当 LLM 只有工具调用但无文字时，从工具结果提取摘要作为兜底发送给用户。

---

## 为什么不走"对接机器人会话"路径

用户提到的"对接给机器人"方向的问题：
- `inbound/channel.go` 处理器设计用于处理"用户发来的消息"，注入 Bot 主动消息需要绕过 identity/routing 层
- 可能破坏会话历史的逻辑顺序
- 需要给 `Resolver` 加 `ChannelInboundProcessor` 引用，引入循环依赖风险

**结论：** 当前修复（prompt 强制 + Go fallback）覆盖了所有主要场景，是更轻量且有效的方案。
