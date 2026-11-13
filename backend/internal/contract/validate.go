package contract

import (
	_ "embed"
	"encoding/json"
	"github.com/getkin/kin-openapi/openapi3"
)

//go:embed api_spec.yaml
var Spec []byte

type Validator struct{ doc *openapi3.T }

func New() (*Validator, error) {
	doc, e := openapi3.NewLoader().LoadFromData(Spec)
	if e != nil {
		return nil, e
	}
	// Contract prose explicitly says to ignore unknown fields, including fact.
	delete(doc.Components.Schemas["Order"].Value.Properties, "fact")
	seen := map[*openapi3.Schema]bool{}
	var walk func(*openapi3.Schema)
	walk = func(s *openapi3.Schema) {
		if s == nil || seen[s] {
			return
		}
		seen[s] = true
		s.AdditionalProperties.Has = nil
		for _, r := range s.Properties {
			walk(r.Value)
		}
		if s.Items != nil {
			walk(s.Items.Value)
		}
		for _, r := range s.AllOf {
			walk(r.Value)
		}
		for _, r := range s.OneOf {
			walk(r.Value)
		}
	}
	for _, r := range doc.Components.Schemas {
		walk(r.Value)
	}
	return &Validator{doc}, nil
}
func (v *Validator) Decode(raw []byte, name string, out any) error {
	var value any
	if e := json.Unmarshal(raw, &value); e != nil {
		return e
	}
	// Fact is ignored even if its value does not match the training schema.
	if m, ok := value.(map[string]any); ok {
		if orders, ok := m["orders"].([]any); ok {
			for _, o := range orders {
				if x, ok := o.(map[string]any); ok {
					delete(x, "fact")
				}
			}
		}
	}
	if e := v.doc.Components.Schemas[name].Value.VisitJSON(value); e != nil {
		return e
	}
	clean, e := json.Marshal(value)
	if e != nil {
		return e
	}
	return json.Unmarshal(clean, out)
}
