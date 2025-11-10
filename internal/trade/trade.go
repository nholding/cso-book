package trade

import (
	"github.com/nholding/cso-book/internal/audit"
)

// TradeBase contains fields common to both Purchase and Sale trades.
// Each trade references a single PeriodID (month, quarter, or year).
type TradeBase struct {
	ID                string          `json:"id"`
	BusinessKey       string          `json:"business_key"`
	PeriodID          string          `json:"period_id"`
	ProductID         string          `json:"product_id"`
	VolumeMT          float64         `json:"volume_mt"`
	PricePerMT        float64         `json:"price_per_mt"`
	Currency          string          `json:"currency"`
	BrokerID          string          `json:"broker_id"`
	BrokerContractID  string          `json:"broker_contract_id"`
	StorageLocationID string          `json:"storage_location_id"`
	AuditInfo         audit.AuditInfo `json:"audit"`
}

type Purchase struct {
	TradeBase
	SellerID string `json:"seller_id"`
	HolderID string `json:"holder_id"`
	OwnerID  string `json:"owner_id"`
}

type Sale struct {
	TradeBase
	BuyerID            string `json:"buyer_id"`
	BuyerBeneficiaryID string `json:"buyer_beneficiary_id"`
}

func NewPurchase(id string) *Purchase {
	p := Purchase{}

	return &p
}
