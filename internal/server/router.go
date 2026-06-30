package server

import (
	"github.com/gin-gonic/gin"

	"travel-agent/internal/auth"
	"travel-agent/internal/plans"
	"travel-agent/internal/travel"
)

// RouterDeps bundles every wired-in dependency. main.go assembles concrete
// stores and services, then hands them in.
type RouterDeps struct {
	Travel                       *travel.TravelPlanService
	Auth                         *auth.Handler
	Plans                        *plans.Handler
	AllowedOrigins               []string
	AllowAnonymousPlanGeneration bool
}

func NewRouter(deps RouterDeps) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(RequestID(), Recovery(), Logger(), CORS(deps.AllowedOrigins))

	travelHandler := travel.NewHandler(deps.Travel)
	v1 := router.Group("/api/v1")

	// Auth endpoints
	if deps.Auth != nil {
		v1.POST("/auth/register", deps.Auth.Register)
		v1.POST("/auth/login", deps.Auth.Login)
		v1.POST("/auth/logout", deps.Auth.Logout)
		v1.GET("/auth/me", deps.Auth.Optional(), deps.Auth.Me)
	}

	// Travel generation. Optional auth lets us tag tasks with user_id when
	// signed in. When AllowAnonymousPlanGeneration is false the handler
	// rejects unauthenticated callers; the wiring layer enforces that
	// invariant by inspecting the context.
	travelGroup := v1.Group("")
	if deps.Auth != nil {
		travelGroup.Use(deps.Auth.Optional())
	}
	if !deps.AllowAnonymousPlanGeneration && deps.Auth != nil {
		travelGroup.Use(deps.Auth.RequireAuth())
	}
	travelGroup.POST("/travel/plans", travelHandler.CreatePlan)
	travelGroup.GET("/travel/plans/:task_id", travelHandler.GetPlan)
	travelGroup.GET("/travel/plans/:task_id/stream", travelHandler.StreamPlan)
	travelGroup.POST("/travel/chat", travelHandler.Chat)
	travelGroup.POST("/travel/chat/stream", travelHandler.ChatStream)

	// Private plan library
	if deps.Auth != nil && deps.Plans != nil {
		me := v1.Group("/me", deps.Auth.RequireAuth())
		me.POST("/plans", deps.Plans.Save)
		me.GET("/plans", deps.Plans.List)
		me.GET("/plans/:plan_id", deps.Plans.Get)
		me.PATCH("/plans/:plan_id", deps.Plans.Patch)
		me.DELETE("/plans/:plan_id", deps.Plans.Delete)
		me.GET("/plans/:plan_id/conversation", deps.Plans.GetConversation)
		me.POST("/plans/:plan_id/publish", deps.Plans.Publish)
		me.POST("/plans/:plan_id/unpublish", deps.Plans.Unpublish)
		me.GET("/current", deps.Plans.Current)
	}

	// Public plan discovery
	if deps.Plans != nil {
		pub := v1.Group("/public")
		if deps.Auth != nil {
			pub.Use(deps.Auth.Optional())
		}
		pub.GET("/plans", deps.Plans.ListPublic)
		pub.GET("/plans/:public_plan_id", deps.Plans.GetPublic)
		pubAuthed := v1.Group("/public")
		if deps.Auth != nil {
			pubAuthed.Use(deps.Auth.RequireAuth())
		}
		pubAuthed.POST("/plans/:public_plan_id/save", deps.Plans.SavePublicAsCopy)
	}

	return router
}
