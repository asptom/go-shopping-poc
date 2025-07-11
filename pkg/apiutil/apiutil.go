package apiutil

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

// RFC3339Format is the standard format for timestamps in JSON APIs.
const RFC3339Format = "2006-01-02T15:04:05Z07:00"

// NullString converts a Go string to pgtype.Text, handling nulls.
func NullString(s string) pgtype.Text {
	if s == "" {
		return pgtype.Text{Valid: false}
	}
	return pgtype.Text{String: s, Valid: true}
}

// NullTimestampFromString parses a string in RFC3339 format and returns pgtype.Timestamp.
func NullTimestampFromString(s string) pgtype.Timestamp {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return pgtype.Timestamp{Valid: false}
	}
	return pgtype.Timestamp{Time: t, Valid: true}
}
