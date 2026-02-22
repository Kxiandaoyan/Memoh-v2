#!/bin/sh
# Memoh Upgrade Script
# Upgrades an existing Docker Compose deployment to the latest version.
# Usage: ./scripts/upgrade.sh [--no-backup] [--no-pull] [-y|--yes]
set -e

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
CYAN='\033[0;36m'
NC='\033[0m'

NO_BACKUP=false
NO_PULL=false
SILENT=false

for arg in "$@"; do
  case "$arg" in
    --no-backup) NO_BACKUP=true ;;
    --no-pull)   NO_PULL=true ;;
    -y|--yes)    SILENT=true ;;
  esac
done

# Auto-silent if no TTY
if [ "$SILENT" = false ] && ! [ -e /dev/tty ]; then
  SILENT=true
fi

# ── Locate project root ──────────────────────────────────────────────
# Works for both ./scripts/upgrade.sh and curl ... | sh
find_project_root() {
  # 1) Try $0-based detection (works when script is run directly)
  if [ -f "$0" ]; then
    local dir
    dir="$(cd "$(dirname "$0")" && pwd)"
    if [ -f "$dir/../docker-compose.yml" ]; then
      echo "$(cd "$dir/.." && pwd)"
      return
    fi
  fi
  # 2) Current directory
  if [ -f "./docker-compose.yml" ] && [ -d "./.git" ]; then
    pwd; return
  fi
  # 3) Memoh-v2 subdirectory under current dir
  if [ -f "./Memoh-v2/docker-compose.yml" ]; then
    echo "$(cd ./Memoh-v2 && pwd)"; return
  fi
  # 4) Default install location ~/memoh/Memoh-v2
  local default_loc="${HOME:-/tmp}/memoh/Memoh-v2"
  if [ -f "$default_loc/docker-compose.yml" ]; then
    echo "$default_loc"; return
  fi
  # 5) ~/Memoh-v2
  if [ -f "${HOME:-/tmp}/Memoh-v2/docker-compose.yml" ]; then
    echo "${HOME:-/tmp}/Memoh-v2"; return
  fi
  return 1
}

PROJECT_ROOT="$(find_project_root)" || {
  echo "${RED}Error: Cannot find Memoh project directory.${NC}"
  echo ""
  echo "Looked in:"
  echo "  • Current directory ($(pwd))"
  echo "  • $(pwd)/Memoh-v2/"
  echo "  • ${HOME:-/tmp}/memoh/Memoh-v2/"
  echo "  • ${HOME:-/tmp}/Memoh-v2/"
  echo ""
  echo "Please cd into the Memoh project root first, then re-run."
  exit 1
}

cd "$PROJECT_ROOT"
echo "  Project root: ${CYAN}${PROJECT_ROOT}${NC}"

echo "${GREEN}========================================${NC}"
echo "${GREEN}   Memoh Upgrade${NC}"
echo "${GREEN}========================================${NC}"
echo ""

# ── Pre-flight checks ────────────────────────────────────────────────
if ! command -v docker >/dev/null 2>&1; then
  echo "${RED}Error: Docker is not installed${NC}"
  exit 1
fi
if ! docker compose version >/dev/null 2>&1; then
  echo "${RED}Error: Docker Compose v2 is required${NC}"
  exit 1
fi

# Check if services are running
RUNNING=$(docker compose ps --status running -q 2>/dev/null | wc -l | tr -d ' ')
if [ "$RUNNING" = "0" ]; then
  echo "${YELLOW}Warning: No Memoh services are currently running.${NC}"
  echo "If this is a fresh install, use ${CYAN}scripts/install.sh${NC} instead."
  if [ "$SILENT" = false ]; then
    printf "Continue anyway? [y/N]: " > /dev/tty
    read -r confirm < /dev/tty || true
    case "$confirm" in
      y|Y|yes|YES) ;;
      *) echo "Cancelled."; exit 0 ;;
    esac
  fi
fi

# ── Record current version ───────────────────────────────────────────
OLD_COMMIT="unknown"
if command -v git >/dev/null 2>&1 && [ -d ".git" ]; then
  OLD_COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
fi
echo "  Current commit: ${CYAN}${OLD_COMMIT}${NC}"

# ── Optional: Backup database ────────────────────────────────────────
if [ "$NO_BACKUP" = false ]; then
  PG_CONTAINER=$(docker compose ps -q postgres 2>/dev/null || true)
  if [ -n "$PG_CONTAINER" ]; then
    PG_RUNNING=$(docker inspect -f '{{.State.Running}}' "$PG_CONTAINER" 2>/dev/null || echo "false")
    if [ "$PG_RUNNING" = "true" ]; then
      BACKUP_DIR="${PROJECT_ROOT}/backups"
      mkdir -p "$BACKUP_DIR"
      BACKUP_FILE="${BACKUP_DIR}/memoh_pre_upgrade_$(date +%Y%m%d_%H%M%S).sql.gz"
      echo ""
      echo "${CYAN}Backing up database...${NC}"
      if docker compose exec -T postgres pg_dump -U memoh memoh 2>/dev/null | gzip > "$BACKUP_FILE"; then
        BACKUP_SIZE=$(ls -lh "$BACKUP_FILE" 2>/dev/null | awk '{print $5}')
        echo "${GREEN}✓ Database backed up to ${BACKUP_FILE} (${BACKUP_SIZE})${NC}"
      else
        echo "${YELLOW}⚠ Database backup failed (non-fatal, continuing)${NC}"
        rm -f "$BACKUP_FILE"
      fi
    else
      echo "${YELLOW}⚠ PostgreSQL not running, skipping backup${NC}"
    fi
  else
    echo "${YELLOW}⚠ PostgreSQL container not found, skipping backup${NC}"
  fi
fi

# ── Pull latest code ─────────────────────────────────────────────────
if [ "$NO_PULL" = false ]; then
  if command -v git >/dev/null 2>&1 && [ -d ".git" ]; then
    echo ""
    echo "${CYAN}Pulling latest code...${NC}"
    if git pull --ff-only; then
      NEW_COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
      if [ "$OLD_COMMIT" = "$NEW_COMMIT" ]; then
        echo "${GREEN}✓ Already up to date (${NEW_COMMIT})${NC}"
      else
        echo "${GREEN}✓ Updated: ${OLD_COMMIT} → ${NEW_COMMIT}${NC}"
      fi
    else
      echo "${YELLOW}⚠ git pull --ff-only failed. You may have local changes.${NC}"
      echo "  Resolve manually, then re-run with: ./scripts/upgrade.sh --no-pull"
      exit 1
    fi
  else
    echo "${YELLOW}⚠ Not a git repository, skipping code pull${NC}"
  fi
fi

# ── Rebuild and restart services ──────────────────────────────────────
echo ""
echo "${CYAN}Rebuilding and restarting services...${NC}"
docker compose up -d --build --remove-orphans

# ── Wait for PostgreSQL to be healthy ─────────────────────────────────
echo ""
echo "${CYAN}Waiting for PostgreSQL to be ready...${NC}"
RETRIES=30
while [ $RETRIES -gt 0 ]; do
  if docker compose exec -T postgres pg_isready -U memoh >/dev/null 2>&1; then
    echo "${GREEN}✓ PostgreSQL is ready${NC}"
    break
  fi
  RETRIES=$((RETRIES - 1))
  sleep 2
done
if [ $RETRIES -eq 0 ]; then
  echo "${RED}Error: PostgreSQL did not become ready in time${NC}"
  echo "Check logs: docker compose logs postgres"
  exit 1
fi

# ── Run database migrations ───────────────────────────────────────────
echo ""
echo "${CYAN}Running database migrations...${NC}"
MIGRATION_FAILED=false
for migration_file in db/migrations/*.up.sql; do
  if [ -f "$migration_file" ]; then
    FNAME=$(basename "$migration_file")
    # Migration files are mounted at /migrations inside the postgres container.
    if docker compose exec -T postgres psql -U memoh -d memoh -f "/migrations/${FNAME}" >/dev/null 2>&1; then
      echo "  ${GREEN}✓${NC} ${FNAME}"
    else
      echo "  ${RED}✗${NC} ${FNAME}"
      MIGRATION_FAILED=true
    fi
  fi
done

if [ "$MIGRATION_FAILED" = true ]; then
  echo ""
  echo "${YELLOW}⚠ Some migrations reported errors (may be normal for IF NOT EXISTS statements)${NC}"
else
  echo "${GREEN}✓ All migrations applied${NC}"
fi

# ── Sync TOOLS.md and skills to all existing bots ────────────────────
echo ""
echo "${CYAN}Syncing TOOLS.md and skills to existing bots...${NC}"
SYNC_OK=true

# Sync via server container (has memoh_data mounted at /opt/memoh/data)
BOTS_DIR="/opt/memoh/data/bots"
TEMPLATE_TOOLS="cmd/mcp/template/TOOLS.md"
SKILLS_DEFAULTS="internal/skills/defaults"

if [ -f "$TEMPLATE_TOOLS" ] && docker compose exec -T server test -d "$BOTS_DIR" 2>/dev/null; then
  # Copy TOOLS.md into server container temp location
  docker compose cp "$TEMPLATE_TOOLS" server:/tmp/_upgrade_TOOLS.md 2>/dev/null || true

  # Copy skills defaults directory
  if [ -d "$SKILLS_DEFAULTS" ]; then
    docker compose cp "$SKILLS_DEFAULTS" server:/tmp/_upgrade_skills_defaults 2>/dev/null || true
  fi

  # Run sync inside server container
  docker compose exec -T server sh -c '
    BOTS_DIR="/opt/memoh/data/bots"
    TOOLS_SRC="/tmp/_upgrade_TOOLS.md"
    SKILLS_SRC="/tmp/_upgrade_skills_defaults"
    synced=0
    for bot_dir in "$BOTS_DIR"/*/; do
      [ -d "$bot_dir" ] || continue
      bot_id=$(basename "$bot_dir")
      # Skip non-UUID directories
      case "$bot_id" in ????????-????-????-????-????????????) ;; *) continue ;; esac

      # Update TOOLS.md (always overwrite with latest)
      if [ -f "$TOOLS_SRC" ]; then
        data_dir="$bot_dir"
        # Find the data mount point (could be root or /data subdirectory)
        if [ -d "$bot_dir/data" ]; then data_dir="$bot_dir/data"; fi
        cp "$TOOLS_SRC" "$data_dir/TOOLS.md" 2>/dev/null || true
      fi

      # Sync skills (copy new ones, skip existing to preserve customizations)
      if [ -d "$SKILLS_SRC" ]; then
        skills_dir="$bot_dir/.skills"
        mkdir -p "$skills_dir" 2>/dev/null || true
        for skill in "$SKILLS_SRC"/*/; do
          [ -d "$skill" ] || continue
          skill_name=$(basename "$skill")
          if [ ! -d "$skills_dir/$skill_name" ]; then
            cp -a "$skill" "$skills_dir/$skill_name" 2>/dev/null || true
          fi
        done
      fi

      synced=$((synced + 1))
    done
    # Cleanup
    rm -f "$TOOLS_SRC" 2>/dev/null || true
    rm -rf "$SKILLS_SRC" 2>/dev/null || true
    echo "$synced"
  ' 2>/dev/null && {
    echo "${GREEN}✓ TOOLS.md and skills synced to existing bots${NC}"
  } || {
    echo "${YELLOW}⚠ Sync failed (non-fatal, bots will use cached versions)${NC}"
    SYNC_OK=false
  }
else
  echo "${YELLOW}⚠ Skipping sync (template or bots directory not found)${NC}"
fi

# ── Wait for all services to be healthy ───────────────────────────────
echo ""
echo "${CYAN}Waiting for services to stabilize...${NC}"
sleep 5

# Show final status
echo ""
docker compose ps
echo ""

# ── Health check ──────────────────────────────────────────────────────
echo "${CYAN}Running health checks...${NC}"
HEALTH_OK=true

# Check server
if docker compose exec -T server wget -q --spider http://127.0.0.1:8080/health 2>/dev/null; then
  echo "  ${GREEN}✓${NC} Server (API)       — healthy"
else
  echo "  ${YELLOW}⚠${NC} Server (API)       — not yet responding (may still be starting)"
  HEALTH_OK=false
fi

# Check agent
if docker compose exec -T agent wget -q --spider http://127.0.0.1:8081/ 2>/dev/null; then
  echo "  ${GREEN}✓${NC} Agent Gateway      — healthy"
else
  echo "  ${YELLOW}⚠${NC} Agent Gateway      — not yet responding (may still be starting)"
  HEALTH_OK=false
fi

echo ""
echo "${GREEN}========================================${NC}"
echo "${GREEN}   Upgrade complete!${NC}"
echo "${GREEN}========================================${NC}"
echo ""
echo "  Web UI:          http://localhost:8082"
echo "  API:             http://localhost:8080"
echo "  Agent Gateway:   http://localhost:8081"
echo ""
if [ "$HEALTH_OK" = false ]; then
  echo "${YELLOW}Some services may still be starting. Check with:${NC}"
  echo "  docker compose logs -f"
fi
echo "Commands:"
echo "  docker compose ps       # Status"
echo "  docker compose logs -f  # Logs"
echo "  docker compose down     # Stop"
echo ""
