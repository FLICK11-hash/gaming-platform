// Package dbutil provides small conversions between pgx's pgtype wrappers
// and plain Go types used across handlers and domain services.
package dbutil

import (
	"github.com/jackc/pgx/v5/pgtype"
)

// IntToPGInt4 converts a plain int to a pgtype.Int4.
func IntToPGInt4(v int) pgtype.Int4 {
	return pgtype.Int4{Int32: int32(v), Valid: true}
}

// PGInt4ToInt converts a pgtype.Int4 to a plain int, defaulting to 0 if null.
func PGInt4ToInt(v pgtype.Int4) int {
	if !v.Valid {
		return 0
	}
	return int(v.Int32)
}

// Int64ToPGInt8 converts a plain int64 to a pgtype.Int8.
func Int64ToPGInt8(v int64) pgtype.Int8 {
	return pgtype.Int8{Int64: v, Valid: true}
}

// PGInt8ToInt64 converts a pgtype.Int8 to a plain int64, defaulting to 0 if null.
func PGInt8ToInt64(v pgtype.Int8) int64 {
	if !v.Valid {
		return 0
	}
	return v.Int64
}

// TextToPGText converts a plain string to a pgtype.Text.
func TextToPGText(s string) pgtype.Text {
	return pgtype.Text{String: s, Valid: s != ""}
}

// PGTextToString converts a pgtype.Text to a plain string, defaulting to "".
func PGTextToString(v pgtype.Text) string {
	return v.String
}
