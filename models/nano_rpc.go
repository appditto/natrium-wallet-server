package models

// Requests
type AccountInfoAction struct {
	Action         string `json:"action"`
	Account        string `json:"account"`
	Pending        bool   `json:"pending"`
	Representative bool   `json:"representative"`
}

// Responses
