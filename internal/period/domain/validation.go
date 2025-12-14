package domain

import (
	"fmt"
	"sort"
	"time"
)

// DetectOverlaps
// validates that no two periods of the same granularity overlap.
//
// It returns a slice of human-readable error messages.
//
// HOW IT WORKS:
//   - Group periods by granularity (YEARLY/CALENDAR, QUARTERLY, MONTHLY)
//   - For each group:
//   - Sort by StartDate
//   - Compare each period with the next one
//   - If StartDate < previous.EndDate → OVERLAP
//
// EXAMPLE USAGE:
//
//	errs := DetectOverlaps(periods)
//	for _, e := range errs {
//	    fmt.Println(e)
//	}
//
// EXPECTED OUTPUT (if an overlap exists):
//
//	"Overlap detected (MONTHLY): 2026-MAR overlaps with 2026-APR"
//
// ============================================================================
func DetectOverlaps(periods []*Period) []string {

	// --- 1. Group periods by granularity -----------------------------------
	grouped := map[PeriodGranularity][]*Period{
		Calendar: {},
		Quarter:  {},
		Month:    {},
	}

	for _, p := range periods {
		grouped[p.Granularity] = append(grouped[p.Granularity], p)
	}

	var errs []string

	// --- 2. Validate overlaps inside each granularity group -----------------
	for granularity, list := range grouped {

		// Sort by StartDate (oldest first)
		sort.Slice(list, func(i, j int) bool {
			return list[i].StartDate.Before(list[j].StartDate)
		})

		for i := 1; i < len(list); i++ {
			prev := list[i-1]
			curr := list[i]

			// Overlap if: curr.Start < prev.End
			if curr.StartDate.Before(prev.EndDate) {
				errs = append(errs, fmt.Sprintf(
					"Overlap detected (%s): %s (%s → %s) overlaps with %s (%s → %s)",
					granularity,
					prev.ID,
					fmtDate(prev.StartDate),
					fmtDate(prev.EndDate),
					curr.ID,
					fmtDate(curr.StartDate),
					fmtDate(curr.EndDate),
				))
			}
		}
	}

	return errs
}

// Utility to format time for nicer error messages
func fmtDate(t time.Time) string {
	return t.Format("2006-01-02")
}
