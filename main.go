package main

import (
	"context"
	"log"
	"os"
	"os/signal"

	"bethapi/api/database"
	"bethapi/api/handlers"
	"bethapi/api/middleware"
	"bethapi/api/repository"
	"bethapi/api/services"
	"bethapi/config"
	"bethapi/worker"

	"github.com/hibiken/asynq"
	"github.com/labstack/echo/v4"
	echomiddleware "github.com/labstack/echo/v4/middleware"
	"google.golang.org/genai"
)

func main() {
	// 1. Load config
	config.LoadConfig()

	// 2. Init DBs
	database.ConnectMongo()
	database.InitAsynq()
	services.InitStorage()
	services.InitEmail()

	// 3. Init Services & Repos
	userRepo := repository.NewUserRepository()
	authService := services.NewAuthService(userRepo)
	// transRepo := repository.NewTransactionRepository()
	// creditService := billing.NewCreditService(userRepo, transRepo)

	// 4. Set up Google GenAI Client
	ctx := context.Background()
	genAIClient, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  config.AppConfig.GoogleAIKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		log.Fatalf("Failed to create GenAI client: %v", err)
	}

	// 5. Start Shadow Background Worker (Conceptual)
	go startWorker(genAIClient, userRepo)

	// 6. Echo Setup
	e := echo.New()
	e.Use(echomiddleware.Logger())
	e.Use(echomiddleware.Recover())
	e.Use(echomiddleware.CORSWithConfig(echomiddleware.CORSConfig{
		AllowOrigins: config.AppConfig.AllowedOrigins,
		AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, echo.HeaderAuthorization, "X-API-KEY"},
	}))

	// Handlers
	authHandler := handlers.NewAuthHandler(authService)
	genHandler := handlers.NewGenerateHandler(database.AsynqClient)

	// Routes
	v1 := e.Group("/api/v1")

	// Auth
	v1.POST("/signup", authHandler.Signup)
	v1.POST("/login", authHandler.Login)
	v1.GET("/me", authHandler.GetMe, middleware.CombinedAuthMiddleware)

	// Video Generation
	gen := v1.Group("/generate", middleware.CombinedAuthMiddleware, middleware.CreditCheckMiddleware)
	gen.POST("", genHandler.Generate)

	// Jobs & SSE
	v1.GET("/jobs/:id/stream", genHandler.JobStream)

	// Start Server
	go func() {
		if err := e.Start(":" + config.AppConfig.Port); err != nil {
			log.Printf("Shutting down the server: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit
	log.Println("Shutting down BethAPI...")
}

func startWorker(client *genai.Client, userRepo *repository.UserRepository) {
	srv := asynq.NewServer(
		database.GetRedisOpt(),
		asynq.Config{Concurrency: 10},
	)

	videoWorker := worker.NewVideoWorker(client, userRepo)
	mux := asynq.NewServeMux()
	mux.HandleFunc(worker.TypeVideoGeneration, videoWorker.ProcessTask)

	if err := srv.Run(mux); err != nil {
		log.Fatalf("could not run asynq server: %v", err)
	}
}
