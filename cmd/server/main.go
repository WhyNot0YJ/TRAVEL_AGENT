package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"

	_ "github.com/go-sql-driver/mysql"

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
	if cfg.SQLEnabled && cfg.SQLDSN != "" {
		db, err := sql.Open("mysql", cfg.SQLDSN)
		if err != nil {
			log.Printf("mysql open failed, keeping %T task store: %v", store, err)
		} else {
			db.SetMaxOpenConns(cfg.SQLMaxOpenConns)
			db.SetMaxIdleConns(cfg.SQLMaxIdleConns)
			db.SetConnMaxLifetime(cfg.SQLConnMaxLifetime)
			if err := db.PingContext(context.Background()); err != nil {
				_ = db.Close()
				log.Printf("mysql ping failed, keeping %T task store: %v", store, err)
			} else {
				store = travel.NewMySQLTaskStore(db)
				log.Printf("mysql task persistence enabled max_open_conns=%d max_idle_conns=%d", cfg.SQLMaxOpenConns, cfg.SQLMaxIdleConns)
			}
		}
	} else if cfg.SQLEnabled {
		log.Printf("TRAVEL_AGENT_SQL_ENABLED=true but TRAVEL_AGENT_SQL_DSN is empty; keeping %T task store", store)
	}
	service := travel.NewTravelPlanService(planner, store, limiter)
	router := server.NewRouter(service)

	log.Printf("travel agent server listening on %s with planner=%s", cfg.HTTPAddr, cfg.Planner)
	if err := http.ListenAndServe(cfg.HTTPAddr, router); err != nil {
		log.Fatalf("server stopped: %v", err)
	}
}
