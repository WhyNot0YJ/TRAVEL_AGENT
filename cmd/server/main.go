package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"

	_ "github.com/go-sql-driver/mysql"

	"travel-agent/internal/agent/eino"
	"travel-agent/internal/auth"
	"travel-agent/internal/config"
	"travel-agent/internal/plans"
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

	var db *sql.DB
	if cfg.SQLEnabled && cfg.SQLDSN != "" {
		opened, err := sql.Open("mysql", cfg.SQLDSN)
		if err != nil {
			log.Printf("mysql open failed, keeping %T task store: %v", store, err)
		} else {
			opened.SetMaxOpenConns(cfg.SQLMaxOpenConns)
			opened.SetMaxIdleConns(cfg.SQLMaxIdleConns)
			opened.SetConnMaxLifetime(cfg.SQLConnMaxLifetime)
			if err := opened.PingContext(context.Background()); err != nil {
				_ = opened.Close()
				log.Printf("mysql ping failed, keeping %T task store: %v", store, err)
			} else {
				db = opened
				store = travel.NewMySQLTaskStore(db)
				log.Printf("mysql task persistence enabled max_open_conns=%d max_idle_conns=%d", cfg.SQLMaxOpenConns, cfg.SQLMaxIdleConns)
			}
		}
	} else if cfg.SQLEnabled {
		log.Printf("TRAVEL_AGENT_SQL_ENABLED=true but TRAVEL_AGENT_SQL_DSN is empty; keeping %T task store", store)
	}

	travelService := travel.NewTravelPlanService(planner, store, limiter, eino.NewTravelInfoExtractor())

	// Stage 21 — auth + plan library wiring. The dev fallback uses memory
	// stores so local laptops without MySQL still get the full UX.
	var authHandler *auth.Handler
	var plansHandler *plans.Handler
	if cfg.AuthEnabled {
		var userStore auth.UserStore = auth.NewMemoryUserStore()
		var sessionStore auth.SessionStore = auth.NewMemorySessionStore()
		var planStore plans.PlanStore = plans.NewMemoryPlanStore()
		var publicStore plans.PublicPlanStore = plans.NewMemoryPublicPlanStore()
		if db != nil {
			userStore = auth.NewMySQLUserStore(db)
			sessionStore = auth.NewMySQLSessionStore(db)
			planStore = plans.NewMySQLPlanStore(db)
			publicStore = plans.NewMySQLPublicPlanStore(db)
			log.Printf("mysql auth + plan persistence enabled")
		} else {
			log.Printf("auth + plan persistence using in-memory dev stores; data lost on restart")
		}
		authService := auth.NewService(userStore, sessionStore, auth.Config{
			PasswordMinLength: cfg.PasswordMinLength,
			SessionTTL:        cfg.SessionTTL,
		})
		authHandler = auth.NewHandler(authService, auth.CookieConfig{
			Name:   cfg.SessionCookieName,
			Domain: cfg.CookieDomain,
			Secure: cfg.CookieSecure,
		})

		plansService := plans.NewService(planStore, publicStore, &travelTaskAdapter{store: store}, &authorAdapter{users: userStore})
		plansHandler = plans.NewHandler(plansService, plans.SimpleCurrentTaskLookup(func(userID string) *plans.RunningTask {
			// Best-effort lookup — store doesn't yet support filter-by-user,
			// so we return nil and rely on the latest_plan fallback. The
			// adapter is wired so future store extensions can fill this in.
			return nil
		}))
	}

	router := server.NewRouter(server.RouterDeps{
		Travel:                       travelService,
		Auth:                         authHandler,
		Plans:                        plansHandler,
		AllowedOrigins:               cfg.AllowedOrigins,
		AllowAnonymousPlanGeneration: cfg.AllowAnonymousPlanGeneration,
	})

	log.Printf("travel agent server listening on %s with planner=%s auth_enabled=%t allow_anonymous=%t",
		cfg.HTTPAddr, cfg.Planner, cfg.AuthEnabled, cfg.AllowAnonymousPlanGeneration)
	if err := http.ListenAndServe(cfg.HTTPAddr, router); err != nil {
		log.Fatalf("server stopped: %v", err)
	}
}

// travelTaskAdapter bridges the travel TaskStore into plans.TaskLookup so the
// plans package never imports internal/travel directly.
type travelTaskAdapter struct {
	store travel.TaskStore
}

func (a *travelTaskAdapter) LookupTask(ctx context.Context, taskID string) (plans.TaskSnapshot, error) {
	task, err := a.store.Get(ctx, taskID)
	if err != nil {
		return plans.TaskSnapshot{}, err
	}
	return plans.TaskSnapshot{
		TaskID:  task.ID,
		UserID:  task.UserID,
		Status:  string(task.Status),
		Plan:    task.Plan,
		Request: task.Request,
	}, nil
}

// authorAdapter resolves a user_id into a display name without forcing the
// plans package to depend on auth.
type authorAdapter struct {
	users auth.UserStore
}

func (a *authorAdapter) DisplayName(ctx context.Context, userID string) string {
	if a == nil || a.users == nil || userID == "" {
		return ""
	}
	user, err := a.users.GetByID(ctx, userID)
	if err != nil {
		return ""
	}
	return user.DisplayName
}
