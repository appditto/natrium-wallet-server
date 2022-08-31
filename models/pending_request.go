package models

type PendingRequest struct {
	Action               string  `json:"action"`
	Account              string  `json:"account"`
	Source               bool    `json:"source"`
	Count                int     `json:"count"`
	IncludeActive        bool    `json:"include_active"`
	Threshold            *string `json:"threshold,omitempty"`
	IncludeOnlyConfirmed *bool   `json:"include_only_confirmed,omitempty"`
}
