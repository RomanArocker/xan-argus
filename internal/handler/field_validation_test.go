package handler

import (
	"encoding/json"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/xan-com/xan-pythia/internal/model"
)

func makeFieldDef(id string, name string, fieldType string) model.FieldDefinition {
	var uid pgtype.UUID
	uid.Scan(id)
	return model.FieldDefinition{ID: uid, Name: name, FieldType: fieldType}
}

func TestValidateFieldValues(t *testing.T) {
	fields := []model.FieldDefinition{
		makeFieldDef("00000000-0000-0000-0000-000000000001", "RAM", "number"),
		makeFieldDef("00000000-0000-0000-0000-000000000002", "Serial", "text"),
		makeFieldDef("00000000-0000-0000-0000-000000000003", "Active", "boolean"),
		makeFieldDef("00000000-0000-0000-0000-000000000004", "PurchaseDate", "date"),
	}

	tests := []struct {
		name    string
		values  string
		wantErr bool
	}{
		{"empty object", `{}`, false},
		{"null", `null`, false},
		{"valid number", `{"00000000-0000-0000-0000-000000000001": 16}`, false},
		{"valid text", `{"00000000-0000-0000-0000-000000000002": "SN-123"}`, false},
		{"valid boolean", `{"00000000-0000-0000-0000-000000000003": true}`, false},
		{"valid date", `{"00000000-0000-0000-0000-000000000004": "2026-01-15"}`, false},
		{"unknown key", `{"00000000-0000-0000-0000-999999999999": "x"}`, true},
		{"wrong type for number", `{"00000000-0000-0000-0000-000000000001": "not-a-number"}`, true},
		{"wrong type for boolean", `{"00000000-0000-0000-0000-000000000003": "yes"}`, true},
		{"bad date format", `{"00000000-0000-0000-0000-000000000004": "15/01/2026"}`, true},
		{"invalid json", `not-json`, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := validateFieldValues(json.RawMessage(tt.values), fields)
			if tt.wantErr && msg == "" {
				t.Error("expected error, got empty string")
			}
			if !tt.wantErr && msg != "" {
				t.Errorf("expected no error, got %q", msg)
			}
		})
	}
}
