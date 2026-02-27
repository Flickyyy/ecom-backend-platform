package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

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

	db, err := pgxpool.New(ctx, cfg.DB.DSN())
	if err != nil {
		log.Error("connect to database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	if err := db.Ping(ctx); err != nil {
		log.Error("ping database", "error", err)
		os.Exit(1)
	}
	log.Info("connected to PostgreSQL")

	// Redis
	rdb := redis.NewClient(&redis.Options{Addr: cfg.Redis.Addr})
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Error("connect to redis", "error", err)
		os.Exit(1)
	}
	defer rdb.Close()
	log.Info("connected to Redis")

	// RabbitMQ
	amqpConn, err := amqp.Dial(cfg.RabbitMQ.URL)
	if err != nil {
		log.Error("connect to rabbitmq", "error", err)
		os.Exit(1)
	}
	defer amqpConn.Close()

	amqpCh, err := amqpConn.Channel()
	if err != nil {
		log.Error("open rabbitmq channel", "error", err)
		os.Exit(1)
	}
	defer amqpCh.Close()
	log.Info("connected to RabbitMQ")

	if err := worker.SetupQueues(amqpCh); err != nil {
		log.Error("setup queues", "error", err)
		os.Exit(1)
	}

	// Repos
	userRepo := repository.NewUserRepository(db)
	productRepo := repository.NewProductRepository(db)
	cartRepo := repository.NewCartRepository(db)
	orderRepo := repository.NewOrderRepository(db)

	// Services
	authSvc := service.NewAuthService(userRepo, cfg.JWT.Secret, cfg.JWT.Expiration)
	productSvc := service.NewProductService(productRepo, rdb)
	cartSvc := service.NewCartService(cartRepo, productRepo)
	orderSvc := service.NewOrderService(orderRepo, cartRepo, productRepo, amqpCh)

	// Worker
	orderWorker := worker.NewOrderWorker(amqpCh, orderRepo, rdb, log)
	orderWorker.Start(ctx)

	// Handlers
	authH := handler.NewAuthHandler(authSvc)
	productH := handler.NewProductHandler(productSvc)
	cartH := handler.NewCartHandler(cartSvc)
	orderH := handler.NewOrderHandler(orderSvc)

	// Router
	r := gin.Default()
	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})
	r.GET("/readyz", func(c *gin.Context) {
		if err := db.Ping(c.Request.Context()); err != nil {
			c.JSON(503, gin.H{"status": "not ready", "error": "postgres"})
			return
		}
		if err := rdb.Ping(c.Request.Context()).Err(); err != nil {
			c.JSON(503, gin.H{"status": "not ready", "error": "redis"})
			return
		}
		c.JSON(200, gin.H{"status": "ready"})
	})

	v1 := r.Group("/api/v1")
	v1.POST("/auth/register", authH.Register)
	v1.POST("/auth/login", authH.Login)

	v1.GET("/products", productH.List)
	v1.GET("/products/:id", productH.GetByID)

	admin := v1.Group("", middleware.AuthMiddleware(cfg.JWT.Secret), middleware.AdminOnly())
	admin.POST("/products", productH.Create)
	admin.PUT("/products/:id", productH.Update)
	admin.DELETE("/products/:id", productH.Delete)

	auth := v1.Group("", middleware.AuthMiddleware(cfg.JWT.Secret))
	auth.GET("/cart", cartH.GetCart)
	auth.POST("/cart/items", cartH.AddItem)
	auth.PUT("/cart/items/:id", cartH.UpdateItem)
	auth.DELETE("/cart/items/:id", cartH.DeleteItem)
	auth.POST("/orders", orderH.CreateOrder)
	auth.GET("/orders", orderH.ListOrders)
	auth.GET("/orders/:id", orderH.GetOrder)

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Server.Port),
		Handler: r,
	}

	go func() {
		log.Info("starting server", "port", cfg.Server.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("shutting down...")
	orderWorker.Stop()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("shutdown error", "error", err)
	}
	log.Info("server stopped")
}
