package pgp

import (
	"time"

	"github.com/andreka2000000/Smart_dispatching_of_field_brigades/backend/internal/model"
)

// DetectPGP возвращает ID нарядов, которые попали в ПГП (групповые проблемы)
func DetectPGP(orders []model.Order, window time.Duration, minOrders int) []uint {
	// Группируем по AccessNodeID
	nodeOrders := make(map[uint][]model.Order)
	for _, o := range orders {
		if o.AccessNodeID != nil && o.Profile == "zltp" { // ПГП ищем только по неисправностям
			nodeOrders[*o.AccessNodeID] = append(nodeOrders[*o.AccessNodeID], o)
		}
	}

	var pgpOrderIDs []uint
	now := time.Now()

	for _, ordersInNode := range nodeOrders {
		// Считаем, сколько нарядов пришло за последние N часов
		recentCount := 0
		var recentIDs []uint
		for _, o := range ordersInNode {
			if now.Sub(o.RegisteredAt) <= window {
				recentCount++
				recentIDs = append(recentIDs, o.ID)
			}
		}

		// Если порог превышен — добавляем в список ПГП
		if recentCount >= minOrders {
			pgpOrderIDs = append(pgpOrderIDs, recentIDs...)
		}
	}

	return pgpOrderIDs
}
