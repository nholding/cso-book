package trade

import (
	"github.com/nholding/cso-book/internal/audit"
	"github.com/nholding/cso-book/internal/utils"
	"time"
)

// TradeBase contains fields common to both Purchase and Sale trades.
// Each trade references a single PeriodID (month, quarter, or year).
// TODO: omitempty, including broker when we sell directly
type TradeBase struct {
	ID          string `json:"id"`
	BusinessKey string `json:"business_key"`
	Status      string `json:"status"`

	//OwnerID       string `json:"owner_id"`       // e.g. Petronas
	//HolderID      string `json:"holder_id"`      // e.g. ORIM
	SellerID string `json:"seller_id"` // e.g. IOT
	BuyerID  string `json:"buyer_id"`  // Portland
	//BeneficiaryID string `json:"beneficiary_id"` // BP
	PeriodID string `json:"period_id"`

	//StorageLocationID string  `json:"storage_location_id"`
	//ProductID         string  `json:"product_id"`
	//VolumeMT          float64 `json:"volume_mt"`
	//PricePerMT        float64 `json:"price_per_mt"`
	////	Currency          string  `json:"currency"`

	//BrokerID         string `json:"broker_id"`
	//BrokerContractID string `json:"broker_contract_id"`

	TransactionDate time.Time       `json:"transaction_date"`
	AuditInfo       audit.AuditInfo `json:"audit"`
}

type Purchase struct {
	TradeBase
}

type CsoTicket struct {
	TradeBase
}

func NewTradeBase(periodID, status, sellerID, buyerID, createdBy string) TradeBase {
	t := TradeBase{
		ID: utils.GenerateStableID(),
		BusinessKey: utils.GenerateBusinessKey("C1", map[string]string{
			//"product": productID,
			"period": periodID,
		}),

		Status: status,
		//OwnerID: ownerID,
		//HolderID, holderID,
		SellerID: sellerID,
		BuyerID:  buyerID,
		//BeneficiaryID:     beneficiaryID,
		PeriodID: periodID,
		//StorageLocationID: storageLocationID,
		//ProductID:         productID,
		//VolumeMT:          volumeMT,
		//PricePerMT:        pricePerMT,
		//BrokerID:          brokerID,
		//BrokerContractID:  brokerContractID,
		TransactionDate: time.Now(), // TODO: Use nanoseconds?
		AuditInfo: audit.AuditInfo{
			CreatedBy: createdBy,
			CreatedAt: time.Now(), // TODO: Use nanoseconds?
		},
	}

	return t
}

func NewPurchase(periodID, status, sellerID, buyerID, createdBy string) *Purchase {
	tb := NewTradeBase(periodID, status, sellerID, buyerID, "system")
	p := Purchase{
		TradeBase: tb,
	}

	return &p
}

func NewCsoTicket(periodID, status, sellerID, buyerID, createdBy string) *CsoTicket {
	tb := NewTradeBase(periodID, status, sellerID, buyerID, "system")
	ct := CsoTicket{
		TradeBase: tb,
	}

	return &ct
}
