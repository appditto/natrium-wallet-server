package models

type ProcessJsonBlock struct {
	Link           string  `json:"link" mapstructure:"link"`
	Type           string  `json:"type" mapstructure:"type"`
	Previous       string  `json:"previous" mapstructure:"previous"`
	Balance        string  `json:"balance" mapstructure:"balance"`
	Work           *string `json:"work,omitempty" mapstructure:"work,omitempty"`
	Account        string  `json:"account" mapstructure:"account"`
	Signature      string  `json:"signature" mapstructure:"signature"`
	Representative string  `json:"representative" mapstructure:"representative"`
}

type ProcessRequestStringBlock struct {
	Action    string  `json:"action" mapstructure:"action"`
	Block     *string `json:"block,omitempty" mapstructure:"block,omitempty"`
	JsonBlock *bool   `json:"json_block,omitempty" mapstructure:"json_block,omitempty"`
	DoWork    *bool   `json:"do_work,omitempty" mapstructure:"do_work,omitempty"`
	SubType   *string `json:"subtype,omitempty" mapstructure:"subtype,omitempty"`
}

type ProcessRequestJsonBlock struct {
	Action  string            `json:"action" mapstructure:"action"`
	Block   *ProcessJsonBlock `json:"block,omitempty" mapstructure:"block,omitempty"`
	DoWork  *bool             `json:"do_work,omitempty" mapstructure:"do_work,omitempty"`
	SubType *string           `json:"subtype,omitempty" mapstructure:"subtype,omitempty"`
}
