//go:build integration

package repository

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flicky/go-ecommerce-api/internal/model"
)

func setupTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://postgres:postgres@localhost:5432/ecommerce?sslmode=disable"
	}
	pool, err := pgxpool.New(context.Background(), dsn)
	require.NoError(t, err)
	t.Cleanup(func() { pool.Close() })
	return pool
}

func TestProductRepository_Integration(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewProductRepository(pool)
	ctx := context.Background()

	// Create
	p := &model.Product{
		Name: "Integration Test Product", Description: "test",
		Price: decimal.NewFromFloat(19.99), Stock: 50,
	}
	err := repo.Create(ctx, p)
	require.NoError(t, err)
	assert.NotEmpty(t, p.ID)

	// Read
	found, err := repo.GetByID(ctx, p.ID)
	require.NoError(t, err)
	require.NotNil(t, found)
	assert.Equal(t, p.Name, found.Name)
	assert.True(t, p.Price.Equal(found.Price))

	// Update
	found.Stock = 42
	err = repo.Update(ctx, found)
	require.NoError(t, err)

	updated, _ := repo.GetByID(ctx, p.ID)
	assert.Equal(t, 42, updated.Stock)

	// List
	products, total, err := repo.List(ctx, 10, 0)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, total, 1)
	assert.GreaterOrEqual(t, len(products), 1)

	// Delete
	err = repo.Delete(ctx, p.ID)
	require.NoError(t, err)

	deleted, err := repo.GetByID(ctx, p.ID)
	require.NoError(t, err)
	assert.Nil(t, deleted)
}
