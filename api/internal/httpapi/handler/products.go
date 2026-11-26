package handler

import (
	"context"
	"net/http"
	"regexp"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"github.com/YaelBadillo/product-catalog/api/internal/domain"
	"github.com/YaelBadillo/product-catalog/api/internal/service"
)

const productsPath = "/api/v1/products/"

var uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// ProductService is what the product handlers need from the service layer.
type ProductService interface {
	Create(ctx context.Context, in domain.ProductInput) (domain.Product, error)
	Get(ctx context.Context, id string) (domain.Product, error)
	List(ctx context.Context, p service.ListParams) (service.ProductPage, error)
}

// Products serves the /products routes.
type Products struct {
	svc      ProductService
	validate *validator.Validate
}

// NewProducts returns the product handlers.
func NewProducts(svc ProductService, validate *validator.Validate) *Products {
	return &Products{svc: svc, validate: validate}
}

// Register adds the product routes to g (the /api/v1 group).
func (h *Products) Register(g *gin.RouterGroup) {
	g.GET("/products", h.List)
	g.POST("/products", h.Create)
	g.GET("/products/:id", h.Get)
}

// List handles GET /products: one page and its meta.
func (h *Products) List(c *gin.Context) {
	params, err := ParseListParams(c.Request.URL.Query())
	if err != nil {
		_ = c.Error(err)
		return
	}
	page, err := h.svc.List(c.Request.Context(), params)
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, toListResponse(page, params))
}

// Create handles POST /products: 201 with Location and the product.
func (h *Products) Create(c *gin.Context) {
	var req CreateProductRequest
	if err := DecodeJSON(c, &req, h.validate); err != nil {
		_ = c.Error(err)
		return
	}
	product, err := h.svc.Create(c.Request.Context(), req.toInput())
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.Header("Location", productsPath+product.ID)
	c.JSON(http.StatusCreated, toResponse(product))
}

// Get handles GET /products/:id.
func (h *Products) Get(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	product, err := h.svc.Get(c.Request.Context(), id)
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, toResponse(product))
}

// pathID returns the :id parameter, or reports a RequestError if it is not a
// UUID.
func pathID(c *gin.Context) (string, bool) {
	id := c.Param("id")
	if !uuidPattern.MatchString(id) {
		_ = c.Error(&RequestError{Field: "id", Message: "must be a UUID"})
		return "", false
	}
	return id, true
}
