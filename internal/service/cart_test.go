package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flicky/go-ecommerce-api/internal/model"
	"github.com/flicky/go-ecommerce-api/internal/repository"
)

type mockCartRepo struct {
	carts map[uuid.UUID]*model.Cart
	items map[uuid.UUID]*model.CartItem
}

func newMockCartRepo() *mockCartRepo {
	return &mockCartRepo{carts: make(map[uuid.UUID]*model.Cart), items: make(map[uuid.UUID]*model.CartItem)}
}

func (m *mockCartRepo) GetOrCreateCart(_ context.Context, userID uuid.UUID) (*model.Cart, error) {
	for _, c := range m.carts {
		if c.UserID == userID {
			return c, nil
		}
	}
	cart := &model.Cart{ID: uuid.New(), UserID: userID, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	m.carts[cart.ID] = cart
	return cart, nil
}

func (m *mockCartRepo) GetCartWithItems(_ context.Context, cartID uuid.UUID) (*model.Cart, error) {
	cart, ok := m.carts[cartID]
	if !ok {
		return nil, nil
	}
	cart.Items = nil
	for _, item := range m.items {
		if item.CartID == cartID {
			cart.Items = append(cart.Items, *item)
		}
	}
	return cart, nil
}

func (m *mockCartRepo) AddItem(_ context.Context, item *model.CartItem) error {
	item.ID = uuid.New()
	item.CreatedAt = time.Now()
	item.UpdatedAt = time.Now()
	m.items[item.ID] = item
	return nil
}

func (m *mockCartRepo) UpdateItem(_ context.Context, item *model.CartItem) error {
	if existing, ok := m.items[item.ID]; ok {
		existing.Quantity = item.Quantity
	}
	return nil
}

func (m *mockCartRepo) DeleteItem(_ context.Context, itemID uuid.UUID) error {
	if _, ok := m.items[itemID]; !ok {
		return pgx.ErrNoRows
	}
	delete(m.items, itemID)
	return nil
}

func (m *mockCartRepo) ClearCart(_ context.Context, cartID uuid.UUID) error {
	for id, item := range m.items {
		if item.CartID == cartID {
			delete(m.items, id)
		}
	}
	return nil
}

var _ repository.CartRepository = (*mockCartRepo)(nil)

func TestCartService_GetCart(t *testing.T) {
	svc := NewCartService(newMockCartRepo(), newMockProductRepo())
	userID := uuid.New()

	cart, err := svc.GetCart(context.Background(), userID)
	require.NoError(t, err)
	assert.Equal(t, userID, cart.UserID)
}

func TestCartService_AddItem(t *testing.T) {
	cartRepo := newMockCartRepo()
	productRepo := newMockProductRepo()
	svc := NewCartService(cartRepo, productRepo)

	pid := uuid.New()
	productRepo.products[pid] = &model.Product{ID: pid, Stock: 100}

	err := svc.AddItem(context.Background(), uuid.New(), pid, 2)
	require.NoError(t, err)
	assert.Len(t, cartRepo.items, 1)
}

func TestCartService_AddItem_ProductNotFound(t *testing.T) {
	svc := NewCartService(newMockCartRepo(), newMockProductRepo())
	err := svc.AddItem(context.Background(), uuid.New(), uuid.New(), 2)
	assert.ErrorIs(t, err, ErrProductNotFound)
}

func TestCartService_DeleteItem(t *testing.T) {
	cartRepo := newMockCartRepo()
	svc := NewCartService(cartRepo, newMockProductRepo())

	userID := uuid.New()
	cart, _ := cartRepo.GetOrCreateCart(context.Background(), userID)
	item := &model.CartItem{ID: uuid.New(), CartID: cart.ID, ProductID: uuid.New(), Quantity: 1, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	cartRepo.items[item.ID] = item

	err := svc.DeleteItem(context.Background(), userID, item.ID)
	require.NoError(t, err)
	assert.Empty(t, cartRepo.items)
}
