package api

import (
	"net/http"
	"time"

	"github.com/andreka2000000/Smart_dispatching_of_field_brigades/backend/internal/model"
	models "github.com/andreka2000000/Smart_dispatching_of_field_brigades/backend/internal/model"
	"github.com/andreka2000000/Smart_dispatching_of_field_brigades/backend/internal/planner"
	"github.com/andreka2000000/Smart_dispatching_of_field_brigades/backend/internal/store"
	"github.com/gin-gonic/gin"
)

func BuildPlan(c *gin.Context) {
	var orders []model.Order
	var staff []model.Staff

	store.DB.Where("status = ?", "pending").Preload("WorkType").Find(&orders)
	store.DB.Preload("ServiceCenter").Find(&staff)

	if len(orders) == 0 || len(staff) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Нет данных для планирования"})
		return
	}

	planDate := time.Now()
	routes, unassigned := planner.BuildPlan(orders, staff, planDate)

	plan, err := planner.SavePlanToDB(store.DB, routes, unassigned, planDate, "initial")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка сохранения: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "План успешно построен и сохранен",
		"plan_id": plan.ID,
		"routes":  routes,
	})
}

// GET /api/plan  — получить актуальный план
func GetPlan(c *gin.Context) {
	var plan models.Plan
	err := store.DB.
		Preload("Routes").
		Preload("Routes.Staff").
		Preload("Routes.Visits").
		Preload("Routes.Visits.Order").
		Order("version DESC").
		First(&plan).Error
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "План ещё не построен"})
		return
	}
	c.JSON(http.StatusOK, plan)
}

// POST /api/events  — принять событие (новый наряд, отмена, задержка)
type EventPayload struct {
	Type    string                 `json:"type"` // "new_order" | "cancel" | "delay" | "staff_unavailable"
	Payload map[string]interface{} `json:"payload"`
}

func HandleEvent(c *gin.Context) {
	var ev EventPayload
	if err := c.ShouldBindJSON(&ev); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// TODO: вызов planner.Replan(...)
	c.JSON(http.StatusOK, gin.H{"message": "Событие принято", "type": ev.Type})
}
