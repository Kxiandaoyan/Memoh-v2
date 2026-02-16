package main

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/Kxiandaoyan/Memoh-v2/internal/logger"
	"github.com/Kxiandaoyan/Memoh-v2/internal/version"
	gomcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// File tools (read/write/list/edit) are provided by the agent's MCP tool gateway, not this binary.
	server := gomcp.NewServer(
		&gomcp.Implementation{Name: "memoh-mcp", Version: version.GetInfo()},
		nil,
	)
	err := server.Run(ctx, &gomcp.StdioTransport{})
	if ctx.Err() != nil {
		return
	}
	if err == nil {
		logger.Warn("mcp server exited without error; waiting for shutdown signal")
		<-ctx.Done()
		return
	}
	if errors.Is(err, io.EOF) {
		logger.Warn("mcp stdio closed; waiting for shutdown signal")
		<-ctx.Done()
		return
	}
	logger.Error("mcp server failed", slog.Any("error", err))
	os.Exit(1)
}
