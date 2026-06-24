package main

import (
	"log"
	"net/http"
	"os"

	"go-deployment1/src/client"

	"github.com/gin-gonic/gin"
)

type Prediction struct {
	HomeTeam  string `json:"home_team"`
	AwayTeam  string `json:"away_team"`
	HomeGoals int32  `json:"home_goals"`
	AwayGoals int32  `json:"away_goals"`
	Username  string `json:"username"`
	Timestamp string `json:"timestamp"`
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	grpcServerAddr := os.Getenv("GRPC_SERVER_ADDR")
	if grpcServerAddr == "" {
		grpcServerAddr = "go-deployment2-service.sopes1.svc.cluster.local:50051"
	}

	grpcClient := client.NewGRPCClient(grpcServerAddr)

	r := gin.Default()

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy", "service": "go-deployment1"})
	})

	r.POST("/predict", func(c *gin.Context) {
		var pred Prediction
		if err := c.ShouldBindJSON(&pred); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		log.Printf("Predicción recibida de Rust: %s vs %s | %d-%d | user: %s",
			pred.HomeTeam, pred.AwayTeam, pred.HomeGoals, pred.AwayGoals, pred.Username)

		err := grpcClient.SendPrediction(
			pred.HomeTeam,
			pred.AwayTeam,
			pred.HomeGoals,
			pred.AwayGoals,
			pred.Username,
			pred.Timestamp,
		)

		if err != nil {
			log.Printf("Error enviando a gRPC Server: %v", err)
			c.JSON(http.StatusOK, gin.H{
				"success": true,
				"message": "Predicción recibida (gRPC Server no disponible aún)",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "Predicción enviada al gRPC Server correctamente",
		})
	})

	log.Printf("Go Deployment 1 escuchando en puerto %s", port)
	log.Printf("gRPC Server addr: %s", grpcServerAddr)
	r.Run(":" + port)
}
