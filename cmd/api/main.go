package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/redis/go-redis/v9"

	"github.com/flicky/go-ecommerce-api/internal/config"
	"github.com/flicky/go-ecommerce-api/internal/handler"
	"github.com/flicky/go-ecommerce-api/internal/middleware"
	"github.com/flicky/go-ecommerce-api/internal/repository"
	"github.com/flicky/go-ecommerce-api/internal/service"
	"github.com/flicky/go-ecommerce-api/internal/worker"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	cfg, err := config.Load()
	if err != nil {
		log.Error("load config", "error", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// PostgreSQL
	poolCfg, err := pgxpool.ParseConfig(cfg.DB.DSN())
	if err != nil {
		log.Error("parse db config", "error", err)
		os.Exit(1)
	}
	poolCfg.MaxConns = cfg.DB.MaxConns

	dbPool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		log.Error("connect to database", "error", err)
		os.Exit(1)
	}
	defer dbPool.Close()

	if err := dbPool.Ping(ctx); err != nil {
		log.Error("ping database", "error", err)
		os.Exit(1)
	}
	log.Info("connected to PostgreSQL")

	// Redis
	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	defer redisClient.Close()

	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Error("connect to Redis", "error", err)
		os.Exit(1)
	}
	log.Info("connected to Redis")

	// RabbitMQ
	amqpConn, err := amqp.Dial(cfg.RabbitMQ.URL)
	if err != nil {
		log.Error("connect to RabbitMQ", "error", err)
		os.Exit(1)
	}
	defer amqpConn.Close()

	amqpCh, err := amqpConn.Channel()
	if err != nil {
		log.Error("open RabbitMQ channel", "error", err)
		os.Exit(1)
	}
	defer amqpCh.Close()

	if err := worker.SetupRabbitMQ(amqpCh); err != nil {
		log.Error("setup RabbitMQ", "error", err)
		os.Exit(1)
	}
	log.Info("connected to RabbitMQ")

	// Repositories
	userRepo := repository.NewUserRepository(dbPool)
	productRepo := repository.NewProductRepository(dbPool)
	cartRepo := repository.NewCartRepository(dbPool)
	orderRepo := repository.NewOrderRepository(dbPool)

	// Services
	authSvc := service.NewAuthService(userRepo, cfg.JWT.Secret, cfg.JWT.Expiration)
	productSvc := service.NewProductService(productRepo, redisClient)
	cartSvc := service.NewCartService(cartRepo, productRepo)
	orderSvc := service.NewOrderService(orderRepo, cartRepo, productRepo, amqpCh)

	// Handlers
	authH := handler.NewAuthHandler(authSvc)
	productH := handler.NewProductHandler(productSvc)
	cartH := handler.NewCartHandler(cartSvc, productSvc)
	orderH := handler.NewOrderHandler(orderSvc)
	healthH := handler.NewHealthHandler(dbPool, redisClient, amqpConn)

	// Worker
	orderWorker := worker.NewOrderWorker(amqpCh, orderRepo, productRepo, redisClient, log)

	// Router
	router := gin.Default()
	router.GET("/healthz", healthH.Healthz)
	router.GET("/readyz", healthH.Readyz)

	v1 := router.Group("/api/v1")
	{
		auth := v1.Group("/auth")
		auth.POST("/register", authH.Register)
		auth.POST("/login", authH.Login)

		products := v1.Group("/products")
		products.GET("", productH.List)
		products.GET("/:id", productH.GetByID)

		admin := products.Group("", middleware.AuthMiddleware(cfg.JWT.Secret), middleware.AdminOnly())
		admin.POST("", productH.Create)
		admin.PUT("/:id", productH.Update)
		admin.DELETE("/:id", productH.Delete)

		cart := v1.Group("/cart", middleware.AuthMiddleware(cfg.JWT.Secret))
		cart.GET("", cartH.GetCart)
		cart.POST("/items", cartH.AddItem)
		cart.PUT("/items/:id", cartH.UpdateItem)
		cart.DELETE("/items/:id", cartH.DeleteItem)

		orders := v1.Group("/orders", middleware.AuthMiddleware(cfg.JWT.Secret))
		orders.POST("", orderH.CreateOrder)
		orders.GET("", orderH.ListOrders)
		orders.GET("/:id", orderH.GetOrder)
	}

	if err := orderWorker.Start(ctx); err != nil {
		log.Error("start order worker", "error", err)
		os.Exit(1)
	}

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	go func() {
		log.Info("starting HTTP server", "port", cfg.Server.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("shutting down...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("server shutdown", "error", err)
	}

	orderWorker.Stop()
	time.Sleep(500 * time.Millisecond)
	cancel()
	log.Info("server stopped")
}
