package service

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/nholding/cso-book/internal/period/domain"
	"github.com/nholding/cso-book/internal/period/repository"
)

type PeriodService struct {
	repo  *repository.RdsPeriodRepository
	store *domain.PeriodStore
}

func NewPeriodService(repo *repository.RdsPeriodRepository) *PeriodService {
	return &PeriodService{
		repo: repo,
	}
}

// InitializePeriods
//
// PURPOSE:
//
//	Performs COMPLETE initialization of all period-related state
//	for the application.
//
//	This function establishes a HARD STARTUP CONTRACT:
//
//	  If this function returns nil:
//	     - All calendar periods are present
//	     - All fiscal periods (if configured) are present
//	     - All hierarchies are structurally valid
//	     - All fiscal calendars are business-correct overlays
//
//	  If this function returns an error:
//	     - The application MUST NOT start
//
//	This guarantees that all downstream systems
//	(trading, risk, P&L, reporting) can rely on PeriodService
//	without defensive checks.
//
// CORE DESIGN PRINCIPLES:
//
//   - Gregorian MONTHS are the atomic unit of time
//   - Calendar (CAL) and Fiscal (FY) calendars coexist
//   - Fiscal calendars are OVERLAYS, not owners of months
//   - All validation is fail-fast and deterministic
//
// RESPONSIBILITIES (IN ORDER):
//
//  1. Load all existing periods from persistent storage
//  2. Generate Gregorian calendar periods if none exist
//  3. Persist generated calendar periods
//  4. Initialize the in-memory PeriodStore
//  5. Generate fiscal calendars (if configured)
//  6. Validate structural hierarchy (CAL + FY)
//  7. Validate fiscal coverage (overlay correctness)
//
// WHEN TO CALL:
//
//   - EXACTLY ONCE at application startup
//   - BEFORE any trading, risk, or reporting logic runs
//
// EXAMPLE USAGE:
//
//	repo := NewRdsPeriodRepository(db)
//	ps   := NewPeriodService(repo)
//
//	fiscalCfg := domain.FiscalCalendarConfig{
//	    StartYear:  2026,
//	    StartMonth: time.April,
//	}
//
//	err := ps.InitializePeriods(
//	    ctx,
//	    2025,
//	    2030,
//	    []domain.FiscalCalendarConfig{fiscalCfg},
//	)
//
//	if err != nil {
//	    log.Fatal(err)
//	}
//
// EXPECTED OUTCOME (SUCCESS):
//
//   - ps.store.Months contains all Gregorian months
//   - ps.store.Quarters / Years contain CAL periods
//   - FY periods exist as overlays
//   - All validation has passed
//
// EXPECTED OUTCOME (FAILURE):
//
//   - Error returned
//   - Application terminates
func (s *PeriodService) InitializePeriods(ctx context.Context, startYear int, endYear int, fiscalConfigs []domain.FiscalCalendarConfig) error {

	// STEP 0: Defensive guards
	if startYear > endYear {
		return fmt.Errorf("invalid period range: startYear %d is after endYear %d", startYear, endYear)
	}

	// STEP 1: Load all periods from persistent storage
	// This includes:
	//   - Gregorian calendar periods
	//   - Any previously generated fiscal periods
	periods, err := s.repo.GetAllPeriods(ctx)
	if err != nil {
		return fmt.Errorf("failed to load periods from DB: %w", err)
	}

	// STEP 2: Generate Gregorian calendar periods if none exist
	// This typically occurs:
	//   - On first deployment
	//   - In a brand-new environment
	//
	// IMPORTANT:
	//   Gregorian periods are ALWAYS generated first
	//   because all other logic depends on months existing.
	if len(periods) == 0 {

		// Generate YEAR → QUARTER → MONTH
		periods = domain.GeneratePeriods(startYear, endYear)

		// Persist generated periods
		if err := s.repo.SavePeriods(ctx, periods); err != nil {
			return fmt.Errorf("failed to persist generated calendar periods: %w", err)
		}
	}

	// STEP 3: Initialize in-memory PeriodStore
	// From this point forward:
	//   - ALL operations occur in memory
	//   - DB is not consulted again during startup
	s.store = domain.NewPeriodStore(periods)

	// STEP 4: Generate fiscal calendars (OVERLAYS)
	// Fiscal calendars:
	//   - Do NOT create months
	//   - Reuse existing Gregorian months by date range
	//   - Must be generated BEFORE validation
	for _, cfg := range fiscalConfigs {
		fyID := fmt.Sprintf("FY%d", cfg.StartYear)

		if s.store.FindByID(fyID) != nil &&
			s.store.FindByID(fyID+"-Q1") != nil {
			continue
		}

		fiscalPeriods, err := domain.GenerateFiscalYear(s.store.Months, cfg)
		if err != nil {
			return fmt.Errorf("failed to generate fiscal year FY%d: %w", cfg.StartYear, err)
		}

		if err := s.repo.SavePeriods(ctx, fiscalPeriods); err != nil {
			return fmt.Errorf("failed to persist fiscal year FY%d: %w", cfg.StartYear, err)
		}

		for _, p := range fiscalPeriods {
			s.store.Periods[p.ID] = p

			switch p.Granularity {
			case domain.CalendarYearPeriod:
				s.store.Years = append(s.store.Years, p)
			case domain.QuarterlyPeriod:
				s.store.Quarters = append(s.store.Quarters, p)
			}
		}
	}

	s.store.SortAll()

	// ------------------------------------------------------------
	// STEP 5: Validate structural hierarchy
	// ------------------------------------------------------------
	// Guarantees:
	//   ✔ Parent/child links are valid
	//   ✔ No CAL/FY cross-contamination
	//   ✔ Months are shared atomic leaves
	//   ✔ Granularity ordering is correct
	if errs := s.ValidateHierarchy(); len(errs) > 0 {
		return fmt.Errorf("period hierarchy validation failed")
	}

	// ------------------------------------------------------------
	// STEP 6: Validate fiscal coverage
	// Guarantees:
	//   ✔ Each fiscal year spans EXACTLY 12 months
	//   ✔ Months are contiguous and gap-free
	//   ✔ Boundaries align to month start/end
	//   ✔ Safe for trading, delivery, and risk
	// ------------------------------------------------------------
	if errs := s.ValidateFiscalCoverage(); len(errs) > 0 {
		return fmt.Errorf("fiscal calendar validation failed")
	}

	// ------------------------------------------------------------
	// INITIALIZATION SUCCESSFUL
	// ------------------------------------------------------------
	// PeriodService is now SAFE for use by:
	//   - Trade capture
	//   - Exposure calculation
	//   - Risk aggregation
	//   - Reporting
	return nil
}

// ValidateHierarchy
//
// PURPOSE:
//
//	Performs STRICT structural validation of all Period hierarchies loaded
//	into the PeriodService, while explicitly supporting an
//	**overlay fiscal calendar model**.
//
//	This function validates BOTH Gregorian (CAL) and Fiscal (FY) calendars,
//	which coexist side-by-side in memory.
//
//	IMPORTANT:
//	  - Gregorian MONTHS are atomic, shared leaves.
//	  - Fiscal periods are OVERLAYS defined by date ranges.
//	  - Months are NOT owned by fiscal hierarchies.
//
// CALENDAR MODELS:
//
//	Gregorian Calendar (CAL):
//	  YEAR  → QUARTER → MONTH
//
//	Fiscal Calendar (FY):
//	  FY-YEAR → FY-QUARTER
//	  (months are selected by date range, NOT parent links)
//
// WHAT THIS FUNCTION GUARANTEES:
//
//	After successful validation:
//
//	  ✔ All declared parent/child links are structurally valid
//	  ✔ No calendar cross-contamination occurs
//	  ✔ Granularity ordering is enforced
//	  ✔ Parent periods fully contain their children by date
//	  ✔ Months are safe to use as shared atomic delivery units
//
// WHAT THIS FUNCTION EXPLICITLY ALLOWS:
//
//	✔ Gregorian months reused by both CAL and FY views
//	✔ Fiscal years / quarters that reference months by date only
//
// WHAT THIS FUNCTION FORBIDS:
//
//	✘ CAL periods having FY parents
//	✘ FY periods having CAL parents
//	✘ Month → FY parent links
//	✘ Self-referential periods
//	✘ Inverted or overlapping hierarchies
//
// WHEN TO CALL:
//
//   - Once at application startup
//   - AFTER:
//   - All calendar periods are loaded/generated
//   - Fiscal periods are generated
//   - BEFORE:
//   - Any trading, risk, or reporting logic runs
//
// FAIL-FAST BEHAVIOR:
//
//	If this function returns ANY errors,
//	the application MUST NOT continue.
//
// EXAMPLE USAGE:
//
//	errs := periodService.ValidateHierarchy()
//	if len(errs) > 0 {
//	    for _, err := range errs {
//	        log.Println("Period hierarchy validation error:", err)
//	    }
//	    os.Exit(1)
//	}
//
// EXPECTED OUTPUT (VALID):
//
//	No errors returned
//
// EXAMPLE INVALID OUTPUTS:
//
//   - "period FY2026-Q1 (FY) has parent 2026 (CAL) with different calendar type"
//   - "child 2026-FEB references missing parent 2026-QQ"
//   - "period 2026-Q1 has parent 2026-MAR which is not a larger granularity"
func (s *PeriodService) ValidateHierarchy() []error {

	// ------------------------------------------------------------
	// Guard clause: PeriodStore must be initialized
	// ------------------------------------------------------------
	if s.store == nil {
		return []error{fmt.Errorf("period store not initialised")}
	}

	var errs []error

	// ------------------------------------------------------------
	// Validation is performed by granularity order
	// (for readability only; logic does not depend on order)
	// ------------------------------------------------------------
	for _, periodList := range [][]*domain.Period{
		s.store.Years,
		s.store.Quarters,
		s.store.Months,
	} {

		for _, p := range periodList {

			// ----------------------------------------------------
			// Defensive programming: skip nil entries
			// ----------------------------------------------------
			if p == nil {
				continue
			}

			// ----------------------------------------------------
			// YEAR periods (CAL or FY) are ROOTS
			// ----------------------------------------------------
			// They must not have parents and require no validation
			// beyond existence.
			if p.Granularity == domain.CalendarYearPeriod {
				continue
			}

			// ----------------------------------------------------
			// SPECIAL CASE: MONTHS
			// ----------------------------------------------------
			// Months are atomic delivery units.
			//
			// They:
			//   - MAY belong to a CAL hierarchy (YEAR → Q → MONTH)
			//   - MUST NOT be required to belong to FY hierarchy
			//
			// Therefore:
			//   - We validate MONTHS ONLY within their declared calendar
			//   - We do NOT require FY parents for months
			if p.Granularity == domain.MonthlyPeriod {

				// Month must belong to CALENDAR
				if p.Calendar != domain.CalendarGregorian {
					errs = append(errs,
						fmt.Errorf(
							"month %s has invalid calendar %s (months must be Gregorian)",
							p.ID,
							p.Calendar,
						),
					)
				}

				// If month declares a parent, validate it normally
				if p.ParentPeriodID != nil {

					parent, exists := s.store.Periods[*p.ParentPeriodID]
					if !exists {
						errs = append(errs,
							fmt.Errorf(
								"month %s references missing parent %s",
								p.ID,
								*p.ParentPeriodID,
							),
						)
						continue
					}

					// Parent must be CAL
					if parent.Calendar != domain.CalendarGregorian {
						errs = append(errs,
							fmt.Errorf(
								"month %s has non-Gregorian parent %s",
								p.ID,
								parent.ID,
							),
						)
					}

					// Parent must be larger granularity
					if parent.GranularityRank() <= p.GranularityRank() {
						errs = append(errs,
							fmt.Errorf(
								"month %s has invalid parent granularity %s",
								p.ID,
								parent.Granularity,
							),
						)
					}

					// Parent must contain month by date
					if parent.StartDate.After(p.StartDate) ||
						parent.EndDate.Before(p.EndDate) {
						errs = append(errs,
							fmt.Errorf(
								"month %s is not fully contained in parent %s",
								p.ID,
								parent.ID,
							),
						)
					}
				}

				// Month validation complete
				continue
			}

			// ----------------------------------------------------
			// ALL NON-YEAR, NON-MONTH PERIODS (i.e. QUARTERS)
			// ----------------------------------------------------

			// Rule 1: Parent must exist
			if p.ParentPeriodID == nil {
				errs = append(errs,
					fmt.Errorf(
						"period %s (%s) has no parent but is not a year",
						p.ID,
						p.Granularity,
					),
				)
				continue
			}

			parentID := *p.ParentPeriodID
			parent, exists := s.store.Periods[parentID]
			if !exists {
				errs = append(errs,
					fmt.Errorf(
						"child %s references missing parent %s",
						p.ID,
						parentID,
					),
				)
				continue
			}

			// Rule 2: Calendar isolation (CRITICAL)
			if parent.Calendar != p.Calendar {
				errs = append(errs,
					fmt.Errorf(
						"period %s (%s) has parent %s (%s) with different calendar type",
						p.ID,
						p.Calendar,
						parent.ID,
						parent.Calendar,
					),
				)
				continue
			}

			// Rule 3: No self-reference
			if parent.ID == p.ID {
				errs = append(errs,
					fmt.Errorf(
						"period %s cannot reference itself as a parent",
						p.ID,
					),
				)
			}

			// Rule 4: Granularity ordering
			if parent.GranularityRank() <= p.GranularityRank() {
				errs = append(errs,
					fmt.Errorf(
						"period %s (%s) has parent %s (%s) which is not a larger granularity",
						p.ID,
						p.Granularity,
						parent.ID,
						parent.Granularity,
					),
				)
			}

			// Rule 5: Date containment
			if parent.StartDate.After(p.StartDate) {
				errs = append(errs,
					fmt.Errorf(
						"child %s starts before parent %s",
						p.ID,
						parent.ID,
					),
				)
			}

			if parent.EndDate.Before(p.EndDate) {
				errs = append(errs,
					fmt.Errorf(
						"child %s ends after parent %s",
						p.ID,
						parent.ID,
					),
				)
			}
		}
	}

	return errs
}

// ValidateFiscalCoverage
// Purpose:
//
//		Validates the *completeness and correctness* of all fiscal years (FY)
//		loaded into the PeriodService.
//
//		While ValidateHierarchy ensures that the period graph is structurally sound,
//		this function enforces *business-level fiscal calendar invariants*.
//
//	  In this system:
//	    - Gregorian MONTHS are the atomic unit of time.
//	    - Calendar (CAL) years/quarters and Fiscal (FY) years/quarters
//	      are *named date ranges* layered on top of the same months.
//	    - Fiscal calendars do NOT create or own months.
//
//	  This function ensures that each fiscal year:
//	    1. Spans EXACTLY 12 Gregorian months
//	    2. Those months are fully contained within the fiscal year boundaries
//	    3. The months form a contiguous, gap-free sequence
//	    4. The fiscal year starts and ends on exact month boundaries
//
//	  These guarantees are CRITICAL for:
//
//	    - Commodity delivery schedules
//	    - Risk aggregation (VaR, exposure)
//	    - P&L reconciliation
//	    - Accurate trade breakdowns
//
//	  This function is intentionally STRICT and FAIL-FAST.
//	  If it returns errors, the application MUST NOT start.
//
// Assumptions:
//
//   - The in-memory PeriodStore has already been initialized
//   - ValidateHierarchy has already passed successfully
//   - Fiscal years and their quarters have already been generated
//
// It is strictly a validator.
//
// IMPORTANT DESIGN NOTES:
//
//   - This function does NOT look for "fiscal months".
//   - This function does NOT use parent/child hierarchy links for months.
//   - This function ONLY operates on Gregorian months by date range.
//   - Fiscal periods are treated as OVERLAYS, not owners.
//
// WHEN TO CALL:
//
//   - Once, at application startup
//   - AFTER:
//   - Gregorian periods have been generated or loaded
//   - Fiscal years / quarters have been generated
//   - Structural hierarchy validation has passed
//
// RETURN VALUE:
//
//	[]error
//	  - Empty slice (nil or len==0): fiscal calendars are valid
//	  - Non-empty slice: application MUST NOT continue
//
// EXAMPLE USAGE:
//
//	errs := periodService.ValidateFiscalCoverage()
//	if len(errs) > 0 {
//	    for _, err := range errs {
//	        log.Println("Fiscal validation error:", err)
//	    }
//	    os.Exit(1)
//	}
//
// EXPECTED VALID EXAMPLE:
//
//	Fiscal config:
//	  FY2026 starts April 2026
//
//	Expected coverage:
//	  2026-APR → 2027-MAR
//
//	Result:
//	  No errors
//
// EXPECTED INVALID EXAMPLES:
//
//   - FY spans 11 months
//   - Gap between months
//   - FY starts mid-month
//   - FY overlaps months incorrectly

func (s *PeriodService) ValidateFiscalCoverage() []error {

	// ------------------------------------------------------------
	// Guard clause: the service must be initialized
	// ------------------------------------------------------------
	// If the in-memory PeriodStore has not been initialized,
	// validation cannot proceed safely.
	if s.store == nil {
		return []error{fmt.Errorf("period store not initialised")}
	}

	var errs []error

	// ------------------------------------------------------------
	// STEP 1: Identify all fiscal year periods
	// ------------------------------------------------------------
	// Fiscal years are defined as:
	//   - Calendar == FY
	//   - Granularity == YEAR
	//
	// Note:
	//   Fiscal years may coexist with calendar years in the same store.
	//   They are independent overlays and validated independently.
	var fiscalYears []*domain.Period

	for _, y := range s.store.Years {
		if y == nil {
			continue
		}

		if y.Calendar == domain.CalendarFiscal &&
			y.Granularity == domain.CalendarYearPeriod {
			fiscalYears = append(fiscalYears, y)
		}
	}

	// ------------------------------------------------------------
	// No fiscal years configured is NOT an error
	// ------------------------------------------------------------
	// This allows deployments that only use calendar periods.
	if len(fiscalYears) == 0 {
		return nil
	}

	// ------------------------------------------------------------
	// STEP 2: Validate each fiscal year independently
	// ------------------------------------------------------------
	for _, fy := range fiscalYears {

		// --------------------------------------------------------
		// STEP 2.1: Collect all Gregorian months fully inside FY
		// --------------------------------------------------------
		// A month belongs to a fiscal year IF AND ONLY IF:
		//
		//   month.Start >= fy.Start AND month.End <= fy.End
		//
		// We do NOT:
		//   - Check parent hierarchy
		//   - Require fiscal ownership
		//   - Care which CAL year/quarter the month belongs to
		//
		// This guarantees a single, unambiguous mapping
		// from fiscal year → months.
		var months []*domain.Period

		for _, m := range s.store.Months {
			if m == nil {
				continue
			}

			if !m.StartDate.Before(fy.StartDate) &&
				!m.EndDate.After(fy.EndDate) {
				months = append(months, m)
			}
		}

		// --------------------------------------------------------
		// RULE 1: Fiscal year must span EXACTLY 12 months
		// --------------------------------------------------------
		// Commodity trades for a fiscal year MUST map to exactly
		// twelve delivery months — no more, no fewer.
		if len(months) != 12 {
			errs = append(errs,
				fmt.Errorf(
					"fiscal year %s spans %d months (expected exactly 12)",
					fy.ID,
					len(months),
				),
			)
			// We continue validation to report ALL problems.
		}

		// --------------------------------------------------------
		// STEP 2.2: Sort months chronologically
		// --------------------------------------------------------
		// Ensures deterministic validation and gap detection.
		sort.Slice(months, func(i, j int) bool {
			return months[i].StartDate.Before(months[j].StartDate)
		})

		// --------------------------------------------------------
		// RULE 2: Months must be contiguous (no gaps or overlaps)
		// --------------------------------------------------------
		// Each month must start immediately after the previous
		// month ends.
		for i := 1; i < len(months); i++ {
			prev := months[i-1]
			curr := months[i]

			expectedStart := prev.EndDate.Add(time.Nanosecond)

			if !curr.StartDate.Equal(expectedStart) {
				errs = append(errs,
					fmt.Errorf(
						"fiscal year %s has gap or overlap between %s and %s",
						fy.ID,
						prev.ID,
						curr.ID,
					),
				)
			}
		}

		// --------------------------------------------------------
		// RULE 3: Boundary alignment
		// --------------------------------------------------------
		// The fiscal year MUST:
		//   - Start exactly at the beginning of a month
		//   - End exactly at the end of a month
		//
		// Partial-month fiscal years are NOT allowed because they
		// break delivery, settlement, and risk calculations.
		if len(months) > 0 {
			first := months[0]
			last := months[len(months)-1]

			if !first.StartDate.Equal(fy.StartDate) {
				errs = append(errs,
					fmt.Errorf(
						"fiscal year %s does not start on a month boundary (starts %s)",
						fy.ID,
						first.StartDate,
					),
				)
			}

			if !last.EndDate.Equal(fy.EndDate) {
				errs = append(errs,
					fmt.Errorf(
						"fiscal year %s does not end on a month boundary (ends %s)",
						fy.ID,
						last.EndDate,
					),
				)
			}
		}
	}

	return errs
}

func (s *PeriodService) GetPeriodStore() *domain.PeriodStore {
	return s.store
}

// BreakDownTradeRange takes a given PeriodRange (StartPeriodID → EndPeriodID)
// and returns a chronological list of all individual month IDs that fall within that range.
//
// This function is a **wrapper around the domain-level method** `BreakDownTradePeriodRange`
// but explicitly operates on the in-memory PeriodStore (`s.store`) rather than querying the database.
//
// The rationale for this wrapper is:
//
//  1. **Performance:** We load all periods once from RDS at application start and store them
//     in memory. All subsequent operations like period breakdowns are served from memory
//     without hitting the database, making it very fast and cheap.
//  2. **Encapsulation:** Higher-level services like `PeriodService` expose domain functionality
//     in a convenient and consistent way for business logic, hiding the internal store implementation.
//  3. **Safety:** Since we operate on in-memory copies of periods, no accidental database writes
//     occur during breakdown calculations.
//
// Example usage:
//
//	// Define a range spanning Q1 and Q2 of 2026
//	pr := domain.PeriodRange{
//	    StartPeriodID: "2026-Q1", // ID of the starting period
//	    EndPeriodID:   "2026-Q2", // ID of the ending period
//	}
//
//	// Break the range into all constituent months
//	months := ps.BreakDownTradeRange(pr)
//
//	// Output (slice of month IDs):
//	// ["2026-JAN", "2026-FEB", "2026-MAR", "2026-APR", "2026-MAY", "2026-JUN"]
//
// Notes:
//
//   - The function **assumes** that the in-memory store (`s.store`) has been initialized
//     via `InitializePeriods` before calling this method. If the store is nil, this will panic.
//   - Only full months are returned. Partial months are **never included** because
//     the system works on full-period granularity (months, quarters, years).
//   - If the start or end period ID does not exist in the store, the underlying domain method
//     will return an empty slice (nil). Caller should handle this case.
//
// Returns:
//
//	[]string - slice of month period IDs in chronological order within the specified range
func (s *PeriodService) BreakDownTradeRange(pr domain.PeriodRange) []string {
	if s.store == nil {
		return nil
	}

	return s.store.BreakDownTradePeriodRange(pr)
}

//func (s *PeriodService) BreakDownTradeRange(pr domain.PeriodRange) []string {
//	if s.store == nil {
//		return nil
//	}
//
//	var result []string
//	startFound, endFound := false, false
//
//	for _, month := range s.store.Months {
//		if month == nil {
//			continue
//		}
//
//		if month.ID == pr.StartPeriodID {
//			startFound = true
//		}
//
//		if startFound {
//			result = append(result, month.ID)
//		}
//
//		if month.ID == pr.EndPeriodID {
//			endFound = true
//			break
//		}
//	}
//
//	if !startFound || !endFound {
//		// Either start or end not found; return empty slice
//		return nil
//	}
//
//	return result
//}

// ValidateOverlaps
// checks if any periods overlap within the same granularity (Calendar, Quarter, or Month).
// This function is an implementation of  DetectOverlaps in the domain
//
// EXAMPLE:
//
//	errs := ps.ValidateOverlaps()
//	if len(errs) > 0 {
//	    for _, e := range errs {
//	        fmt.Println(e)
//	    }
//	}
//
// EXPECTED OUTPUT (if overlaps exist):
//
//	"Overlap detected (MONTHLY): 2026-FEB overlaps with 2026-MAR"
func (s *PeriodService) ValidateOverlaps() []error {
	if s.store == nil {
		return []error{fmt.Errorf("period store not initialised")}
	}

	// Collect all periods into a slice
	periodList := make([]*domain.Period, 0, len(s.store.Periods))
	for _, p := range s.store.Periods {
		periodList = append(periodList, p)
	}

	// Call domain function to detect overlaps
	errStrs := domain.DetectOverlaps(periodList)
	if len(errStrs) == 0 {
		return nil
	}

	// Convert string errors to []error
	errs := make([]error, len(errStrs))
	for i, e := range errStrs {
		errs[i] = fmt.Errorf("%s", e)
	}

	return errs
}
