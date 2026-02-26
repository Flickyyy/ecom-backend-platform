package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/redis/go-redis/v9"

	"github.com/flicky/go-ecommerce-api/internal/model"
	"github.com/flicky/go-ecommerce-api/internal/repository"
)

const (
	orderQueueName = "orders"
	dlxExchange    = "orders.dlx"
	dlqQueueName   = "orders.dlq"
	idempotencyTTL = 24 * time.Hour
)

type OrderWorker struct {
	channel     *amqp.Channel
	orderRepo   repository.OrderRepository
	productRepo repository.ProductRepository
	redisClient *redis.Client
	log         *slog.Logger
	done        chan struct{}
}

func NewOrderWorker(
	ch *amqp.Channel,
	orderRepo repository.OrderRepository,
	productRepo repository.ProductRepository,
	redisClient *redis.Client,
	log *slog.Logger,
) *OrderWorker {
	return &OrderWorker{
		channel:     ch,
		orderRepo:   orderRepo,
		productRepo: productRepo,
		redisClient: redisClient,
		log:         log,
		done:        make(chan struct{}),
	}
}

// SetupRabbitMQ declares exchanges, queues, and bindings (DLX/DLQ).
func SetupRabbitMQ(ch *amqp.Channel) error {
	if err := ch.ExchangeDeclare(dlxExchange, "direct", true, false, false, false, nil); err != nil {
		return fmt.Errorf("declare DLX: %w", err)
	}
	if _, err := ch.QueueDeclare(dlqQueueName, true, false, false, false, nil); err != nil {
		return fmt.Errorf("declare DLQ: %w", err)
	}
	if err := ch.QueueBind(dlqQueueName, orderQueueName, dlxExchange, false, nil); err != nil {
		return fmt.Errorf("bind DLQ: %w", err)
	}
	if _, err := ch.QueueDeclare(orderQueueName, true, false, false, false, amqp.Table{
		"x-dead-letter-exchange":    dlxExchange,
		"x-dead-letter-routing-key": orderQueueName,
	}); err != nil {
		return fmt.Errorf("declare order queue: %w", err)
	}
	if err := ch.Qos(1, 0, false); err != nil {
		return fmt.Errorf("set QoS: %w", err)
	}
	return nil
}

func (w *OrderWorker) Start(ctx context.Context) error {
	msgs, err := w.channel.Consume(orderQueueName, "", false, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("start consuming: %w", err)
	}

	go func() {
		for {
			select {
			case msg, ok := <-msgs:
				if !ok {
					return
				}
				w.processMessage(ctx, msg)
			case <-w.done:
				return
			case <-ctx.Done():
				return
			}
		}
	}()

	w.log.Info("order worker started")
	return nil
}

func (w *OrderWorker) Stop() { close(w.done) }

func (w *OrderWorker) processMessage(ctx context.Context, msg amqp.Delivery) {
	var orderMsg model.OrderMessage
	if err := json.Unmarshal(msg.Body, &orderMsg); err != nil {
		w.log.Error("unmarshal order message", "error", err)
		_ = msg.Nack(false, false)
		return
	}

	log := w.log.With("order_id", orderMsg.OrderID, "user_id", orderMsg.UserID)

	// Idempotency check via Redis
	idempotencyKey := "order_processed:" + orderMsg.OrderID.String()
	exists, err := w.redisClient.Exists(ctx, idempotencyKey).Result()
	if err != nil {
		log.Error("check idempotency key", "error", err)
		_ = msg.Nack(false, true)
		return
	}
	if exists > 0 {
		log.Info("order already processed, skipping")
		_ = msg.Ack(false)
		return
	}

	if err := w.processOrder(ctx, orderMsg.OrderID); err != nil {
		log.Error("process order failed", "error", err)
		_ = msg.Nack(false, false) // → DLQ
		return
	}

	if err := w.redisClient.Set(ctx, idempotencyKey, "1", idempotencyTTL).Err(); err != nil {
		log.Error("set idempotency key", "error", err)
	}

	_ = msg.Ack(false)
	log.Info("order processed successfully")
}

func (w *OrderWorker) processOrder(ctx context.Context, orderID uuid.UUID) error {
	order, err := w.orderRepo.GetByID(ctx, orderID)
	if err != nil {
		return fmt.Errorf("get order: %w", err)
	}
	if order == nil {
		return fmt.Errorf("order not found: %s", orderID)
	}

	tx, err := w.orderRepo.BeginTx(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
		}
	}()

	if err = w.orderRepo.UpdateStatus(ctx, tx, orderID, model.OrderStatusProcessing); err != nil {
		return fmt.Errorf("set processing: %w", err)
	}

	if err = w.orderRepo.CreateItems(ctx, tx, order.Items); err != nil {
		_ = w.orderRepo.UpdateStatus(ctx, tx, orderID, model.OrderStatusFailed)
		return fmt.Errorf("create items: %w", err)
	}

	for _, item := range order.Items {
		if err = w.productRepo.DecrementStock(ctx, tx, item.ProductID, item.Quantity); err != nil {
			_ = w.orderRepo.UpdateStatus(ctx, tx, orderID, model.OrderStatusFailed)
			return fmt.Errorf("decrement stock: %w", err)
		}
	}

	if err = w.orderRepo.UpdateStatus(ctx, tx, orderID, model.OrderStatusCompleted); err != nil {
		return fmt.Errorf("set completed: %w", err)
	}

	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}
	return nil
}
