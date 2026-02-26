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

type OrderRepository interface {
	Create(ctx context.Context, order *model.Order) error
	CreateItems(ctx context.Context, tx pgx.Tx, items []model.OrderItem) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.Order, error)
	ListByUserID(ctx context.Context, userID uuid.UUID) ([]model.Order, error)
	UpdateStatus(ctx context.Context, tx pgx.Tx, id uuid.UUID, status model.OrderStatus) error
	BeginTx(ctx context.Context) (pgx.Tx, error)
}

type pgOrderRepo struct{ pool *pgxpool.Pool }

func NewOrderRepository(pool *pgxpool.Pool) OrderRepository {
	return &pgOrderRepo{pool: pool}
}

func (r *pgOrderRepo) Create(ctx context.Context, order *model.Order) error {
	order.ID = uuid.New()
	query := `INSERT INTO orders (id, user_id, status, total_price, created_at, updated_at)
			  VALUES ($1, $2, $3, $4, NOW(), NOW()) RETURNING created_at, updated_at`
	err := r.pool.QueryRow(ctx, query, order.ID, order.UserID, order.Status, order.TotalPrice).Scan(&order.CreatedAt, &order.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create order: %w", err)
	}
	return nil
}

func (r *pgOrderRepo) CreateItems(ctx context.Context, tx pgx.Tx, items []model.OrderItem) error {
	for i := range items {
		items[i].ID = uuid.New()
		err := tx.QueryRow(ctx,
			`INSERT INTO order_items (id, order_id, product_id, quantity, price, created_at)
			 VALUES ($1, $2, $3, $4, $5, NOW()) RETURNING created_at`,
			items[i].ID, items[i].OrderID, items[i].ProductID, items[i].Quantity, items[i].Price,
		).Scan(&items[i].CreatedAt)
		if err != nil {
			return fmt.Errorf("create order item: %w", err)
		}
	}
	return nil
}

func (r *pgOrderRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.Order, error) {
	order := &model.Order{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, user_id, status, total_price, created_at, updated_at FROM orders WHERE id = $1`, id,
	).Scan(&order.ID, &order.UserID, &order.Status, &order.TotalPrice, &order.CreatedAt, &order.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get order: %w", err)
	}

	rows, err := r.pool.Query(ctx,
		`SELECT id, order_id, product_id, quantity, price, created_at FROM order_items WHERE order_id = $1`, id,
	)
	if err != nil {
		return nil, fmt.Errorf("get order items: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var item model.OrderItem
		if err := rows.Scan(&item.ID, &item.OrderID, &item.ProductID, &item.Quantity, &item.Price, &item.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan order item: %w", err)
		}
		order.Items = append(order.Items, item)
	}
	return order, nil
}

func (r *pgOrderRepo) ListByUserID(ctx context.Context, userID uuid.UUID) ([]model.Order, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, user_id, status, total_price, created_at, updated_at FROM orders WHERE user_id = $1 ORDER BY created_at DESC`, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("list orders: %w", err)
	}
	defer rows.Close()

	var orders []model.Order
	for rows.Next() {
		var o model.Order
		if err := rows.Scan(&o.ID, &o.UserID, &o.Status, &o.TotalPrice, &o.CreatedAt, &o.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan order: %w", err)
		}
		orders = append(orders, o)
	}
	return orders, nil
}

func (r *pgOrderRepo) UpdateStatus(ctx context.Context, tx pgx.Tx, id uuid.UUID, status model.OrderStatus) error {
	ct, err := tx.Exec(ctx, `UPDATE orders SET status = $2, updated_at = NOW() WHERE id = $1`, id, status)
	if err != nil {
		return fmt.Errorf("update order status: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *pgOrderRepo) BeginTx(ctx context.Context) (pgx.Tx, error) {
	return r.pool.Begin(ctx)
}
