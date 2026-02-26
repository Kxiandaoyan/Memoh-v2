FROM golang:1.25-alpine AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .
ARG TARGETARCH
ARG COMMIT_HASH=unknown
RUN CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH:-amd64} \
    go build -trimpath -ldflags "-s -w -X github.com/Kxiandaoyan/Memoh-v2/internal/version.CommitHash=${COMMIT_HASH}" -o /out/mcp ./cmd/mcp

FROM alpine:latest

ARG TZ=UTC
ENV TZ=${TZ}

# Base utilities
RUN apk add --no-cache grep curl bash tzdata gcompat \
    && ln -snf "/usr/share/zoneinfo/${TZ}" /etc/localtime \
    && echo "${TZ}" > /etc/timezone

# Node.js + npm (provides npx for JS/TS MCP servers)
RUN apk add --no-cache nodejs npm

# Chromium + xvfb + dependencies (headless browser for agent-browser)
RUN apk add --no-cache chromium libstdc++ harfbuzz nss freetype xvfb-run

# ClawHub CLI + agent-browser + actionbook (skills + browser + web operation manuals)
RUN npm install -g clawhub agent-browser @actionbookdev/cli
ENV PLAYWRIGHT_SKIP_BROWSER_DOWNLOAD=1
ENV AGENT_BROWSER_EXECUTABLE_PATH=/usr/bin/chromium-browser

# CJK fonts (Chinese/Japanese/Korean PDF support)
RUN apk add --no-cache font-noto-cjk fontconfig && fc-cache -f

# Python 3 + pip + uv (provides uvx for Python MCP servers)
RUN apk add --no-cache python3 py3-pip && \
    curl -LsSf https://astral.sh/uv/install.sh | sh && \
    ln -sf /root/.local/bin/uv /usr/local/bin/uv && \
    ln -sf /root/.local/bin/uvx /usr/local/bin/uvx

# OpenViking context database (optional, enabled per-bot)
# Needs build toolchain: Go+make for AGFS server, gcc/g++/cmake for pybind11 C++ extensions
RUN apk add --no-cache --virtual .ov-build-deps \
        go make gcc g++ musl-dev python3-dev cmake git && \
    pip install --break-system-packages openviking && \
    apk del .ov-build-deps

WORKDIR /app
COPY --from=build /out/mcp /opt/mcp
COPY cmd/mcp/template /opt/mcp-template
ENTRYPOINT ["/bin/sh","-lc","bootstrap(){ [ -e /app/mcp ] || { mkdir -p /app; [ -f /opt/mcp ] && cp -a /opt/mcp /app/mcp 2>/dev/null || true; }; [ -d /tmp ] || mkdir -p /tmp; chmod 1777 /tmp 2>/dev/null || true; }; bootstrap; if [ -x /app/mcp ]; then exec /app/mcp \"$@\"; fi; exec /opt/mcp \"$@\"","--"]
