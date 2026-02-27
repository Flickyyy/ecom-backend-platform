package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flicky/go-ecommerce-api/internal/model"
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
	m.orders[order.ID] = order
	return nil
}

func (m *mockOrderRepo) ProcessOrder(_ context.Context, _ uuid.UUID, _ []model.OrderItem) error {
	return nil
}

func (m *mockOrderRepo) UpdateStatus(_ context.Context, id uuid.UUID, status string) error {
	if o, ok := m.orders[id]; ok {
		o.Status = status
	}
	return nil
}

func (m *mockOrderRepo) GetByID(_ context.Context, id uuid.UUID) (*model.Order, error) {
	return m.orders[id], nil
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

func TestOrderService_CreateOrder_EmptyCart(t *testing.T) {
	svc := NewOrderService(newMockOrderRepo(), newMockCartRepo(), newMockProductRepo(), nil)
	_, err := svc.CreateOrder(context.Background(), uuid.New())
	assert.ErrorIs(t, err, ErrEmptyCart)
}

func TestOrderService_GetByID(t *testing.T) {
	repo := newMockOrderRepo()
	userID := uuid.New()
	orderID := uuid.New()
	repo.orders[orderID] = &model.Order{
		ID: orderID, UserID: userID, Status: "completed",
		TotalPrice: decimal.NewFromFloat(99.99), CreatedAt: time.Now(),
	}
	svc := NewOrderService(repo, nil, nil, nil)
	order, err := svc.GetByID(context.Background(), orderID, userID)
	require.NoError(t, err)
	assert.Equal(t, orderID, order.ID)
}

func TestOrderService_GetByID_NotFound(t *testing.T) {
	svc := NewOrderService(newMockOrderRepo(), nil, nil, nil)
	_, err := svc.GetByID(context.Background(), uuid.New(), uuid.New())
	assert.ErrorIs(t, err, ErrOrderNotFound)
}
