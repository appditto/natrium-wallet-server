package models

import (
	"encoding/json"
	"errors"
)

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

type BlockRequest struct {
	Action    string `json:"action"`
	Hash      string `json:"hash"`
	JsonBlock bool   `json:"json_block"`
}

type WorkGenerate struct {
	Action     string `json:"action"`
	Hash       string `json:"hash"`
	Difficulty string `json:"difficulty"`
}

// Responses
type ReceivableResponse struct {
	Blocks map[string]string
}

// UnmarshalJSON is a custom unmarshaler for ReceivableResponse,
// handling the case where "blocks" can be a JSON object or an empty string.
func (r *ReceivableResponse) UnmarshalJSON(data []byte) error {
	// First, try unmarshaling into the expected format.
	var obj struct {
		Blocks map[string]string `json:"blocks"`
	}
	err := json.Unmarshal(data, &obj)
	if err == nil {
		r.Blocks = obj.Blocks
		return nil
	}

	// If the first attempt fails, check if "blocks" is an empty string.
	var altObj struct {
		Blocks json.RawMessage `json:"blocks"`
	}
	err = json.Unmarshal(data, &altObj)
	if err != nil {
		return err
	}

	// Check if blocks is an empty string.
	if string(altObj.Blocks) == `""` {
		r.Blocks = make(map[string]string) // Initialize Blocks as an empty map.
		return nil
	}

	return errors.New("unexpected format for blocks")
}

type BlockContents struct {
	Type           string `json:"type"`
	Account        string `json:"account"`
	Previous       string `json:"previous"`
	Representative string `json:"representative"`
	Balance        string `json:"balance"`
	Link           string `json:"link"`
	LinkAsAccount  string `json:"link_as_account"`
	Signature      string `json:"signature"`
	Work           string `json:"work"`
}

type BlockResponse struct {
	BlockAccount   string        `json:"block_account"`
	Amount         string        `json:"amount"`
	Balance        string        `json:"balance"`
	Height         string        `json:"height"`
	LocalTimestamp string        `json:"local_timestamp"`
	Successor      string        `json:"successor"`
	Confirmed      string        `json:"confirmed"`
	Contents       BlockContents `json:"contents"`
	Subtype        string        `json:"subtype"`
}

type WorkResponse struct {
	Work       string `json:"work"`
	Difficulty string `json:"difficulty"`
	Multiplier string `json:"multiplier"`
	Hash       string `json:"hash"`
}
