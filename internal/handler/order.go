package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/flicky/go-ecommerce-api/internal/dto"
	"github.com/flicky/go-ecommerce-api/internal/middleware"
	"github.com/flicky/go-ecommerce-api/internal/model"
	"github.com/flicky/go-ecommerce-api/internal/service"
)

type OrderHandler struct {
	svc *service.OrderService
}

func NewOrderHandler(svc *service.OrderService) *OrderHandler {
	return &OrderHandler{svc: svc}
}

func (h *OrderHandler) CreateOrder(c *gin.Context) {
	order, err := h.svc.CreateOrder(c.Request.Context(), middleware.GetUserID(c))
	if err != nil {
		if errors.Is(err, service.ErrEmptyCart) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "cart is empty"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	c.JSON(http.StatusCreated, toOrderResponse(order))
}

func (h *OrderHandler) ListOrders(c *gin.Context) {
	orders, err := h.svc.ListByUserID(c.Request.Context(), middleware.GetUserID(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	resp := make([]dto.OrderResponse, len(orders))
	for i := range orders {
		resp[i] = toOrderResponse(&orders[i])
	}
	c.JSON(http.StatusOK, resp)
}

func (h *OrderHandler) GetOrder(c *gin.Context) {
	orderID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	order, err := h.svc.GetByID(c.Request.Context(), orderID, middleware.GetUserID(c))
	if err != nil {
		if errors.Is(err, service.ErrOrderNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "order not found"})
			return
		}
		if errors.Is(err, service.ErrOrderAccessDenied) {
			c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	c.JSON(http.StatusOK, toOrderResponse(order))
}

func toOrderResponse(o *model.Order) dto.OrderResponse {
	items := make([]dto.OrderItemResponse, len(o.Items))
	for i, item := range o.Items {
		items[i] = dto.OrderItemResponse{
			ProductID: item.ProductID, Quantity: item.Quantity, Price: item.Price,
		}
	}
	return dto.OrderResponse{
		ID: o.ID, Status: o.Status, TotalPrice: o.TotalPrice,
		Items: items, CreatedAt: o.CreatedAt,
	}
}
