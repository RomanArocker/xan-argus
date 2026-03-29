package importer

import (
	"bytes"
	"testing"
)

func TestGenerateTemplate(t *testing.T) {
	cfg := customersConfig()
	exp := &Exporter{}
	var buf bytes.Buffer
	err := exp.WriteTemplate(&buf, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	output := buf.Bytes()
	// Check BOM
	if len(output) < 3 || output[0] != 0xEF || output[1] != 0xBB || output[2] != 0xBF {
		t.Fatal("expected UTF-8 BOM")
	}
	// Check headers
	content := string(output[3:])
	if content != "name,contact_email,notes\r\n" {
		t.Fatalf("unexpected: %q", content)
	}
}
