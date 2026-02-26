package repository

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flicky/go-ecommerce-api/internal/model"
)

func TestUserRepo_CreateAndGetByEmail(t *testing.T) {
	cleanupTable(t, "order_items", "orders", "cart_items", "carts", "users")

	repo := NewUserRepository(testPool)
	ctx := context.Background()

	user := &model.User{
		Email: "test@example.com", Password: "hashed",
		FirstName: "John", LastName: "Doe", Role: "customer",
	}
	require.NoError(t, repo.Create(ctx, user))
	assert.NotEqual(t, uuid.Nil, user.ID)

	found, err := repo.GetByEmail(ctx, "test@example.com")
	require.NoError(t, err)
	require.NotNil(t, found)
	assert.Equal(t, user.ID, found.ID)
}

func TestProductRepo_CRUD(t *testing.T) {
	cleanupTable(t, "order_items", "orders", "cart_items", "carts", "products")

	repo := NewProductRepository(testPool)
	ctx := context.Background()

	product := &model.Product{
		Name: "Test", Description: "Desc",
		Price: decimal.NewFromFloat(29.99), Stock: 100,
	}
	require.NoError(t, repo.Create(ctx, product))
	assert.NotEqual(t, uuid.Nil, product.ID)

	found, err := repo.GetByID(ctx, product.ID)
	require.NoError(t, err)
	assert.Equal(t, "Test", found.Name)

	product.Name = "Updated"
	require.NoError(t, repo.Update(ctx, product))

	found, _ = repo.GetByID(ctx, product.ID)
	assert.Equal(t, "Updated", found.Name)

	require.NoError(t, repo.Delete(ctx, product.ID))
	found, _ = repo.GetByID(ctx, product.ID)
	assert.Nil(t, found)
}

func TestCartRepo_AddAndGetItems(t *testing.T) {
	cleanupTable(t, "order_items", "orders", "cart_items", "carts", "products", "users")

	userRepo := NewUserRepository(testPool)
	productRepo := NewProductRepository(testPool)
	cartRepo := NewCartRepository(testPool)
	ctx := context.Background()

	user := &model.User{
		Email: "cart@example.com", Password: "h", FirstName: "C", LastName: "U", Role: "customer",
	}
	require.NoError(t, userRepo.Create(ctx, user))

	product := &model.Product{
		Name: "P", Description: "D", Price: decimal.NewFromFloat(15), Stock: 10,
	}
	require.NoError(t, productRepo.Create(ctx, product))

	cart, err := cartRepo.GetOrCreateCart(ctx, user.ID)
	require.NoError(t, err)

	require.NoError(t, cartRepo.AddItem(ctx, &model.CartItem{
		CartID: cart.ID, ProductID: product.ID, Quantity: 2,
	}))

	cartWithItems, err := cartRepo.GetCartWithItems(ctx, cart.ID)
	require.NoError(t, err)
	require.Len(t, cartWithItems.Items, 1)
	assert.Equal(t, 2, cartWithItems.Items[0].Quantity)
}

func TestOrderRepo_CreateAndGet(t *testing.T) {
	cleanupTable(t, "order_items", "orders", "cart_items", "carts", "products", "users")

	userRepo := NewUserRepository(testPool)
	productRepo := NewProductRepository(testPool)
	orderRepo := NewOrderRepository(testPool)
	ctx := context.Background()

	user := &model.User{
		Email: "order@example.com", Password: "h", FirstName: "O", LastName: "U", Role: "customer",
	}
	require.NoError(t, userRepo.Create(ctx, user))

	product := &model.Product{
		Name: "P", Description: "D", Price: decimal.NewFromFloat(25), Stock: 10,
	}
	require.NoError(t, productRepo.Create(ctx, product))

	order := &model.Order{
		UserID: user.ID, Status: model.OrderStatusCreated,
		TotalPrice: decimal.NewFromFloat(50),
	}
	require.NoError(t, orderRepo.Create(ctx, order))
	assert.NotEqual(t, uuid.Nil, order.ID)

	tx, err := orderRepo.BeginTx(ctx)
	require.NoError(t, err)

	require.NoError(t, orderRepo.CreateItems(ctx, tx, []model.OrderItem{
		{OrderID: order.ID, ProductID: product.ID, Quantity: 2, Price: product.Price},
	}))
	require.NoError(t, tx.Commit(ctx))

	found, err := orderRepo.GetByID(ctx, order.ID)
	require.NoError(t, err)
	assert.Equal(t, model.OrderStatusCreated, found.Status)
	require.Len(t, found.Items, 1)
}
