# 模型与提供方

## 进入模型管理

点击左侧导航的 **模型管理** 进入。

## 页面布局

- **左侧面板**：提供方列表，支持搜索和按类型筛选。
- **右侧面板**：选中提供方后，展示其下的模型列表和配置。

## LLM 提供方

### 支持的提供方类型

| 类型 | 说明 |
|------|------|
| openai | OpenAI 官方 API |
| openai-compat | OpenAI 兼容 API（如第三方转发服务） |
| anthropic | Anthropic（Claude 系列） |
| google | Google AI（Gemini 系列） |
| azure | Azure OpenAI |
| bedrock | AWS Bedrock |
| mistral | Mistral AI |
| xai | xAI（Grok 系列） |
| ollama | Ollama 本地模型 |
| dashscope | 阿里云 DashScope（通义系列） |

### 筛选提供方

- **搜索框**：按名称搜索。
- **类型下拉**：按提供方类型筛选（如只看 openai 类型的）。

### 添加提供方

点击 **添加提供方** 按钮，在弹出的对话框中填写：
- 提供方名称
- API 类型（从上述支持的类型中选择）
- API Key / 密钥
- API Base URL（可选，某些类型需要）

### 管理模型

选中某个提供方后，右侧面板显示该提供方下的所有模型。

每个模型需要指定：
- **模型 ID**：对应 API 中的模型标识（如 `gpt-4o`、`claude-3.5-sonnet`）。
- **模型类型**：`chat`（对话模型）或 `embedding`（嵌入模型）。
- 其他特定于提供方的配置。

## 模型在 Bot 中的使用

在 Bot 设置中选择模型时，下拉列表会根据模型类型自动筛选：
- **对话模型**：`chat` 类型的模型。
- **记忆模型**：`chat` 类型的模型（用于记忆提取）。
- **嵌入模型**：`embedding` 类型的模型（用于向量化）。
- **VLM 模型**：`chat` 类型的模型（用于视觉任务）。

## 搜索提供方

除了 LLM 模型，Memoh 还支持配置搜索引擎提供方。

点击左侧导航的 **搜索提供方** 进入。

搜索提供方用于 Bot 的 `web_search` 工具，配置后 Bot 可以通过搜索引擎查找实时信息。

目前支持的搜索提供方包括 Brave Search 等，需要提供对应的 API Key。
