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
		v1.GET("/travel/plans/:id", handler.GetPlan)
	}
	return router
}
