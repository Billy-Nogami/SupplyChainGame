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
	"supply-chain-simulator/internal/infrastructure/system"
	httptransport "supply-chain-simulator/internal/interface/http"
	"supply-chain-simulator/internal/usecase"
)

func main() {
	cfg := config.Load()
	roomStore := memory.NewRoomStore()
	sessionStore := memory.NewSessionStore()
	decisionStore := memory.NewDecisionStore()
	scenarioRepo := memory.NewScenarioRepository()
	exporter := export.NewXLSXExporter()
	idGenerator := system.NewIDGenerator()
	clock := system.SystemClock{}

	roomService := usecase.NewRoomService(roomStore, idGenerator, clock)
	gameService := usecase.NewGameService(roomStore, sessionStore, decisionStore, scenarioRepo, exporter, idGenerator, clock)
	server := httptransport.NewServer(roomService, gameService)
	handler := httptransport.WithCORS(server.Handler(), cfg.AllowedOrigins)

	httpServer := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Printf("api listening on %s", httpServer.Addr)
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
