package main

import (
	"flag"
	"log"

	"github.com/extrasalt/event-pipeline/api"
	"github.com/extrasalt/event-pipeline/pipeline"
)

func main() {
	mode := flag.String("m", "api", "run mode: api or pipeline")
	flag.Parse()

	switch *mode {
	case "api":
		srv := api.NewServer()
		log.Println("starting API server on :8080")
		if err := srv.Run(); err != nil {
			log.Fatalf("server error: %v", err)
		}
	case "pipeline":
		pipeline.Run()
	default:
		log.Fatalf("unknown mode: %s", *mode)
	}
}
