# 流程日志

## 进入流程日志页面

点击左侧导航的 **流程日志** 进入。

## 页面布局

- **左侧面板**：筛选控件和统计信息。
- **右侧面板**：日志表格。

## 筛选功能

### 按 Bot 筛选

顶部下拉框选择要查看的 Bot，只显示该 Bot 的日志。

### 按步骤筛选

下拉框可按处理步骤筛选：

| 步骤 | 说明 |
|------|------|
| user_message_received | 收到用户消息 |
| history_loaded | 加载对话历史 |
| memory_searched | 搜索记忆库 |
| memory_loaded | 加载记忆内容 |
| prompt_built | 构建提示词 |
| llm_request_sent | 发送 LLM 请求 |
| llm_response_received | 收到 LLM 回复 |
| response_sent | 发送回复给用户 |
| memory_stored | 存储新记忆 |
| stream_started | 开始流式响应 |
| stream_completed | 流式响应完成 |
| stream_error | 流式响应出错 |

### 关键词搜索

搜索框支持按消息内容、步骤名、Trace ID 进行模糊搜索。

## 日志表格

每条日志记录包含：

| 列 | 说明 |
|------|------|
| 时间 | 日志记录的时间 |
| 级别 | info / warn / error，用颜色区分 |
| 步骤 | 处理步骤名称 |
| 消息 | 日志内容 |
| 耗时 | 该步骤的处理时间（毫秒） |

### 展开详情

点击某条日志可以展开查看完整的 JSON 数据，包含该步骤的详细参数和返回值。

## 统计面板

左侧面板底部显示统计信息：
- 日志总数
- 错误数量
- 警告数量

## 一轮对话的完整流程

一轮正常的对话处理流程包含以下步骤：

1. **user_message_received** — 系统收到用户消息。
2. **history_loaded** — 加载该用户与 Bot 的对话历史。
3. **memory_searched** — 根据当前消息在记忆库中语义搜索相关记忆。
4. **prompt_built** — 将系统提示词、记忆、历史和用户消息组装成完整的 Prompt。
5. **llm_request_sent** — 将 Prompt 发送给 AI 模型。
6. **stream_started** — 开始接收模型的流式响应。
7. **stream_completed** — 流式响应接收完毕。
8. **response_sent** — 将最终回复发送给用户。
9. **memory_stored** — 从本轮对话中提取并存储新记忆。

如果某些步骤没有出现，可能的原因包括：
- 未配置嵌入模型 → `memory_searched` 不会触发。
- 记忆库为空 → `memory_loaded` 可能被跳过。
- 使用非流式响应 → `stream_started` / `stream_completed` 被替换为 `llm_response_received`。
