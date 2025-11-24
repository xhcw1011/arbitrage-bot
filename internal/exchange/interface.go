package exchange

// Exchange defines the common interface for all exchanges
type Exchange interface {
	// Market Data
	GetFundingRate(symbol string) (float64, error)
	GetPrice(symbol string) (float64, error)

	// Account
	GetBalance(asset string) (float64, error)
	GetPosition(symbol string) (*Position, error)

	// Trading
	PlaceOrder(req *OrderRequest) (*OrderResponse, error)
	CancelOrder(symbol, orderID string) error
}

type Position struct {
	Symbol        string
	Size          float64
	EntryPrice    float64
	UnrealizedPnL float64
}

type OrderRequest struct {
	Symbol     string
	Side       string // "buy" or "sell"
	Size       float64
	Price      float64
	Type       string // "limit" or "market"
	ReduceOnly bool
}

type OrderResponse struct {
	OrderID string
	Status  string
}
