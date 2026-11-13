package model

import (
	"time"

	"gorm.io/gorm"
)

// ============ Справочники ============

// ServiceCenter — сервисный центр (депо сотрудника)
type ServiceCenter struct {
	gorm.Model
	Name      string  `json:"name" gorm:"not null"`
	Address   string  `json:"address"`
	Lat       float64 `json:"lat"`
	Lon       float64 `json:"lon"`
	OpenTime  string  `json:"open_time"`  // "09:00"
	CloseTime string  `json:"close_time"` // "18:00"
}

// AccessNode — узел доступа (ОЛТ, коммутатор) для выявления ПГП
type AccessNode struct {
	gorm.Model
	Name   string  `json:"name"`
	Lat    float64 `json:"lat"`
	Lon    float64 `json:"lon"`
	Houses string  `json:"houses"` // JSON-массив адресов, которые он обслуживает
}

// WorkType — справочник типов работ
type WorkType struct {
	gorm.Model
	Code           string `json:"code" gorm:"uniqueIndex"` // "install_pon_mkd"
	Name           string `json:"name"`
	Profile        string `json:"profile"`         // "installation" | "zltp" | "both"
	DurationMin    int    `json:"duration_min"`    // норматив в минутах
	DeadlineHours  int    `json:"deadline_hours"`  // контрольный срок в часах (0 = только слот)
	RequiresHeight bool   `json:"requires_height"` // нужен допуск к высоте (ВЛС)
}

// ============ Сотрудник ============

// Staff — выездной сотрудник
type Staff struct {
	gorm.Model
	ExternalID      string        `json:"external_id" gorm:"uniqueIndex"` // ID из staff.json
	Name            string        `json:"name"`
	ServiceCenterID uint          `json:"service_center_id"`
	ServiceCenter   ServiceCenter `json:"service_center" gorm:"foreignKey:ServiceCenterID"`

	// Смена
	ShiftStart string `json:"shift_start"` // "09:00"
	ShiftEnd   string `json:"shift_end"`   // "18:00"
	LunchStart string `json:"lunch_start"` // "13:00"
	LunchEnd   string `json:"lunch_end"`   // "14:00"

	// Допуски
	Profiles      string `json:"profiles"`       // JSON: ["installation", "zltp"]
	Technologies  string `json:"technologies"`   // JSON: ["PON", "FTTB"]
	HeightAllowed bool   `json:"height_allowed"` // допуск к работам на высоте

	// Рюкзак АО (текущие остатки)
	BagONT    int `json:"bag_ont"`
	BagRouter int `json:"bag_router"`
	BagTV     int `json:"bag_tv"`

	IsActive bool `json:"is_active" gorm:"default:true"`
}

// ============ Наряд ============

// Order — наряд (заявка)
type Order struct {
	gorm.Model
	ExternalID string `json:"external_id" gorm:"uniqueIndex"` // ID из orders_train.json

	// Адрес и координаты
	Address string  `json:"address"`
	Lat     float64 `json:"lat"`
	Lon     float64 `json:"lon"`

	// Тип работ
	WorkTypeID uint     `json:"work_type_id"`
	WorkType   WorkType `json:"work_type" gorm:"foreignKey:WorkTypeID"`
	Technology string   `json:"technology"` // "PON" | "FTTB" | "xDSL" | "DOCSIS"
	Profile    string   `json:"profile"`    // "installation" | "zltp" | "both"

	// Слот (2 часа)
	SlotStart time.Time `json:"slot_start"`
	SlotEnd   time.Time `json:"slot_end"`

	// Контрольный срок
	RegisteredAt time.Time  `json:"registered_at"`
	Deadline     *time.Time `json:"deadline,omitempty"`

	// Оборудование, требуемое на наряде
	NeedsONT    bool `json:"needs_ont"`
	NeedsRouter bool `json:"needs_router"`
	NeedsTV     bool `json:"needs_tv"`

	// Признаки
	IsRepeat       bool `json:"is_repeat"`       // повторное обращение
	RequiresHeight bool `json:"requires_height"` // ВЛС

	// Узел доступа (для ПГП)
	AccessNodeID *uint       `json:"access_node_id,omitempty"`
	AccessNode   *AccessNode `json:"access_node,omitempty" gorm:"foreignKey:AccessNodeID"`

	// Статус
	Status string `json:"status" gorm:"default:'pending'"`
	// "pending" | "assigned" | "in_progress" | "done" | "unassigned" | "pgp"

	// Причина, если нераспределён
	UnassignedReason string `json:"unassigned_reason,omitempty"`
	// "NO_QUALIFIED_STAFF" | "NO_SHIFT_TIME" | "SLOT_UNREACHABLE" | "NO_EQUIPMENT" | "PGP_TRANSFERRED" | "CAPACITY_EXCEEDED"

	// ПГП
	FaultGroupID *uint       `json:"fault_group_id,omitempty"`
	FaultGroup   *FaultGroup `json:"fault_group,omitempty" gorm:"foreignKey:FaultGroupID"`
}

// ============ ПГП ============

// FaultGroup — группа ПГП (несколько заявок с одного узла)
type FaultGroup struct {
	gorm.Model
	AccessNodeID uint       `json:"access_node_id"`
	AccessNode   AccessNode `json:"access_node" gorm:"foreignKey:AccessNodeID"`
	Status       string     `json:"status" gorm:"default:'detected'"` // "detected" | "transferred"
	Orders       []Order    `json:"orders,omitempty" gorm:"foreignKey:FaultGroupID"`
}

// ============ План смены ============

// Plan — версия плана на день
type Plan struct {
	gorm.Model
	Version    int       `json:"version"`
	PlanDate   time.Time `json:"plan_date"` // какой день планируем
	Reason     string    `json:"reason"`    // "initial" | "event_new_order" | "event_cancel" | ...
	Routes     []Route   `json:"routes,omitempty" gorm:"foreignKey:PlanID"`
	Unassigned []Order   `json:"-" gorm:"-"` // нераспределенные (не храним в БД как связь, просто вычисляем)

	// Метрики
	InSlotPercent     float64 `json:"in_slot_percent"`
	InDeadlinePercent float64 `json:"in_deadline_percent"`
	TotalTravelMin    int     `json:"total_travel_min"`
	LoadCV            float64 `json:"load_cv"` // коэффициент вариации загрузки
}

// Route — маршрут одного сотрудника в рамках плана
type Route struct {
	gorm.Model
	PlanID  uint    `json:"plan_id"`
	StaffID uint    `json:"staff_id"`
	Staff   Staff   `json:"staff" gorm:"foreignKey:StaffID"`
	Visits  []Visit `json:"visits,omitempty" gorm:"foreignKey:RouteID"`

	TravelMin int `json:"travel_min"` // суммарное время в пути
	WorkMin   int `json:"work_min"`   // суммарное время работ
	TotalMin  int `json:"total_min"`  // весь маршрут
}

// Visit — один визит в маршруте
type Visit struct {
	gorm.Model
	RouteID uint   `json:"route_id"`
	OrderID *uint  `json:"order_id,omitempty"` // null, если это заезд в СЦ
	Order   *Order `json:"order,omitempty" gorm:"foreignKey:OrderID"`

	Sequence int `json:"sequence"` // порядок визита в маршруте (0 = старт в СЦ)

	// Плановые времена
	PlannedArrival time.Time `json:"planned_arrival"`
	PlannedEnd     time.Time `json:"planned_end"`

	// Тип визита
	VisitType string `json:"visit_type"`
	// "order" (наряд) | "service_center" (заезд за АО) | "lunch" (обед)

	// Риски
	IsAtRisk       bool   `json:"is_at_risk"`
	RiskReason     string `json:"risk_reason,omitempty"`
	Recommendation string `json:"recommendation,omitempty"` // "reschedule" | "transfer" | ""
}
