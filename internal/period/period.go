package period

import (
	"fmt"
	"strings"
	"time"

	"github.com/nholding/cso-book/internal/audit"
)

// PeriodGranularity identifies the logical resolution of a Period, defines what kind of period a trade covers.
// It allows you to filter or aggregate trades differently
// based on whether they are monthly, quarterly, or yearly deals.
type PeriodGranularity string

const (
	// MonthlyPeriod represents a single calendar month, e.g. January 2026.
	MonthlyPeriod PeriodGranularity = "MONTHLY"

	// QuarterlyPeriod represents a three-month quarter, e.g. Q1 2026.
	QuarterlyPeriod PeriodGranularity = "QUARTERLY"

	// CalendarYearPeriod represents a full calendar year, e.g. Calendar 2026.
	CalendarYearPeriod PeriodGranularity = "CALENDAR"
)

// Period defines a specific period of time for purchases and sales. It represents 'Years', 'Quarters', and 'Months.
// It includes a start and end date as well as a label for easy identification.
// The `ID` field is included to uniquely identify the period for reference purposes.
//
// This is intentionally generic so that the same struct can represent:
// - A single month (e.g., January 2026)
// - A quarter (e.g., Q2 2026)
// - A full calendar year (e.g., CAL 2026)
//
// Fields:
//   - StartDate: The start of the period (e.g., 2026-01-01)
//   - EndDate: The end of the period (e.g., 2026-03-31)
//   - Granularity: Type of the period (monthly, quarterly, calendar)
//   - Label: Human-readable name (e.g., "Q1 2026" or "January 2026")
//
// Periods form a strict parent-child hierarchy:
//
//	2026 (Year)
//	  ├── Q1-2026 (Quarter)
//	  │     ├── JAN-2026 (Month)
//	  │     ├── FEB-2026 (Month)
//	  │     └── MAR-2026 (Month)
//	  ├── Q2-2026
//	  ├── Q3-2026
//	  └── Q4-2026
//
// Each `Period` has:
// - A unique ID (e.g., "JAN-2026" or "Q1-2026")
// - StartDate and EndDate covering that period fully (inclusive)
// - Optional ParentPeriodID (nil for top-level years)
//
// Period represents a generic time unit in the trading calendar.
type Period struct {
	ID             string            // Unique period identifier (e.g., "2026-Q1")
	Name           string            // Human-readable label (e.g., "Q1 2026")
	Granularity    PeriodGranularity // Granularity of the period (Monthly, quarterly, Calendar)
	ParentPeriodID *string           // / Points to parent (Quarter → Year, Month → Quarter)
	ChildPeriodIDs []string          // IDs of child periods (e.g., year has quarters, quarter has months
	StartDate      time.Time         // Period start (UTC, inclusive)
	EndDate        time.Time         // Period end (UTC, inclusive)
	CreatedBy      string
	AuditInfo      audit.AuditInfo `json:"audit"`
}

func GeneratePeriods(startYear, endYear int) []Period {
	var periods []Period
	systemUser := "system@internal.local"

	// --- periodMap for quick parent lookup ---
	// Maps period ID -> pointer to Period object in periods slice
	periodMap := make(map[string]*Period)

	// --- Loop through each year ---
	for y := startYear; y <= endYear; y++ {
		yearID := fmt.Sprintf("%d", y)
		yearStart := time.Date(y, 1, 1, 0, 0, 0, 0, time.UTC)
		yearEnd := time.Date(y+1, 1, 1, 0, 0, 0, 0, time.UTC).Add(-time.Nanosecond)

		// Create year period
		yearPeriod := Period{
			ID:             yearID,
			Name:           fmt.Sprintf("%d", y),
			Granularity:    CalendarYearPeriod,
			ParentPeriodID: nil,
			ChildPeriodIDs: []string{}, // will populate with quarters
			StartDate:      yearStart,
			EndDate:        yearEnd,
			AuditInfo:      *audit.NewAuditInfo(systemUser),
		}

		// Append year period to slice and map
		periods = append(periods, yearPeriod)
		periodMap[yearID] = &periods[len(periods)-1] // pointer to the slice element

		// --- Generate Quarters for this Year ---
		for q := 1; q <= 4; q++ {
			qID := fmt.Sprintf("%d-Q%d", y, q)         // e.g., "2025-Q1"
			qStart := yearStart.AddDate(0, (q-1)*3, 0) // Start of quarter
			qEnd := qStart.AddDate(0, 3, 0).Add(-time.Nanosecond)

			quarterPeriod := Period{
				ID:             qID,
				Name:           fmt.Sprintf("Q%d %d", q, y),
				Granularity:    QuarterlyPeriod,
				ParentPeriodID: &yearID,
				ChildPeriodIDs: []string{}, // will populate with months
				StartDate:      qStart,
				EndDate:        qEnd,
				AuditInfo:      *audit.NewAuditInfo(systemUser),
			}

			// --- Generate Months for this Quarter ---
			for m := 0; m < 3; m++ {
				monthStart := qStart.AddDate(0, m, 0)
				monthEnd := monthStart.AddDate(0, 1, 0).Add(-time.Nanosecond)
				monthID := strings.ToUpper(monthStart.Format("2006-Jan")) // e.g., "2025-JAN"

				monthPeriod := Period{
					ID:             monthID,
					Name:           monthStart.Format("January 2006"), // e.g., "January 2025"
					Granularity:    MonthlyPeriod,
					ParentPeriodID: &qID,
					ChildPeriodIDs: []string{}, // months have no children
					StartDate:      monthStart,
					EndDate:        monthEnd,
					AuditInfo:      *audit.NewAuditInfo(systemUser),
				}

				// Append month to slice and map
				periods = append(periods, monthPeriod)
				periodMap[monthID] = &periods[len(periods)-1]

				// Link month to parent quarter
				quarterPeriod.ChildPeriodIDs = append(quarterPeriod.ChildPeriodIDs, monthID)
			}

			// Append quarter to slice and map
			periods = append(periods, quarterPeriod)
			periodMap[qID] = &periods[len(periods)-1]

			// Link quarter to parent year
			yearPtr := periodMap[yearID]
			yearPtr.ChildPeriodIDs = append(yearPtr.ChildPeriodIDs, qID)
		}
	}

	return periods
}
