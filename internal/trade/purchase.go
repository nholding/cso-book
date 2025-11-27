package trade

import (
	"github.com/nholding/cso-book/internal/period"
)

// Purchase
// Represents a purchase trade .
type Purchase struct {
	TradeBase
	SupplierID string
}

func NewPurchase(ps period.PeriodStore, supplierName string, pr period.PeriodRange, volumeMT, pricePerMT float64, currency, createdBy string) (Purchase, []TradeBreakdown) {
	// User does NOT provide status. The new purchase ALWAYS starts as Pending.
	p := Purchase{
		TradeBase:  *NewTradeBase(pr, volumeMT, pricePerMT, currency, createdBy),
		SupplierID: "TestSupplierID",
	}

	breakdowns := CreateTradeBreakdowns(p.TradeBase, &ps, createdBy)

	return p, breakdowns
}

func (p *Purchase) UpdateAvailabilityFee(newAvailabilityFee float64) {
	p.TradeBase.PricePerMT = newAvailabilityFee
}
