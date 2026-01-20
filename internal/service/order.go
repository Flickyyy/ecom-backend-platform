package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/shopspring/decimal"

	"github.com/flicky/go-ecommerce-api/internal/model"
	"github.com/flicky/go-ecommerce-api/internal/repository"
)

var (
	ErrEmptyCart         = errors.New("cart is empty")
	ErrOrderNotFound     = errors.New("order not found")
	ErrOrderAccessDenied = errors.New("access denied")
)

type OrderService struct {
	orderRepo   repository.OrderRepository
	cartRepo    repository.CartRepository
	productRepo repository.ProductRepository
	amqpCh      *amqp.Channel
}

func NewOrderService(orderRepo repository.OrderRepository, cartRepo repository.CartRepository, productRepo repository.ProductRepository, amqpCh *amqp.Channel) *OrderService {
	return &OrderService{orderRepo: orderRepo, cartRepo: cartRepo, productRepo: productRepo, amqpCh: amqpCh}
}

func (s *OrderService) CreateOrder(ctx context.Context, userID uuid.UUID) (*model.Order, error) {
	cart, err := s.cartRepo.GetOrCreateCart(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get cart: %w", err)
	}
	cartWithItems, err := s.cartRepo.GetCartWithItems(ctx, cart.ID)
	if err != nil {
		return nil, fmt.Errorf("get cart items: %w", err)
	}
	if cartWithItems == nil || len(cartWithItems.Items) == 0 {
		return nil, ErrEmptyCart
	}

	var total decimal.Decimal
	var items []model.OrderItem
	for _, ci := range cartWithItems.Items {
		product, err := s.productRepo.GetByID(ctx, ci.ProductID)
		if err != nil || product == nil {
			return nil, fmt.Errorf("product %s not found", ci.ProductID)
		}
		total = total.Add(product.Price.Mul(decimal.NewFromInt(int64(ci.Quantity))))
		items = append(items, model.OrderItem{
			ProductID: ci.ProductID, Quantity: ci.Quantity, Price: product.Price,
		})
	}

	order := &model.Order{UserID: userID, Status: "pending", TotalPrice: total, Items: items}
	if err := s.orderRepo.Create(ctx, order); err != nil {
		return nil, fmt.Errorf("create order: %w", err)
	}

	// Publish to RabbitMQ for async processing
	msg, _ := json.Marshal(model.OrderMessage{OrderID: order.ID, UserID: userID})
	if s.amqpCh != nil {
		_ = s.amqpCh.PublishWithContext(ctx, "", "orders", false, false, amqp.Publishing{
			ContentType:  "application/json",
			Body:         msg,
			DeliveryMode: amqp.Persistent,
		})
	}

	_ = s.cartRepo.ClearCart(ctx, cart.ID)
	return order, nil
}

func (s *OrderService) GetByID(ctx context.Context, orderID, userID uuid.UUID) (*model.Order, error) {
	order, err := s.orderRepo.GetByID(ctx, orderID)
	if err != nil {
		return nil, fmt.Errorf("get order: %w", err)
	}
	if order == nil {
		return nil, ErrOrderNotFound
	}
	if order.UserID != userID {
		return nil, ErrOrderAccessDenied
	}
	return order, nil
}

func (s *OrderService) ListByUserID(ctx context.Context, userID uuid.UUID) ([]model.Order, error) {
	return s.orderRepo.ListByUserID(ctx, userID)
}
