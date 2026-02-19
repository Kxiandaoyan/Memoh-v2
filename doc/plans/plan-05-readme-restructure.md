# README 拆分文档结构化

> 状态：✅ 全部完成（5/5 tasks）

---

## 当前问题

根目录 `README.md` 长达 691 行，`README_EN.md` 长达 664 行。用户在 GitHub 着陆页下拉很难浏览完。详细内容应移入 `doc/`，根目录 README 只作为简洁的着陆页。

---

## 新文档结构

### 根目录 README.md（精简着陆页，约 120 行）

保留：
- 头部 + badges + 一句话描述
- 截图预览
- 快速开始（GitHub 着陆页必须有）
- 架构图（可视化，紧凑）
- 功能亮点（单行 bullet list，链接到详情页，NOT 完整描述）
- 指向 doc/ 的链接
- 技术栈（简短表格）
- 致谢

### doc/ 新增文件

从 README 中提取到这些新文档：

| 文件 | 内容 |
|------|------|
| `doc/features.md` | 完整 12 节功能指南 |
| `doc/concepts.md` | 模型类型、人设体系、Provider 配置示例 |
| `doc/known-limitations.md` | 所有已知限制条目 |
| `doc/comparison.md` | OpenClaw 42 项对比表 |
| `doc/installation.md` | 完整安装、升级、卸载、数据库管理、数据迁移 |

以及对应的英文版本（`_en.md` 后缀）。

### doc/README.md（统一导航中心）

新结构：
1. **项目文档**（新增，来自 README 拆分）—— features、concepts、installation、known-limitations、comparison
2. **使用教程**（现有 01-18 文档，保持不变）
3. **内部文档**（现有 FEATURE_AUDIT、PROMPTS_INVENTORY）

---

## 导航链接

- 根目录 README.md：顶部导航栏链接到 `doc/` 页面
- 根目录 README_EN.md：相同结构，链接到 `_en` 变体
- doc/README.md：在现有"使用教程"上方添加"项目文档"节
