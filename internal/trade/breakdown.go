package trade

import (
	"github.com/google/uuid"
	"github.com/nholding/cso-book/internal/audit"
	"github.com/nholding/cso-book/internal/period"

	"time"
)

// TradeBreakdown represents a single month slice of a multi-month trade.
// For example, if we sell Q1 2026 (covering Jan–Mar), we will have 3 TradeBreakdowns: one for each month.
// Include in TradeBreakdown only what is needed for reporting and monthly calculations.
//
// Each breakdown is calculated independently, with its own value
// (Volume * Price).
type TradeBreakdown struct {
	ID            string          `json:"id"`
	BusinessKey   string          `json:"business_key"`
	ParentTradeID string          `json:"parent_trade_id"`
	PeriodID      string          `json:"period_id"`
	StartDate     time.Time       `json:"start_date"`
	EndDate       time.Time       `json:"end_date"`
	VolumeMT      float64         `json:"volume_mt"`
	PricePerMT    float64         `json:"price_per_mt"`
	Currency      string          `json:"currency"`
	Value         float64         `json:"value"`
	AuditInfo     audit.AuditInfo `json:"audit"`
}

// CreateTradeBreakdowns generates monthly breakdowns for a trade, based on its PeriodID (which could be a quarter or year).
//
// Example:
//
//	sale := Sale{TradeBase{ID: "S1", PeriodID: "2026-Q1", VolumeMT: 10000, PricePerMT: 3.50, Currency: "EUR"}}
//	bds := CreateTradeBreakdowns(sale.TradeBase, periods) -> Returns 3 monthly breakdowns (Jan, Feb, Mar)
func CreateTradeBreakdowns(trade TradeBase, allPeriods []period.Period, createdBy string) []TradeBreakdown {
	monthIDs := period.BreakDownTradePeriod(trade.PeriodID, allPeriods)
	var breakdowns []TradeBreakdown

	for _, monthID := range monthIDs {
		p := period.FindPeriodByID(allPeriods, monthID)
		if p == nil {
			continue
		}
		bd := TradeBreakdown{
			ID:            uuid.NewString(),
			ParentTradeID: trade.ID,
			PeriodID:      p.ID,
			StartDate:     p.StartDate,
			EndDate:       p.EndDate,
			VolumeMT:      trade.VolumeMT,
			PricePerMT:    trade.PricePerMT,
			Currency:      trade.Currency,
			Value:         trade.VolumeMT * trade.PricePerMT,
			AuditInfo:     *audit.NewAuditInfo(createdBy),
		}
		breakdowns = append(breakdowns, bd)
	}

	return breakdowns
}
