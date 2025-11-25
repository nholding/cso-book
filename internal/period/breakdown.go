package period

// BreakDownTradePeriod returns all month IDs for a single period (month, quarter, or year)
//
// Examples:
//
//	// YEAR
//	ps.BreakDownTradePeriod("2026")
//	→ ["2026-JAN" ... "2026-DEC"]
//
//	// QUARTER
//	ps.BreakDownTradePeriod("2026-Q3")
//	→ ["2026-JUL", "2026-AUG", "2026-SEP"]
//
//	// MONTH (identity)
//	ps.BreakDownTradePeriod("2026-FEB")
//	→ ["2026-FEB"]
func (ps *PeriodStore) BreakDownTradePeriod(periodID string) []string {
	p := ps.FindByID(periodID)
	if p == nil {
		return nil
	}
	if p.Granularity == MonthlyPeriod {
		return []string{p.ID}
	}

	var monthIDs []string
	for _, childID := range p.ChildPeriodIDs {
		child := ps.FindByID(childID)
		if child == nil {
			continue
		}
		if child.Granularity == QuarterlyPeriod {
			monthIDs = append(monthIDs, ps.BreakDownTradePeriod(child.ID)...)
		} else if child.Granularity == MonthlyPeriod {
			monthIDs = append(monthIDs, child.ID)
		}
	}
	return monthIDs
}

// BreakDownTradePeriodRange
// The BreakDownTradePeriodRange function is used to break down a range of periods into all the individual months that fall within the range.
// Given a period range StartPeriodID → EndPeriodID, returns ALL month period IDs in that range.
// The function checks the given period range (e.g., "2026-Q1" to "2026-Q2") and returns all the month
// IDs that fall within that range. Since we only deal with full months, there's no need to handle partial months
// or overlapping periods.
//
// Example:
//
//	pr := PeriodRange{
//	    StartPeriodID: "2026-Q1",
//	    EndPeriodID:   "2026-Q2",
//	}
//	months := ps.BreakDownTradePeriodRange(pr)
//
// Output:
//
//	[
//	    "2026-JAN", "2026-FEB", "2026-MAR",
//	    "2026-APR", "2026-MAY", "2026-JUN"
//	]
func (ps *PeriodStore) BreakDownTradePeriodRange(pr PeriodRange) []string {
	startPeriod := ps.FindByID(pr.StartPeriodID)
	endPeriod := ps.FindByID(pr.EndPeriodID)

	// If either start or end period is invalid (not found), return an empty slice
	if startPeriod == nil || endPeriod == nil {
		return nil
	}

	// --- Step 1: Determine actual start and end dates ---
	startDate := startPeriod.StartDate
	endDate := endPeriod.EndDate

	// Prepare a slice to collect the month IDs that fall within the period range
	var monthIDs []string
	for _, m := range ps.Months {
		// We simply check if the month's start date is between the start and end period's range
		// We do not need to worry about partial months because all trades are for full months.
		// This ensures that a trade is evenly spread across the months in the range.
		if !m.StartDate.Before(startDate) && !m.StartDate.After(endDate) {
			// If the month's start date is within the range, add it to the result list
			monthIDs = append(monthIDs, m.ID)
		}
	}
	return monthIDs
}
