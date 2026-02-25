#!/bin/sh

MCP_IMAGE="${MCP_IMAGE:-docker.io/library/memoh-mcp:latest}"

# Start containerd in background
mkdir -p /run/containerd
containerd &
CONTAINERD_PID=$!

# Wait for containerd to be fully responsive
echo "Waiting for containerd..."
for i in $(seq 1 30); do
  if ctr version >/dev/null 2>&1; then
    break
  fi
  sleep 1
done

if ! ctr version >/dev/null 2>&1; then
  echo "ERROR: containerd not responsive after 30s"
  exit 1
fi
echo "containerd is running"

# Always import bundled MCP image tar so startup picks up image updates
echo "Importing MCP image into containerd..."
for tar in /opt/images/*.tar; do
  if [ -f "$tar" ]; then
    ctr -n default images import --all-platforms "$tar" 2>&1 || true
  fi
done

# Verify image availability after import.
# For this ctr version, explicit "images unpack" is not available; snapshot
# preparation is handled when creating containers from the image.
if ctr -n default images check "name==${MCP_IMAGE}" 2>/dev/null | grep -q "${MCP_IMAGE}"; then
  echo "MCP image ready: ${MCP_IMAGE}"
else
  echo "WARNING: MCP image not available after import, will try pull at runtime"
fi

echo "containerd is ready"
wait $CONTAINERD_PID
