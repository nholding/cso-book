package trade

import (
	"github.com/nholding/cso-book/internal/audit"
	"github.com/nholding/cso-book/internal/period"

	"fmt"
	"time"
)

// DRAFT: A trader has created the trade internally but has not yet received external confirmation.
// PENDING: The trader has reached a verbal agreement with counterparty, but the contractual confirmation (recap) is not signed yet.
// CONFIRMED: The counterparty has: a) confirmed the deal, b) recap has been exchanged, and c) deal is contractually binding
// VOIDED: Trade is explicitly cancelled, but was previously confirmed or pending.
// SUPERSEDED: Used when a trade gets replaced by a revised version (e.g., amended volume or new price).
//
// Example lifecycle:
// 1. Trader sets up trade → DRAFT
// 2. Negotiation ongoing → DRAFT
// 3. Trader & counterparty verbally agree → PENDING
// 4. Recap exchanged & confirmed → CONFIRMED
// 5. If buyer rejects recap → CANCELLED
// 6. If revised trade issued → SUPERSEDED
const (
	TradeStatusDraft      TradeStatus = "DRAFT"
	TradeStatusPending    TradeStatus = "PENDING-CONFIRMATION"
	TradeStatusConfirmed  TradeStatus = "CONFIRMED"
	TradeStatusCancelled  TradeStatus = "CANCELLED"
	TradeStatusSuperseded TradeStatus = "SUPERSEDED"
)

type TradeStatus string

type TradeStatusHistory struct {
	OldStatus TradeStatus `json:"oldStatus"`
	NewStatus TradeStatus `json:"newStatus"`
	ChangedAt time.Time   `json:"changedAt"`
	ChangedBy string      `json:"changedBy"`
	Reason    string      `json:"reason,omitempty"` // optional, must be provided for cancellations
}

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
	ID          string               `json:"id"`
	PeriodRange period.PeriodRange   `json:"periodRange"`
	VolumeMT    float64              `json:"volumeMT"`
	PricePerMT  float64              `json:"pricePerMT"`
	Currency    string               `json:"currency"`
	Status      TradeStatus          `json:"status"`
	StatusAudit []TradeStatusHistory `json:"statusAudit"`
	AuditInfo   audit.AuditInfo      `json:"auditInfo"`
}

func NewTradeBase(pr period.PeriodRange, volumeMT, pricePerMT float64, currency, createdBy string) *TradeBase {
	tb := TradeBase{
		ID:          "test",
		PeriodRange: pr,
		VolumeMT:    volumeMT,
		PricePerMT:  pricePerMT,
		Currency:    currency,
		Status:      TradeStatusDraft,
		StatusAudit: []TradeStatusHistory{
			{
				OldStatus: TradeStatusDraft,
				NewStatus: TradeStatusDraft,
				ChangedAt: time.Now().UTC(),
				ChangedBy: createdBy,
				Reason:    "trade creation",
			},
		},
		AuditInfo: *audit.NewAuditInfo(createdBy),
	}

	return &tb
}

// Method to update trade status for any TradeBase (Purchase/Sale)
func (t *TradeBase) UpdateTradeStatus(newStatus TradeStatus, reason, changedBy string) error {
	// Ensure the new status is valid
	if newStatus != "PENDING" && newStatus != "CONFIRMED" && newStatus != "CANCELLED" && newStatus != "SUPERSEDED" {
		return fmt.Errorf("invalid status: %s", newStatus)
	}

	now := time.Now().UTC()
	oldStatus := t.Status

	// Record in status history
	t.StatusAudit = append(t.StatusAudit, TradeStatusHistory{
		OldStatus: oldStatus,
		NewStatus: newStatus,
		ChangedAt: now,
		ChangedBy: changedBy,
		Reason:    reason,
	})

	return nil
}
