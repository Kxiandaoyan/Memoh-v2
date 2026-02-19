# Installation & Upgrade

> Back to [Documentation](./README.md) · [Project Home](../README_EN.md)

---

## One-Click Install

```bash
curl -fsSL https://raw.githubusercontent.com/Kxiandaoyan/Memoh-v2/main/scripts/install.sh | sh
```

The install script: detects Docker → detects previous installation (optional cleanup) → clones repo → generates config.toml → builds and starts all services.

Supports interactive configuration for workspace, data directory, admin password, etc. Add `-y` for silent mode.

## Upgrade (No Data Loss)

```bash
curl -fsSL https://raw.githubusercontent.com/Kxiandaoyan/Memoh-v2/main/scripts/upgrade.sh | sh
```

The script auto-locates the Memoh project directory. Or run directly:

```bash
cd ~/memoh/Memoh-v2 && ./scripts/upgrade.sh
```

Upgrade flow: auto-backup database → `git pull` → rebuild Docker images → run database migrations → health check.

**All data (PostgreSQL, Qdrant, bot files) is stored in Docker named volumes and host directories. Upgrades never lose data.**

| Flag | Description |
|------|-------------|
| `--no-backup` | Skip pre-upgrade database backup |
| `--no-pull` | Skip git pull (if code was updated manually) |
| `-y` | Silent mode, skip all confirmation prompts |

## Uninstall

```bash
curl -fsSL https://raw.githubusercontent.com/Kxiandaoyan/Memoh-v2/main/scripts/uninstall.sh | sh
```

| Flag | Description |
|------|-------------|
| `--keep-data` | Keep Docker volumes (database, vector DB, bot data preserved) |
| `--keep-images` | Keep built Docker images |
| `-y` | Silent mode |

A final database backup is auto-created in `backups/` before uninstalling.

## Database Management

```bash
./scripts/db-up.sh      # Run database migrations (incremental)
./scripts/db-drop.sh     # Rollback all tables (dangerous, requires confirmation)
```

## Data Migration

```bash
# Backup on old server
docker compose exec -T postgres pg_dump -U memoh memoh | gzip > memoh-backup.sql.gz

# Restore on new server
gunzip -c memoh-backup.sql.gz | docker compose exec -T postgres psql -U memoh memoh
```

Bot file data is in the host `data/bots/` directory — simply copy it over.

## Script Reference

| Script | Purpose |
|--------|---------|
| `scripts/install.sh` | One-click install (fresh deployment) |
| `scripts/upgrade.sh` | One-click upgrade (data preserved) |
| `scripts/uninstall.sh` | Uninstall (optional data retention) |
| `scripts/db-up.sh` | Database migration |
| `scripts/db-drop.sh` | Database rollback |
| `scripts/compile-mcp.sh` | Compile MCP binary and hot-reload into container |
