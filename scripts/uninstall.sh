#!/bin/sh
# Memoh Uninstall Script
# Removes Memoh containers, images, volumes, and optionally data/source code.
# Usage: ./scripts/uninstall.sh [-y|--yes] [--keep-data] [--keep-images]
set -e

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
CYAN='\033[0;36m'
NC='\033[0m'

SILENT=false
KEEP_DATA=false
KEEP_IMAGES=false

for arg in "$@"; do
  case "$arg" in
    -y|--yes)         SILENT=true ;;
    --keep-data)      KEEP_DATA=true ;;
    --keep-images)    KEEP_IMAGES=true ;;
  esac
done

# Auto-silent if no TTY
if [ "$SILENT" = false ] && ! [ -e /dev/tty ]; then
  SILENT=true
fi

# ── Locate project root ──────────────────────────────────────────────
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$PROJECT_ROOT"

if [ ! -f "docker-compose.yml" ]; then
  echo "${RED}Error: docker-compose.yml not found in ${PROJECT_ROOT}${NC}"
  echo "Please run this script from the Memoh project root."
  exit 1
fi

echo "${RED}========================================${NC}"
echo "${RED}   Memoh Uninstall${NC}"
echo "${RED}========================================${NC}"
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

# ── Show what will be removed ─────────────────────────────────────────
echo "This will remove:"
echo "  ${CYAN}• All Memoh containers${NC} (server, agent, web, postgres, qdrant, containerd)"
if [ "$KEEP_DATA" = false ]; then
  echo "  ${RED}• All Docker volumes${NC} (database, qdrant, containerd data)"
  echo "    ⚠  This includes all your bots, messages, memories, and configs!"
else
  echo "  ${GREEN}• Docker volumes will be KEPT${NC} (database, qdrant, containerd data)"
fi
if [ "$KEEP_IMAGES" = false ]; then
  echo "  ${CYAN}• All Memoh Docker images${NC}"
else
  echo "  ${GREEN}• Docker images will be KEPT${NC}"
fi
echo ""

# ── Confirmation ──────────────────────────────────────────────────────
if [ "$SILENT" = false ]; then
  if [ "$KEEP_DATA" = false ]; then
    echo "${RED}WARNING: This will permanently delete all your data!${NC}"
    printf "Type 'yes' to confirm: " > /dev/tty
    read -r confirm < /dev/tty || true
    if [ "$confirm" != "yes" ]; then
      echo "Cancelled."
      exit 0
    fi
  else
    printf "Proceed? [y/N]: " > /dev/tty
    read -r confirm < /dev/tty || true
    case "$confirm" in
      y|Y|yes|YES) ;;
      *) echo "Cancelled."; exit 0 ;;
    esac
  fi
  echo ""
fi

# ── Optional: Backup before uninstall ─────────────────────────────────
if [ "$KEEP_DATA" = false ]; then
  PG_CONTAINER=$(docker compose ps -q postgres 2>/dev/null || true)
  if [ -n "$PG_CONTAINER" ]; then
    PG_RUNNING=$(docker inspect -f '{{.State.Running}}' "$PG_CONTAINER" 2>/dev/null || echo "false")
    if [ "$PG_RUNNING" = "true" ]; then
      BACKUP_DIR="${PROJECT_ROOT}/backups"
      mkdir -p "$BACKUP_DIR"
      BACKUP_FILE="${BACKUP_DIR}/memoh_final_$(date +%Y%m%d_%H%M%S).sql.gz"
      echo "${CYAN}Creating final database backup before removal...${NC}"
      if docker compose exec -T postgres pg_dump -U memoh memoh 2>/dev/null | gzip > "$BACKUP_FILE"; then
        BACKUP_SIZE=$(ls -lh "$BACKUP_FILE" 2>/dev/null | awk '{print $5}')
        echo "${GREEN}✓ Final backup saved to ${BACKUP_FILE} (${BACKUP_SIZE})${NC}"
      else
        echo "${YELLOW}⚠ Backup failed (continuing with uninstall)${NC}"
        rm -f "$BACKUP_FILE"
      fi
      echo ""
    fi
  fi
fi

# ── Stop and remove containers ────────────────────────────────────────
echo "${CYAN}Stopping all Memoh services...${NC}"
if [ "$KEEP_DATA" = false ]; then
  docker compose down -v --remove-orphans 2>/dev/null || true
  echo "${GREEN}✓ Containers stopped and volumes removed${NC}"
else
  docker compose down --remove-orphans 2>/dev/null || true
  echo "${GREEN}✓ Containers stopped (volumes preserved)${NC}"
fi

# ── Remove Docker images ──────────────────────────────────────────────
if [ "$KEEP_IMAGES" = false ]; then
  echo ""
  echo "${CYAN}Removing Memoh Docker images...${NC}"
  IMAGES=$(docker images --filter "label=com.docker.compose.project=memoh" -q 2>/dev/null || true)
  if [ -z "$IMAGES" ]; then
    # Fallback: match by name pattern
    IMAGES=$(docker images "memoh-*" -q 2>/dev/null || true)
  fi
  if [ -n "$IMAGES" ]; then
    echo "$IMAGES" | xargs docker rmi -f 2>/dev/null || true
    echo "${GREEN}✓ Memoh images removed${NC}"
  else
    echo "  No Memoh images found"
  fi

  # Clean up dangling images from the build
  DANGLING=$(docker images -f "dangling=true" -q 2>/dev/null || true)
  if [ -n "$DANGLING" ]; then
    echo "${CYAN}Cleaning up dangling images...${NC}"
    echo "$DANGLING" | xargs docker rmi -f 2>/dev/null || true
    echo "${GREEN}✓ Dangling images cleaned${NC}"
  fi
fi

# ── Summary ───────────────────────────────────────────────────────────
echo ""
echo "${GREEN}========================================${NC}"
echo "${GREEN}   Memoh has been uninstalled${NC}"
echo "${GREEN}========================================${NC}"
echo ""
echo "Removed:"
echo "  ${GREEN}✓${NC} All Memoh containers"
if [ "$KEEP_DATA" = false ]; then
  echo "  ${GREEN}✓${NC} All Docker volumes (data deleted)"
else
  echo "  ${YELLOW}–${NC} Docker volumes preserved (use 'docker volume ls' to see them)"
fi
if [ "$KEEP_IMAGES" = false ]; then
  echo "  ${GREEN}✓${NC} All Memoh Docker images"
else
  echo "  ${YELLOW}–${NC} Docker images preserved (use 'docker images memoh-*' to see them)"
fi
echo ""
echo "Still on disk:"
echo "  ${YELLOW}•${NC} Source code:  ${PROJECT_ROOT}"
if [ -d "${PROJECT_ROOT}/backups" ]; then
  echo "  ${YELLOW}•${NC} Backups:     ${PROJECT_ROOT}/backups/"
fi
echo ""
echo "To fully remove, delete the project directory:"
echo "  ${CYAN}rm -rf ${PROJECT_ROOT}${NC}"
echo ""
echo "To reinstall later:"
echo "  ${CYAN}curl -fsSL https://raw.githubusercontent.com/Kxiandaoyan/Memoh-v2/main/scripts/install.sh | sh${NC}"
echo ""
