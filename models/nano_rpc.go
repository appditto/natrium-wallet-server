package models

// Requests
type AccountInfoAction struct {
	Action         string `json:"action"`
	Account        string `json:"account"`
	Pending        bool   `json:"pending"`
	Representative bool   `json:"representative"`
}

type ReceivableRequest struct {
	Action               string `json:"action"`
	Account              string `json:"account"`
	Threshold            string `json:"threshold"`
	Count                int    `json:"count"`
	IncludeOnlyConfirmed bool   `json:"include_only_confirmed"`
}

// Responses
type ReceivableResponse struct {
	Blocks []string `json:"blocks"`
}
