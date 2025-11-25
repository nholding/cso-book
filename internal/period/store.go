package period

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
func NewPeriodStore(periods []Period) *PeriodStore {
	store := &PeriodStore{
		Periods: make(map[string]*Period),
	}

	for i := range periods {
		p := &periods[i]
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
	// Sort Quarters by StartDate
	sort.Slice(store.Quarters, func(i, j int) bool {
		return store.Quarters[i].StartDate.Before(store.Quarters[j].StartDate)
	})
	// Sort Years by StartDate
	sort.Slice(store.Years, func(i, j int) bool {
		return store.Years[i].StartDate.Before(store.Years[j].StartDate)
	})

	return store
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
