package utils

import (
	"time"
)

// DateInRange checks if a date lies between two boundaries (inclusive).
func DateInRange(date, start, end time.Time) bool {
	return (date.Equal(start) || date.After(start)) && (date.Equal(end) || date.Before(end))
}
