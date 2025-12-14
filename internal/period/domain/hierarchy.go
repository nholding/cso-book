package domain

//This file now contains ONLY utilities related to constructing or
// managing the PERIOD HIERARCHY (Year → Quarter → Month).

// AddChild
// Adds a child period to a parent Period struct. Prevents duplicates.
// This is used while generating periods (typically in
// GeneratePeriods(startYear, endYear)). It ensures:
//
// - No duplicates
// - Correct structural linking (Year → Quarter, Quarter → Month)
//
// Example:
//
//	year := Period{ID: "2026", Granularity: CalendarYearPeriod}
//	AddChild(&year, "2026-Q1")
//
// Result:
//
//	year.ChildPeriodIDs == ["2026-Q1"]
func AddChild(parent *Period, childID string) {
	for _, existing := range parent.ChildPeriodIDs {
		if existing == childID {
			return // Prevent duplicates
		}
	}
	parent.ChildPeriodIDs = append(parent.ChildPeriodIDs, childID)
}

// contains
// Helper to check membership in a slice
// Simple helper to check if a slice contains a string.

// This is used by PeriodStore's breakdown functions and kept here
// because other period-related helpers may use it too.
//
// Example:
//
//	ok := contains([]string{"A", "B"}, "A") // true
//	ok := contains([]string{"A", "B"}, "C") // false
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
