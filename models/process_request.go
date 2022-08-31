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

type ProcessRequest struct {
	Action    string            `json:"action"`
	Block     *string           `json:"block,omitempty"`
	JsonBlock *ProcessJsonBlock `json:"json_block,omitempty"`
	DoWork    *bool             `json:"do_work,omitempty"`
	SubType   *string           `json:"subtype,omitempty"`
}
