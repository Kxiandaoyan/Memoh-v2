# 安装与升级

> 返回 [文档首页](./README.md) · [项目首页](../README.md)

---

## 一键安装

```bash
curl -fsSL https://raw.githubusercontent.com/Kxiandaoyan/Memoh-v2/main/scripts/install.sh | sh
```

安装脚本自动：检测 Docker → 检测旧版本（可选清理）→ 克隆代码 → 生成 config.toml → 构建并启动所有服务。

支持交互式配置工作目录、数据目录、管理员密码等；加 `-y` 跳过交互。

## 升级（不丢数据）

```bash
curl -fsSL https://raw.githubusercontent.com/Kxiandaoyan/Memoh-v2/main/scripts/upgrade.sh | sh
```

脚本自动查找 Memoh 项目目录，无需手动 `cd`。也可在项目目录下直接执行：

```bash
cd ~/memoh/Memoh-v2 && ./scripts/upgrade.sh
```

升级流程：自动备份数据库 → `git pull` → 重建 Docker 镜像 → 数据库迁移 → 健康检查。

**所有数据（PostgreSQL、Qdrant、Bot 文件）存储在 Docker named volumes 和宿主机目录中，升级不会丢失任何数据。**

| 参数 | 说明 |
|------|------|
| `--no-backup` | 跳过升级前数据库备份 |
| `--no-pull` | 跳过 git pull（已手动更新代码时） |
| `-y` | 静默模式，跳过所有确认提示 |

## 卸载

```bash
curl -fsSL https://raw.githubusercontent.com/Kxiandaoyan/Memoh-v2/main/scripts/uninstall.sh | sh
```

| 参数 | 说明 |
|------|------|
| `--keep-data` | 保留 Docker volumes（数据库、向量库、Bot 数据不删） |
| `--keep-images` | 保留已构建的 Docker 镜像 |
| `-y` | 静默模式 |

卸载前自动创建数据库最终备份到 `backups/` 目录。

## 数据库管理

```bash
./scripts/db-up.sh      # 执行数据库迁移（增量）
./scripts/db-drop.sh     # 回滚所有表（危险操作，需确认）
```

## 数据迁移

```bash
# 旧服务器备份
docker compose exec -T postgres pg_dump -U memoh memoh | gzip > memoh-backup.sql.gz

# 新服务器恢复
gunzip -c memoh-backup.sql.gz | docker compose exec -T postgres psql -U memoh memoh
```

Bot 文件数据在宿主机 `data/bots/` 目录下，直接拷贝即可。

## 脚本一览

| 脚本 | 用途 |
|------|------|
| `scripts/install.sh` | 一键安装（全新部署） |
| `scripts/upgrade.sh` | 一键升级（保留数据） |
| `scripts/uninstall.sh` | 卸载（可选保留数据） |
| `scripts/db-up.sh` | 数据库迁移 |
| `scripts/db-drop.sh` | 数据库回滚 |
| `scripts/compile-mcp.sh` | 编译 MCP 二进制并热更新到容器 |
