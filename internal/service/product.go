package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"github.com/flicky/go-ecommerce-api/internal/dto"
	"github.com/flicky/go-ecommerce-api/internal/model"
	"github.com/flicky/go-ecommerce-api/internal/repository"
)

var ErrProductNotFound = errors.New("product not found")

const productCacheTTL = 60 * time.Second

type ProductService struct {
	productRepo repository.ProductRepository
	redisClient *redis.Client
}

func NewProductService(productRepo repository.ProductRepository, redisClient *redis.Client) *ProductService {
	return &ProductService{productRepo: productRepo, redisClient: redisClient}
}

func (s *ProductService) Create(ctx context.Context, req dto.CreateProductRequest) (*dto.ProductResponse, error) {
	product := &model.Product{
		Name:        req.Name,
		Description: req.Description,
		Price:       req.Price,
		Stock:       req.Stock,
	}
	if err := s.productRepo.Create(ctx, product); err != nil {
		return nil, fmt.Errorf("create product: %w", err)
	}
	resp := toProductResponse(product)
	return &resp, nil
}

func (s *ProductService) GetByID(ctx context.Context, id uuid.UUID) (*dto.ProductResponse, error) {
	cacheKey := "product:" + id.String()

	// Try cache
	if s.redisClient != nil {
		if cached, err := s.redisClient.Get(ctx, cacheKey).Result(); err == nil {
			var resp dto.ProductResponse
			if json.Unmarshal([]byte(cached), &resp) == nil {
				return &resp, nil
			}
		}
	}

	product, err := s.productRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get product: %w", err)
	}
	if product == nil {
		return nil, ErrProductNotFound
	}

	resp := toProductResponse(product)

	// Write to cache
	if s.redisClient != nil {
		if data, err := json.Marshal(resp); err == nil {
			s.redisClient.Set(ctx, cacheKey, data, productCacheTTL)
		}
	}

	return &resp, nil
}

func (s *ProductService) List(ctx context.Context, req dto.ListProductsRequest) (*dto.ProductListResponse, error) {
	offset := (req.Page - 1) * req.Limit
	products, total, err := s.productRepo.List(ctx, req.Limit, offset, req.Search, req.Sort, req.Order)
	if err != nil {
		return nil, fmt.Errorf("list products: %w", err)
	}

	var items []dto.ProductResponse
	for _, p := range products {
		items = append(items, toProductResponse(&p))
	}

	return &dto.ProductListResponse{Products: items, Total: total, Page: req.Page, Limit: req.Limit}, nil
}

func (s *ProductService) Update(ctx context.Context, id uuid.UUID, req dto.UpdateProductRequest) (*dto.ProductResponse, error) {
	product, err := s.productRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get product: %w", err)
	}
	if product == nil {
		return nil, ErrProductNotFound
	}

	if req.Name != nil {
		product.Name = *req.Name
	}
	if req.Description != nil {
		product.Description = *req.Description
	}
	if req.Price != nil {
		product.Price = *req.Price
	}
	if req.Stock != nil {
		product.Stock = *req.Stock
	}

	if err := s.productRepo.Update(ctx, product); err != nil {
		return nil, fmt.Errorf("update product: %w", err)
	}

	s.invalidateCache(ctx, id)
	resp := toProductResponse(product)
	return &resp, nil
}

func (s *ProductService) Delete(ctx context.Context, id uuid.UUID) error {
	if err := s.productRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete product: %w", err)
	}
	s.invalidateCache(ctx, id)
	return nil
}

func (s *ProductService) invalidateCache(ctx context.Context, id uuid.UUID) {
	if s.redisClient != nil {
		s.redisClient.Del(ctx, "product:"+id.String())
	}
}

func toProductResponse(p *model.Product) dto.ProductResponse {
	return dto.ProductResponse{
		ID:          p.ID,
		Name:        p.Name,
		Description: p.Description,
		Price:       p.Price,
		Stock:       p.Stock,
		CreatedAt:   p.CreatedAt,
		UpdatedAt:   p.UpdatedAt,
	}
}
