package models

// Callback received from nano node
type CallbackBlock struct {
	LinkAsAccount string `json:"link_as_account"`
	Balance       string `json:"balance"`
	Previous      string `json:"previous"`
	Subtype       string `json:"subtype"`
}

type Callback struct {
	Hash    string `json:"hash"`
	Block   string `json:"block"`
	Account string `json:"account"`
	Amount  string `json:"amount"`
	Subtype string `json:"subtype"`
	IsSend  string `json:"is_send"`
}
