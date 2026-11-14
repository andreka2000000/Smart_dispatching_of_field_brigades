// Package dataset reads the file formats defined by the organizer contract.
package dataset

import (
	"encoding/csv"
	"fmt"
	"io"
	"magistral/backend/internal/application"
	"magistral/backend/internal/contract"
	"magistral/backend/internal/domain"
	"magistral/backend/internal/dto"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Data struct {
	Reference dto.ReferenceData
	Matrix    dto.TravelMatrix
	Day       dto.DayInput
}

func Load(dir, date, now string) (Data, error) {
	var out Data
	if _, e := time.Parse("2006-01-02", date); e != nil {
		return out, fmt.Errorf("invalid date: %w", e)
	}
	clock, e := domain.NewClock(now)
	if e != nil {
		return out, e
	}
	v, e := contract.New()
	if e != nil {
		return out, e
	}
	read := func(file, schema string, target any) error {
		raw, e := os.ReadFile(filepath.Join(dir, file))
		if e == nil {
			e = v.Decode(raw, schema, target)
		}
		if e != nil {
			return fmt.Errorf("%s: %w", file, e)
		}
		return nil
	}
	for _, item := range []struct {
		file, schema string
		target       any
	}{
		{"service_centers.json", "ServiceCentersFile", &out.Reference.ServiceCenters},
		{"staff.json", "StaffFile", &out.Reference.Staff},
		{"access_nodes.json", "AccessNodesFile", &out.Reference.AccessNodes},
		{"travel_matrix.json", "TravelMatrix", &out.Matrix},
	} {
		if e = read(item.file, item.schema, item.target); e != nil {
			return out, e
		}
	}
	file, e := os.Open(filepath.Join(dir, "work_types.csv"))
	if e != nil {
		return out, e
	}
	defer file.Close()
	out.Reference.WorkTypes, e = ReadWorks(file)
	if e != nil {
		return out, e
	}
	// Validate parsed CSV against the original JSON schema too.
	if e = application.ValidateReference(out.Reference); e != nil {
		return out, e
	}
	if _, e = domain.NewMatrix(out.Matrix); e != nil {
		return out, e
	}
	if e = application.Coverage(out.Reference, out.Matrix); e != nil {
		return out, e
	}
	var orders []dto.Order
	if e = read("orders_train.json", "OrdersFile", &orders); e != nil {
		return out, e
	}
	out.Day = dto.DayInput{Date: date, Now: dto.DateTime(now), Orders: []dto.Order{}, AbsentStaffIds: []string{}}
	for _, o := range orders {
		slot, e := domain.NewClock(string(o.Slot.Start))
		if e != nil {
			return out, e
		}
		registered, e := domain.NewClock(string(o.RegisteredAt))
		if e != nil {
			return out, e
		}
		if slot.Now.Format("2006-01-02") == date && !registered.Now.After(clock.Now) {
			out.Day.Orders = append(out.Day.Orders, o)
		}
	}
	if e = application.ValidateDay(date, &out.Day, out.Reference); e != nil {
		return out, e
	}
	// Schema validation catches CSV enums, bounds and nullable fields.
	if e = validateValue(v, "ReferenceData", out.Reference); e != nil {
		return out, e
	}
	if e = validateValue(v, "DayInput", out.Day); e != nil {
		return out, e
	}
	if _, e = os.Stat(filepath.Join(dir, "events_example.json")); e == nil {
		var scenario any
		if e = read("events_example.json", "EventScenarioFile", &scenario); e != nil {
			return out, e
		}
	}
	return out, nil
}

func ReadWorks(source io.Reader) ([]dto.WorkType, error) {
	reader := csv.NewReader(source)
	header, e := reader.Read()
	if e != nil {
		return nil, e
	}
	columns := map[string]int{}
	for i, k := range header {
		if _, ok := columns[k]; ok {
			return nil, fmt.Errorf("duplicate CSV column %s", k)
		}
		columns[k] = i
	}
	for _, k := range []string{"code", "name", "profiles", "duration_min", "max_start_delay_min", "deadline_kind", "deadline_value", "repeat_deadline_value", "pgp_relevant"} {
		if _, ok := columns[k]; !ok {
			return nil, fmt.Errorf("missing CSV column %s (UTF-8 without BOM required)", k)
		}
	}
	out := []dto.WorkType{}
	for row := 2; ; row++ {
		record, e := reader.Read()
		if e == io.EOF {
			break
		}
		if e != nil {
			return nil, fmt.Errorf("CSV row %d: %w", row, e)
		}
		get := func(k string) string { return record[columns[k]] }
		integer := func(k string) (*int64, error) {
			s := get(k)
			if s == "" {
				return nil, nil
			}
			n, e := strconv.ParseInt(s, 10, 64)
			return &n, e
		}
		duration, e := integer("duration_min")
		if e != nil || duration == nil {
			return nil, fmt.Errorf("CSV row %d: invalid duration_min", row)
		}
		delay, e := integer("max_start_delay_min")
		if e != nil {
			return nil, e
		}
		deadline, e := integer("deadline_value")
		if e != nil {
			return nil, e
		}
		repeat, e := integer("repeat_deadline_value")
		if e != nil {
			return nil, e
		}
		pgp, e := strconv.ParseBool(get("pgp_relevant"))
		if e != nil {
			return nil, e
		}
		profiles := []dto.Profile{}
		for _, p := range strings.Split(get("profiles"), "|") {
			profiles = append(profiles, dto.Profile(p))
		}
		out = append(out, dto.WorkType{Code: dto.WorkTypeCode(get("code")), Name: get("name"), Profiles: profiles, DurationMin: *duration, MaxStartDelayMin: delay, DeadlineKind: get("deadline_kind"), DeadlineValue: deadline, RepeatDeadlineValue: repeat, PgpRelevant: pgp})
	}
	return out, nil
}
