package main

import (
	"BecomeOverMan/internal/handlers"
	"BecomeOverMan/internal/integrations"
	"log"
	"log/slog"

	_ "github.com/golang-migrate/migrate/v4/database/postgres"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"

	"BecomeOverMan/internal/config"
	_ "BecomeOverMan/internal/models"
	"BecomeOverMan/internal/repositories"
	"BecomeOverMan/internal/services"
)

func main() {
	slog.SetLogLoggerLevel(slog.LevelDebug) // Включаем DEBUG-логирование

	db, err := sqlx.Connect("postgres", config.Cfg.DatabaseURL)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	// gRPC-клиент для сервиса рекомендаций
	grpcClient, err := integrations.NewRecommendationGRPCClient(config.Cfg.GRPCRecommendationAddr)
	if err != nil {
		slog.Warn("Failed to connect to recommendation gRPC service, some features will be unavailable", "error", err)
	}
	if grpcClient != nil {
		defer grpcClient.Close()
	}

	techRepo := repositories.NewTechRepository(db)
	techService := services.NewTechService(techRepo)

	userRepo := repositories.NewUserRepository(db)
	userService := services.NewUserService(userRepo)

	questRepo := repositories.NewQuestRepository(db)
	questService := services.NewQuestService(questRepo, userRepo, grpcClient)

	r := gin.Default()
	// Настройка CORS
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		AllowCredentials: true,
	}))

	{
		handlers.RegisterTechRoutes(r, techService, grpcClient)

		handlers.RegisterUserRoutes(r, userService)
		handlers.RegisterQuestRoutes(r, questService)
	}

	if err := r.Run("0.0.0.0:8080"); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
