package models

import (
	"encoding/json"
	"testing"

	"github.com/mitchellh/mapstructure"
	"github.com/stretchr/testify/assert"
)

func TestEncodeAccountHistory(t *testing.T) {
	request := AccountHistory{
		Action:  "account_history",
		Account: "1",
	}
	expected := `{"action":"account_history","account":"1"}`
	serialized, _ := json.Marshal(request)
	assert.Equal(t, expected, string(serialized))

	// With count
	count := 15
	request = AccountHistory{
		Action:  "account_history",
		Account: "1",
		Count:   &count,
	}
	expected = `{"action":"account_history","account":"1","count":15}`
	serialized, _ = json.Marshal(request)
	assert.Equal(t, expected, string(serialized))
}

func TestDecodeAccountHistoryRequest(t *testing.T) {
	encoded := `{"action":"account_history","account":"1"}`
	var decoded AccountHistory
	json.Unmarshal([]byte(encoded), &decoded)
	assert.Equal(t, "account_history", decoded.Action)
	assert.Equal(t, "1", decoded.Account)
	assert.Equal(t, (*int)(nil), decoded.Count)

	// With count
	encoded = `{"action":"account_history","account":"1","count":15}`
	json.Unmarshal([]byte(encoded), &decoded)
	assert.Equal(t, "account_history", decoded.Action)
	assert.Equal(t, "1", decoded.Account)
	assert.Equal(t, 15, *decoded.Count)

	// With count as string
	encoded = `{"action":"account_history","account":"1","count":"15"}`
	json.Unmarshal([]byte(encoded), &decoded)
	assert.Equal(t, "account_history", decoded.Action)
	assert.Equal(t, "1", decoded.Account)
	assert.Equal(t, 15, *decoded.Count)
}

func TestMapStructureDecodeAccountHistoryRequest(t *testing.T) {
	request := map[string]interface{}{
		"action":  "account_history",
		"account": "1",
	}
	var decoded AccountHistory
	mapstructure.Decode(request, &decoded)
	assert.Equal(t, "account_history", decoded.Action)
	assert.Equal(t, "1", decoded.Account)
	assert.Equal(t, (*int)(nil), decoded.Count)

	// With count as integer
	request = map[string]interface{}{
		"action":  "account_history",
		"account": "1",
		"count":   15,
	}
	mapstructure.Decode(request, &decoded)
	assert.Equal(t, "account_history", decoded.Action)
	assert.Equal(t, "1", decoded.Account)
	assert.Equal(t, 15, *decoded.Count)

	// With count as string
	request = map[string]interface{}{
		"action":  "account_history",
		"account": "1",
		"count":   "15",
	}
	mapstructure.Decode(request, &decoded)
	assert.Equal(t, "account_history", decoded.Action)
	assert.Equal(t, "1", decoded.Account)
	assert.Equal(t, 15, *decoded.Count)
}
