package dto

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// Auth

type RegisterRequest struct {
	Email     string `json:"email" binding:"required,email"`
	Password  string `json:"password" binding:"required,min=8"`
	FirstName string `json:"first_name" binding:"required"`
	LastName  string `json:"last_name" binding:"required"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type AuthResponse struct {
	Token string       `json:"token"`
	User  UserResponse `json:"user"`
}

type UserResponse struct {
	ID        uuid.UUID `json:"id"`
	Email     string    `json:"email"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	Role      string    `json:"role"`
}

// Product

type CreateProductRequest struct {
	Name        string          `json:"name" binding:"required"`
	Description string          `json:"description"`
	Price       decimal.Decimal `json:"price" binding:"required"`
	Stock       int             `json:"stock" binding:"required,min=0"`
}

type UpdateProductRequest struct {
	Name        string          `json:"name" binding:"required"`
	Description string          `json:"description"`
	Price       decimal.Decimal `json:"price" binding:"required"`
	Stock       int             `json:"stock" binding:"required,min=0"`
}

type ProductResponse struct {
	ID          uuid.UUID       `json:"id"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Price       decimal.Decimal `json:"price"`
	Stock       int             `json:"stock"`
	CreatedAt   time.Time       `json:"created_at"`
}

type ProductListResponse struct {
	Products []ProductResponse `json:"products"`
	Total    int               `json:"total"`
}

// Cart

type AddCartItemRequest struct {
	ProductID uuid.UUID `json:"product_id" binding:"required"`
	Quantity  int       `json:"quantity" binding:"required,min=1"`
}

type UpdateCartItemRequest struct {
	Quantity int `json:"quantity" binding:"required,min=1"`
}

type CartResponse struct {
	ID    uuid.UUID          `json:"id"`
	Items []CartItemResponse `json:"items"`
}

type CartItemResponse struct {
	ID        uuid.UUID `json:"id"`
	ProductID uuid.UUID `json:"product_id"`
	Quantity  int       `json:"quantity"`
}

// Order

type OrderResponse struct {
	ID         uuid.UUID           `json:"id"`
	Status     string              `json:"status"`
	TotalPrice decimal.Decimal     `json:"total_price"`
	Items      []OrderItemResponse `json:"items"`
	CreatedAt  time.Time           `json:"created_at"`
}

type OrderItemResponse struct {
	ProductID uuid.UUID       `json:"product_id"`
	Quantity  int             `json:"quantity"`
	Price     decimal.Decimal `json:"price"`
}
