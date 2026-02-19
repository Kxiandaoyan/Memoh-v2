# Memoh-v2 文档中心

> 返回 [项目首页](../README.md) · [English](../README_EN.md)

---

## 项目文档

| 文档 | 说明 |
|------|------|
| [功能详解](features.md) | 12 项核心功能完整介绍，Bot 管理、记忆、容器、进化、MCP 等 |
| [概念手册](concepts.md) | 模型类型（Chat vs Embedding）、常见模型配置表、人设体系、Provider 配置 |
| [安装与升级](installation.md) | 一键安装 / 升级 / 卸载 / 数据库管理 / 数据迁移 / 脚本一览 |
| [已知局限性](known-limitations.md) | 当前不足的客观评估与解决方案 |
| [与 OpenClaw 对比](comparison.md) | 42 项全面对比，Memoh-v2 胜 27 项 |

**English versions:**
[Features](features_en.md) · [Concepts](concepts_en.md) · [Installation](installation_en.md) · [Limitations](known-limitations_en.md) · [Comparison](comparison_en.md)

---

## 使用教程

本目录包含 Memoh-v2 的完整使用教程和技巧，帮助你快速上手并充分利用平台的所有功能。

| 编号 | 文档 | 说明 |
|------|------|------|
| 01 | [快速上手](01-quick-start.md) | 登录、导航、创建第一个 Bot |
| 02 | [Bot 管理](02-bot-management.md) | Bot 列表、创建、模板、卡片信息 |
| 03 | [Bot 设置详解](03-bot-settings.md) | 模型、行为、功能开关、危险区域 |
| 04 | [对话功能](04-chat.md) | Web 端对话、消息类型、Token 显示 |
| 05 | [平台接入](05-channel-integration.md) | Telegram / 飞书 / Discord 接入与群聊行为 |
| 06 | [记忆系统](06-memory-system.md) | 记忆浏览、编辑、压缩、搜索可视化 |
| 07 | [容器与文件](07-container-files.md) | 容器管理、快照、人格文件编辑 |
| 08 | [MCP 工具与技能](08-mcp-skills.md) | 内置工具、多路 MCP 服务器、技能管理 |
| 09 | [自进化与心跳](09-evolution-heartbeat.md) | 自进化机制、心跳配置、实验记录 |
| 10 | [子智能体](10-subagents.md) | 创建和管理子智能体 |
| 11 | [定时任务](11-schedules.md) | 查看和管理 Bot 的定时任务 |
| 12 | [模型与提供方](12-models-providers.md) | 添加 LLM 提供方和模型配置 |
| 13 | [Token 用量](13-token-usage.md) | 用量统计、趋势图、模型分布 |
| 14 | [流程日志](14-process-logs.md) | 查看每轮对话的处理步骤 |
| 15 | [系统设置](15-system-settings.md) | 语言、主题、时区、证书 |
| 16 | [管理员设置](16-admin-settings.md) | 个人资料、密码、清除绑定 |
| 17 | [使用技巧](17-tips.md) | 实用技巧与最佳实践 |
| 18 | [OpenViking 上下文数据库](18-openviking.md) | 知识库管理、分层上下文、内置工具 |

---

## 项目内部文档

| 文档 | 说明 |
|------|------|
| [项目完成进度](FEATURE_AUDIT.md) | 93 项功能逐一审计，对标代码验证完成度评估 |
| [项目提示词](PROMPTS_INVENTORY.md) | 全部提示词完整清单、参数、数据流、Token 估算 |

---

## 开发计划归档

记录本项目每次重大改造阶段的设计方案与实施细节，便于后续参考和继续迭代。

| 计划 | 说明 | 状态 |
|------|------|------|
| [超越 OpenClaw 三梯队改造](plans/plan-01-surpass-openclaw.md) | MMR 重排序、Embedding 缓存、工具循环检测、子智能体持久化、心跳活跃时段、Prompt 双模式、群组防抖 | ✅ 已完成 |
| [增强日志系统与单轮导出](plans/plan-02-enhanced-logging.md) | 丰富 12 个日志步骤数据字段，新增 trace 导出 API 和前端复制按鈕 | ✅ 已完成 |
| [智能化 / Token / 记忆全面审计](plans/plan-03-intelligence-audit.md) | 14 项 P0-P2 改进，Token 估算、软截断、记忆阈値、MMR、时间衰减、Embedding 缓存等 | ✅ 已完成 |
| [定时任务通知可靠性修复](plans/plan-04-schedule-reliability.md) | Prompt 强制 send 指令 + Go 侧工具结果兜底发送 | ✅ 已完成 |
| [README 拆分文档结构化](plans/plan-05-readme-restructure.md) | 将 README 拆分为精简的导航页 + doc/ 多文档结构 | ✅ 已完成 |
| [已知局限性全面解决计划](plans/plan-06-limitations-resolved.md) | 8 项修复：防抖可配置、子智能体历史 UI、BM25 持久化、记忆去重、进化 diff+回退、容器人设自愈等 | ✅ 已完成 |
