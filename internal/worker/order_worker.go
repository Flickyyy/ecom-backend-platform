package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/redis/go-redis/v9"

	"github.com/flicky/go-ecommerce-api/internal/model"
	"github.com/flicky/go-ecommerce-api/internal/repository"
)

type OrderWorker struct {
	ch        *amqp.Channel
	orderRepo repository.OrderRepository
	redis     *redis.Client
	log       *slog.Logger
	done      chan struct{}
}

func NewOrderWorker(ch *amqp.Channel, orderRepo repository.OrderRepository, redis *redis.Client, log *slog.Logger) *OrderWorker {
	return &OrderWorker{ch: ch, orderRepo: orderRepo, redis: redis, log: log, done: make(chan struct{})}
}

func SetupQueues(ch *amqp.Channel) error {
	if err := ch.ExchangeDeclare("orders.dlx", "direct", true, false, false, false, nil); err != nil {
		return fmt.Errorf("declare DLX: %w", err)
	}
	if _, err := ch.QueueDeclare("orders.dlq", true, false, false, false, nil); err != nil {
		return fmt.Errorf("declare DLQ: %w", err)
	}
	if err := ch.QueueBind("orders.dlq", "orders", "orders.dlx", false, nil); err != nil {
		return fmt.Errorf("bind DLQ: %w", err)
	}
	if _, err := ch.QueueDeclare("orders", true, false, false, false, amqp.Table{
		"x-dead-letter-exchange":    "orders.dlx",
		"x-dead-letter-routing-key": "orders",
	}); err != nil {
		return fmt.Errorf("declare queue: %w", err)
	}
	return ch.Qos(1, 0, false)
}

func (w *OrderWorker) Start(ctx context.Context) error {
	msgs, err := w.ch.Consume("orders", "", false, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("consume: %w", err)
	}
	go func() {
		for {
			select {
			case msg, ok := <-msgs:
				if !ok {
					return
				}
				w.handle(ctx, msg)
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

func (w *OrderWorker) handle(ctx context.Context, msg amqp.Delivery) {
	var m model.OrderMessage
	if err := json.Unmarshal(msg.Body, &m); err != nil {
		w.log.Error("unmarshal", "error", err)
		msg.Nack(false, false)
		return
	}

	// Idempotency check
	key := "order_processed:" + m.OrderID.String()
	if n, _ := w.redis.Exists(ctx, key).Result(); n > 0 {
		w.log.Info("already processed", "order_id", m.OrderID)
		msg.Ack(false)
		return
	}

	order, err := w.orderRepo.GetByID(ctx, m.OrderID)
	if err != nil || order == nil {
		w.log.Error("get order", "error", err)
		msg.Nack(false, false)
		return
	}

	if err := w.orderRepo.ProcessOrder(ctx, m.OrderID, order.Items); err != nil {
		w.log.Error("process order", "error", err, "order_id", m.OrderID)
		_ = w.orderRepo.UpdateStatus(ctx, m.OrderID, "failed")
		msg.Nack(false, false) // → DLQ
		return
	}

	w.redis.Set(ctx, key, "1", 24*time.Hour)
	msg.Ack(false)
	w.log.Info("order processed", "order_id", m.OrderID)
}
