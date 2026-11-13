package planner

import (
	"sort"
	"time"

	"gorm.io/gorm"

	"github.com/andreka2000000/Smart_dispatching_of_field_brigades/backend/internal/geo"
	"github.com/andreka2000000/Smart_dispatching_of_field_brigades/backend/internal/model"
)

type Route struct {
	Staff  model.Staff
	Visits []Visit
}

type Visit struct {
	Order          model.Order
	PlannedArrival time.Time
	PlannedEnd     time.Time
	TravelMin      int
	IsAtRisk       bool
	RiskReason     string
	Recommendation string
}

// BuildPlan строит базовый план на день (жадная вставка)
func BuildPlan(orders []model.Order, staff []model.Staff, planDate time.Time) ([]Route, []model.Order) {
	var unassigned []model.Order
	var routes []Route

	// Создаем маршруты для каждого сотрудника
	for _, s := range staff {
		routes = append(routes, Route{Staff: s, Visits: []Visit{}})
	}

	// Сортируем наряды: сначала те, у кого слот раньше (чтобы не опаздывать)
	sort.Slice(orders, func(i, j int) bool {
		return orders[i].SlotStart.Before(orders[j].SlotStart)
	})

	// Пытаемся вставить каждый наряд
	for _, order := range orders {
		assigned := false

		for i := range routes {
			staff := routes[i].Staff

			// 1. Проверка допусков (профиль, технология, высота)
			if !isQualified(staff, order) {
				continue
			}

			// 2. Пытаемся вставить в маршрут
			if inserted, visit := tryInsert(&routes[i], order, planDate); inserted {
				routes[i].Visits = append(routes[i].Visits, visit)
				assigned = true
				break // Наряд назначен, идем к следующему
			}
		}

		// Если ни один сотрудник не подошел — наряд в нераспределенные
		if !assigned {
			order.Status = "unassigned"
			order.UnassignedReason = "NO_QUALIFIED_STAFF_OR_TIME"
			unassigned = append(unassigned, order)
		}
	}

	// Сортируем визиты внутри маршрутов по времени
	for i := range routes {
		sort.Slice(routes[i].Visits, func(a, b int) bool {
			return routes[i].Visits[a].PlannedArrival.Before(routes[i].Visits[b].PlannedArrival)
		})
	}

	return routes, unassigned
}

// isQualified проверяет, подходит ли сотрудник под наряд
func isQualified(staff model.Staff, order model.Order) bool {
	// Проверка профиля (упрощенно, в реале это JSON-массивы)
	// TODO: распарсить JSON в массив строк
	if order.Profile != staff.Profiles && staff.Profiles != "both" && order.Profile != "both" {
		// Здесь временная заглушка, пока не распарсим JSON
	}

	// Проверка допуска к высоте
	if order.RequiresHeight && !staff.HeightAllowed {
		return false
	}

	return true
}

// tryInsert пытается вставить наряд в маршрут с учетом времени и слотов
func tryInsert(route *Route, order model.Order, planDate time.Time) (bool, Visit) {
	currentTime := geo.ParseTime(planDate, route.Staff.ShiftStart)
	currentLat := route.Staff.ServiceCenter.Lat
	currentLon := route.Staff.ServiceCenter.Lon

	for _, v := range route.Visits {
		currentTime = v.PlannedEnd
		currentLat = v.Order.Lat
		currentLon = v.Order.Lon
	}

	travelMin := geo.TravelTimeMin(currentLat, currentLon, order.Lat, order.Lon)
	arrivalTime := currentTime.Add(time.Duration(travelMin) * time.Minute)

	if arrivalTime.Before(order.SlotStart) {
		arrivalTime = order.SlotStart
	}
	if arrivalTime.After(order.SlotEnd) {
		return false, Visit{}
	}

	endTime := arrivalTime.Add(time.Duration(order.WorkType.DurationMin) * time.Minute)
	shiftEnd := geo.ParseTime(planDate, route.Staff.ShiftEnd)
	if endTime.After(shiftEnd) {
		return false, Visit{}
	}

	// --- НОВАЯ ЛОГИКА РИСК-МОНИТОРА ---
	isAtRisk := false
	riskReason := ""
	recommendation := ""

	// Если приезжаем в слот, но до конца слота остается меньше 30 минут — это риск
	// (Математика: время прибытия + норматив работы > конец слота)
	if endTime.After(order.SlotEnd.Add(-30 * time.Minute)) {
		isAtRisk = true
		riskReason = "Визит может выйти за границы слота. Высокий риск опоздания."
		recommendation = "transfer" // Рекомендуем передать другому сотруднику
	}

	return true, Visit{
		Order:          order,
		PlannedArrival: arrivalTime,
		PlannedEnd:     endTime,
		TravelMin:      travelMin,
		IsAtRisk:       isAtRisk,
		RiskReason:     riskReason,
		Recommendation: recommendation,
	}
}

// SavePlanToDB сохраняет маршруты и визиты в базу данных
func SavePlanToDB(db *gorm.DB, routes []Route, unassigned []model.Order, planDate time.Time, reason string) (*model.Plan, error) {
	plan := model.Plan{
		Version:  int(time.Now().Unix()),
		PlanDate: planDate,
		Reason:   reason,
	}

	if err := db.Create(&plan).Error; err != nil {
		return nil, err
	}

	// Сохраняем маршруты и визиты
	for _, r := range routes {
		route := model.Route{
			PlanID:  plan.ID,
			StaffID: r.Staff.ID,
		}
		db.Create(&route)

		for i, v := range r.Visits {
			visit := model.Visit{
				RouteID:        route.ID,
				OrderID:        &v.Order.ID,
				Sequence:       i + 1,
				PlannedArrival: v.PlannedArrival,
				PlannedEnd:     v.PlannedEnd,
				VisitType:      "order",
				IsAtRisk:       v.IsAtRisk,
				RiskReason:     v.RiskReason,
				Recommendation: v.Recommendation,
			}
			db.Create(&visit)
		}
	}

	// Не забудь добавить импорт "gorm.io/gorm" в начале файла!
	return &plan, nil
}
