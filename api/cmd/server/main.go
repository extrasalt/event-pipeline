package main

import (
	"log"

	"github.com/extrasalt/event-pipeline/api"
)

func main() {
	srv := api.NewServer()
	log.Println("starting API server on :8080")
	if err := srv.Run(); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
