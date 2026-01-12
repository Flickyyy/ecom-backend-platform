package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flicky/go-ecommerce-api/internal/dto"
	"github.com/flicky/go-ecommerce-api/internal/model"
)

type mockProductRepo struct {
	products map[uuid.UUID]*model.Product
}

func newMockProductRepo() *mockProductRepo {
	return &mockProductRepo{products: make(map[uuid.UUID]*model.Product)}
}

func (m *mockProductRepo) Create(_ context.Context, p *model.Product) error {
	p.ID = uuid.New()
	p.CreatedAt = time.Now()
	p.UpdatedAt = time.Now()
	m.products[p.ID] = p
	return nil
}

func (m *mockProductRepo) GetByID(_ context.Context, id uuid.UUID) (*model.Product, error) {
	return m.products[id], nil
}

func (m *mockProductRepo) List(_ context.Context, limit, offset int) ([]model.Product, int, error) {
	var all []model.Product
	for _, p := range m.products {
		all = append(all, *p)
	}
	return all, len(all), nil
}

func (m *mockProductRepo) Update(_ context.Context, p *model.Product) error {
	m.products[p.ID] = p
	return nil
}

func (m *mockProductRepo) Delete(_ context.Context, id uuid.UUID) error {
	delete(m.products, id)
	return nil
}

func TestProductService_Create(t *testing.T) {
	svc := NewProductService(newMockProductRepo(), nil)
	resp, err := svc.Create(context.Background(), dto.CreateProductRequest{
		Name: "Test", Price: decimal.NewFromFloat(9.99), Stock: 100,
	})
	require.NoError(t, err)
	assert.Equal(t, "Test", resp.Name)
	assert.Equal(t, 100, resp.Stock)
}

func TestProductService_GetByID_NotFound(t *testing.T) {
	svc := NewProductService(newMockProductRepo(), nil)
	_, err := svc.GetByID(context.Background(), uuid.New())
	assert.ErrorIs(t, err, ErrProductNotFound)
}

func TestProductService_Delete(t *testing.T) {
	repo := newMockProductRepo()
	id := uuid.New()
	repo.products[id] = &model.Product{ID: id}
	svc := NewProductService(repo, nil)
	err := svc.Delete(context.Background(), id)
	require.NoError(t, err)
	assert.Empty(t, repo.products)
}
