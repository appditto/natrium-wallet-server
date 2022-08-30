package models

// Callback received from nano node
type CallbackBlock struct {
	LinkAsAccount string `json:"link_as_account"`
	Balance       string `json:"balance"`
	Previous      string `json:"previous"`
}

type Callback struct {
	Hash  string `json:"hash"`
	Block CallbackBlock
}
