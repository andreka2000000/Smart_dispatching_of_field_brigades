package application

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	c "magistral/backend/internal/constraints"
	"magistral/backend/internal/dto"
	"magistral/backend/internal/storage/postgres"
	"time"
)

type Approval struct {
	Date       string           `json:"date"`
	BuildingID string           `json:"building_id"`
	NodeID     string           `json:"access_node_id"`
	WorkType   dto.WorkTypeCode `json:"work_type_code"`
	Slot       dto.TimeWindow   `json:"slot"`
	Equipment  dto.Equipment    `json:"required_equipment"`
	Note       string           `json:"decision_note"`
}

func (s *Service) Approve(ctx context.Context, id string, input Approval) (dto.EventResult, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	var result dto.EventResult
	e := s.Store.WithDayLock(ctx, "approval:"+id, func() error {
		var status, service string
		e := s.Store.Pool.QueryRow(ctx, "SELECT status,service FROM service_requests WHERE id=$1", id).Scan(&status, &service)
		if errors.Is(e, pgx.ErrNoRows) {
			return &Error{404, "REQUEST_NOT_FOUND", "Обращение не найдено."}
		}
		if e != nil {
			return e
		}
		if status == "APPROVED" {
			var raw []byte
			var same bool
			e = s.Store.Pool.QueryRow(ctx, "SELECT l.result,coalesce(d.generation=l.generation,false) FROM request_orders l LEFT JOIN days d ON d.date::text=l.day WHERE l.request_id=$1", id).Scan(&raw, &same)
			if e != nil {
				return e
			}
			if !same {
				return &Error{409, "REQUEST_DAY_REPLACED", "Учебные данные этого наряда заменены. Повторное одобрение запрещено."}
			}
			if e = json.Unmarshal(raw, &result); e != nil {
				return e
			}
			result.Duplicate = true
			return nil
		}
		if status != "NEW" && status != "IN_REVIEW" {
			return &Error{409, "REQUEST_CHANGED", "Обращение уже закрыто или отклонено."}
		}
		if _, e = time.Parse("2006-01-02", input.Date); e != nil {
			return invalid("Выберите день смены.")
		}
		if len([]rune(input.Note)) > 2000 {
			return invalid("Комментарий слишком длинный.")
		}
		for _, n := range []int64{input.Equipment.ONT, input.Equipment.ROUTER, input.Equipment.STB} {
			if n < 0 || n > 100 {
				return invalid("Количество оборудования: 0–100.")
			}
		}
		var order dto.Order
		extension := &eventExtension{}
		extension.prepare = func(in postgres.EventInputs, ev *dto.EventInput) error {
			var node *dto.AccessNode
			var building *dto.Building
			var work *dto.WorkType
			for i := range in.Reference.AccessNodes {
				n := &in.Reference.AccessNodes[i]
				if n.AccessNodeId == input.NodeID {
					node = n
					for j := range n.Buildings {
						if n.Buildings[j].BuildingId == input.BuildingID {
							building = &n.Buildings[j]
						}
					}
				}
			}
			for i := range in.Reference.WorkTypes {
				if in.Reference.WorkTypes[i].Code == input.WorkType {
					work = &in.Reference.WorkTypes[i]
				}
			}
			if node == nil || building == nil || work == nil {
				return invalid("Выберите адрес, узел и норматив из справочника.")
			}
			if (work.Code == "INST_PON_MKD" && (node.Technology != "PON" || building.BuildingType != "MKD")) || (work.Code == "INST_FTTB_MKD" && (node.Technology != "FTTB" || building.BuildingType != "MKD")) || ((work.Code == "INST_PRIVATE_VLS" || work.Code == "SURVEY_PRIVATE") && building.BuildingType != "PRIVATE") {
				return invalid("Выбранный норматив не соответствует технологии или типу здания.")
			}
			profile := dto.Profile("INSTALL")
			if service == "REPAIR" {
				profile = "LTP3"
			}
			if !c.Has(work.Profiles, profile) {
				return invalid("Норматив не соответствует услуге обращения.")
			}
			start, e := time.Parse(time.RFC3339, string(input.Slot.Start))
			if e != nil {
				return invalid("Неверное начало окна.")
			}
			end, e := time.Parse(time.RFC3339, string(input.Slot.End))
			if e != nil || end.Sub(start) != 2*time.Hour {
				return invalid("Укажите двухчасовое окно визита.")
			}
			latest := end
			if work.MaxStartDelayMin != nil {
				latest = minTime(end, start.Add(time.Duration(*work.MaxStartDelayMin)*time.Minute))
			}
			registered := c.Time(in.Day.Now)
			deadline := end
			if work.DeadlineKind == "HOURS" && work.DeadlineValue != nil {
				deadline = registered.Add(time.Duration(*work.DeadlineValue) * time.Hour)
			}
			if work.DeadlineKind == "WORKING_DAYS" && work.DeadlineValue != nil {
				deadline = registered
				for days := int64(0); days < *work.DeadlineValue; {
					deadline = deadline.AddDate(0, 0, 1)
					if deadline.Weekday() != time.Saturday && deadline.Weekday() != time.Sunday {
						days++
					}
				}
			}
			used := map[string]bool{}
			for _, o := range in.Day.Orders {
				used[o.OrderId] = true
			}
			orderID := ""
			for number := 1; number <= 9999; number++ {
				candidate := fmt.Sprintf("ORD-%s-%04d", registered.Format("20060102"), number)
				if !used[candidate] {
					orderID = candidate
					break
				}
			}
			if orderID == "" {
				return invalid("Исчерпаны номера нарядов дня.")
			}
			order = dto.Order{OrderId: orderID, WorkTypeCode: work.Code, Technology: node.Technology, BuildingId: building.BuildingId, Address: building.Address, Lat: building.Lat, Lon: building.Lon, BuildingType: building.BuildingType, RequiresHeight: building.OverheadLine, AccessNodeId: node.AccessNodeId, RegisteredAt: in.Day.Now, Slot: input.Slot, LatestStart: c.Stamp(latest), Deadline: c.Stamp(deadline), RequiredEquipment: input.Equipment}
			ev.Timestamp = in.Day.Now
			ev.Order = &order
			return nil
		}
		extension.commit = func(tx pgx.Tx, in postgres.EventInputs, r dto.EventResult) error {
			changed, e := tx.Exec(ctx, "UPDATE service_requests SET status='APPROVED',decision_note=$2,updated_at=now() WHERE id=$1 AND status IN ('NEW','IN_REVIEW')", id, input.Note)
			if e != nil {
				return e
			}
			if changed.RowsAffected() != 1 {
				return &Error{409, "REQUEST_CHANGED", "Обращение изменилось. Обновите список."}
			}
			raw, e := json.Marshal(r)
			if e != nil {
				return e
			}
			_, e = tx.Exec(ctx, "INSERT INTO request_orders(request_id,day,generation,order_id,result) VALUES($1,$2,$3,$4,$5)", id, input.Date, in.Generation, order.OrderId, raw)
			return e
		}
		result, e = s.event(ctx, input.Date, dto.EventInput{EventId: "approval:" + id, Type: "NEW_ORDER"}, "", extension)
		return e
	})
	return result, e
}
func minTime(a, b time.Time) time.Time {
	if a.Before(b) {
		return a
	}
	return b
}
