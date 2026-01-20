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
	ProcessOrder(ctx context.Context, orderID uuid.UUID, items []model.OrderItem) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.Order, error)
	ListByUserID(ctx context.Context, userID uuid.UUID) ([]model.Order, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status string) error
}

type pgOrderRepo struct{ pool *pgxpool.Pool }

func NewOrderRepository(pool *pgxpool.Pool) OrderRepository {
	return &pgOrderRepo{pool: pool}
}

func (r *pgOrderRepo) Create(ctx context.Context, order *model.Order) error {
	order.ID = uuid.New()
	err := r.pool.QueryRow(ctx,
		`INSERT INTO orders (id, user_id, status, total_price, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, NOW(), NOW()) RETURNING created_at`,
		order.ID, order.UserID, order.Status, order.TotalPrice,
	).Scan(&order.CreatedAt)
	if err != nil {
		return fmt.Errorf("insert order: %w", err)
	}
	return nil
}

func (r *pgOrderRepo) ProcessOrder(ctx context.Context, orderID uuid.UUID, items []model.OrderItem) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	for i := range items {
		items[i].ID = uuid.New()
		items[i].OrderID = orderID
		_, err = tx.Exec(ctx,
			`INSERT INTO order_items (id, order_id, product_id, quantity, price, created_at)
			 VALUES ($1, $2, $3, $4, $5, NOW())`,
			items[i].ID, items[i].OrderID, items[i].ProductID, items[i].Quantity, items[i].Price,
		)
		if err != nil {
			return fmt.Errorf("insert order item: %w", err)
		}

		ct, err := tx.Exec(ctx,
			`UPDATE products SET stock = stock - $2, updated_at = NOW() WHERE id = $1 AND stock >= $2`,
			items[i].ProductID, items[i].Quantity,
		)
		if err != nil {
			return fmt.Errorf("decrement stock: %w", err)
		}
		if ct.RowsAffected() == 0 {
			return fmt.Errorf("insufficient stock for product %s", items[i].ProductID)
		}
	}

	_, err = tx.Exec(ctx,
		`UPDATE orders SET status = 'completed', updated_at = NOW() WHERE id = $1`, orderID,
	)
	if err != nil {
		return fmt.Errorf("update order status: %w", err)
	}
	return tx.Commit(ctx)
}

func (r *pgOrderRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE orders SET status = $2, updated_at = NOW() WHERE id = $1`, id, status,
	)
	return err
}

func (r *pgOrderRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.Order, error) {
	order := &model.Order{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, user_id, status, total_price, created_at FROM orders WHERE id = $1`, id,
	).Scan(&order.ID, &order.UserID, &order.Status, &order.TotalPrice, &order.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get order: %w", err)
	}

	rows, err := r.pool.Query(ctx,
		`SELECT id, product_id, quantity, price FROM order_items WHERE order_id = $1`, id,
	)
	if err != nil {
		return nil, fmt.Errorf("get order items: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var item model.OrderItem
		if err := rows.Scan(&item.ID, &item.ProductID, &item.Quantity, &item.Price); err != nil {
			return nil, fmt.Errorf("scan order item: %w", err)
		}
		item.OrderID = order.ID
		order.Items = append(order.Items, item)
	}
	return order, nil
}

func (r *pgOrderRepo) ListByUserID(ctx context.Context, userID uuid.UUID) ([]model.Order, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, status, total_price, created_at FROM orders WHERE user_id = $1 ORDER BY created_at DESC`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("list orders: %w", err)
	}
	defer rows.Close()

	var orders []model.Order
	for rows.Next() {
		var o model.Order
		o.UserID = userID
		if err := rows.Scan(&o.ID, &o.Status, &o.TotalPrice, &o.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan order: %w", err)
		}
		orders = append(orders, o)
	}
	return orders, nil
}
