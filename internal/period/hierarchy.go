package period

import (
	"time"
)

func AddChild(parent *Period, childID string) {
	for _, existing := range parent.ChildPeriodIDs {
		if existing == childID {
			return // Child already added -> preventing duplicates
		}
	}

	parent.ChildPeriodIDs = append(parent.ChildPeriodIDs, childID)
}

// FindPeriodByID searches for a Period in a slice and returns a pointer. Returns nil if no match is found.
//
// Example:
//
//	p := FindPeriodByID(periods, "2026-Q1")
//	fmt.Println(p.Name) // → "Q1 2026"
func FindPeriodByID(periods []Period, id string) *Period {
	for i := range periods {
		if periods[i].ID == id {
			return &periods[i]
		}
	}

	return nil
}

// BreakDownTradePeriod converts a higher-level Period (Quarter or Year) into its monthly sub-periods.
// This is essential for translating trades that span multiple months into individual monthly payments.
//
// Example:
//
//		months := BreakDownTradePeriod("2026-Q1", periods)
//	 fmt.Println(months) → ["2026-JAN", "2026-FEB", "2026-MAR"]
func BreakDownTradePeriod(parentID string, periods []Period) []string {
	parent := FindPeriodByID(periods, parentID)
	if parent == nil {
		return nil
	}

	// If this is already a month, just return it directly
	if parent.Granularity == MonthlyPeriod {
		return []string{parentID}
	}

	var monthIDs []string
	for _, childID := range parent.ChildPeriodIDs {
		child := FindPeriodByID(periods, childID)
		if child == nil {
			continue
		}

		// If the child is a quarter, we recursively dive into its months
		if child.Granularity == QuarterlyPeriod {
			monthIDs = append(monthIDs, BreakDownTradePeriod(child.ID, periods)...)
		} else if child.Granularity == MonthlyPeriod {
			monthIDs = append(monthIDs, childID)
		}
	}

	return monthIDs
}

// FindPeriodsForDate finds the corresponding Month, Quarter, and Year IDs for a given date.
// Useful for assigning trades to the correct periods.
//
// Example:
//
//	m, q, y := FindPeriodsForDate(periods, time.Date(2026, 2, 10, 0, 0, 0, 0, time.UTC))
//	fmt.Println(m, q, y)
//	// → "2026-FEB", "2026-Q1", "2026"
func FindPeriodsForDate(periods []Period, date time.Time) (monthID, quarterID, yearID string) {
	for _, p := range periods {
		if date.After(p.StartDate) && date.Before(p.EndDate) || date.Equal(p.StartDate) || date.Equal(p.EndDate) {
			switch p.Granularity {
			case MonthlyPeriod:
				monthID = p.ID
			case QuarterlyPeriod:
				quarterID = p.ID
			case CalendarYearPeriod:
				yearID = p.ID
			}
		}
	}
	return
}
