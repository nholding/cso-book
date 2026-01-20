package domain

// BreakDownTradePeriodRange
// The core function of BreakDownTradePeriodRange is to take a PeriodRange
// (whether it's a single period, a multi-period range, or a full calendar)
// and break it down into the list of all the individual months that the trade spans.
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
// Output: [ "2026-JAN", "2026-FEB", "2026-MAR", "2026-APR", "2026-MAY", "2026-JUN" ]
func (ps *PeriodStore) BreakDownTradePeriodRange(pr PeriodRange) []string {
	startPeriod := ps.FindByID(pr.StartPeriodID)
	endPeriod := ps.FindByID(pr.EndPeriodID)

	// If either start or end period is invalid (not found), return nil
	if startPeriod == nil || endPeriod == nil {
		return nil
	}

	// Guard against reversed ranges (start after end)
	if startPeriod.StartDate.After(endPeriod.EndDate) {
		return nil
	}

	// Prepare a slice to collect the month IDs that fall fully within the period range
	var monthIDs []string

	for _, m := range ps.Months {
		// A month is included IFF it is fully contained in the range:
		//   month.Start >= range.Start AND month.End <= range.End
		if !m.StartDate.Before(startPeriod.StartDate) && !m.EndDate.After(endPeriod.EndDate) {
			monthIDs = append(monthIDs, m.ID)
		}
	}

	return monthIDs
}
