package models

// rpc account_history
type AccountHistory struct {
	Action  string 	`json:"action" mapstructure:"action"`
	Account string 	`json:"account" mapstructure:"account"`
	Count   *int   	`json:"count,omitempty" mapstructure:"count,omitempty"`
	Raw     *bool  	`json:"raw,omitempty" mapstructure:"raw,omitempty"`
	Reverse *bool  	`json:"reverse,omitempty" mapstructure:"reverse,omitempty"`
	Head 	*string	`json:"head,omitempty" mapstructure:"head,omitempty"`
}
