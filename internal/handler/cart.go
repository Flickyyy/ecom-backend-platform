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
	cartService    *service.CartService
	productService *service.ProductService
}

func NewCartHandler(cartService *service.CartService, productService *service.ProductService) *CartHandler {
	return &CartHandler{cartService: cartService, productService: productService}
}

func (h *CartHandler) GetCart(c *gin.Context) {
	userID := middleware.GetUserID(c)

	cart, err := h.cartService.GetCart(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	resp := dto.CartResponse{ID: cart.ID, Items: make([]dto.CartItemResponse, 0)}
	for _, item := range cart.Items {
		product, err := h.productService.GetByID(c.Request.Context(), item.ProductID)
		if err != nil {
			continue
		}
		resp.Items = append(resp.Items, dto.CartItemResponse{
			ID:        item.ID,
			ProductID: item.ProductID,
			Name:      product.Name,
			Price:     product.Price,
			Quantity:  item.Quantity,
		})
	}

	c.JSON(http.StatusOK, resp)
}

func (h *CartHandler) AddItem(c *gin.Context) {
	userID := middleware.GetUserID(c)

	var req dto.AddCartItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.cartService.AddItem(c.Request.Context(), userID, req.ProductID, req.Quantity); err != nil {
		if errors.Is(err, service.ErrProductNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "product not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "item added to cart"})
}

func (h *CartHandler) UpdateItem(c *gin.Context) {
	userID := middleware.GetUserID(c)

	itemID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid item ID"})
		return
	}

	var req dto.UpdateCartItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.cartService.UpdateItem(c.Request.Context(), userID, itemID, req.Quantity); err != nil {
		if errors.Is(err, service.ErrCartItemNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "cart item not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "cart item updated"})
}

func (h *CartHandler) DeleteItem(c *gin.Context) {
	userID := middleware.GetUserID(c)

	itemID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid item ID"})
		return
	}

	if err := h.cartService.DeleteItem(c.Request.Context(), userID, itemID); err != nil {
		if errors.Is(err, service.ErrWrongCart) {
			c.JSON(http.StatusNotFound, gin.H{"error": "cart item not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.Status(http.StatusNoContent)
}
