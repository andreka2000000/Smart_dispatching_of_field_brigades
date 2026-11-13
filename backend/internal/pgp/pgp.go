package pgp

import (
	"time"

	"github.com/andreka2000000/Smart_dispatching_of_field_brigades/backend/internal/model"
	"gorm.io/gorm"
)

// DetectPGP возвращает ID нарядов, которые попали в ПГП, и создает для них группы
func DetectPGP(db *gorm.DB, orders []model.Order) map[uint]bool {
	// Группируем по AccessNodeID
	nodeOrders := make(map[uint][]model.Order)

	for _, o := range orders {
		if o.AccessNodeID != nil && o.Profile == "zltp" {
			nodeOrders[*o.AccessNodeID] = append(nodeOrders[*o.AccessNodeID], o)
		}
	}

	pgpOrders := make(map[uint]bool) // Сюда сохраняем ID заявок, которые ПГП
	now := time.Now()

	for nodeID, ordersInNode := range nodeOrders {
		// Считаем, сколько нарядов пришло за последние 2 часа
		recentCount := 0
		var recentOrders []model.Order

		for _, o := range ordersInNode {
			if now.Sub(o.RegisteredAt) <= 2*time.Hour {
				recentCount++
				recentOrders = append(recentOrders, o)
			}
		}

		// Если 3 и больше — создаем группу ПГП
		if recentCount >= 3 {
			// 1. Создаем FaultGroup в БД
			fg := model.FaultGroup{
				AccessNodeID: nodeID,
				Status:       "detected",
			}
			db.Create(&fg)

			// 2. Помечаем заявки как ПГП и привязываем к группе
			for _, o := range recentOrders {
				pgpOrders[o.ID] = true

				// Обновляем в БД (чтобы фронт видел)
				db.Model(&o).Updates(map[string]interface{}{
					"fault_group_id":    fg.ID,
					"status":            "pgp",
					"unassigned_reason": "PGP_TRANSFERRED",
				})
			}
		}
	}

	return pgpOrders
}
