package model

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type User struct {
	ID        uuid.UUID
	Email     string
	Password  string
	FirstName string
	LastName  string
	Role      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Product struct {
	ID          uuid.UUID
	Name        string
	Description string
	Price       decimal.Decimal
	Stock       int
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type Cart struct {
	ID     uuid.UUID
	UserID uuid.UUID
	Items  []CartItem
}

type CartItem struct {
	ID        uuid.UUID
	CartID    uuid.UUID
	ProductID uuid.UUID
	Quantity  int
}

type Order struct {
	ID         uuid.UUID
	UserID     uuid.UUID
	Status     string
	TotalPrice decimal.Decimal
	Items      []OrderItem
	CreatedAt  time.Time
}

type OrderItem struct {
	ID        uuid.UUID
	OrderID   uuid.UUID
	ProductID uuid.UUID
	Quantity  int
	Price     decimal.Decimal
}

type OrderMessage struct {
	OrderID uuid.UUID `json:"order_id"`
	UserID  uuid.UUID `json:"user_id"`
}
