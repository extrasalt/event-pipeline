package main

import (
	"log"

	"github.com/extrasalt/event-pipeline/api"
)

func main() {
	store := api.NewStore()
	srv := api.NewServer(store)
	log.Println("starting API server on :8081")
	if err := srv.Run(":8081"); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
