package domain

import (
	"sort"
)

// PeriodStore stores/caches all periods in memory for fast lookups and efficient breakdowns.
// Intended to reduce RDS queries: load all periods at app startup.
//
// Example usage:
//
//	ps := NewPeriodStore(periods)
//	jan2026 := ps.FindByID("2026-JAN")
//	fmt.Println(jan2026.Name) // → "January 2026"
type PeriodStore struct {
	Periods  map[string]*Period // Lookup by ID
	Months   []*Period          // Chronologically sorted months
	Quarters []*Period          // Optional, sorted quarters
	Years    []*Period          // Optional, sorted years
}

// NewPeriodStore initializes a PeriodStore from a slice of Periods.
// It builds both a lookup map and a chronologically sorted months slice.
//
// Example:
//
//	periods := GeneratePeriods(2026, 2026)
//	store := NewPeriodStore(periods)
//	jan := store.FindByID("2026-JAN")
func NewPeriodStore(periods []*Period) *PeriodStore {
	store := &PeriodStore{
		Periods: make(map[string]*Period),
	}

	for _, p := range periods {
		store.Periods[p.ID] = p

		switch p.Granularity {
		case MonthlyPeriod:
			store.Months = append(store.Months, p)
		case QuarterlyPeriod:
			store.Quarters = append(store.Quarters, p)
		case CalendarYearPeriod:
			store.Years = append(store.Years, p)
		}
	}

	// Sort Months by StartDate
	sort.Slice(store.Months, func(i, j int) bool {
		return store.Months[i].StartDate.Before(store.Months[j].StartDate)
	})

	sort.Slice(store.Quarters, func(i, j int) bool {
		return store.Quarters[i].StartDate.Before(store.Quarters[j].StartDate)
	})

	sort.Slice(store.Years, func(i, j int) bool {
		return store.Years[i].StartDate.Before(store.Years[j].StartDate)
	})

	return store
}

// SortAll
//
//	Sorts all PeriodStore slices (Months, Quarters, Years) chronologically by StartDate.
//
// When to call:
//   - After manually adding periods to the store
//   - After generating fiscal years/quarters dynamically
//
// Notes:
//   - Sorting Months is critical for correct behavior of
//     BreakDownTradePeriodRange.
//   - Sorting Years and Quarters ensures validation and
//     traversal logic works predictably.
func (ps *PeriodStore) SortAll() {
	sort.Slice(ps.Months, func(i, j int) bool {
		return ps.Months[i].StartDate.Before(ps.Months[j].StartDate)
	})

	sort.Slice(ps.Quarters, func(i, j int) bool {
		return ps.Quarters[i].StartDate.Before(ps.Quarters[j].StartDate)
	})

	sort.Slice(ps.Years, func(i, j int) bool {
		return ps.Years[i].StartDate.Before(ps.Years[j].StartDate)
	})
}

// FindByID retrieves a period pointer by ID
//
// Example:
//
//	p := store.FindByID("2026-JAN")
//	fmt.Println(p.Name) // → "January 2026"
func (ps *PeriodStore) FindByID(id string) *Period {
	if p, ok := ps.Periods[id]; ok {
		return p
	}
	return nil
}

// Creates a PeriodStore from hardcoded periods. Used for development purposes only.
//
// EXAMPLE: Use this during development BEFORE hooking up AWS.
//
//	ps := period.NewMockPeriodStore()
//	fmt.Println(ps.FindByID("2026-Q1"))
//
// OUTPUT:
//
//	&Period{ID:"2026-Q1", ... }
//
// You can adjust the year range easily while developing.
// ----------------------------------------------------------
func NewMockPeriodStore(startYear, endYear int) *PeriodStore {
	periods := GeneratePeriods(startYear, endYear)
	return NewPeriodStore(periods)
}
