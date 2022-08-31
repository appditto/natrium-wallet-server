package models

// rpc account_history
type AccountHistory struct {
	BaseRequest
	Account string `json:"account"`
	Count   *int   `json:"count,omitempty"`
}
