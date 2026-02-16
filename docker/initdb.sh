#!/bin/bash
set -e

# Only run .up.sql migration files in sorted order.
# This script is placed in /docker-entrypoint-initdb.d/ and called by PostgreSQL
# on first database initialization only.
#
# NOTE: If this file is not executable (+x), PostgreSQL will `source` it rather
# than run it as a subprocess. We therefore avoid bare `exit` — use `return`
# first (works when sourced) and fall back to `exit` (works when executed).

MIGRATIONS_DIR="/migrations"

if [ ! -d "$MIGRATIONS_DIR" ]; then
    echo "initdb: no migrations directory found, skipping"
    return 0 2>/dev/null || exit 0
fi

for f in $(ls "$MIGRATIONS_DIR"/*.up.sql 2>/dev/null | sort); do
    echo "initdb: running $(basename "$f")"
    psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" -f "$f"
done

echo "initdb: all migrations applied"
