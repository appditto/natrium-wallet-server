package models

import (
	"encoding/json"
	"testing"

	"github.com/mitchellh/mapstructure"
	"github.com/stretchr/testify/assert"
)

func TestEncodeAccountSubscribe(t *testing.T) {
	request := AccountSubscribe{
		Action:  "account_subscribe",
		Account: "1",
	}
	expected := `{"action":"account_subscribe","account":"1","fcm_token_v2":"","notification_enabled":false}`
	serialized, _ := json.Marshal(request)
	assert.Equal(t, expected, string(serialized))
}

func TestDecodeAccountSubscribeRequest(t *testing.T) {
	encoded := `{"action":"account_subscribe","account":"1"}`
	var decoded AccountSubscribe
	json.Unmarshal([]byte(encoded), &decoded)
	assert.Equal(t, "account_subscribe", decoded.Action)
	assert.Equal(t, "1", decoded.Account)
}

func TestMapStructureDecodeAccountSubscribeRequest(t *testing.T) {
	request := map[string]interface{}{
		"action":  "account_subscribe",
		"account": "1",
	}
	var decoded AccountSubscribe
	mapstructure.Decode(request, &decoded)
	assert.Equal(t, "account_subscribe", decoded.Action)
	assert.Equal(t, "1", decoded.Account)
}
