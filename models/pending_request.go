package models

type PendingRequest struct {
	Action               string  `json:"action" mapstructure:"action"`
	Account              string  `json:"account" mapstructure:"account"`
	Source               bool    `json:"source" mapstructure:"source"`
	Count                int     `json:"count" mapstructure:"count"`
	IncludeActive        bool    `json:"include_active" mapstructure:"include_active"`
	Threshold            *string `json:"threshold,omitempty" mapstructure:"threshold,omitempty"`
	IncludeOnlyConfirmed *bool   `json:"include_only_confirmed,omitempty" mapstructure:"include_only_confirmed,omitempty"`
}
