#!/bin/bash
# migrate-skills.sh - Sync default skills to all existing bots
#
# This script migrates the 18 default skills to all existing bots by calling
# the skills sync API endpoint for each bot. The sync uses force=false to
# preserve any bot-specific customizations.
#
# Prerequisites:
# - Backend server must be running (default: http://localhost:8080)
# - Valid authentication token (if required)
# - jq installed for JSON processing (optional, for better output)
#
# Usage:
#   ./scripts/migrate-skills.sh [OPTIONS]
#
# Options:
#   --force         Force overwrite existing skills (default: false)
#   --server URL    Backend server URL (default: http://localhost:8080)
#   --token TOKEN   Authentication token (if required)
#   --help          Show this help message
#
# Examples:
#   # Sync skills to all bots (preserve customizations)
#   ./scripts/migrate-skills.sh
#
#   # Force overwrite all skills
#   ./scripts/migrate-skills.sh --force
#
#   # Use custom server URL
#   ./scripts/migrate-skills.sh --server http://prod.example.com:8080

set -e  # Exit on error

# Script directory (absolute path)
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# Default configuration
FORCE="${FORCE:-false}"
SERVER_URL="${SERVER_URL:-http://localhost:8080}"
AUTH_TOKEN="${AUTH_TOKEN:-}"
DATA_DIR="${DATA_DIR:-$PROJECT_ROOT/data}"

# Colors for output (if terminal supports it)
if [ -t 1 ]; then
  RED='\033[0;31m'
  GREEN='\033[0;32m'
  YELLOW='\033[1;33m'
  BLUE='\033[0;34m'
  NC='\033[0m' # No Color
else
  RED=''
  GREEN=''
  YELLOW=''
  BLUE=''
  NC=''
fi

# Print colored output
log_info() {
  echo -e "${BLUE}[INFO]${NC} $*"
}

log_success() {
  echo -e "${GREEN}[SUCCESS]${NC} $*"
}

log_warn() {
  echo -e "${YELLOW}[WARN]${NC} $*"
}

log_error() {
  echo -e "${RED}[ERROR]${NC} $*"
}

# Show help message
show_help() {
  cat << EOF
migrate-skills.sh - Sync default skills to all existing bots

This script migrates the 18 default skills to all existing bots by calling
the skills sync API endpoint for each bot.

Usage:
  ./scripts/migrate-skills.sh [OPTIONS]

Options:
  --force         Force overwrite existing skills (default: false)
  --server URL    Backend server URL (default: http://localhost:8080)
  --token TOKEN   Authentication token (if required)
  --help          Show this help message

Environment Variables:
  FORCE           Same as --force flag (true/false)
  SERVER_URL      Same as --server option
  AUTH_TOKEN      Same as --token option
  DATA_DIR        Data directory path (default: ./data)

Examples:
  # Sync skills to all bots (preserve customizations)
  ./scripts/migrate-skills.sh

  # Force overwrite all skills
  ./scripts/migrate-skills.sh --force

  # Use custom server URL
  ./scripts/migrate-skills.sh --server http://prod.example.com:8080

  # With authentication token
  ./scripts/migrate-skills.sh --token "your-jwt-token-here"

EOF
}

# Parse command-line arguments
while [[ $# -gt 0 ]]; do
  case $1 in
    --force)
      FORCE="true"
      shift
      ;;
    --server)
      SERVER_URL="$2"
      shift 2
      ;;
    --token)
      AUTH_TOKEN="$2"
      shift 2
      ;;
    --help)
      show_help
      exit 0
      ;;
    *)
      log_error "Unknown option: $1"
      show_help
      exit 1
      ;;
  esac
done

# Validate configuration
log_info "Configuration:"
log_info "  Server URL: $SERVER_URL"
log_info "  Force mode: $FORCE"
log_info "  Data directory: $DATA_DIR"
if [ -n "$AUTH_TOKEN" ]; then
  log_info "  Auth token: (set)"
else
  log_info "  Auth token: (not set)"
fi
echo ""

# Check if server is running
log_info "Checking server connectivity..."
if ! curl -sf --max-time 5 "$SERVER_URL/health" > /dev/null 2>&1; then
  log_error "Backend server is not reachable at $SERVER_URL"
  log_error "Please ensure the server is running and the URL is correct"
  log_error "You can start the server with: ./agent"
  exit 1
fi
log_success "Server is reachable"
echo ""

# Find all bot directories
log_info "Discovering bots from data directory..."
BOTS_DIR="$DATA_DIR/bots"

if [ ! -d "$BOTS_DIR" ]; then
  log_error "Bots directory not found: $BOTS_DIR"
  log_error "Please ensure the data directory exists and contains bots"
  exit 1
fi

# Get list of bot IDs from directory
BOT_IDS=()
for bot_dir in "$BOTS_DIR"/*/; do
  if [ -d "$bot_dir" ]; then
    bot_id=$(basename "$bot_dir")
    # Skip hidden directories
    if [[ "$bot_id" != .* ]]; then
      BOT_IDS+=("$bot_id")
    fi
  fi
done

if [ ${#BOT_IDS[@]} -eq 0 ]; then
  log_warn "No bots found in $BOTS_DIR"
  log_warn "Nothing to migrate"
  exit 0
fi

log_success "Found ${#BOT_IDS[@]} bot(s)"
echo ""

# Prepare curl options
CURL_OPTS=(-sf --max-time 30)
if [ -n "$AUTH_TOKEN" ]; then
  CURL_OPTS+=(-H "Authorization: Bearer $AUTH_TOKEN")
fi
CURL_OPTS+=(-H "Content-Type: application/json")

# Sync skills for each bot
SUCCESS_COUNT=0
FAIL_COUNT=0
SKIP_COUNT=0

log_info "Starting skill migration..."
echo ""

for bot_id in "${BOT_IDS[@]}"; do
  log_info "Processing bot: $bot_id"

  # Check if bot has .skills directory
  skills_dir="$BOTS_DIR/$bot_id/.skills"
  if [ ! -d "$skills_dir" ]; then
    log_warn "  No .skills directory found, will be created by sync"
  fi

  # Call sync API
  API_URL="$SERVER_URL/bots/$bot_id/container/skills/sync?force=$FORCE"

  if response=$(curl "${CURL_OPTS[@]}" -X POST "$API_URL" 2>&1); then
    # Check if jq is available for prettier output
    if command -v jq &> /dev/null; then
      count=$(echo "$response" | jq -r '.count // 0')
      log_success "  Synced $count skill(s) for bot $bot_id"
    else
      log_success "  Synced skills for bot $bot_id"
    fi
    ((SUCCESS_COUNT++))
  else
    log_error "  Failed to sync skills for bot $bot_id"
    log_error "  URL: $API_URL"
    if [ -n "$response" ]; then
      log_error "  Response: $response"
    fi
    ((FAIL_COUNT++))
  fi

  echo ""
done

# Summary
echo "========================================"
log_info "Migration Summary"
echo "========================================"
log_info "Total bots: ${#BOT_IDS[@]}"
log_success "Successful: $SUCCESS_COUNT"

if [ $FAIL_COUNT -gt 0 ]; then
  log_error "Failed: $FAIL_COUNT"
fi

if [ $SKIP_COUNT -gt 0 ]; then
  log_warn "Skipped: $SKIP_COUNT"
fi

echo "========================================"

if [ $FAIL_COUNT -gt 0 ]; then
  exit 1
else
  log_success "Migration completed successfully!"
  exit 0
fi
