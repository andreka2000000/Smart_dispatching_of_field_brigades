package api

import (
	"net/http"

	models "github.com/andreka2000000/Smart_dispatching_of_field_brigades/backend/internal/model"
	"github.com/andreka2000000/Smart_dispatching_of_field_brigades/backend/internal/store"
	"github.com/gin-gonic/gin"
)

// POST /api/data/service-centers
func CreateServiceCenter(c *gin.Context) {
	var sc models.ServiceCenter
	if err := c.ShouldBindJSON(&sc); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	store.DB.Create(&sc)
	c.JSON(http.StatusOK, sc)
}

// POST /api/data/access-nodes
func CreateAccessNode(c *gin.Context) {
	var an models.AccessNode
	if err := c.ShouldBindJSON(&an); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	store.DB.Create(&an)
	c.JSON(http.StatusOK, an)
}

// POST /api/data/work-types
func CreateWorkType(c *gin.Context) {
	var wt models.WorkType
	if err := c.ShouldBindJSON(&wt); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	store.DB.Create(&wt)
	c.JSON(http.StatusOK, wt)
}

// POST /api/data/staff
func CreateStaff(c *gin.Context) {
	var s models.Staff
	if err := c.ShouldBindJSON(&s); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	store.DB.Create(&s)
	c.JSON(http.StatusOK, s)
}

// POST /api/data/orders  (batch загрузка нарядов)
func CreateOrders(c *gin.Context) {
	var orders []models.Order
	if err := c.ShouldBindJSON(&orders); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	store.DB.Create(&orders)
	c.JSON(http.StatusOK, gin.H{"created": len(orders)})
}

// GET /api/data/staff
func GetStaff(c *gin.Context) {
	var staff []models.Staff
	store.DB.Preload("ServiceCenter").Find(&staff)
	c.JSON(http.StatusOK, staff)
}

// GET /api/data/orders
func GetOrders(c *gin.Context) {
	var orders []models.Order
	store.DB.Preload("WorkType").Find(&orders)
	c.JSON(http.StatusOK, orders)
}
