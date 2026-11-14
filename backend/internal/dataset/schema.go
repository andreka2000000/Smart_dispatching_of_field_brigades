package dataset

import (
	"encoding/json"
	"magistral/backend/internal/contract"
)

func validateValue(v *contract.Validator, schema string, value any) error {
	raw, e := json.Marshal(value)
	if e != nil {
		return e
	}
	var target any
	return v.Decode(raw, schema, &target)
}
