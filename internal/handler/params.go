package handler

import (
	"fmt"

	"github.com/jackc/pgx/v5/pgtype"
)

func parseUUID(s string) (pgtype.UUID, error) {
	var id pgtype.UUID
	if err := id.Scan(s); err != nil {
		return id, fmt.Errorf("invalid UUID: %w", err)
	}
	return id, nil
}

// uuidToStr formats a pgtype.UUID as a lowercase hex string.
func uuidToStr(u pgtype.UUID) string {
	if !u.Valid {
		return ""
	}
	b := u.Bytes
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
