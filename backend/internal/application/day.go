package application

import (
	"context"
	"github.com/jackc/pgx/v5"
	"magistral/backend/internal/dto"
	"magistral/backend/internal/storage/postgres"
	"time"
)

// CreateDay adds a fresh operational shift. Unlike the organizer PUT endpoint,
// it never replaces an existing day, its events, or its plan generation.
func (s *Service) CreateDay(ctx context.Context, date string) (dto.DayInput, error) {
	day := dto.DayInput{Date: date, Now: dto.DateTime(date + "T08:30:00+05:00"), Orders: []dto.Order{}, AbsentStaffIds: []string{}}
	if _, e := time.Parse("2006-01-02", date); e != nil {
		return day, &Error{400, "INVALID_DATE", "Выберите дату смены."}
	}
	e := s.Store.Update(ctx, func(tx pgx.Tx) error {
		var exists bool
		if e := tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM days WHERE date::text=$1)", date).Scan(&exists); e != nil {
			return e
		}
		if exists {
			return &Error{409, "DAY_EXISTS", "Смена на эту дату уже существует. Её данные сохранены."}
		}
		reference, e := postgres.Read[dto.ReferenceData](ctx, tx, "reference_data")
		if e != nil {
			return e
		}
		matrix, e := postgres.Read[dto.TravelMatrix](ctx, tx, "travel_matrices")
		if e != nil {
			return e
		}
		if reference == nil || matrix == nil {
			return &Error{409, "REFERENCE_NOT_LOADED", "Сначала загрузите справочники и дорожную матрицу."}
		}
		return postgres.ReplaceDay(ctx, tx, day)
	})
	return day, e
}
