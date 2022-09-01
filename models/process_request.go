package models

type ProcessJsonBlock struct {
	Link           string  `json:"link"`
	Type           string  `json:"type"`
	Previous       string  `json:"previous"`
	Balance        string  `json:"balance"`
	Work           *string `json:"work,omitempty"`
	Account        string  `json:"account"`
	Signature      string  `json:"signature"`
	Representative string  `json:"representative"`
}

type ProcessRequestStringBlock struct {
	Action    string  `json:"action"`
	Block     *string `json:"block,omitempty"`
	JsonBlock *bool   `json:"json_block,omitempty"`
	DoWork    *bool   `json:"do_work,omitempty"`
	SubType   *string `json:"subtype,omitempty"`
}

type ProcessRequestJsonBlock struct {
	Action  string            `json:"action"`
	Block   *ProcessJsonBlock `json:"block,omitempty"`
	DoWork  *bool             `json:"do_work,omitempty"`
	SubType *string           `json:"subtype,omitempty"`
}
