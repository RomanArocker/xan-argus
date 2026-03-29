package importer

import (
	"context"
	"strings"
	"testing"
)

// --- parseHeaders tests ---

func TestParseHeaders_Valid(t *testing.T) {
	cfg := &EntityConfig{
		Name:      "test",
		Table:     "test",
		MatchKeys: []string{"name"},
		Columns: []ColumnConfig{
			{Header: "name", DBColumn: "name", Required: true, Type: "text"},
			{Header: "email", DBColumn: "email", Type: "text"},
		},
	}
	r := strings.NewReader("name,email\n")
	mappings, err := parseHeaders(r, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(mappings) != 2 {
		t.Fatalf("expected 2 mappings, got %d", len(mappings))
	}
	if mappings[0].csvIndex != 0 || mappings[0].config.Header != "name" {
		t.Errorf("expected first mapping to be 'name' at index 0, got %+v", mappings[0])
	}
	if mappings[1].csvIndex != 1 || mappings[1].config.Header != "email" {
		t.Errorf("expected second mapping to be 'email' at index 1, got %+v", mappings[1])
	}
}

func TestParseHeaders_MissingRequired(t *testing.T) {
	cfg := &EntityConfig{
		Name:  "test",
		Table: "test",
		Columns: []ColumnConfig{
			{Header: "name", DBColumn: "name", Required: true, Type: "text"},
			{Header: "email", DBColumn: "email", Type: "text"},
		},
	}
	// CSV has only 'email', missing required 'name'.
	r := strings.NewReader("email\n")
	_, err := parseHeaders(r, cfg)
	if err == nil {
		t.Fatal("expected error for missing required column, got nil")
	}
	if !strings.Contains(err.Error(), "name") {
		t.Errorf("error should mention 'name', got: %v", err)
	}
}

func TestParseHeaders_UnknownColumnsIgnored(t *testing.T) {
	cfg := &EntityConfig{
		Name:  "test",
		Table: "test",
		Columns: []ColumnConfig{
			{Header: "name", DBColumn: "name", Required: true, Type: "text"},
		},
	}
	// CSV has 'name' plus unknown 'garbage'.
	r := strings.NewReader("name,garbage,another_unknown\n")
	mappings, err := parseHeaders(r, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(mappings) != 1 {
		t.Fatalf("expected 1 mapping (unknown columns ignored), got %d", len(mappings))
	}
	if mappings[0].config.Header != "name" {
		t.Errorf("expected mapping 'name', got %q", mappings[0].config.Header)
	}
}

// --- parseValue tests ---

func TestParseValue_Text(t *testing.T) {
	col := ColumnConfig{Header: "notes", DBColumn: "notes", Type: "text"}
	got, err := parseValue(context.Background(), "hello world", col, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "hello world" {
		t.Errorf("expected %q, got %v", "hello world", got)
	}
}

func TestParseValue_Int(t *testing.T) {
	col := ColumnConfig{Header: "count", DBColumn: "count", Type: "int"}
	got, err := parseValue(context.Background(), "42", col, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != 42 {
		t.Errorf("expected 42, got %v", got)
	}
}

func TestParseValue_IntInvalid(t *testing.T) {
	col := ColumnConfig{Header: "count", DBColumn: "count", Type: "int"}
	_, err := parseValue(context.Background(), "not-a-number", col, nil)
	if err == nil {
		t.Fatal("expected error for invalid int, got nil")
	}
}

func TestParseValue_Bool(t *testing.T) {
	col := ColumnConfig{Header: "active", DBColumn: "active", Type: "bool"}
	tests := []struct {
		input    string
		expected bool
	}{
		{"true", true},
		{"1", true},
		{"yes", true},
		{"false", false},
		{"0", false},
		{"no", false},
		{"TRUE", true},
		{"YES", true},
	}
	for _, tc := range tests {
		got, err := parseValue(context.Background(), tc.input, col, nil)
		if err != nil {
			t.Errorf("input %q: unexpected error: %v", tc.input, err)
			continue
		}
		if got != tc.expected {
			t.Errorf("input %q: expected %v, got %v", tc.input, tc.expected, got)
		}
	}
}

func TestParseValue_BoolInvalid(t *testing.T) {
	col := ColumnConfig{Header: "active", DBColumn: "active", Type: "bool"}
	_, err := parseValue(context.Background(), "maybe", col, nil)
	if err == nil {
		t.Fatal("expected error for invalid bool, got nil")
	}
}

func TestParseValue_Date(t *testing.T) {
	col := ColumnConfig{Header: "start_date", DBColumn: "start_date", Type: "date"}
	got, err := parseValue(context.Background(), "2024-06-15", col, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got == nil {
		t.Fatal("expected non-nil date value")
	}
}

func TestParseValue_DateInvalid(t *testing.T) {
	col := ColumnConfig{Header: "start_date", DBColumn: "start_date", Type: "date"}
	_, err := parseValue(context.Background(), "15/06/2024", col, nil)
	if err == nil {
		t.Fatal("expected error for invalid date format, got nil")
	}
}

func TestParseValue_JSON(t *testing.T) {
	col := ColumnConfig{Header: "metadata", DBColumn: "metadata", Type: "json"}
	got, err := parseValue(context.Background(), `{"key":"value"}`, col, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != `{"key":"value"}` {
		t.Errorf("expected JSON passthrough, got %v", got)
	}
}

func TestParseValue_JSONInvalid(t *testing.T) {
	col := ColumnConfig{Header: "metadata", DBColumn: "metadata", Type: "json"}
	_, err := parseValue(context.Background(), `{not valid json}`, col, nil)
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

func TestStripBOM(t *testing.T) {
	withBOM := []byte{0xEF, 0xBB, 0xBF, 'h', 'e', 'l', 'l', 'o'}
	got := stripBOM(withBOM)
	if string(got) != "hello" {
		t.Errorf("expected BOM stripped, got %q", got)
	}

	noBOM := []byte("hello")
	got = stripBOM(noBOM)
	if string(got) != "hello" {
		t.Errorf("expected unchanged, got %q", got)
	}
}
