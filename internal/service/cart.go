package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/flicky/go-ecommerce-api/internal/model"
	"github.com/flicky/go-ecommerce-api/internal/repository"
)

var (
	ErrCartNotFound     = errors.New("cart not found")
	ErrCartItemNotFound = errors.New("cart item not found")
	ErrWrongCart        = errors.New("item does not belong to this cart")
)

type CartService struct {
	cartRepo    repository.CartRepository
	productRepo repository.ProductRepository
}

func NewCartService(cartRepo repository.CartRepository, productRepo repository.ProductRepository) *CartService {
	return &CartService{cartRepo: cartRepo, productRepo: productRepo}
}

func (s *CartService) GetCart(ctx context.Context, userID uuid.UUID) (*model.Cart, error) {
	cart, err := s.cartRepo.GetOrCreateCart(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get or create cart: %w", err)
	}
	return s.cartRepo.GetCartWithItems(ctx, cart.ID)
}

func (s *CartService) AddItem(ctx context.Context, userID, productID uuid.UUID, quantity int) error {
	product, err := s.productRepo.GetByID(ctx, productID)
	if err != nil {
		return fmt.Errorf("get product: %w", err)
	}
	if product == nil {
		return ErrProductNotFound
	}

	cart, err := s.cartRepo.GetOrCreateCart(ctx, userID)
	if err != nil {
		return fmt.Errorf("get or create cart: %w", err)
	}

	return s.cartRepo.AddItem(ctx, &model.CartItem{
		CartID:    cart.ID,
		ProductID: productID,
		Quantity:  quantity,
	})
}

func (s *CartService) UpdateItem(ctx context.Context, userID, itemID uuid.UUID, quantity int) error {
	cart, err := s.cartRepo.GetOrCreateCart(ctx, userID)
	if err != nil {
		return fmt.Errorf("get cart: %w", err)
	}

	cartWithItems, err := s.cartRepo.GetCartWithItems(ctx, cart.ID)
	if err != nil {
		return fmt.Errorf("get cart items: %w", err)
	}

	if !containsItem(cartWithItems.Items, itemID) {
		return ErrCartItemNotFound
	}

	return s.cartRepo.UpdateItem(ctx, &model.CartItem{ID: itemID, Quantity: quantity})
}

func (s *CartService) DeleteItem(ctx context.Context, userID, itemID uuid.UUID) error {
	cart, err := s.cartRepo.GetOrCreateCart(ctx, userID)
	if err != nil {
		return fmt.Errorf("get cart: %w", err)
	}

	cartWithItems, err := s.cartRepo.GetCartWithItems(ctx, cart.ID)
	if err != nil {
		return fmt.Errorf("get cart items: %w", err)
	}

	if !containsItem(cartWithItems.Items, itemID) {
		return ErrWrongCart
	}

	return s.cartRepo.DeleteItem(ctx, itemID)
}

func containsItem(items []model.CartItem, id uuid.UUID) bool {
	for _, item := range items {
		if item.ID == id {
			return true
		}
	}
	return false
}
