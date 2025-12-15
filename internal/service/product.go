package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/flicky/go-ecommerce-api/internal/dto"
	"github.com/flicky/go-ecommerce-api/internal/model"
	"github.com/flicky/go-ecommerce-api/internal/repository"
)

var ErrProductNotFound = errors.New("product not found")

type ProductService struct {
	repo repository.ProductRepository
}

func NewProductService(repo repository.ProductRepository) *ProductService {
	return &ProductService{repo: repo}
}

func (s *ProductService) Create(ctx context.Context, req dto.CreateProductRequest) (*dto.ProductResponse, error) {
	product := &model.Product{
		Name: req.Name, Description: req.Description,
		Price: req.Price, Stock: req.Stock,
	}
	if err := s.repo.Create(ctx, product); err != nil {
		return nil, fmt.Errorf("create product: %w", err)
	}
	resp := toProductResponse(product)
	return &resp, nil
}

func (s *ProductService) GetByID(ctx context.Context, id uuid.UUID) (*dto.ProductResponse, error) {
	product, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get product: %w", err)
	}
	if product == nil {
		return nil, ErrProductNotFound
	}
	resp := toProductResponse(product)
	return &resp, nil
}

func (s *ProductService) List(ctx context.Context, page, limit int) (*dto.ProductListResponse, error) {
	offset := (page - 1) * limit
	products, total, err := s.repo.List(ctx, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list products: %w", err)
	}
	items := make([]dto.ProductResponse, len(products))
	for i, p := range products {
		items[i] = toProductResponse(&p)
	}
	return &dto.ProductListResponse{Products: items, Total: total}, nil
}

func (s *ProductService) Update(ctx context.Context, id uuid.UUID, req dto.UpdateProductRequest) (*dto.ProductResponse, error) {
	product, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get product: %w", err)
	}
	if product == nil {
		return nil, ErrProductNotFound
	}

	product.Name = req.Name
	product.Description = req.Description
	product.Price = req.Price
	product.Stock = req.Stock

	if err := s.repo.Update(ctx, product); err != nil {
		return nil, fmt.Errorf("update product: %w", err)
	}
	resp := toProductResponse(product)
	return &resp, nil
}

func (s *ProductService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}

func toProductResponse(p *model.Product) dto.ProductResponse {
	return dto.ProductResponse{
		ID: p.ID, Name: p.Name, Description: p.Description,
		Price: p.Price, Stock: p.Stock, CreatedAt: p.CreatedAt,
	}
}
