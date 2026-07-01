package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/extrasalt/event-pipeline/api"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := api.ConfigFromEnv()

	store, err := api.NewStore(ctx, cfg)
	if err != nil {
		log.Fatalf("store init: %v", err)
	}

	srv := api.NewServer(store)
	httpServer := &http.Server{Addr: ":8081", Handler: srv}

	go func() {
		log.Println("starting API server on :8081")
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	log.Printf("received signal %s, shutting down...", sig)

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("server shutdown error: %v", err)
	}

	cancel()
	store.Close()
	log.Println("server stopped")
}
