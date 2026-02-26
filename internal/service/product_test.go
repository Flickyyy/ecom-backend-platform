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

	"github.com/flicky/go-ecommerce-api/internal/dto"
	"github.com/flicky/go-ecommerce-api/internal/model"
	"github.com/flicky/go-ecommerce-api/internal/repository"
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
	p, ok := m.products[id]
	if !ok {
		return nil, nil
	}
	return p, nil
}

func (m *mockProductRepo) List(_ context.Context, limit, offset int, _, _, _ string) ([]model.Product, int, error) {
	var all []model.Product
	for _, p := range m.products {
		all = append(all, *p)
	}
	total := len(all)
	if offset >= total {
		return nil, total, nil
	}
	end := offset + limit
	if end > total {
		end = total
	}
	return all[offset:end], total, nil
}

func (m *mockProductRepo) Update(_ context.Context, p *model.Product) error {
	if _, ok := m.products[p.ID]; !ok {
		return nil
	}
	p.UpdatedAt = time.Now()
	m.products[p.ID] = p
	return nil
}

func (m *mockProductRepo) Delete(_ context.Context, id uuid.UUID) error {
	if _, ok := m.products[id]; !ok {
		return pgx.ErrNoRows
	}
	delete(m.products, id)
	return nil
}

func (m *mockProductRepo) DecrementStock(_ context.Context, _ pgx.Tx, id uuid.UUID, qty int) error {
	p, ok := m.products[id]
	if !ok {
		return pgx.ErrNoRows
	}
	p.Stock -= qty
	return nil
}

var _ repository.ProductRepository = (*mockProductRepo)(nil)

func TestProductService_Create(t *testing.T) {
	repo := newMockProductRepo()
	svc := &ProductService{productRepo: repo}

	resp, err := svc.Create(context.Background(), dto.CreateProductRequest{
		Name: "Test", Description: "Desc", Price: decimal.NewFromFloat(9.99), Stock: 100,
	})
	require.NoError(t, err)
	assert.Equal(t, "Test", resp.Name)
	assert.Equal(t, 100, resp.Stock)
}

func TestProductService_GetByID(t *testing.T) {
	repo := newMockProductRepo()
	id := uuid.New()
	repo.products[id] = &model.Product{
		ID: id, Name: "Test", Price: decimal.NewFromFloat(9.99),
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}

	svc := &ProductService{productRepo: repo}

	resp, err := svc.GetByID(context.Background(), id)
	require.NoError(t, err)
	assert.Equal(t, "Test", resp.Name)
}

func TestProductService_GetByID_NotFound(t *testing.T) {
	svc := &ProductService{productRepo: newMockProductRepo()}
	_, err := svc.GetByID(context.Background(), uuid.New())
	assert.ErrorIs(t, err, ErrProductNotFound)
}

func TestProductService_Update(t *testing.T) {
	repo := newMockProductRepo()
	id := uuid.New()
	repo.products[id] = &model.Product{
		ID: id, Name: "Old", Description: "Desc",
		Price: decimal.NewFromFloat(9.99), Stock: 10,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}

	svc := &ProductService{productRepo: repo}
	newName := "New"
	resp, err := svc.Update(context.Background(), id, dto.UpdateProductRequest{Name: &newName})
	require.NoError(t, err)
	assert.Equal(t, "New", resp.Name)
}

func TestProductService_Delete(t *testing.T) {
	repo := newMockProductRepo()
	id := uuid.New()
	repo.products[id] = &model.Product{ID: id}

	svc := &ProductService{productRepo: repo}
	err := svc.Delete(context.Background(), id)
	require.NoError(t, err)
	assert.Empty(t, repo.products)
}
