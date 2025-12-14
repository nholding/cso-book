package domain

import (
	"fmt"
	//	"sort"
	"strings"
	"time"

	"github.com/nholding/cso-book/internal/audit"
)

// PeriodGranularity identifies the logical resolution of a Period, defines what kind of period a trade covers.
// It allows you to filter or aggregate trades differently based on whether they are monthly, quarterly, or yearly deals.
type PeriodGranularity string

const (
	MonthlyPeriod      PeriodGranularity = "MONTHLY"
	QuarterlyPeriod    PeriodGranularity = "QUARTERLY"
	CalendarYearPeriod PeriodGranularity = "CALENDAR"
)

// Period defines a specific period of time for purchases and sales. It represents 'Years', 'Quarters', and 'Months.
// The `ID` field is included to uniquely identify the period for reference purposes.
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
type Period struct {
	ID             string            // Unique period identifier (e.g., "2026-Q1")
	Name           string            // Human-readable label (e.g., "Q1 2026")
	Granularity    PeriodGranularity // Granularity of the period (Monthly, quarterly, Calendar)
	ParentPeriodID *string           // / Points to parent (Quarter → Year, Month → Quarter)
	ChildPeriodIDs []string          // IDs of child periods (e.g., year has quarters, quarter has months); not stored in the DB
	StartDate      time.Time         // Period start (UTC, inclusive)
	EndDate        time.Time         // Period end (UTC, inclusive)
	AuditInfo      *audit.AuditInfo
}

// PeriodRange represents a range of Periods for a trade. PeriodRange allows a Trade to span multiple periods (e.g., Q1 + Q2)
// It allows a trade to span multiple months, quarters, or even years.
//
// Example usage:
//
//	// Single quarter trade
//	pr1 := PeriodRange{
//	    StartPeriodID: "2026-Q1",
//	    EndPeriodID:   "2026-Q1",
//	}
//
//	// Multi-quarter trade (Q1+Q2)
//	pr2 := PeriodRange{
//	    StartPeriodID: "2026-Q1",
//	    EndPeriodID:   "2026-Q2",
//	}
type PeriodRange struct {
	StartPeriodID string // ID of the starting period (e.g., "2026-Q1")
	EndPeriodID   string // ID of the ending period (e.g., "2026-Q2")
}

// GeneratePeriods creates years, quarters, and months for a range of years.
//
// Example:
//
//	periods := GeneratePeriods(2026, 2026)
//
//	// Outcome (IDs):
//	// "2026" -> year
//	// "2026-Q1", "2026-Q2", "2026-Q3", "2026-Q4" -> quarters
//	// "2026-JAN", "2026-FEB", "2026-MAR", ... -> months
func GeneratePeriods(startYear, endYear int) []Period {
	var periods []Period
	systemUser := "system@internal.local"

	for y := startYear; y <= endYear; y++ {
		yearID := fmt.Sprintf("%d", y)
		yearStart := time.Date(y, 1, 1, 0, 0, 0, 0, time.UTC)
		yearEnd := time.Date(y+1, 1, 1, 0, 0, 0, 0, time.UTC).Add(-time.Nanosecond)

		yearPeriod := Period{
			ID:             yearID,
			Name:           fmt.Sprintf("%d", y),
			Granularity:    CalendarYearPeriod,
			ParentPeriodID: nil,
			ChildPeriodIDs: []string{},
			StartDate:      yearStart,
			EndDate:        yearEnd,
			AuditInfo:      *audit.NewAuditInfo(systemUser),
		}
		periods = append(periods, yearPeriod)

		// Generate quarters
		for q := 1; q <= 4; q++ {
			qID := fmt.Sprintf("%d-Q%d", y, q)
			qStart := yearStart.AddDate(0, (q-1)*3, 0)
			qEnd := qStart.AddDate(0, 3, 0).Add(-time.Nanosecond)

			quarterPeriod := Period{
				ID:             qID,
				Name:           fmt.Sprintf("Q%d %d", q, y),
				Granularity:    QuarterlyPeriod,
				ParentPeriodID: &yearID,
				ChildPeriodIDs: []string{},
				StartDate:      qStart,
				EndDate:        qEnd,
				AuditInfo:      *audit.NewAuditInfo(systemUser),
			}

			// Generate months
			for m := 0; m < 3; m++ {
				monthStart := qStart.AddDate(0, m, 0)
				monthEnd := monthStart.AddDate(0, 1, 0).Add(-time.Nanosecond)
				monthID := strings.ToUpper(monthStart.Format("2006-Jan"))

				monthPeriod := Period{
					ID:             monthID,
					Name:           monthStart.Format("January 2006"),
					Granularity:    MonthlyPeriod,
					ParentPeriodID: &qID,
					ChildPeriodIDs: []string{},
					StartDate:      monthStart,
					EndDate:        monthEnd,
					AuditInfo:      *audit.NewAuditInfo(systemUser),
				}

				quarterPeriod.ChildPeriodIDs = append(quarterPeriod.ChildPeriodIDs, monthID)
				periods = append(periods, monthPeriod)
			}

			yearPeriod.ChildPeriodIDs = append(yearPeriod.ChildPeriodIDs, qID)
			periods = append(periods, quarterPeriod)
		}
	}
	return periods
}

// Validate checks the period for consistency and returns an error if invalid.
func (p *Period) Validate() error {
	if p.ID == "" {
		fmt.Errorf("period ID cannot be empty")
	}
	if p.Name == "" {
		return fmt.Errorf("period name cannot be empty")
	}
	if p.Granularity != "CALENDAR" && p.Granularity != "QUARTERLY" && p.Granularity != "MONTHLY" {
		return fmt.Errorf("invalid granularity, must be CALENDAR, QUARTERLY, or MONTHLY")
	}
	if !p.StartDate.Before(p.EndDate) {
		return fmt.Errorf("start date must be before end date")
	}
	return nil
}

// GranularityRank
// Purpose:
//
//	Maps granularity enums to numeric ranks to allow
//	consistent comparisons such as:
//
//	     MONTHLY (1) < QUARTERLY (2) < CALENDAR (3)
//
// Used by hierarchy validation.
// ================================================
func (p *Period) GranularityRank() int {
	switch p.Granularity {
	case GranularityMonthly:
		return 1
	case GranularityQuarterly:
		return 2
	case GranularityCalendar:
		return 3
	default:
		return 99 // any unknown granularity is considered invalid
	}
}
