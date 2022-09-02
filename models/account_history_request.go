package models

// rpc account_history
type AccountHistory struct {
	BaseRequest
	Account string `json:"account" mapstructure:"account"`
	Count   *int   `json:"count,omitempty" mapstructure:"count,omitempty"`
}
