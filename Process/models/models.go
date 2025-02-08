package models

type Orders struct {
	OrderId   string
	ClientId  string
	ISIN      string
	Quantity  int64
	BuySell   string
	OrderDate string
}

type Trades struct {
	TradeDate      string
	BrokerCode     string
	ISIN           string
	BuySell        rune
	Quantity       int64
	Price          float64
	NetAmount      float64
	Brokerage      float64
	SettlementDate string
	STT            int32
	ExchangeCode   string
	Depository     string
}

type ReconciliationResult struct {
	OrderID          string
	ClientID         string
	Symbol           string
	OrderQuantity    int
	ExecutedQuantity int
	OrderPrice       float64
	ExecutionPrice   float64
	Slippage         float64
	BrokerageAmount  float64
	STT              float64
	TotalCost        float64
	Status           string
}
