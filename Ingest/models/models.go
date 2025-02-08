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
