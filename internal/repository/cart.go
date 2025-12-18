package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/flicky/go-ecommerce-api/internal/model"
)

type CartRepository interface {
	GetOrCreateCart(ctx context.Context, userID uuid.UUID) (*model.Cart, error)
	GetCartWithItems(ctx context.Context, cartID uuid.UUID) (*model.Cart, error)
	AddItem(ctx context.Context, item *model.CartItem) error
	UpdateItem(ctx context.Context, item *model.CartItem) error
	DeleteItem(ctx context.Context, itemID uuid.UUID) error
	ClearCart(ctx context.Context, cartID uuid.UUID) error
}

type pgCartRepo struct{ pool *pgxpool.Pool }

func NewCartRepository(pool *pgxpool.Pool) CartRepository {
	return &pgCartRepo{pool: pool}
}

func (r *pgCartRepo) GetOrCreateCart(ctx context.Context, userID uuid.UUID) (*model.Cart, error) {
	cart := &model.Cart{UserID: userID}
	err := r.pool.QueryRow(ctx, `SELECT id FROM carts WHERE user_id = $1`, userID).Scan(&cart.ID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			cart.ID = uuid.New()
			_, err = r.pool.Exec(ctx,
				`INSERT INTO carts (id, user_id, created_at, updated_at) VALUES ($1, $2, NOW(), NOW())`,
				cart.ID, userID,
			)
			if err != nil {
				return nil, fmt.Errorf("create cart: %w", err)
			}
			return cart, nil
		}
		return nil, fmt.Errorf("get cart: %w", err)
	}
	return cart, nil
}

func (r *pgCartRepo) GetCartWithItems(ctx context.Context, cartID uuid.UUID) (*model.Cart, error) {
	cart := &model.Cart{ID: cartID}
	err := r.pool.QueryRow(ctx, `SELECT user_id FROM carts WHERE id = $1`, cartID).Scan(&cart.UserID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get cart: %w", err)
	}

	rows, err := r.pool.Query(ctx,
		`SELECT id, product_id, quantity FROM cart_items WHERE cart_id = $1`, cartID,
	)
	if err != nil {
		return nil, fmt.Errorf("get cart items: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var item model.CartItem
		if err := rows.Scan(&item.ID, &item.ProductID, &item.Quantity); err != nil {
			return nil, fmt.Errorf("scan cart item: %w", err)
		}
		item.CartID = cartID
		cart.Items = append(cart.Items, item)
	}
	return cart, nil
}

func (r *pgCartRepo) AddItem(ctx context.Context, item *model.CartItem) error {
	item.ID = uuid.New()
	_, err := r.pool.Exec(ctx,
		`INSERT INTO cart_items (id, cart_id, product_id, quantity, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, NOW(), NOW())
		 ON CONFLICT (cart_id, product_id) DO UPDATE SET quantity = cart_items.quantity + $4, updated_at = NOW()`,
		item.ID, item.CartID, item.ProductID, item.Quantity,
	)
	if err != nil {
		return fmt.Errorf("add cart item: %w", err)
	}
	return nil
}

func (r *pgCartRepo) UpdateItem(ctx context.Context, item *model.CartItem) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE cart_items SET quantity = $2, updated_at = NOW() WHERE id = $1`,
		item.ID, item.Quantity,
	)
	if err != nil {
		return fmt.Errorf("update cart item: %w", err)
	}
	return nil
}

func (r *pgCartRepo) DeleteItem(ctx context.Context, itemID uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM cart_items WHERE id = $1`, itemID)
	if err != nil {
		return fmt.Errorf("delete cart item: %w", err)
	}
	return nil
}

func (r *pgCartRepo) ClearCart(ctx context.Context, cartID uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM cart_items WHERE cart_id = $1`, cartID)
	if err != nil {
		return fmt.Errorf("clear cart: %w", err)
	}
	return nil
}
