package main

import (
	"net/http"
	"os"

	handlers "github.com/andreka2000000/Smart_dispatching_of_field_brigades/backend/internal/api"
	storage "github.com/andreka2000000/Smart_dispatching_of_field_brigades/backend/internal/store"
	"github.com/gin-gonic/gin"
	"github.com/rs/cors"
)

func main() {
	storage.InitDB()

	r := gin.Default()

	// Ping
	r.GET("/api/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "rotelcom-dispatcher"})
	})

	// Данные
	r.POST("/api/data/service-centers", handlers.CreateServiceCenter)
	r.POST("/api/data/access-nodes", handlers.CreateAccessNode)
	r.POST("/api/data/work-types", handlers.CreateWorkType)
	r.POST("/api/data/staff", handlers.CreateStaff)
	r.POST("/api/data/orders", handlers.CreateOrders)
	r.GET("/api/data/staff", handlers.GetStaff)
	r.GET("/api/data/orders", handlers.GetOrders)

	// План
	r.POST("/api/plan/build", handlers.BuildPlan)
	r.GET("/api/plan", handlers.GetPlan)

	// События
	r.POST("/api/events", handlers.HandleEvent)

	// CORS
	c := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"*"},
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	println("🚀 Сервер на :" + port)
	http.ListenAndServe(":"+port, c.Handler(r))
}
