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
	ErrOrderAccessDenied = errors.New("access denied to this order")
)

type OrderService struct {
	orderRepo   repository.OrderRepository
	cartRepo    repository.CartRepository
	productRepo repository.ProductRepository
	amqpChannel *amqp.Channel
}

func NewOrderService(
	orderRepo repository.OrderRepository,
	cartRepo repository.CartRepository,
	productRepo repository.ProductRepository,
	amqpChannel *amqp.Channel,
) *OrderService {
	return &OrderService{
		orderRepo:   orderRepo,
		cartRepo:    cartRepo,
		productRepo: productRepo,
		amqpChannel: amqpChannel,
	}
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

	// Calculate total and build order items
	totalPrice := decimal.NewFromInt(0)
	var orderItems []model.OrderItem
	for _, item := range cartWithItems.Items {
		product, err := s.productRepo.GetByID(ctx, item.ProductID)
		if err != nil {
			return nil, fmt.Errorf("get product: %w", err)
		}
		if product == nil {
			return nil, fmt.Errorf("product %s not found", item.ProductID)
		}

		totalPrice = totalPrice.Add(product.Price.Mul(decimal.NewFromInt(int64(item.Quantity))))
		orderItems = append(orderItems, model.OrderItem{
			ProductID: item.ProductID,
			Quantity:  item.Quantity,
			Price:     product.Price,
		})
	}

	order := &model.Order{UserID: userID, Status: model.OrderStatusCreated, TotalPrice: totalPrice}
	if err := s.orderRepo.Create(ctx, order); err != nil {
		return nil, fmt.Errorf("create order: %w", err)
	}

	for i := range orderItems {
		orderItems[i].OrderID = order.ID
	}
	order.Items = orderItems

	// Publish to RabbitMQ
	msgBytes, err := json.Marshal(model.OrderMessage{OrderID: order.ID, UserID: userID})
	if err != nil {
		return nil, fmt.Errorf("marshal order message: %w", err)
	}
	err = s.amqpChannel.PublishWithContext(ctx, "", "orders", false, false, amqp.Publishing{
		ContentType:  "application/json",
		Body:         msgBytes,
		DeliveryMode: amqp.Persistent,
	})
	if err != nil {
		return nil, fmt.Errorf("publish order message: %w", err)
	}

	if err := s.cartRepo.ClearCart(ctx, cart.ID); err != nil {
		return nil, fmt.Errorf("clear cart: %w", err)
	}

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
	orders, err := s.orderRepo.ListByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list orders: %w", err)
	}
	return orders, nil
}
