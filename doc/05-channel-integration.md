# 平台接入

Memoh 支持将 Bot 接入多个即时通讯平台。目前支持的渠道适配器有：

| 渠道 | 说明 |
|------|------|
| Telegram | 通过 Telegram Bot API 接入 |
| 飞书 (Feishu/Lark) | 通过飞书开放平台接入 |
| 本地 (Local/Web) | 内置的 Web 对话界面 |

## 配置渠道

1. 进入 Bot 详情页 → **渠道** 标签。
2. 页面会列出所有可用的渠道类型，每种渠道旁边显示配置状态。
3. 点击某个渠道，进入配置面板。

### 配置字段

不同渠道的配置字段不同，由后端定义的 `config_schema` 动态生成。

**Telegram**：
- **Bot API Token**（必填，密文字段）：从 @BotFather 获取的 Bot API Token。

**飞书**：
- 根据飞书开放平台提供的 App ID、App Secret 等信息配置。

每个渠道还有一个 **状态开关**（active / inactive），用于控制是否启用该渠道。

### 配置步骤（以 Telegram 为例）

1. 在 Telegram 中找到 @BotFather，发送 `/newbot` 创建一个 Telegram Bot。
2. 获取 Bot API Token。
3. 在 Memoh 的 Bot 详情页 → 渠道 → Telegram，粘贴 Token。
4. 将状态设为 **active**，保存。
5. 在 Telegram 中找到你的 Bot，发送消息即可开始对话。

## 私聊与群聊

### 私聊 (Private Chat)

直接在 Telegram 中给 Bot 发私信，Bot 会自动响应所有消息。

### 群聊 (Group Chat)

将 Bot 拉入 Telegram 群组后，Bot 的群聊行为取决于 **群聊需要 @提及** 设置（在 Bot 设置页面配置）。

#### 群聊需要 @提及 = 开启（默认）

Bot 只在以下情况才会响应：

- **@提及**：在群消息中 @Bot 的用户名。
- **引用回复**：回复 Bot 发出的某条消息。
- **命令**：发送以 `/` 开头的命令。

其他普通群消息会被忽略。

#### 群聊需要 @提及 = 关闭

Bot 会响应群聊中 **所有人类用户** 的消息。

**防止无限循环**：即使关闭了此设置，Bot 也 **不会** 回复其他 Bot 发出的消息。这是为了防止多个 Bot 在群里互相触发，导致无限对话循环和 Token 浪费。

**手动触发 Bot 间对话**：如果你需要让 Bot B 回应 Bot A 的内容，可以：
- **引用回复**：引用 Bot A 的消息，并 @Bot B。此时 Bot B 会收到触发并回复。
- @提及和引用回复始终有效，不受 `is_from_bot` 限制。

### Bot 类型与群聊的关系

- **personal（私人）Bot**：即使被拉入群聊，也 **不会** 在群组中响应任何消息。只在私聊中工作。
- **public（公开）Bot**：可以在群聊中正常工作，行为由上述设置控制。

## Telegram 特殊说明

### Privacy Mode

Telegram Bot 默认开启 Privacy Mode，在此模式下 Bot 只能收到 @提及和命令消息。如需让 Bot 响应群内所有消息（配合"群聊需要 @提及 = 关闭"），需要：

1. 在 Telegram 中找到 @BotFather。
2. 发送 `/mybots`，选择你的 Bot。
3. 选择 **Bot Settings** → **Group Privacy** → **Turn off**。

关闭后 Bot 才能接收到群内所有消息。

### 消息与 Web 端的关系

- Telegram 私聊消息会同步显示在 Web 端的对话界面中。
- Telegram **群聊** 消息 **不会** 显示在 Web 端对话界面中，避免混淆。
- 如需查看群聊消息，可到 Bot 详情页的 **历史** 标签查看完整消息记录。

### Token 用量显示

Bot 在 Telegram 中回复时，消息末尾会附带 Token 用量提示，格式如 `⚡ 11.6k`，与 Web 端保持一致。

## 渠道身份绑定

用户可以通过 **绑定码** 将不同平台的账号关联到同一个 Memoh 用户。详情参考 [管理员设置 - 渠道绑定](16-admin-settings.md#渠道身份绑定)。
