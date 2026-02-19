# OpenViking 上下文数据库

OpenViking 是一个专为 AI Agent 设计的上下文数据库，由字节跳动火山引擎团队开源。它通过文件系统范式（`viking://` URI）统一管理记忆、资源和技能，并提供分层上下文加载（L0 摘要 → L1 概览 → L2 完整内容）和语义搜索能力。

## 启用方式

1. 进入 Bot 详情页 → **设置** 标签
2. 找到 **启用 OpenViking** 开关
3. 开启后系统自动：
   - 生成 `ov.conf` 配置文件（基于当前系统模型设置）
   - 在容器内初始化 OpenViking 数据目录
   - 注册原生 `ov_*` 工具供 Bot 使用

## 原生工具

启用后，Bot 可以直接使用以下工具（无需编写 Python 脚本）：

| 工具名 | 功能说明 |
|--------|---------|
| `ov_initialize` | 初始化数据目录（首次使用时自动调用） |
| `ov_find` | 快速语义搜索，返回匹配的 URI 和相关度分数 |
| `ov_search` | 高级检索，带意图分析和分层递归搜索 |
| `ov_read` | 读取 viking:// URI 的完整内容（L2） |
| `ov_abstract` | 获取 L0 摘要（~100 tokens，一句话总结） |
| `ov_overview` | 获取 L1 概览（~2k tokens，结构和关键信息） |
| `ov_ls` | 列出 viking:// 目录下的内容 |
| `ov_tree` | 获取目录树视图 |
| `ov_add_resource` | 添加资源（URL、文件或目录），自动解析和索引 |
| `ov_rm` | 删除指定 viking:// URI 的资源 |
| `ov_session_commit` | 提交对话记录，提取长期记忆 |

## viking:// 文件系统

OpenViking 使用虚拟文件系统管理所有上下文：

```
viking://
├── resources/              # 外部资源（文档、代码库、网页等）
│   ├── my_project/
│   │   ├── docs/
│   │   └── src/
│   └── ...
├── user/                   # 用户相关
│   └── memories/           # 用户偏好、习惯
│       ├── preferences/
│       └── ...
└── agent/                  # Agent 相关
    ├── skills/             # Agent 技能
    ├── memories/           # Agent 任务经验
    └── instructions/       # Agent 指令
```

## 三层上下文结构

OpenViking 自动将内容处理为三个层级，按需加载以节省 Token：

| 层级 | 名称 | Token 量 | 用途 |
|------|------|----------|------|
| L0 | Abstract（摘要） | ~100 | 快速识别相关性 |
| L1 | Overview（概览） | ~2,000 | 理解结构和关键信息，用于规划 |
| L2 | Details（详情） | 完整内容 | 深入阅读，按需加载 |

## 自动集成

### 上下文注入

每次对话时，如果 Bot 启用了 OpenViking，系统会自动：
1. 对用户消息进行语义搜索
2. 获取最相关的 L0 摘要
3. 作为上下文注入到对话中

这让 Bot 在回答时能利用 OpenViking 中存储的知识，而不需要 Bot 主动调用工具。

### 会话记忆提取

每轮对话完成后，系统自动将对话内容提交到 OpenViking Session，触发：
- 对话归档
- 长期记忆提取（用户偏好、Agent 经验等）
- 自动更新到 `viking://user/memories/` 和 `viking://agent/memories/`

## 典型使用流程

### 为 Bot 添加知识库

```
用户：请把这个文档加入你的知识库
Bot（自动调用 ov_add_resource）：
  → 添加文档 URL 到 OpenViking
  → 等待处理完成
  → 资源已添加到 viking://resources/
```

### 搜索知识库

```
用户：你知道关于 API 认证的内容吗？
Bot（自动调用 ov_find / ov_search）：
  → 在 viking://resources/ 中搜索 "API 认证"
  → 找到相关文档
  → 使用 ov_overview 获取概览
  → 基于概览回答用户问题
```

### 深入阅读

```
用户：给我看完整的认证文档
Bot（自动调用 ov_read）：
  → 读取 viking://resources/.../auth.md 的完整内容
  → 展示给用户
```

## 配置文件

`ov.conf` 自动生成在 Bot 的数据目录中，包含：

```json
{
  "embedding": {
    "dense": {
      "api_base": "...",
      "api_key": "...",
      "provider": "openai",
      "dimension": 1536,
      "model": "text-embedding-3-small"
    }
  },
  "vlm": {
    "api_base": "...",
    "api_key": "...",
    "provider": "openai",
    "model": "gpt-4o"
  }
}
```

- **embedding.dense**: 用于向量化和语义检索的 Embedding 模型
- **vlm**: 用于内容理解和摘要生成的 VLM/Chat 模型
- 配置从 Bot 的模型设置自动填充，也可在 Web UI 的「文件」标签中手动编辑

## 何时开启 OpenViking

### 适合开启的场景

- **Bot 需要查询外部文档**：技术文档、项目规格、知识库、书籍等大型文本资源
- **Bot 需要管理代码库**：将代码仓库加入知识库，让 Bot 能回答代码相关问题
- **Bot 需要多层次知识检索**：先用 L0 摘要快速筛选，再按需加载 L2 完整内容，节省 Token
- **长期持续积累知识**：Bot 会将每轮对话中学到的重要信息自动沉淀到 OpenViking

### 不需要开启的场景

- **纯聊天助手**：只需要对话记忆（用户偏好、习惯等），内置记忆系统已够用
- **没有外部知识库需求**：Bot 只处理即时问题，不需要检索历史文档
- **对话简单、消息短**：OpenViking 的价值在于处理大规模结构化知识，简单对话场景性价比低

### 决策建议

```
需要查询大型文档/代码库？
    ├── 是 → 开启 OpenViking，添加资源到 viking://resources/
    └── 否 → 只需内置记忆（Qdrant + BM25），更轻量
```

> **提示**：两套系统可以同时启用。如果不确定，先不开启 OpenViking，待有明确的知识库需求再开启。

---

## 与现有记忆系统的关系

Memoh 有两套记忆系统，它们互补使用：

| 特性 | 内置记忆 (Qdrant) | OpenViking |
|------|-------------------|------------|
| 存储 | 向量数据库 | 虚拟文件系统 + 向量索引 |
| 搜索 | 语义搜索 + BM25 | 分层递归检索 |
| 内容 | 从对话中提取的关键信息 | 文档、代码库、网页等外部资源 + 对话记忆 |
| 层级 | 扁平存储 | L0/L1/L2 三层按需加载 |
| 适合 | 对话记忆、用户偏好 | 知识库管理、长文档、项目文档 |

两者可同时启用，互不冲突。
