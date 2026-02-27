package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/flicky/go-ecommerce-api/internal/dto"
	"github.com/flicky/go-ecommerce-api/internal/middleware"
	"github.com/flicky/go-ecommerce-api/internal/service"
)

type CartHandler struct {
	svc *service.CartService
}

func NewCartHandler(svc *service.CartService) *CartHandler {
	return &CartHandler{svc: svc}
}

func (h *CartHandler) GetCart(c *gin.Context) {
	cart, err := h.svc.GetCart(c.Request.Context(), middleware.GetUserID(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	items := make([]dto.CartItemResponse, 0, len(cart.Items))
	for _, item := range cart.Items {
		items = append(items, dto.CartItemResponse{
			ID: item.ID, ProductID: item.ProductID, Quantity: item.Quantity,
		})
	}
	c.JSON(http.StatusOK, dto.CartResponse{ID: cart.ID, Items: items})
}

func (h *CartHandler) AddItem(c *gin.Context) {
	var req dto.AddCartItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.svc.AddItem(c.Request.Context(), middleware.GetUserID(c), req.ProductID, req.Quantity); err != nil {
		if errors.Is(err, service.ErrProductNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "product not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"message": "item added"})
}

func (h *CartHandler) UpdateItem(c *gin.Context) {
	itemID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var req dto.UpdateCartItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.svc.UpdateItem(c.Request.Context(), middleware.GetUserID(c), itemID, req.Quantity); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "item updated"})
}

func (h *CartHandler) DeleteItem(c *gin.Context) {
	itemID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	if err := h.svc.DeleteItem(c.Request.Context(), middleware.GetUserID(c), itemID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	c.Status(http.StatusNoContent)
}
