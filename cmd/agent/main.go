package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"
	_ "time/tzdata"

	containerd "github.com/containerd/containerd/v2/client"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"golang.org/x/crypto/bcrypt"

	"github.com/Kxiandaoyan/Memoh-v2/internal/accounts"
	"github.com/Kxiandaoyan/Memoh-v2/internal/automation"
	"github.com/Kxiandaoyan/Memoh-v2/internal/bind"
	"github.com/Kxiandaoyan/Memoh-v2/internal/boot"
	"github.com/Kxiandaoyan/Memoh-v2/internal/bots"
	"github.com/Kxiandaoyan/Memoh-v2/internal/channel"
	"github.com/Kxiandaoyan/Memoh-v2/internal/channel/adapters/feishu"
	"github.com/Kxiandaoyan/Memoh-v2/internal/channel/adapters/local"
	"github.com/Kxiandaoyan/Memoh-v2/internal/channel/adapters/telegram"
	"github.com/Kxiandaoyan/Memoh-v2/internal/channel/identities"
	"github.com/Kxiandaoyan/Memoh-v2/internal/channel/inbound"
	"github.com/Kxiandaoyan/Memoh-v2/internal/channel/route"
	"github.com/Kxiandaoyan/Memoh-v2/internal/config"
	ctr "github.com/Kxiandaoyan/Memoh-v2/internal/containerd"
	"github.com/Kxiandaoyan/Memoh-v2/internal/conversation"
	"github.com/Kxiandaoyan/Memoh-v2/internal/conversation/flow"
	"github.com/Kxiandaoyan/Memoh-v2/internal/db"
	dbsqlc "github.com/Kxiandaoyan/Memoh-v2/internal/db/sqlc"
	"github.com/Kxiandaoyan/Memoh-v2/internal/embeddings"
	"github.com/Kxiandaoyan/Memoh-v2/internal/globalsettings"
	"github.com/Kxiandaoyan/Memoh-v2/internal/handlers"
	"github.com/Kxiandaoyan/Memoh-v2/internal/logger"
	"github.com/Kxiandaoyan/Memoh-v2/internal/mcp"
	mcpcontainer "github.com/Kxiandaoyan/Memoh-v2/internal/mcp/providers/container"
	mcpdirectory "github.com/Kxiandaoyan/Memoh-v2/internal/mcp/providers/directory"
	mcphistory "github.com/Kxiandaoyan/Memoh-v2/internal/mcp/providers/history"
	mcpmemory "github.com/Kxiandaoyan/Memoh-v2/internal/mcp/providers/memory"
	mcpmessage "github.com/Kxiandaoyan/Memoh-v2/internal/mcp/providers/message"
	mcpschedule "github.com/Kxiandaoyan/Memoh-v2/internal/mcp/providers/schedule"
	mcpadmin "github.com/Kxiandaoyan/Memoh-v2/internal/mcp/providers/admin"
	mcpopenviking "github.com/Kxiandaoyan/Memoh-v2/internal/mcp/providers/openviking"
	mcpweb "github.com/Kxiandaoyan/Memoh-v2/internal/mcp/providers/web"
	mcpfederation "github.com/Kxiandaoyan/Memoh-v2/internal/mcp/sources/federation"
	"github.com/Kxiandaoyan/Memoh-v2/internal/memory"
	"github.com/Kxiandaoyan/Memoh-v2/internal/message"
	"github.com/Kxiandaoyan/Memoh-v2/internal/message/event"
	"github.com/Kxiandaoyan/Memoh-v2/internal/models"
	"github.com/Kxiandaoyan/Memoh-v2/internal/policy"
	"github.com/Kxiandaoyan/Memoh-v2/internal/preauth"
	"github.com/Kxiandaoyan/Memoh-v2/internal/processlog"
	"github.com/Kxiandaoyan/Memoh-v2/internal/providers"
	"github.com/Kxiandaoyan/Memoh-v2/internal/heartbeat"
	"github.com/Kxiandaoyan/Memoh-v2/internal/schedule"
	"github.com/Kxiandaoyan/Memoh-v2/internal/searchproviders"
	"github.com/Kxiandaoyan/Memoh-v2/internal/server"
	"github.com/Kxiandaoyan/Memoh-v2/internal/settings"
	"github.com/Kxiandaoyan/Memoh-v2/internal/subagent"
	"github.com/Kxiandaoyan/Memoh-v2/internal/templates"
	"github.com/Kxiandaoyan/Memoh-v2/internal/version"
)

func main() {
	fx.New(
		fx.Provide(
			provideConfig,
			boot.ProvideRuntimeConfig,
			provideLogger,
			provideContainerdClient,
			provideDBConn,
			provideDBQueries,

			// containerd & mcp infrastructure
			fx.Annotate(ctr.NewDefaultService, fx.As(new(ctr.Service))),
			provideMCPManager,

			// memory pipeline
			provideMemoryLLM,
			provideEmbeddingsResolver,
			provideEmbeddingSetup,
			provideTextEmbedderForMemory,
			provideQdrantStore,
			memory.NewBM25Indexer,
			provideMemoryService,

			// domain services (auto-wired)
			models.NewService,
			bots.NewService,
			accounts.NewService,
			settings.NewService,
			providers.NewService,
			searchproviders.NewService,
			policy.NewService,
			preauth.NewService,
			mcp.NewConnectionService,
			subagent.NewService,
			conversation.NewService,
			identities.NewService,
			bind.NewService,
			event.NewHub,

			// global settings (timezone, etc.)
			provideGlobalSettings,

			// services requiring provide functions
			provideRouteService,
			provideMessageService,

			// channel infrastructure
			local.NewRouteHub,
			provideChannelRegistry,
			channel.NewService,
			provideChannelRouter,
			provideChannelManager,

			// process log service
			provideProcessLogService,

			// shared cron pool for schedule + heartbeat
			provideCronPool,

			// conversation flow
			provideChatResolver,
			provideScheduleTriggerer,
			schedule.NewService,
			provideHeartbeatTriggerer,
			heartbeat.NewEngine,

			// containerd handler & tool gateway
			provideContainerdHandler,
			provideToolGatewayService,

			// http handlers (group:"server_handlers")
			provideServerHandler(handlers.NewPingHandler),
			provideServerHandler(provideAuthHandler),
			provideServerHandler(provideMemoryHandler),
			provideServerHandler(handlers.NewEmbeddingsHandler),
			provideServerHandler(provideProcessLogHandler),
			provideServerHandler(provideMessageHandler),
			provideServerHandler(handlers.NewSwaggerHandler),
			provideServerHandler(handlers.NewProvidersHandler),
			provideServerHandler(handlers.NewSearchProvidersHandler),
			provideServerHandler(handlers.NewModelsHandler),
			provideServerHandler(handlers.NewSettingsHandler),
			provideServerHandler(providePromptsHandler),
			provideServerHandler(handlers.NewPreauthHandler),
			provideServerHandler(handlers.NewBindHandler),
			provideServerHandler(handlers.NewScheduleHandler),
			provideServerHandler(handlers.NewHeartbeatHandler),
			provideServerHandler(handlers.NewSubagentHandler),
			provideServerHandler(provideSubagentRunsHandler),
			provideServerHandler(handlers.NewTokenUsageHandler),
			provideServerHandler(handlers.NewDiagnosticsHandler),
			provideServerHandler(handlers.NewGlobalSettingsHandler),
			provideServerHandler(handlers.NewChannelHandler),
			provideServerHandler(provideUsersHandler),
			provideServerHandler(handlers.NewMCPHandler),
			provideServerHandler(provideSharedFilesHandler),
			provideServerHandler(templates.NewHandler),
			provideServerHandler(provideCLIHandler),
			provideServerHandler(provideWebHandler),

			provideServer,
		),
		fx.Invoke(
			startMemoryWarmup,
			startCronPool,
			startScheduleService,
			startHeartbeatEngine,
			startChannelManager,
			startContainerReconciliation,
			startServer,
			wireTriggerSender,
		),
		fx.WithLogger(func(logger *slog.Logger) fxevent.Logger {
			return &fxevent.SlogLogger{Logger: logger.With(slog.String("component", "fx"))}
		}),
	).Run()
}

// ---------------------------------------------------------------------------
// fx helper
// ---------------------------------------------------------------------------

func provideServerHandler(fn any) any {
	return fx.Annotate(
		fn,
		fx.As(new(server.Handler)),
		fx.ResultTags(`group:"server_handlers"`),
	)
}

// ---------------------------------------------------------------------------
// infrastructure providers
// ---------------------------------------------------------------------------

func provideConfig() (config.Config, error) {
	cfgPath := os.Getenv("CONFIG_PATH")
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return config.Config{}, fmt.Errorf("load config: %w", err)
	}
	return cfg, nil
}

func provideLogger(cfg config.Config) *slog.Logger {
	logger.Init(cfg.Log.Level, cfg.Log.Format)
	return logger.L
}

func provideContainerdClient(lc fx.Lifecycle, rc *boot.RuntimeConfig) (*containerd.Client, error) {
	factory := ctr.DefaultClientFactory{SocketPath: rc.ContainerdSocketPath}
	client, err := factory.New(context.Background())
	if err != nil {
		return nil, fmt.Errorf("connect containerd: %w", err)
	}
	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			return client.Close()
		},
	})
	return client, nil
}

func provideDBConn(lc fx.Lifecycle, cfg config.Config) (*pgxpool.Pool, error) {
	conn, err := db.Open(context.Background(), cfg.Postgres)
	if err != nil {
		return nil, fmt.Errorf("db connect: %w", err)
	}
	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			conn.Close()
			return nil
		},
	})
	return conn, nil
}

func provideDBQueries(conn *pgxpool.Pool) *dbsqlc.Queries {
	return dbsqlc.New(conn)
}

func provideMCPManager(log *slog.Logger, service ctr.Service, cfg config.Config, conn *pgxpool.Pool) *mcp.Manager {
	return mcp.NewManager(log, service, cfg.MCP, cfg.Containerd.Namespace, conn)
}

// ---------------------------------------------------------------------------
// memory providers
// ---------------------------------------------------------------------------

func provideMemoryLLM(modelsService *models.Service, queries *dbsqlc.Queries, log *slog.Logger) memory.LLM {
	return &lazyLLMClient{
		modelsService: modelsService,
		queries:       queries,
		timeout:       30 * time.Second,
		logger:        log,
	}
}

func provideEmbeddingsResolver(log *slog.Logger, modelsService *models.Service, queries *dbsqlc.Queries) *embeddings.Resolver {
	return embeddings.NewResolver(log, modelsService, queries, 10*time.Second)
}

type embeddingSetup struct {
	Vectors            map[string]int
	TextModel          models.GetResponse
	MultimodalModel    models.GetResponse
	HasEmbeddingModels bool
}

func provideEmbeddingSetup(log *slog.Logger, modelsService *models.Service) (embeddingSetup, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	vectors, textModel, multimodalModel, hasEmbeddingModels, err := embeddings.CollectEmbeddingVectors(ctx, modelsService)
	if err != nil {
		return embeddingSetup{}, fmt.Errorf("embedding models: %w", err)
	}
	if hasEmbeddingModels && multimodalModel.ModelID == "" {
		log.Warn("No multimodal embedding model configured. Multimodal embedding features will be limited.")
	}
	return embeddingSetup{
		Vectors:            vectors,
		TextModel:          textModel,
		MultimodalModel:    multimodalModel,
		HasEmbeddingModels: hasEmbeddingModels,
	}, nil
}

func provideTextEmbedderForMemory(resolver *embeddings.Resolver, setup embeddingSetup, log *slog.Logger) embeddings.Embedder {
	base := buildTextEmbedder(resolver, setup.TextModel, setup.HasEmbeddingModels, log)
	if base == nil {
		return nil
	}
	return embeddings.NewCachedEmbedder(base, setup.TextModel.ModelID, 24*time.Hour, 10000)
}

func provideQdrantStore(log *slog.Logger, cfg config.Config, setup embeddingSetup) (*memory.QdrantStore, error) {
	qcfg := cfg.Qdrant
	timeout := time.Duration(qcfg.TimeoutSeconds) * time.Second
	if setup.HasEmbeddingModels && len(setup.Vectors) > 0 {
		store, err := memory.NewQdrantStoreWithVectors(log, qcfg.BaseURL, qcfg.APIKey, qcfg.Collection, setup.Vectors, "sparse_hash", timeout)
		if err != nil {
			return nil, fmt.Errorf("qdrant named vectors init: %w", err)
		}
		return store, nil
	}
	store, err := memory.NewQdrantStore(log, qcfg.BaseURL, qcfg.APIKey, qcfg.Collection, setup.TextModel.Dimensions, "sparse_hash", timeout)
	if err != nil {
		return nil, fmt.Errorf("qdrant init: %w", err)
	}
	return store, nil
}

func provideMemoryService(log *slog.Logger, llm memory.LLM, embedder embeddings.Embedder, store *memory.QdrantStore, resolver *embeddings.Resolver, bm25 *memory.BM25Indexer, setup embeddingSetup, pool *pgxpool.Pool) *memory.Service {
	svc := memory.NewService(log, llm, embedder, store, resolver, bm25, setup.TextModel.ModelID, setup.MultimodalModel.ModelID)
	if setup.HasEmbeddingModels && setup.TextModel.ModelID != "" {
		providerKey := setup.TextModel.LlmProviderID
		if providerKey == "" {
			providerKey = "default"
		}
		cache := memory.NewEmbeddingCache(pool, providerKey, setup.TextModel.ModelID, log)
		svc.SetEmbeddingCache(cache)
		log.Info("embedding cache enabled", slog.String("model", setup.TextModel.ModelID))
	}
	return svc
}

// ---------------------------------------------------------------------------
// domain service providers (interface adapters)
// ---------------------------------------------------------------------------

func provideRouteService(log *slog.Logger, queries *dbsqlc.Queries, chatService *conversation.Service) *route.DBService {
	return route.NewService(log, queries, chatService)
}

func provideMessageService(log *slog.Logger, queries *dbsqlc.Queries, hub *event.Hub) *message.DBService {
	return message.NewService(log, queries, hub)
}

func provideScheduleTriggerer(resolver *flow.Resolver) schedule.Triggerer {
	return flow.NewScheduleGateway(resolver)
}

func provideHeartbeatTriggerer(resolver *flow.Resolver) heartbeat.Triggerer {
	return flow.NewHeartbeatGateway(resolver)
}

// ---------------------------------------------------------------------------
// conversation flow
// ---------------------------------------------------------------------------

func provideProcessLogService(log *slog.Logger, queries *dbsqlc.Queries) *processlog.Service {
	return processlog.NewService(log, queries)
}

func provideSubagentRunsHandler(pool *pgxpool.Pool, log *slog.Logger) *handlers.SubagentRunsHandler {
	return handlers.NewSubagentRunsHandler(pool, log)
}

func provideChatResolver(log *slog.Logger, cfg config.Config, gs *globalsettings.Service, modelsService *models.Service, queries *dbsqlc.Queries, memoryService *memory.Service, chatService *conversation.Service, msgService *message.DBService, settingsService *settings.Service, processLogSvc *processlog.Service, containerdHandler *handlers.ContainerdHandler, manager *mcp.Manager) *flow.Resolver {
	resolver := flow.NewResolver(log, modelsService, queries, memoryService, chatService, msgService, settingsService, processLogSvc, cfg.AgentGateway.BaseURL(), 120*time.Second)
	resolver.SetSkillLoader(&skillLoaderAdapter{handler: containerdHandler})
	tz, _ := gs.GetTimezone()
	resolver.SetTimezone(tz)
	gs.OnTimezoneChange(func(tz string, _ *time.Location) {
		resolver.SetTimezone(tz)
	})
	if manager != nil {
		resolver.SetOVSessionExtractor(mcpopenviking.NewSessionExtractor(log, manager, queries))
		resolver.SetOVContextLoader(mcpopenviking.NewContextLoader(log, manager, queries))
	}
	return resolver
}

// ---------------------------------------------------------------------------
// channel providers
// ---------------------------------------------------------------------------

func provideChannelRegistry(log *slog.Logger, hub *local.RouteHub) *channel.Registry {
	registry := channel.NewRegistry()
	registry.MustRegister(telegram.NewTelegramAdapter(log))
	registry.MustRegister(feishu.NewFeishuAdapter(log))
	registry.MustRegister(local.NewCLIAdapter(hub))
	registry.MustRegister(local.NewWebAdapter(hub))
	return registry
}

func provideChannelRouter(log *slog.Logger, registry *channel.Registry, routeService *route.DBService, msgService *message.DBService, resolver *flow.Resolver, identityService *identities.Service, botService *bots.Service, policyService *policy.Service, preauthService *preauth.Service, bindService *bind.Service, rc *boot.RuntimeConfig) *inbound.ChannelInboundProcessor {
	proc := inbound.NewChannelInboundProcessor(log, registry, routeService, msgService, resolver, identityService, botService, policyService, preauthService, bindService, rc.JwtSecret, 5*time.Minute)
	proc.SetGroupDebouncer(message.NewGroupDebouncer(3 * time.Second))
	return proc
}

func provideChannelManager(log *slog.Logger, registry *channel.Registry, channelService *channel.Service, channelRouter *inbound.ChannelInboundProcessor) *channel.Manager {
	mgr := channel.NewManager(log, registry, channelService, channelRouter)
	if mw := channelRouter.IdentityMiddleware(); mw != nil {
		mgr.Use(mw)
	}
	return mgr
}

// wireTriggerSender connects channel.Manager to the Resolver as a fallback
// message sender for schedule/heartbeat triggers.
// channelManager depends on channelRouter which depends on resolver, so this
// wiring must happen via fx.Invoke (post-construction) rather than in the
// resolver constructor to avoid a circular dependency.
func wireTriggerSender(resolver *flow.Resolver, channelManager *channel.Manager) {
	resolver.SetTriggerSender(&channelTriggerSender{manager: channelManager})
}

// channelTriggerSender implements flow.TriggerMessageSender using channel.Manager.
type channelTriggerSender struct {
	manager *channel.Manager
}

func (s *channelTriggerSender) SendText(ctx context.Context, botID, platform, target, text string) error {
	ct := channel.ChannelType(strings.ToLower(strings.TrimSpace(platform)))
	return s.manager.Send(ctx, botID, ct, channel.SendRequest{
		Target:  target,
		Message: channel.Message{Text: text},
	})
}

// ---------------------------------------------------------------------------
// containerd handler & tool gateway
// ---------------------------------------------------------------------------

func provideContainerdHandler(log *slog.Logger, service ctr.Service, cfg config.Config, botService *bots.Service, accountService *accounts.Service, policyService *policy.Service, queries *dbsqlc.Queries) *handlers.ContainerdHandler {
	return handlers.NewContainerdHandler(log, service, cfg.MCP, cfg.Containerd.Namespace, botService, accountService, policyService, queries)
}

func provideToolGatewayService(log *slog.Logger, cfg config.Config, channelManager *channel.Manager, registry *channel.Registry, channelService *channel.Service, scheduleService *schedule.Service, memoryService *memory.Service, chatService *conversation.Service, accountService *accounts.Service, settingsService *settings.Service, searchProviderService *searchproviders.Service, manager *mcp.Manager, containerdHandler *handlers.ContainerdHandler, mcpConnService *mcp.ConnectionService, botService *bots.Service, modelService *models.Service, providerService *providers.Service, msgService *message.DBService, queries *dbsqlc.Queries) *mcp.ToolGatewayService {
	messageExec := mcpmessage.NewExecutor(log, channelManager, channelManager, registry)
	directoryExec := mcpdirectory.NewExecutor(log, registry, channelService, registry)
	scheduleExec := mcpschedule.NewExecutor(log, scheduleService)
	memoryExec := mcpmemory.NewExecutor(log, memoryService, chatService, accountService)
	webExec := mcpweb.NewExecutor(log, settingsService, searchProviderService)
	historyExec := mcphistory.NewExecutor(log, msgService)
	execWorkDir := cfg.MCP.DataMount
	if strings.TrimSpace(execWorkDir) == "" {
		execWorkDir = config.DefaultDataMount
	}
	fsExec := mcpcontainer.NewExecutor(log, manager, execWorkDir)

	adminInner := mcpadmin.NewExecutor(log, botService, modelService, providerService)
	adminExec := mcpadmin.NewConditionalExecutor(log, adminInner, queries)

	ovExec := mcpopenviking.NewExecutor(log, manager, queries)

	fedGateway := handlers.NewMCPFederationGateway(log, containerdHandler)
	fedSource := mcpfederation.NewSource(log, fedGateway, mcpConnService)

	svc := mcp.NewToolGatewayService(
		log,
		[]mcp.ToolExecutor{messageExec, directoryExec, scheduleExec, memoryExec, webExec, fsExec, adminExec, ovExec, historyExec},
		[]mcp.ToolSource{fedSource},
	)
	containerdHandler.SetToolGatewayService(svc)
	return svc
}

// ---------------------------------------------------------------------------
// handler providers (interface adaptation / config extraction)
// ---------------------------------------------------------------------------

func providePromptsHandler(log *slog.Logger, botService *bots.Service, accountService *accounts.Service, modelsService *models.Service, queries *dbsqlc.Queries, cfg config.Config, manager *mcp.Manager) *handlers.PromptsHandler {
	h := handlers.NewPromptsHandler(log, botService, accountService, modelsService, queries, cfg)
	if manager != nil {
		ovExec := mcpopenviking.NewExecutor(log, manager, queries)
		h.SetOVInitializer(ovExec)
	}
	return h
}

func provideMemoryHandler(log *slog.Logger, service *memory.Service, chatService *conversation.Service, accountService *accounts.Service, cfg config.Config, manager *mcp.Manager) *handlers.MemoryHandler {
	h := handlers.NewMemoryHandler(log, service, chatService, accountService)
	if manager != nil {
		execWorkDir := cfg.MCP.DataMount
		if strings.TrimSpace(execWorkDir) == "" {
			execWorkDir = config.DefaultDataMount
		}
		h.SetMemoryFS(memory.NewMemoryFS(log, manager, execWorkDir))
	}
	return h
}

func provideProcessLogHandler(log *slog.Logger, botService *bots.Service, processLogSvc *processlog.Service) *handlers.ProcessLogHandler {
	return handlers.NewProcessLogHandler(botService, processLogSvc, log)
}

func provideAuthHandler(log *slog.Logger, accountService *accounts.Service, rc *boot.RuntimeConfig) *handlers.AuthHandler {
	return handlers.NewAuthHandler(log, accountService, rc.JwtSecret, rc.JwtExpiresIn)
}

func provideMessageHandler(log *slog.Logger, resolver *flow.Resolver, chatService *conversation.Service, msgService *message.DBService, botService *bots.Service, accountService *accounts.Service, identityService *identities.Service, hub *event.Hub) *handlers.MessageHandler {
	return handlers.NewMessageHandler(log, resolver, chatService, msgService, botService, accountService, identityService, hub)
}

func provideUsersHandler(log *slog.Logger, accountService *accounts.Service, identityService *identities.Service, botService *bots.Service, routeService *route.DBService, channelService *channel.Service, channelManager *channel.Manager, registry *channel.Registry, heartbeatEngine *heartbeat.Engine) *handlers.UsersHandler {
	return handlers.NewUsersHandler(log, accountService, identityService, botService, routeService, channelService, channelManager, registry, heartbeatEngine)
}

func provideSharedFilesHandler(cfg config.Config) *handlers.SharedFilesHandler {
	return handlers.NewSharedFilesHandler(cfg.MCP)
}

func provideCLIHandler(channelManager *channel.Manager, channelService *channel.Service, chatService *conversation.Service, hub *local.RouteHub, botService *bots.Service, accountService *accounts.Service) *handlers.LocalChannelHandler {
	return handlers.NewLocalChannelHandler(local.CLIType, channelManager, channelService, chatService, hub, botService, accountService)
}

func provideWebHandler(channelManager *channel.Manager, channelService *channel.Service, chatService *conversation.Service, hub *local.RouteHub, botService *bots.Service, accountService *accounts.Service) *handlers.LocalChannelHandler {
	return handlers.NewLocalChannelHandler(local.WebType, channelManager, channelService, chatService, hub, botService, accountService)
}

// ---------------------------------------------------------------------------
// server
// ---------------------------------------------------------------------------

type serverParams struct {
	fx.In

	Logger            *slog.Logger
	RuntimeConfig     *boot.RuntimeConfig
	Config            config.Config
	ServerHandlers    []server.Handler `group:"server_handlers"`
	ContainerdHandler *handlers.ContainerdHandler
}

func provideServer(params serverParams) *server.Server {
	allHandlers := make([]server.Handler, 0, len(params.ServerHandlers)+1)
	allHandlers = append(allHandlers, params.ServerHandlers...)
	allHandlers = append(allHandlers, params.ContainerdHandler)
	return server.NewServer(params.Logger, params.RuntimeConfig.ServerAddr, params.Config.Auth.JWTSecret, allHandlers...)
}

// ---------------------------------------------------------------------------
// lifecycle hooks
// ---------------------------------------------------------------------------

func startMemoryWarmup(lc fx.Lifecycle, memoryService *memory.Service, logger *slog.Logger) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			go func() {
				if err := memoryService.WarmupBM25(context.Background(), 200); err != nil {
					logger.Warn("bm25 warmup failed", slog.Any("error", err))
				}
			}()
			return nil
		},
	})
}

func provideGlobalSettings(log *slog.Logger, queries *dbsqlc.Queries, cfg config.Config) *globalsettings.Service {
	svc := globalsettings.NewService(log, queries, cfg)
	if err := svc.Init(context.Background()); err != nil {
		log.Warn("global settings init failed, using defaults", slog.Any("error", err))
	}
	return svc
}

func provideCronPool(log *slog.Logger, gs *globalsettings.Service) *automation.CronPool {
	_, loc := gs.GetTimezone()
	pool := automation.NewCronPool(log, loc)
	gs.OnTimezoneChange(func(_ string, loc *time.Location) {
		pool.SetLocation(loc)
	})
	return pool
}

func startCronPool(lc fx.Lifecycle, pool *automation.CronPool) {
	lc.Append(fx.Hook{
		OnStart: func(_ context.Context) error {
			pool.Start()
			return nil
		},
		OnStop: func(_ context.Context) error {
			<-pool.Stop().Done()
			return nil
		},
	})
}

func startScheduleService(lc fx.Lifecycle, scheduleService *schedule.Service) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			return scheduleService.Bootstrap(ctx)
		},
	})
}

func startHeartbeatEngine(lc fx.Lifecycle, engine *heartbeat.Engine, pool *pgxpool.Pool, gs *globalsettings.Service) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			engine.SetPool(pool)
			if _, loc := gs.GetTimezone(); loc != nil {
				engine.SetTimezone(loc)
			}
			return engine.Bootstrap(ctx)
		},
		OnStop: func(_ context.Context) error {
			engine.Stop()
			return nil
		},
	})
}

func startChannelManager(lc fx.Lifecycle, channelManager *channel.Manager) {
	ctx, cancel := context.WithCancel(context.Background())
	lc.Append(fx.Hook{
		OnStart: func(_ context.Context) error {
			channelManager.Start(ctx)
			return nil
		},
		OnStop: func(stopCtx context.Context) error {
			cancel()
			return channelManager.Shutdown(stopCtx)
		},
	})
}

func startContainerReconciliation(lc fx.Lifecycle, containerdHandler *handlers.ContainerdHandler, _ *mcp.ToolGatewayService) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			go containerdHandler.ReconcileContainers(ctx)
			return nil
		},
	})
}

func startServer(lc fx.Lifecycle, logger *slog.Logger, srv *server.Server, shutdowner fx.Shutdowner, cfg config.Config, queries *dbsqlc.Queries, botService *bots.Service, containerdHandler *handlers.ContainerdHandler, mcpConnService *mcp.ConnectionService, toolGateway *mcp.ToolGatewayService, heartbeatEngine *heartbeat.Engine, memoryService *memory.Service) {
	fmt.Printf("Starting Memoh Agent %s\n", version.GetInfo())

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			if err := ensureAdminUser(ctx, logger, queries, cfg); err != nil {
				return err
			}
			botService.SetContainerLifecycle(containerdHandler)
			botService.SetHeartbeatSeeder(heartbeatEngine)
			heartbeatEngine.SetMemoryCompactor(memoryService)
			botService.AddRuntimeChecker(mcp.NewConnectionChecker(logger, mcpConnService, toolGateway))

			go func() {
				if err := srv.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
					logger.Error("server failed", slog.Any("error", err))
					_ = shutdowner.Shutdown()
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			if err := srv.Stop(ctx); err != nil && !errors.Is(err, http.ErrServerClosed) {
				return fmt.Errorf("server stop: %w", err)
			}
			return nil
		},
	})
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func buildTextEmbedder(resolver *embeddings.Resolver, textModel models.GetResponse, hasModels bool, log *slog.Logger) embeddings.Embedder {
	if !hasModels {
		return nil
	}
	if textModel.ModelID == "" || textModel.Dimensions <= 0 {
		log.Warn("No text embedding model configured. Text embedding features will be limited.")
		return nil
	}
	return &embeddings.ResolverTextEmbedder{
		Resolver: resolver,
		ModelID:  textModel.ModelID,
		Dims:     textModel.Dimensions,
	}
}

func ensureAdminUser(ctx context.Context, log *slog.Logger, queries *dbsqlc.Queries, cfg config.Config) error {
	if queries == nil {
		return fmt.Errorf("db queries not configured")
	}
	count, err := queries.CountAccounts(ctx)
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	username := strings.TrimSpace(cfg.Admin.Username)
	password := strings.TrimSpace(cfg.Admin.Password)
	email := strings.TrimSpace(cfg.Admin.Email)
	if username == "" || password == "" {
		return fmt.Errorf("admin username/password required in config.toml")
	}
	if password == "change-your-password-here" {
		log.Warn("admin password uses default placeholder; please update config.toml")
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	user, err := queries.CreateUser(ctx, dbsqlc.CreateUserParams{
		IsActive: true,
		Metadata: []byte("{}"),
	})
	if err != nil {
		return fmt.Errorf("create admin user: %w", err)
	}

	emailValue := pgtype.Text{Valid: false}
	if email != "" {
		emailValue = pgtype.Text{String: email, Valid: true}
	}
	displayName := pgtype.Text{String: username, Valid: true}
	dataRoot := pgtype.Text{String: cfg.MCP.DataRoot, Valid: cfg.MCP.DataRoot != ""}

	_, err = queries.CreateAccount(ctx, dbsqlc.CreateAccountParams{
		UserID:       user.ID,
		Username:     pgtype.Text{String: username, Valid: true},
		Email:        emailValue,
		PasswordHash: pgtype.Text{String: string(hashed), Valid: true},
		Role:         "admin",
		DisplayName:  displayName,
		AvatarUrl:    pgtype.Text{Valid: false},
		IsActive:     true,
		DataRoot:     dataRoot,
	})
	if err != nil {
		return err
	}
	log.Info("Admin user created", slog.String("username", username))
	return nil
}

// ---------------------------------------------------------------------------
// lazy LLM client
// ---------------------------------------------------------------------------

type lazyLLMClient struct {
	modelsService *models.Service
	queries       *dbsqlc.Queries
	timeout       time.Duration
	logger        *slog.Logger
}

func (c *lazyLLMClient) Extract(ctx context.Context, req memory.ExtractRequest) (memory.ExtractResponse, error) {
	client, err := c.resolve(ctx)
	if err != nil {
		return memory.ExtractResponse{}, err
	}
	return client.Extract(ctx, req)
}

func (c *lazyLLMClient) Decide(ctx context.Context, req memory.DecideRequest) (memory.DecideResponse, error) {
	client, err := c.resolve(ctx)
	if err != nil {
		return memory.DecideResponse{}, err
	}
	return client.Decide(ctx, req)
}

func (c *lazyLLMClient) Compact(ctx context.Context, req memory.CompactRequest) (memory.CompactResponse, error) {
	client, err := c.resolve(ctx)
	if err != nil {
		return memory.CompactResponse{}, err
	}
	return client.Compact(ctx, req)
}

func (c *lazyLLMClient) DetectLanguage(ctx context.Context, text string) (string, error) {
	client, err := c.resolve(ctx)
	if err != nil {
		return "", err
	}
	return client.DetectLanguage(ctx, text)
}

func (c *lazyLLMClient) resolve(ctx context.Context) (memory.LLM, error) {
	if c.modelsService == nil || c.queries == nil {
		return nil, fmt.Errorf("models service not configured")
	}
	preferred := memory.PreferredModelFromCtx(ctx)
	memoryModel, memoryProvider, err := models.SelectMemoryModel(ctx, c.modelsService, c.queries, preferred)
	if err != nil {
		return nil, err
	}
	clientType := strings.ToLower(strings.TrimSpace(memoryProvider.ClientType))
	switch clientType {
	case "anthropic", "google", "bedrock":
		return nil, fmt.Errorf("memory provider client type %q does not support OpenAI-compatible /chat/completions", memoryProvider.ClientType)
	default:
		// Most providers (openai, openai-compat, azure, deepseek, zai-*, minimax-*,
		// moonshot-*, volcengine*, dashscope, qianfan, groq, ollama, openrouter,
		// together, fireworks, perplexity, xai, mistral, etc.) use the
		// OpenAI-compatible /chat/completions endpoint.
	}
	return memory.NewLLMClient(c.logger, memoryProvider.BaseUrl, memoryProvider.ApiKey, memoryModel.ModelID, c.timeout)
}

// skillLoaderAdapter bridges handlers.ContainerdHandler to flow.SkillLoader.
type skillLoaderAdapter struct {
	handler *handlers.ContainerdHandler
}

func (a *skillLoaderAdapter) LoadSkills(ctx context.Context, botID string) ([]flow.SkillEntry, error) {
	items, err := a.handler.LoadSkills(ctx, botID)
	if err != nil {
		return nil, err
	}
	entries := make([]flow.SkillEntry, len(items))
	for i, item := range items {
		entries[i] = flow.SkillEntry{
			Name:        item.Name,
			Description: item.Description,
			Content:     item.Content,
			Metadata:    item.Metadata,
		}
	}
	return entries, nil
}
