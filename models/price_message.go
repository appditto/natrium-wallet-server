package models

type PriceMessage struct {
	Currency  string  `json:"currency"`
	Price     float64 `json:"price"`
	BtcPrice  float64 `json:"btc"`
	NanoPrice float64 `json:"nano"`
}
