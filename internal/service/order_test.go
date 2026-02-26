package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flicky/go-ecommerce-api/internal/model"
	"github.com/flicky/go-ecommerce-api/internal/repository"
)

type mockOrderRepo struct {
	orders map[uuid.UUID]*model.Order
}

func newMockOrderRepo() *mockOrderRepo {
	return &mockOrderRepo{orders: make(map[uuid.UUID]*model.Order)}
}

func (m *mockOrderRepo) Create(_ context.Context, order *model.Order) error {
	order.ID = uuid.New()
	order.CreatedAt = time.Now()
	order.UpdatedAt = time.Now()
	m.orders[order.ID] = order
	return nil
}

func (m *mockOrderRepo) CreateItems(_ context.Context, _ pgx.Tx, items []model.OrderItem) error {
	for i := range items {
		items[i].ID = uuid.New()
		items[i].CreatedAt = time.Now()
	}
	return nil
}

func (m *mockOrderRepo) GetByID(_ context.Context, id uuid.UUID) (*model.Order, error) {
	o, ok := m.orders[id]
	if !ok {
		return nil, nil
	}
	return o, nil
}

func (m *mockOrderRepo) ListByUserID(_ context.Context, userID uuid.UUID) ([]model.Order, error) {
	var orders []model.Order
	for _, o := range m.orders {
		if o.UserID == userID {
			orders = append(orders, *o)
		}
	}
	return orders, nil
}

func (m *mockOrderRepo) UpdateStatus(_ context.Context, _ pgx.Tx, id uuid.UUID, status model.OrderStatus) error {
	if o, ok := m.orders[id]; ok {
		o.Status = status
		return nil
	}
	return pgx.ErrNoRows
}

func (m *mockOrderRepo) BeginTx(_ context.Context) (pgx.Tx, error) { return nil, nil }

var _ repository.OrderRepository = (*mockOrderRepo)(nil)

func TestOrderService_CreateOrder_EmptyCart(t *testing.T) {
	svc := &OrderService{
		orderRepo: newMockOrderRepo(), cartRepo: newMockCartRepo(), productRepo: newMockProductRepo(),
	}
	_, err := svc.CreateOrder(context.Background(), uuid.New())
	assert.ErrorIs(t, err, ErrEmptyCart)
}

func TestOrderService_GetByID(t *testing.T) {
	repo := newMockOrderRepo()
	userID := uuid.New()
	orderID := uuid.New()
	repo.orders[orderID] = &model.Order{
		ID: orderID, UserID: userID, Status: model.OrderStatusCreated,
		TotalPrice: decimal.NewFromFloat(99.99), CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}

	svc := &OrderService{orderRepo: repo}

	order, err := svc.GetByID(context.Background(), orderID, userID)
	require.NoError(t, err)
	assert.Equal(t, orderID, order.ID)
}

func TestOrderService_GetByID_NotFound(t *testing.T) {
	svc := &OrderService{orderRepo: newMockOrderRepo()}
	_, err := svc.GetByID(context.Background(), uuid.New(), uuid.New())
	assert.ErrorIs(t, err, ErrOrderNotFound)
}

func TestOrderService_GetByID_WrongUser(t *testing.T) {
	repo := newMockOrderRepo()
	orderID := uuid.New()
	repo.orders[orderID] = &model.Order{
		ID: orderID, UserID: uuid.New(), Status: model.OrderStatusCreated,
		TotalPrice: decimal.NewFromFloat(50), CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}

	svc := &OrderService{orderRepo: repo}
	_, err := svc.GetByID(context.Background(), orderID, uuid.New())
	assert.ErrorIs(t, err, ErrOrderAccessDenied)
}
