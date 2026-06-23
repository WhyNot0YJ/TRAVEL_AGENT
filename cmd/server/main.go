package main

import (
	"context"
	"log"
	"net/http"

	"travel-agent/internal/config"
	redisclient "travel-agent/internal/redis"
	"travel-agent/internal/server"
	"travel-agent/internal/travel"
)

func main() {
	cfg := config.Load()
	planner, err := config.BuildPlanner(cfg.Planner)
	if err != nil {
		log.Fatalf("build planner: %v", err)
	}

	store := travel.TaskStore(travel.NewMemoryTaskStore())
	limiter := travel.RateLimiter(travel.NewMemoryRateLimiter(cfg.RateLimitPerMinute))
	if client, err := redisclient.NewClient(context.Background(), cfg); err == nil {
		store = travel.NewRedisTaskStore(client, cfg.CacheTTL)
		limiter = travel.NewRedisRateLimiter(client, cfg.RateLimitPerMinute)
		log.Printf("redis enabled addr=%s db=%d", cfg.RedisAddr, cfg.RedisDB)
	} else {
		log.Printf("redis unavailable, using memory task store and limiter: %v", err)
	}
	service := travel.NewTravelPlanService(planner, store, limiter)
	router := server.NewRouter(service)

	log.Printf("travel agent server listening on %s with planner=%s", cfg.HTTPAddr, cfg.Planner)
	if err := http.ListenAndServe(cfg.HTTPAddr, router); err != nil {
		log.Fatalf("server stopped: %v", err)
	}
}
