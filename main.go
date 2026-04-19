package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"time"

	"bethapi/api/database"
	"bethapi/api/handlers"
	"bethapi/api/middleware"
	"bethapi/api/repository"
	"bethapi/api/services"
	"bethapi/billing"
	"bethapi/config"
	"bethapi/worker"

	"github.com/hibiken/asynq"
	"github.com/labstack/echo/v4"
	echomiddleware "github.com/labstack/echo/v4/middleware"
	"github.com/robfig/cron/v3"
	"github.com/ulule/limiter/v3"
	echo_limiter "github.com/ulule/limiter/v3/drivers/middleware/echo"
	limiter_redis "github.com/ulule/limiter/v3/drivers/store/redis"
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
	jobRepo := repository.NewJobRepository()
	transRepo := repository.NewTransactionRepository()
	
	creditService := billing.NewCreditService(userRepo, transRepo, jobRepo)
	paymentService := billing.NewPaymentService(userRepo, creditService)
	authService := services.NewAuthService(userRepo, database.RedisClient)
	videoService := services.NewVideoService(services.Storage)

	// 4. Set up Google GenAI Client
	ctx := context.Background()
	var genAIClient *genai.Client
	var genAIErr error

	if config.AppConfig.GoogleProjectID != "" {
		genAIClient, genAIErr = genai.NewClient(ctx, &genai.ClientConfig{
			Project:  config.AppConfig.GoogleProjectID,
			Location: config.AppConfig.GoogleLocation,
			Backend:  genai.BackendVertexAI,
		})
	} else {
		genAIClient, genAIErr = genai.NewClient(ctx, &genai.ClientConfig{
			APIKey:  config.AppConfig.GoogleAIKey,
			Backend: genai.BackendGeminiAPI,
		})
	}

	if genAIErr != nil {
		log.Fatalf("Failed to create GenAI client: %v", genAIErr)
	}

	// 5. Start Background Workers
	go startWorker(genAIClient, userRepo, jobRepo, videoService, paymentService, creditService)
	startBillingCron(userRepo, paymentService)
	startPollerCron(genAIClient, jobRepo, creditService)

	// 6. Echo Setup
	e := echo.New()
	
	// Global Rate Limiting (Redis-backed)
	rateStore, _ := limiter_redis.NewStore(database.RedisClient)
	rateLimit := limiter.Rate{Period: 1 * time.Minute, Limit: 60}
	rateInstance := limiter.New(rateStore, rateLimit)
	e.Use(echo_limiter.NewMiddleware(rateInstance))

	e.Use(echomiddleware.RequestID())
	e.Use(echomiddleware.LoggerWithConfig(echomiddleware.LoggerConfig{
		Format: `{"time":"${time_rfc3339_nano}","id":"${id}","remote_ip":"${remote_ip}","host":"${host}",` +
			`"method":"${method}","uri":"${uri}","status":${status},"error":"${error}",` +
			`"latency":${latency},"latency_human":"${latency_human}","bytes_in":${bytes_in},` +
			`"bytes_out":${bytes_out}}` + "\n",
	}))
	e.Use(echomiddleware.Recover())
	e.Use(echomiddleware.CORSWithConfig(echomiddleware.CORSConfig{
		AllowOrigins:     config.AppConfig.AllowedOrigins,
		AllowHeaders:     []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, echo.HeaderAuthorization, "X-API-KEY"},
		AllowCredentials: true,
	}))

	// Handlers
	authHandler := handlers.NewAuthHandler(authService)
	genHandler := handlers.NewGenerateHandler(database.AsynqClient, jobRepo)
	billingHandler := handlers.NewBillingHandler(paymentService, userRepo)

	// Routes
	v1 := e.Group("/api/v1")

	// Auth
	auth := v1.Group("/auth")
	auth.POST("/signup", authHandler.Signup)
	auth.POST("/login", authHandler.Login)
	auth.GET("/me", authHandler.GetMe, middleware.CombinedAuthMiddleware)
	
	// OTP
	auth.POST("/otp/send", authHandler.SendOTP)
	auth.POST("/otp/verify", authHandler.VerifyOTP)
	
	// Google OAuth
	auth.GET("/google", authHandler.GoogleLogin)
	auth.GET("/google/callback", authHandler.GoogleCallback)

	// Billing & Payments
	bill := v1.Group("/billing", middleware.CombinedAuthMiddleware)
	bill.POST("/topup", billingHandler.CreateTopupLink)
	bill.POST("/subscribe", billingHandler.Subscribe)
	v1.POST("/billing/webhook/:provider", billingHandler.HandleWebhook)

	// Video Generation & Composition
	gen := v1.Group("/generate", middleware.CombinedAuthMiddleware, middleware.CreditCheckMiddleware)
	gen.POST("", genHandler.Generate)
	gen.POST("/compose", genHandler.Compose)

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

func startWorker(client *genai.Client, userRepo *repository.UserRepository, jobRepo *repository.JobRepository, videoService *services.VideoService, paymentService *billing.PaymentService, creditService *billing.CreditService) {
	srv := asynq.NewServer(
		database.GetRedisOpt(),
		asynq.Config{
			Concurrency: 10,
			Queues: map[string]int{
				"critical": 6,
				"default":  3,
				"low":      1,
			},
		},
	)

	videoWorker := worker.NewVideoWorker(client, userRepo, jobRepo, creditService)
	compositionWorker := worker.NewCompositionWorker(jobRepo, userRepo, videoService, services.Storage)
	finalizerWorker := worker.NewFinalizerWorker(jobRepo, userRepo)
	
	mux := asynq.NewServeMux()
	mux.HandleFunc(worker.TypeVideoGeneration, videoWorker.ProcessTask)
	mux.HandleFunc(worker.TypeVideoFinalize, finalizerWorker.ProcessTask)
	mux.HandleFunc(worker.TypeVideoComposition, compositionWorker.ProcessTask)

	if err := srv.Run(mux); err != nil {
		log.Fatalf("could not run asynq server: %v", err)
	}
}

func startBillingCron(userRepo *repository.UserRepository, paymentService *billing.PaymentService) {
	c := cron.New()
	billingWorker := worker.NewBillingWorker(userRepo, services.Email, paymentService)

	// Run every hour
	_, err := c.AddFunc("@hourly", func() {
		ctx := context.Background()
		billingWorker.CheckRenewals(ctx)
		billingWorker.HandleGracePeriods(ctx)
	})
	if err != nil {
		log.Fatalf("Could not schedule billing cron: %v", err)
	}

	c.Start()
	log.Println("Billing Worker (Cron) Started - Running hourly")
}

func startPollerCron(client *genai.Client, jobRepo *repository.JobRepository, creditService *billing.CreditService) {
	c := cron.New()
	pollerWorker := worker.NewPollerWorker(client, jobRepo, database.AsynqClient, creditService)

	// Run every 30 seconds to check on active LROs
	_, err := c.AddFunc("@every 30s", func() {
		ctx := context.Background()
		pollerWorker.CheckProcessingJobs(ctx)
	})
	if err != nil {
		log.Fatalf("Could not schedule poller cron: %v", err)
	}

	c.Start()
	log.Println("LRO Poller Worker (Cron) Started - Running every 30s")
}
