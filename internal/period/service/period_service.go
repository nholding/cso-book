package service

import (
	"context"
	"fmt"

	"github.com/nholding/cso-book/internal/period/domain"
	"github.com/nholding/cso-book/internal/period/repository"
)

type PeriodService struct {
	repo  *repository.RdsPeriodRepository
	store domain.PeriodStore
}

func NewPeriodService(repo *repository.RdsPeriodRepository) *PeriodService {
	return &PeriodService{
		repo:  repo,
		store: nil,
	}
}

// InitializePeriods is the STARTUP function that ensures periods are loaded in memory.
// Loads all periods from RDS (or other persistent storage) and populates the in-memory store.
// This method should be called once at application startup to prepare the PeriodService
// for all subsequent operations.
//
// Example usage:
//
//	service := NewPeriodService(repo)
//	store, err := service.InitializePeriods(ctx, 2026, 2040)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	months := store.BreakDownTradePeriodRange(domain.PeriodRange{StartPeriodID:"2026-Q1", EndPeriodID:"2026-Q2"})
func (s *PeriodService) InitializePeriods(ctx context.Context, startYear, endYear int) error {
	periods, err := s.repo.GetAllPeriods(ctx)
	if err != nil {
		return fmt.Errorf("failed to load periods from DB: %v", err)
	}

	if len(periods) == 0 {
		// No periods in DB → generate them
		periods = domain.GeneratePeriods(startYear, endYear)
		periodPtrs := make([]*domain.Period, len(periods))
		for i := range periods {
			periodPtrs[i] = &periods[i]
		}

		// Insert generated periods into RDS
		if err := s.repo.SavePeriods(ctx, periodPtrs); err != nil {
			return fmt.Errorf("failed to insert periods into DB: %w", err)
		}

		// Initialize in-memory store
		s.store = domain.NewPeriodStore(periods)
	} else {

		// Periods Exists: Load into memory store
		s.store = domain.NewPeriodStore(periods)

	}

	return nil
}

// ValidateHierarchy
// Purpose:
//
//	Performs structural validation of the YEAR → QUARTER → MONTH
//	period graph loaded into the PeriodService.
//
// Rules enforced:
//  1. Every non-CALENDAR period must have an existing parent.
//  2. Child periods must be fully contained in the parent date range.
//  3. ParentPeriodID must not point to itself.
//  4. A period may not claim a parent of the same or smaller granularity.
//
// When to call:
//
//	Typically once during application startup, after the repository
//	loads all period records into memory.
//
// Example:
//
//	 After loading all periods from DB:
//	 periodService := period.NewPeriodService(periodRepository)
//
//	 Validate hierarchical correctness:
//		errs := periodService.ValidateHierarchy()
//		if len(errs) > 0 {
//		    for _, err := range errs {
//		        log.Println("Period validation error:", err)
//		    }
//		    os.Exit(1)
//		} else {
//		    log.Println("Periods OK")
//		}
//
// Expected output if hierarchy is good:
//
//	Periods OK
//
// Example output if invalid:
//
//	Period validation error: child 2026-FEB has missing parent 2026-QQ
//	Period validation error: child 2026-JAN range 2026-01-01–2026-02-01 exceeds parent 2026-Q1 range 2026-01-01–2026-03-31
func (s *PeriodService) ValidateHierarchy() []error {
	var errs []error

	for id, p := range s.store {

		// ------------------------------
		// Rule 1: Parent must exist
		// ------------------------------
		if p.Granularity != domain.GranularityCalendar { // years have no parent
			if p.ParentPeriodID == "" {
				errs = append(errs,
					fmt.Errorf("period %s (%s) has no parent but is not CALENDAR", p.ID, p.Granularity),
				)
				continue
			}

			parent, exists := s.store[p.ParentPeriodID]
			if !exists {
				errs = append(errs,
					fmt.Errorf("child %s references missing parent %s", p.ID, p.ParentPeriodID),
				)
				continue
			}

			// ------------------------------
			// Rule 2: A period cannot be its own parent
			// ------------------------------
			if parent.ID == p.ID {
				errs = append(errs,
					fmt.Errorf("period %s cannot reference itself as a parent", p.ID))
			}

			// ------------------------------
			// Rule 3: Granularity must be strictly increasing
			// ------------------------------
			if parent.GranularityRank() >= p.GranularityRank() {
				errs = append(errs,
					fmt.Errorf("period %s (%s) has parent %s (%s) which is not larger granularity",
						p.ID, p.Granularity, parent.ID, parent.Granularity),
				)
			}

			// ------------------------------
			// Rule 4: Parent range must include child range
			// ------------------------------
			// Parent.Start <= Child.Start
			if parent.StartDate.After(p.StartDate) {
				errs = append(errs,
					fmt.Errorf("child %s starts before parent %s", p.ID, parent.ID))
			}

			// Child.End <= Parent.End
			if parent.EndDate.Before(p.EndDate) {
				errs = append(errs,
					fmt.Errorf("child %s ends after parent %s", p.ID, parent.ID))
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
	return s.store.BreakDownTradePeriodRange(pr)
}

package service

import (
	"fmt"

	"github.com/nholding/cso-book/internal/period/domain"
)

// ValidateOverlaps 
// checks if any periods overlap within the same granularity (Calendar, Quarter, or Month).
// This function is an implementation of DetectOverlaps in the domain
//
// EXAMPLE:
//
//    errs := ps.ValidateOverlaps()
//    if len(errs) > 0 {
//        for _, e := range errs {
//            fmt.Println(e)
//        }
//    }
//
// EXPECTED OUTPUT (if overlaps exist):
//
//    "Overlap detected (MONTHLY): 2026-FEB overlaps with 2026-MAR"
// ============================================================================
func (s *PeriodService) ValidateOverlaps() []string {

	if s.store == nil {
		return []string{"period store not initialised"}
	}

	periodList := s.store.AllPeriods()
	errs := domain.DetectOverlaps(periodList)

	if len(errs) == 0 {
		return nil
	}
	return errs
}

