package domain

import (
	"fmt"
	"time"

	"github.com/nholding/cso-book/internal/audit"
)

// FiscalCalendarConfig
//
// Purpose:
//
//	Encapsulates configuration for generating a fiscal year and its quarters.
//	Using a struct makes the function signature cleaner, easier to extend in
//	the future, and ensures clarity about the meaning of StartYear vs StartMonth.
//
// Fields:
//
//	StartYear  - The calendar year in which the fiscal year begins. For example,
//	             if the fiscal year FY2026 starts in April 2026, then StartYear = 2026.
//	StartMonth - The month in which the fiscal year begins (e.g., time.April for FY starting in April).
type FiscalCalendarConfig struct {
	StartYear  int        // the calendar year where the fiscal year begins (e.g., 2026)
	StartMonth time.Month // the month where fiscal year begins (e.g., April)
}

type FiscalCalendar struct {
	FiscalYear int
	StartMonth time.Month
	AuditInfo  *audit.AuditInfo
}

// GenerateFiscalYear
//
// Purpose:
//
//	Generates a fiscal year Period and its constituent fiscal quarters, reusing
//	already-created Gregorian months. No new month Periods are
//	created — only fiscal year and quarter Periods. This is compatible with functions like
//	BreakDownTradeRange or ValidateHierarchy.
//
// Assumptions:
//
//   - The store.Months slice contains all months from GeneratePeriods and is
//     sorted chronologically (earliest → latest).
//   - Fiscal quarters always span 3 months starting from the fiscal year start month.
//   - The function sets proper ParentPeriodID/ChildPeriodIDs relationships.
//
// Parameters:
//
//	cfg   - FiscalCalendarConfig specifying fiscal year start year and month.
//
// Returns:
//
//	[]*Period - Slice of newly created Periods:
//	    1. One fiscal year period (ID = "FY<StartYear>")
//	    2. Four fiscal quarters (IDs = "FY<StartYear>-Q1" … "FY<StartYear>-Q4")
//	Each fiscal quarter points to the corresponding month IDs. The fiscal year
//	period points to its four quarters.
//
// Example usage:
//
//	store := NewMockPeriodStore(2025, 2027) // generates months from 2025–2027
//
//	cfg := FiscalCalendarConfig{
//	    StartYear:  2026,       // FY2026 starts in 2026
//	    StartMonth: time.April, // FY starts in April
//	}
//
//	fyPeriods := GenerateFiscalYear(cfg)
//
// Expected outcome:
//
//	IDs and ranges:
//	fyPeriods[0] = FY2026         → Apr 1, 2026 – Mar 31, 2027
//	fyPeriods[1] = FY2026-Q1      → Apr 1, 2026 – Jun 30, 2026
//	fyPeriods[2] = FY2026-Q2      → Jul 1, 2026 – Sep 30, 2026
//	fyPeriods[3] = FY2026-Q3      → Oct 1, 2026 – Dec 31, 2026
//	fyPeriods[4] = FY2026-Q4      → Jan 1, 2027 – Mar 31, 2027
//
//	Each fiscal quarter's ChildPeriodIDs point to the existing months.
//	Example: FY2026-Q1.ChildPeriodIDs = ["2026-APR", "2026-MAY", "2026-JUN"]
//
// Notes:
//
//   - Fiscal year always spans exactly 12 months.
//   - This function does not modify existing months; it only creates fiscal year and quarter Periods.
//   - Use after generating Gregorian months with GeneratePeriods and before persisting fiscal periods to DB.
func GenerateFiscalYear(months []*Period, cfg FiscalCalendarConfig) ([]*Period, error) {
	var fyPeriods []*Period
	systemUser := "system@internal.local"

	// -------------------------------
	// Step 1: Determine fiscal year start and end
	// -------------------------------
	// The fiscal year starts exactly in StartYear + StartMonth and ends 12 months later
	fyStart := time.Date(cfg.StartYear, cfg.StartMonth, 1, 0, 0, 0, 0, time.UTC)
	fyEnd := fyStart.AddDate(1, 0, 0).Add(-time.Nanosecond)

	// -------------------------------
	// Step 2: Collect all Gregorian months that fall within this fiscal year
	// -------------------------------
	var fyMonths []*Period
	for _, m := range months {
		if m == nil {
			continue
		}

		if m.Calendar != CalendarGregorian || m.Granularity != MonthlyPeriod {
			continue
		}

		if !m.StartDate.Before(fyStart) && !m.EndDate.After(fyEnd) {
			fyMonths = append(fyMonths, m)
		}
	}

	// Safety check: fiscal year should have exactly 12 months
	if len(fyMonths) != 12 {
		return nil, fmt.Errorf("Warning: FY%d expected 12 months, found %d months\n", cfg.StartYear, len(fyMonths))
	}

	// -------------------------------
	// Step 3: Create fiscal year Period
	// -------------------------------
	fyID := fmt.Sprintf("FY%d", cfg.StartYear)

	fyPeriod := &Period{
		ID:             fyID,
		Name:           fmt.Sprintf("Fiscal Year %d", cfg.StartYear),
		Calendar:       CalendarFiscal,
		Granularity:    CalendarYearPeriod,
		StartDate:      fyStart,
		EndDate:        fyEnd,
		ChildPeriodIDs: []string{}, // will be filled with fiscal quarters
		AuditInfo:      audit.NewAuditInfo(systemUser),
	}

	fyPeriods = append(fyPeriods, fyPeriod)

	// -------------------------------
	// Step 4: Create fiscal quarters
	// -------------------------------
	for q := 0; q < 4; q++ {
		// Each quarter spans 3 months
		qStartIndex := q * 3
		qEndIndex := qStartIndex + 3
		qMonths := fyMonths[qStartIndex:qEndIndex]
		qID := fmt.Sprintf("FY%d-Q%d", cfg.StartYear, q+1)

		if qEndIndex > len(fyMonths) {
			return nil, fmt.Errorf("invalid fiscal month slicing for FY%d", cfg.StartYear)
		}

		qMonths = fyMonths[qStartIndex:qEndIndex]

		if len(qMonths) == 0 {
			continue
		}

		quarter := &Period{
			ID:             qID,
			Name:           fmt.Sprintf("FY%d Q%d", cfg.StartYear, q+1),
			Calendar:       CalendarFiscal,
			Granularity:    QuarterlyPeriod,
			ParentPeriodID: &fyID,
			StartDate:      qMonths[0].StartDate,
			EndDate:        qMonths[len(qMonths)-1].EndDate,
			ChildPeriodIDs: []string{},
			AuditInfo:      audit.NewAuditInfo(systemUser),
		}

		// Assign month IDs as children of quarter
		for _, m := range qMonths {
			quarter.ChildPeriodIDs = append(quarter.ChildPeriodIDs, m.ID)
		}

		// Assign quarter ID as child of fiscal year
		fyPeriod.ChildPeriodIDs = append(fyPeriod.ChildPeriodIDs, qID)

		// Add quarter to output
		fyPeriods = append(fyPeriods, quarter)
	}

	return fyPeriods, nil
}

//// GenerateFiscalPeriods generates a full fiscal year (months, quarters, and year period)
//// based on a user-provided fiscal calendar configuration. This allows supporting fiscal years
//// that do not start in January. For example, a fiscal year starting in April 2026.
////
//// Parameters:
////   - cfg: FiscalCalendarConfig struct containing StartYear and StartMonth
////
//// Returns:
////   - A slice of pointers to Period objects, including months, quarters, and the fiscal year.
////
//// Notes:
////   - Month IDs are formatted as "YYYY-MMM" (e.g., "2026-APR")
////   - Quarter IDs are formatted as "YYYY-QX" (e.g., "2026-Q1")
////   - Fiscal year ID is simply the StartYear (e.g., "2026")
////   - All periods are tied to CalendarGregorian (for now, no holiday/fiscal adjustments)
////
//// Example:
////
////	cfg := FiscalCalendarConfig{StartYear: 2026, StartMonth: time.April}
////	periods := GenerateFiscalPeriods(cfg)
////	// periods will include:
////	//   Months: APR 2026 → MAR 2027
////	//   Quarters: Q1 → Q4
////	//   Fiscal Year: FY2026
////
//// GenerateFiscalPeriods creates fiscal periods (Year, Quarters, Months) for a given fiscal year.
//// It reuses existing Month objects from the store, generates fiscal quarters according to the
//// fiscal start month, and creates a fiscal year object.
////
//// Parameters:
////   - fiscalYear: the year in which the fiscal year starts (e.g., 2026)
////   - fiscalStartMonth: the month the fiscal year starts (e.g., time.April)
////   - existingMonths: the slice of all months that have already been generated for this calendar year(s)
////
//// Returns:
////   - a slice of Period pointers representing the fiscal year, its quarters, and months
//func GenerateFiscalPeriods(fiscalYear int, fiscalStartMonth time.Month, existingMonths []*Period) []*Period {
//	var periods []*Period
//
//	// Step 1: Determine the start and end date of the fiscal year
//	// Fiscal year starts on the 1st of fiscalStartMonth
//	startDate := time.Date(fiscalYear, fiscalStartMonth, 1, 0, 0, 0, 0, time.UTC)
//
//	// Fiscal year ends one year later minus 1 day
//	endDate := startDate.AddDate(1, 0, 0).Add(-time.Nanosecond)
//
//	// Step 2: Create the Fiscal Year period
//	fiscalYearPeriod := &Period{
//		ID:             fmt.Sprintf("FY-%d", fiscalYear), // e.g., "FY-2026"
//		Granularity:    CalendarYearPeriod,
//		StartDate:      startDate,
//		EndDate:        endDate,
//		ParentPeriodID: nil, // top-level period
//		Calendar:       CalendarGregorian,
//	}
//
//	periods = append(periods, fiscalYearPeriod)
//
//	// Step 3: Generate Fiscal Quarters
//	// Each quarter is 3 months long; we rotate months starting from fiscalStartMonth
//	monthIndex := int(fiscalStartMonth) - 1 // time.Month is 1-indexed (January=1)
//	for q := 1; q <= 4; q++ {
//		// Determine start and end month indices for the quarter
//		qStartMonth := monthIndex
//		qEndMonth := (monthIndex + 2) % 12
//
//		// Find actual Month objects from existingMonths
//		var quarterStartDate, quarterEndDate time.Time
//		var quarterMonths []*Period
//
//		for _, m := range existingMonths {
//			if m == nil {
//				continue
//			}
//
//			// Determine if this month is part of the current quarter
//			mMonthIndex := int(m.StartDate.Month()) - 1
//			// Handle wrap-around: if fiscalStartMonth is not January, some months belong to next calendar year
//			if qStartMonth <= qEndMonth {
//				if mMonthIndex >= qStartMonth && mMonthIndex <= qEndMonth && m.StartDate.Year() == startDate.Year() || m.StartDate.Year() == startDate.Year()+1 && mMonthIndex <= qEndMonth && qEndMonth < qStartMonth {
//					quarterMonths = append(quarterMonths, m)
//				}
//			} else {
//				// Wrap-around case
//				if mMonthIndex >= qStartMonth || mMonthIndex <= qEndMonth {
//					quarterMonths = append(quarterMonths, m)
//				}
//			}
//		}
//
//		if len(quarterMonths) == 0 {
//			continue // skip empty quarters (should not happen if months exist)
//		}
//
//		// Set quarter start and end dates based on the first and last month in the quarter
//		quarterStartDate = quarterMonths[0].StartDate
//		quarterEndDate = quarterMonths[len(quarterMonths)-1].EndDate
//
//		// Create the quarter period
//		quarterPeriod := &Period{
//			ID:             fmt.Sprintf("FY-%d-Q%d", fiscalYear, q), // e.g., "FY-2026-Q1"
//			Granularity:    QuarterPeriod,
//			StartDate:      quarterStartDate,
//			EndDate:        quarterEndDate,
//			ParentPeriodID: &fiscalYearPeriod.ID, // parent is the fiscal year
//			Calendar:       CalendarGregorian,
//		}
//
//		periods = append(periods, quarterPeriod)
//
//		// Step 4: Link months to their parent quarter
//		for _, m := range quarterMonths {
//			m.ParentPeriodID = &quarterPeriod.ID
//		}
//
//		// Move to next quarter
//		monthIndex = (monthIndex + 3) % 12
//	}
//
//	// Step 5: Append all months (already linked to quarters) to the periods slice
//	periods = append(periods, existingMonths...)
//
//	// Step 6: Return the full slice: fiscal year, quarters, months
//	return periods
//}
