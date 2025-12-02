package trade

// Ticket
// Represents a Ticket sale trade. Distinctive type from Purchase.
type Ticket struct {
	TradeBase
	BuyerID string
}
