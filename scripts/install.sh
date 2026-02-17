#!/bin/sh
set -e

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
CYAN='\033[0;36m'
NC='\033[0m'

REPO="https://github.com/Kxiandaoyan/Memoh-v2.git"
BRANCH="main"
DIR="Memoh-v2"
SILENT=false

# Parse flags
for arg in "$@"; do
  case "$arg" in
    -y|--yes) SILENT=true ;;
  esac
done

# Auto-silent if no TTY available
if [ "$SILENT" = false ] && ! [ -e /dev/tty ]; then
  SILENT=true
fi

echo "${GREEN}========================================${NC}"
echo "${GREEN}   Memoh One-Click Install${NC}"
echo "${GREEN}========================================${NC}"
echo ""

# Check Docker
if ! command -v docker >/dev/null 2>&1; then
    echo "${RED}Error: Docker is not installed${NC}"
    echo "Install Docker first: https://docs.docker.com/get-docker/"
    exit 1
fi
if ! docker compose version >/dev/null 2>&1; then
    echo "${RED}Error: Docker Compose v2 is required${NC}"
    echo "Install: https://docs.docker.com/compose/install/"
    exit 1
fi
echo "${GREEN}✓ Docker and Docker Compose detected${NC}"
echo ""

# Generate random JWT secret
gen_secret() {
  if command -v openssl >/dev/null 2>&1; then
    openssl rand -base64 32
  else
    head -c 32 /dev/urandom | base64 | tr -d '\n'
  fi
}

# Configuration defaults (expand ~ for paths)
WORKSPACE_DEFAULT="${HOME:-/tmp}/memoh"
MEMOH_DATA_DIR_DEFAULT="${HOME:-/tmp}/memoh/data"
ADMIN_USER="admin"
ADMIN_PASS="admin123"
JWT_SECRET="$(gen_secret)"
PG_PASS="memoh123"
WORKSPACE="$WORKSPACE_DEFAULT"
MEMOH_DATA_DIR="$MEMOH_DATA_DIR_DEFAULT"

# ---- Detect previous installation ----
detect_previous_install() {
  PREV_INSTALL=""
  # Check default location
  if [ -f "$WORKSPACE_DEFAULT/$DIR/docker-compose.yml" ] || [ -d "$WORKSPACE_DEFAULT/$DIR/.git" ]; then
    PREV_INSTALL="$WORKSPACE_DEFAULT/$DIR"
  fi
  # Check current directory (only if not inside the default workspace)
  if [ -z "$PREV_INSTALL" ] && [ -f "./docker-compose.yml" ] && [ -d "./.git" ]; then
    PREV_INSTALL="$(pwd)"
  fi
  # Also check ~/Memoh-v2 directly (common clone location)
  if [ -z "$PREV_INSTALL" ] && [ -f "${HOME:-/tmp}/$DIR/docker-compose.yml" ]; then
    PREV_INSTALL="${HOME:-/tmp}/$DIR"
  fi
}

cleanup_previous_install() {
  local prev_dir="$1"
  if [ -z "$prev_dir" ] || [ ! -d "$prev_dir" ]; then
    return
  fi
  echo "${YELLOW}Found previous installation at: ${prev_dir}${NC}"
  echo "${CYAN}Cleaning up...${NC}"

  # Stop running containers
  if [ -f "$prev_dir/docker-compose.yml" ] || [ -f "$prev_dir/compose.yaml" ]; then
    echo "  Stopping Docker containers..."
    (cd "$prev_dir" && docker compose down --remove-orphans 2>/dev/null) || true
  fi

  # Remove old Docker images built by this project
  echo "  Removing old Docker images..."
  docker images --format '{{.Repository}}:{{.Tag}}' | grep -E '^memoh' | while read -r img; do
    docker rmi "$img" 2>/dev/null || true
  done

  # Remove old Docker volumes (only volumes from the "memoh" compose project)
  docker volume ls --format '{{.Name}}' | grep -E '^memoh_' | while read -r vol; do
    docker volume rm "$vol" 2>/dev/null || true
  done

  # Remove old source code (but preserve data directory)
  echo "  Removing old source files..."
  # Preserve config.toml if it exists (user may have customized it)
  if [ -f "$prev_dir/config.toml" ]; then
    cp "$prev_dir/config.toml" "/tmp/memoh-config-backup.toml" 2>/dev/null || true
    echo "  ${CYAN}Backed up config.toml to /tmp/memoh-config-backup.toml${NC}"
  fi
  rm -rf "$prev_dir"

  echo "${GREEN}✓ Previous installation cleaned up${NC}"
  echo ""
}

detect_previous_install

if [ -n "$PREV_INSTALL" ]; then
  if [ "$SILENT" = true ]; then
    cleanup_previous_install "$PREV_INSTALL"
  else
    printf "${YELLOW}Previous installation found at: %s${NC}\n" "$PREV_INSTALL" > /dev/tty
    printf "Remove it and reinstall? [Y/n]: " > /dev/tty
    read -r confirm < /dev/tty || true
    case "$confirm" in
      [nN]*) echo "Keeping existing installation, will update in place." ;;
      *) cleanup_previous_install "$PREV_INSTALL" ;;
    esac
  fi
fi

if [ "$SILENT" = false ]; then
  echo "Configure Memoh (press Enter to use defaults):" > /dev/tty
  echo "" > /dev/tty

  printf "  Workspace (install and clone here) [%s]: " "~/memoh" > /dev/tty
  read -r input < /dev/tty || true
  if [ -n "$input" ]; then
    case "$input" in
      ~) WORKSPACE="${HOME:-/tmp}" ;;
      ~/*) WORKSPACE="${HOME:-/tmp}${input#\~}" ;;
      *) WORKSPACE="$input" ;;
    esac
  fi

  printf "  Data directory (bind mount for containerd/memoh data) [%s]: " "$WORKSPACE/data" > /dev/tty
  read -r input < /dev/tty || true
  if [ -n "$input" ]; then
    case "$input" in
      ~) MEMOH_DATA_DIR="${HOME:-/tmp}" ;;
      ~/*) MEMOH_DATA_DIR="${HOME:-/tmp}${input#\~}" ;;
      *) MEMOH_DATA_DIR="$input" ;;
    esac
  else
    MEMOH_DATA_DIR="$WORKSPACE/data"
  fi

  printf "  Admin username [%s]: " "$ADMIN_USER" > /dev/tty
  read -r input < /dev/tty || true
  [ -n "$input" ] && ADMIN_USER="$input"

  printf "  Admin password [%s]: " "$ADMIN_PASS" > /dev/tty
  read -r input < /dev/tty || true
  [ -n "$input" ] && ADMIN_PASS="$input"

  printf "  JWT secret [auto-generated]: " > /dev/tty
  read -r input < /dev/tty || true
  [ -n "$input" ] && JWT_SECRET="$input"

  printf "  Postgres password [%s]: " "$PG_PASS" > /dev/tty
  read -r input < /dev/tty || true
  [ -n "$input" ] && PG_PASS="$input"

  echo "" > /dev/tty
fi

# Enter workspace (all operations run here)
mkdir -p "$WORKSPACE"
cd "$WORKSPACE"

# Clone or update
if [ -d "$DIR" ]; then
    echo "Updating existing installation in $WORKSPACE..."
    cd "$DIR"
    git pull --ff-only 2>/dev/null || true
else
    echo "Cloning Memoh into $WORKSPACE..."
    git clone --depth 1 -b "$BRANCH" "$REPO" "$DIR"
    cd "$DIR"
fi

# Restore backed-up config if fresh install and backup exists
if [ ! -f config.toml ] && [ -f /tmp/memoh-config-backup.toml ]; then
  echo "${CYAN}Restoring previous config.toml from backup...${NC}"
  cp /tmp/memoh-config-backup.toml config.toml
  rm -f /tmp/memoh-config-backup.toml
else
  # Generate config.toml from template.
  # Use awk for replacements to avoid sed delimiter conflicts with
  # special characters in passwords and JWT secrets.
  cp docker/config/config.docker.toml config.toml
  awk -v au="$ADMIN_USER" -v ap="$ADMIN_PASS" -v js="$JWT_SECRET" -v pp="$PG_PASS" '
    /^\[admin\]/       { in_admin=1; in_pg=0 }
    /^\[postgres\]/    { in_pg=1; in_admin=0 }
    /^\[/              { if (!/^\[admin\]/ && !/^\[postgres\]/) { in_admin=0; in_pg=0 } }
    in_admin && /^username/ { printf "username = \"%s\"\n", au; next }
    in_admin && /^password/ { printf "password = \"%s\"\n", ap; next }
    /^jwt_secret/            { printf "jwt_secret = \"%s\"\n", js; next }
    in_pg && /^password/     { printf "password = \"%s\"\n", pp; next }
    { print }
  ' config.toml > config.toml.tmp && mv config.toml.tmp config.toml
fi

export POSTGRES_PASSWORD="${PG_PASS}"

# Use generated config and data dir
INSTALL_DIR="$(pwd)"
export MEMOH_CONFIG=./config.toml
export MEMOH_DATA_DIR
mkdir -p "$MEMOH_DATA_DIR"

echo ""
echo "${GREEN}Starting services (first build may take a few minutes)...${NC}"
docker compose up -d --build

echo ""
echo "${YELLOW}Waiting for services to become healthy (up to 120s)...${NC}"
MAX_WAIT=120
ELAPSED=0
while [ "$ELAPSED" -lt "$MAX_WAIT" ]; do
  HEALTHY=$(docker compose ps --format json 2>/dev/null | grep -c '"healthy"' || true)
  TOTAL=$(docker compose ps -q 2>/dev/null | wc -l | tr -d ' ')
  # Consider ready when at least server + web are healthy (5 out of 6 have healthchecks)
  if [ "$HEALTHY" -ge 4 ]; then
    break
  fi
  sleep 5
  ELAPSED=$((ELAPSED + 5))
  printf "  ... %ds elapsed (%s/%s healthy)\n" "$ELAPSED" "$HEALTHY" "$TOTAL"
done

if [ "$HEALTHY" -ge 4 ]; then
  echo ""
  echo "${GREEN}========================================${NC}"
  echo "${GREEN}   Memoh is running!${NC}"
  echo "${GREEN}========================================${NC}"
else
  echo ""
  echo "${YELLOW}========================================${NC}"
  echo "${YELLOW}   Memoh is starting (some services${NC}"
  echo "${YELLOW}   may still be initializing)${NC}"
  echo "${YELLOW}========================================${NC}"
  echo ""
  echo "${YELLOW}Check status: docker compose ps${NC}"
  echo "${YELLOW}Check logs:   docker compose logs server${NC}"
fi
echo ""
echo "  Web UI:          http://localhost:8082"
echo "  API:             http://localhost:8080"
echo "  Agent Gateway:   http://localhost:8081"
echo ""
echo "  Admin login:     ${ADMIN_USER} / ${ADMIN_PASS}"
echo ""
echo "Commands:"
echo "  cd ${INSTALL_DIR} && docker compose ps       # Status"
echo "  cd ${INSTALL_DIR} && docker compose logs -f   # Logs"
echo "  cd ${INSTALL_DIR} && docker compose down      # Stop"
