package trade

import (
	"github.com/google/uuid"
	"github.com/nholding/cso-book/internal/audit"
	"github.com/nholding/cso-book/internal/period"
	"time"
)

// TradeBase
// Common fields for both Purchases and Sales. Includes PeriodRange.
//
// Example:
//
//	tb := TradeBase{
//	    ID: "T1",
//	    PeriodRange: period.PeriodRange{
//	        StartPeriodID: "2026-Q1",
//	        EndPeriodID: "2026-Q2",
//	    },
//	    VolumeMT: 10000,
//	    PricePerMT: 3.5,
//	    Currency: "EUR",
//	}
type TradeBase struct {
	ID          string
	PeriodRange period.PeriodRange
	VolumeMT    float64
	PricePerMT  float64
	Currency    string
	AuditInfo   audit.AuditInfo
}

// Purchase
// Represents a purchase trade. Distinctive type from Sale.
//
// Example:
//
//	p := Purchase{
//	    TradeBase: tb,
//	    SupplierID: "COMPANY-123",
//	}
type Purchase struct {
	TradeBase
	SupplierID string
}

// Sale
// Represents a sale trade. Distinctive type from Purchase.
//
// Example:
//
//	s := Sale{
//	    TradeBase: tb,
//	    BuyerID: "COMPANY-456",
//	}
type Sale struct {
	TradeBase
	BuyerID string
}

// TradeBreakdown
// Represents a single monthly slice of a trade.
// Example:
//
//	bd := TradeBreakdown{
//	    ID: uuid.NewString(),
//	    ParentTradeID: "T1",
//	    PeriodID: "2026-JAN",
//	    VolumeMT: 10000,
//	    PricePerMT: 3.5,
//	    Value: 35000,
//	}
type TradeBreakdown struct {
	ID            string
	BusinessKey   string
	ParentTradeID string
	PeriodID      string
	StartDate     time.Time
	EndDate       time.Time
	VolumeMT      float64
	PricePerMT    float64
	Currency      string
	Proceed       float64
	AuditInfo     audit.AuditInfo
}

// CreateTradeBreakdowns generates monthly breakdowns for a trade,
// handling multi-month trades by duplicating the breakdown for each month the trade spans.
// Since we deal with full months only, no partial month handling is needed.
// This function now ensures that for each month a trade spans, the full volume and value are attributed to that month.
//
// Parameters:
//   - trade: TradeBase containing trade details and PeriodRange
//   - ps: *PeriodStore (in-memory, preloaded periods)
//
// Returns:
//   - slice of TradeBreakdown (one per month covered by trade)
//
// Example:
//
//	tb := TradeBase{
//	    ID: "T1",
//	    PeriodRange: period.PeriodRange{
//	        StartPeriodID: "2026-Q1",
//	        EndPeriodID:   "2026-Q2",
//	    },
//	    VolumeMT:   10000,
//	    PricePerMT: 3.5,
//	    Currency:   "EUR",
//	}
//
//	ps := period.NewPeriodStore(allPeriods)
//
//	breakdowns := CreateTradeBreakdowns(tb, ps, "user@internal.local")
//
//	// Output breakdowns (6 months: Jan-Jun 2026):
//	// [
//	//   {PeriodID: "2026-JAN", Value: 35000},
//	//   {PeriodID: "2026-FEB", Value: 35000},
//	//   {PeriodID: "2026-MAR", Value: 35000},
//	//   {PeriodID: "2026-APR", Value: 35000},
//	//   {PeriodID: "2026-MAY", Value: 35000},
//	//   {PeriodID: "2026-JUN", Value: 35000},
//	// ]
func CreateTradeBreakdowns(trade TradeBase, ps *period.PeriodStore, createdBy string) []TradeBreakdown {
	// Prepare an empty slice to store the breakdowns for each month
	var breakdowns []TradeBreakdown

	// Step 1: Flatten PeriodRange into all constituent month IDs
	// Here, we get the list of months that fall within the trade's start and end period range
	// Note: The BreakDownTradePeriodRange function handles multi-month ranges and ensures full month handling.
	monthIDs := ps.BreakDownTradePeriodRange(trade.PeriodRange)

	// Step 2: Create a TradeBreakdown for each month
	// For each month that the trade spans, create a TradeBreakdown
	for _, monthID := range monthIDs {
		p := ps.FindByID(monthID) // Find the period object for this month
		if p == nil {
			continue // skip if month not found (should not happen if periods are preloaded)
		}

		// Here, we simply use the full trade volume for each month in the range
		// There are no fractional calculations since we’re dealing with full months only
		volume := trade.VolumeMT
		value := volume * trade.PricePerMT // Total value for the entire month

		bd := TradeBreakdown{
			ID:            uuid.NewString(),
			ParentTradeID: trade.ID,
			PeriodID:      p.ID,
			StartDate:     p.StartDate,
			EndDate:       p.EndDate,
			VolumeMT:      volume,
			PricePerMT:    trade.PricePerMT,
			Currency:      trade.Currency,
			Proceed:       value,
			AuditInfo:     *audit.NewAuditInfo(createdBy),
		}

		// Append the breakdown for this month to the result slice
		breakdowns = append(breakdowns, bd)
	}

	return breakdowns
}
