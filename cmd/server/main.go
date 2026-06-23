package main

import (
	"log"
	"net/http"

	"travel-agent/internal/config"
	"travel-agent/internal/server"
	"travel-agent/internal/travel"
)

func main() {
	cfg := config.Load()
	planner, err := config.BuildPlanner(cfg.Planner)
	if err != nil {
		log.Fatalf("build planner: %v", err)
	}

	store := travel.NewMemoryPlanStore()
	service := travel.NewTravelPlanService(planner, store)
	router := server.NewRouter(service)

	log.Printf("travel agent server listening on %s with planner=%s", cfg.HTTPAddr, cfg.Planner)
	if err := http.ListenAndServe(cfg.HTTPAddr, router); err != nil {
		log.Fatalf("server stopped: %v", err)
	}
}
