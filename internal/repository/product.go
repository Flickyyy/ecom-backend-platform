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

type ProductRepository interface {
	Create(ctx context.Context, product *model.Product) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.Product, error)
	List(ctx context.Context, limit, offset int, search, sort, order string) ([]model.Product, int, error)
	Update(ctx context.Context, product *model.Product) error
	Delete(ctx context.Context, id uuid.UUID) error
	DecrementStock(ctx context.Context, tx pgx.Tx, productID uuid.UUID, quantity int) error
}

type pgProductRepo struct{ pool *pgxpool.Pool }

func NewProductRepository(pool *pgxpool.Pool) ProductRepository {
	return &pgProductRepo{pool: pool}
}

func (r *pgProductRepo) Create(ctx context.Context, product *model.Product) error {
	product.ID = uuid.New()
	query := `INSERT INTO products (id, name, description, price, stock, created_at, updated_at)
			  VALUES ($1, $2, $3, $4, $5, NOW(), NOW()) RETURNING created_at, updated_at`
	err := r.pool.QueryRow(ctx, query,
		product.ID, product.Name, product.Description, product.Price, product.Stock,
	).Scan(&product.CreatedAt, &product.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create product: %w", err)
	}
	return nil
}

func (r *pgProductRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.Product, error) {
	query := `SELECT id, name, description, price, stock, created_at, updated_at FROM products WHERE id = $1`
	p := &model.Product{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&p.ID, &p.Name, &p.Description, &p.Price, &p.Stock, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get product: %w", err)
	}
	return p, nil
}

func (r *pgProductRepo) List(ctx context.Context, limit, offset int, search, sort, order string) ([]model.Product, int, error) {
	allowedSorts := map[string]bool{"name": true, "price": true, "created_at": true}
	if !allowedSorts[sort] {
		sort = "created_at"
	}
	if order != "asc" && order != "desc" {
		order = "desc"
	}

	var total int
	countQ := `SELECT COUNT(*) FROM products WHERE ($1 = '' OR name ILIKE '%' || $1 || '%' OR description ILIKE '%' || $1 || '%')`
	if err := r.pool.QueryRow(ctx, countQ, search).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count products: %w", err)
	}

	query := fmt.Sprintf(`SELECT id, name, description, price, stock, created_at, updated_at
		FROM products
		WHERE ($1 = '' OR name ILIKE '%%' || $1 || '%%' OR description ILIKE '%%' || $1 || '%%')
		ORDER BY %s %s LIMIT $2 OFFSET $3`, sort, order)

	rows, err := r.pool.Query(ctx, query, search, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list products: %w", err)
	}
	defer rows.Close()

	var products []model.Product
	for rows.Next() {
		var p model.Product
		if err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.Price, &p.Stock, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, 0, fmt.Errorf("scan product: %w", err)
		}
		products = append(products, p)
	}
	return products, total, nil
}

func (r *pgProductRepo) Update(ctx context.Context, product *model.Product) error {
	query := `UPDATE products SET name=$2, description=$3, price=$4, stock=$5, updated_at=NOW()
			  WHERE id=$1 RETURNING updated_at`
	err := r.pool.QueryRow(ctx, query,
		product.ID, product.Name, product.Description, product.Price, product.Stock,
	).Scan(&product.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		return fmt.Errorf("update product: %w", err)
	}
	return nil
}

func (r *pgProductRepo) Delete(ctx context.Context, id uuid.UUID) error {
	ct, err := r.pool.Exec(ctx, `DELETE FROM products WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete product: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *pgProductRepo) DecrementStock(ctx context.Context, tx pgx.Tx, productID uuid.UUID, quantity int) error {
	ct, err := tx.Exec(ctx,
		`UPDATE products SET stock = stock - $2, updated_at = NOW() WHERE id = $1 AND stock >= $2`,
		productID, quantity,
	)
	if err != nil {
		return fmt.Errorf("decrement stock: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("insufficient stock for product %s", productID)
	}
	return nil
}
