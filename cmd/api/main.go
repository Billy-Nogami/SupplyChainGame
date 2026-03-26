package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"supply-chain-simulator/internal/infrastructure/config"
	"supply-chain-simulator/internal/infrastructure/export"
	"supply-chain-simulator/internal/infrastructure/memory"
	redisinfra "supply-chain-simulator/internal/infrastructure/redis"
	"supply-chain-simulator/internal/infrastructure/system"
	httptransport "supply-chain-simulator/internal/interface/http"
	"supply-chain-simulator/internal/usecase"
)

func main() {
	cfg := config.Load()
	scenarioRepo := memory.NewScenarioRepository()
	exporter := export.NewXLSXExporter()
	idGenerator := system.NewIDGenerator()
	clock := system.SystemClock{}

	roomStore, sessionStore, decisionStore, eventPublisher, eventSubscriber, closeInfra := buildInfrastructure(cfg)
	defer closeInfra()

	roomService := usecase.NewRoomService(roomStore, idGenerator, clock, eventPublisher)
	gameService := usecase.NewGameService(roomStore, sessionStore, decisionStore, scenarioRepo, exporter, eventPublisher, idGenerator, clock)
	server := httptransport.NewServer(roomService, gameService, eventSubscriber)
	handler := httptransport.WithRecovery(
		httptransport.WithRequestLogging(
			httptransport.WithCORS(server.Handler(), cfg.AllowedOrigins),
		),
	)

	httpServer := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Printf("api_listening addr=%s redis_enabled=%t cors_origins=%d", httpServer.Addr, cfg.Redis.Enabled, len(cfg.AllowedOrigins))
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		log.Printf("shutdown error: %v", err)
	}
}

func buildInfrastructure(cfg config.App) (
	usecase.RoomStore,
	usecase.SessionStore,
	usecase.DecisionStore,
	usecase.RoomEventPublisher,
	httptransport.RoomEventSubscriber,
	func(),
) {
	if cfg.Redis.Enabled {
		log.Printf("storage_mode=redis addr=%s db=%d ttl=%s", cfg.Redis.Addr, cfg.Redis.DB, cfg.Redis.KeyTTL)
		client := redisinfra.NewClient(redisinfra.Config{
			Addr:     cfg.Redis.Addr,
			Password: cfg.Redis.Password,
			DB:       cfg.Redis.DB,
		})
		if err := redisinfra.Ping(context.Background(), client); err != nil {
			log.Fatalf("redis ping failed: %v", err)
		}

		roomStore := redisinfra.NewRoomStore(client, cfg.Redis.KeyTTL)
		sessionStore := redisinfra.NewSessionStore(client, cfg.Redis.KeyTTL)
		decisionStore := redisinfra.NewDecisionStore(client, cfg.Redis.KeyTTL)
		eventBus := redisinfra.NewRoomEventBus(client)

		return roomStore, sessionStore, decisionStore, eventBus, eventBus, func() {
			_ = client.Close()
		}
	}

	log.Printf("storage_mode=memory")
	roomStore := memory.NewRoomStore()
	sessionStore := memory.NewSessionStore()
	decisionStore := memory.NewDecisionStore()
	eventBus := memory.NewRoomEventBus()

	return roomStore, sessionStore, decisionStore, eventBus, eventBus, func() {}
}
