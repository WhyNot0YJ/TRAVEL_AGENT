package server

import (
	"github.com/gin-gonic/gin"
	"travel-agent/internal/travel"
)

func NewRouter(service *travel.TravelPlanService) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(RequestID(), Recovery(), Logger(), CORS())

	handler := travel.NewHandler(service)
	v1 := router.Group("/api/v1")
	{
		v1.POST("/travel/plans", handler.CreatePlan)
		v1.GET("/travel/plans/:task_id", handler.GetPlan)
		v1.GET("/travel/plans/:task_id/stream", handler.StreamPlan)
		v1.POST("/travel/chat", handler.Chat)
		v1.POST("/travel/chat/stream", handler.ChatStream)
	}
	return router
}
