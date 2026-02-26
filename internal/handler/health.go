package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/redis/go-redis/v9"
)

type HealthHandler struct {
	dbPool      *pgxpool.Pool
	redisClient *redis.Client
	amqpConn    *amqp.Connection
}

func NewHealthHandler(dbPool *pgxpool.Pool, redisClient *redis.Client, amqpConn *amqp.Connection) *HealthHandler {
	return &HealthHandler{dbPool: dbPool, redisClient: redisClient, amqpConn: amqpConn}
}

func (h *HealthHandler) Healthz(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *HealthHandler) Readyz(c *gin.Context) {
	ctx := c.Request.Context()

	if err := h.dbPool.Ping(ctx); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "error", "postgres": "unavailable"})
		return
	}
	if err := h.redisClient.Ping(ctx).Err(); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "error", "redis": "unavailable"})
		return
	}
	if h.amqpConn.IsClosed() {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "error", "rabbitmq": "unavailable"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":   "ok",
		"postgres": "connected",
		"redis":    "connected",
		"rabbitmq": "connected",
	})
}
