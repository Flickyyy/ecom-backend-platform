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
	List(ctx context.Context, limit, offset int) ([]model.Product, int, error)
	Update(ctx context.Context, product *model.Product) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type pgProductRepo struct{ pool *pgxpool.Pool }

func NewProductRepository(pool *pgxpool.Pool) ProductRepository {
	return &pgProductRepo{pool: pool}
}

func (r *pgProductRepo) Create(ctx context.Context, product *model.Product) error {
	product.ID = uuid.New()
	err := r.pool.QueryRow(ctx,
		`INSERT INTO products (id, name, description, price, stock, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, NOW(), NOW()) RETURNING created_at, updated_at`,
		product.ID, product.Name, product.Description, product.Price, product.Stock,
	).Scan(&product.CreatedAt, &product.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create product: %w", err)
	}
	return nil
}

func (r *pgProductRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.Product, error) {
	p := &model.Product{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, name, description, price, stock, created_at, updated_at FROM products WHERE id = $1`, id,
	).Scan(&p.ID, &p.Name, &p.Description, &p.Price, &p.Stock, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get product: %w", err)
	}
	return p, nil
}

func (r *pgProductRepo) List(ctx context.Context, limit, offset int) ([]model.Product, int, error) {
	var total int
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM products`).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count products: %w", err)
	}

	rows, err := r.pool.Query(ctx,
		`SELECT id, name, description, price, stock, created_at, updated_at
		 FROM products ORDER BY created_at DESC LIMIT $1 OFFSET $2`, limit, offset,
	)
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
	err := r.pool.QueryRow(ctx,
		`UPDATE products SET name=$2, description=$3, price=$4, stock=$5, updated_at=NOW()
		 WHERE id=$1 RETURNING updated_at`,
		product.ID, product.Name, product.Description, product.Price, product.Stock,
	).Scan(&product.UpdatedAt)
	if err != nil {
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
		return fmt.Errorf("product not found")
	}
	return nil
}
