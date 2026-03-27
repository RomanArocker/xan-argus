package handler

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/xan-com/xan-pythia/internal/model"
)

func validateFieldValues(rawValues json.RawMessage, fields []model.FieldDefinition) string {
	if len(rawValues) == 0 || string(rawValues) == "{}" || string(rawValues) == "null" {
		return ""
	}

	var values map[string]json.RawMessage
	if err := json.Unmarshal(rawValues, &values); err != nil {
		return "field_values must be a JSON object"
	}

	fieldMap := make(map[string]model.FieldDefinition, len(fields))
	for _, f := range fields {
		idBytes, _ := f.ID.MarshalJSON()
		idStr := string(idBytes)
		if len(idStr) >= 2 && idStr[0] == '"' {
			idStr = idStr[1 : len(idStr)-1]
		}
		fieldMap[idStr] = f
	}

	for key, rawVal := range values {
		fd, ok := fieldMap[key]
		if !ok {
			return fmt.Sprintf("unknown field_values key: %s", key)
		}

		switch fd.FieldType {
		case "text":
			var s string
			if err := json.Unmarshal(rawVal, &s); err != nil {
				return fmt.Sprintf("field %q (%s) must be a string", fd.Name, key)
			}
		case "number":
			var n float64
			if err := json.Unmarshal(rawVal, &n); err != nil {
				return fmt.Sprintf("field %q (%s) must be a number", fd.Name, key)
			}
		case "boolean":
			var b bool
			if err := json.Unmarshal(rawVal, &b); err != nil {
				return fmt.Sprintf("field %q (%s) must be a boolean", fd.Name, key)
			}
		case "date":
			var s string
			if err := json.Unmarshal(rawVal, &s); err != nil {
				return fmt.Sprintf("field %q (%s) must be a date string (YYYY-MM-DD)", fd.Name, key)
			}
			if _, err := time.Parse("2006-01-02", s); err != nil {
				return fmt.Sprintf("field %q (%s) must be in YYYY-MM-DD format", fd.Name, key)
			}
		}
	}

	return ""
}
