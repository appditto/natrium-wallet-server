package models

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/stretchr/testify/assert"
)

func TestEncodeAccountSubscribe(t *testing.T) {
	uuid := uuid.MustParse("f47ac10b-58cc-0372-8567-0e02b2c3d479").String()
	request := AccountSubscribe{
		Action:  "account_subscribe",
		Account: "1",
		Uuid:    &uuid,
	}
	expected := `{"action":"account_subscribe","uuid":"f47ac10b-58cc-0372-8567-0e02b2c3d479","account":"1","fcm_token_v2":"","notification_enabled":false}`
	serialized, _ := json.Marshal(request)
	assert.Equal(t, expected, string(serialized))
}

func TestDecodeAccountSubscribeRequest(t *testing.T) {
	encoded := `{"action":"account_subscribe","uuid":"f47ac10b-58cc-0372-8567-0e02b2c3d479","account":"1"}`
	var decoded AccountSubscribe
	json.Unmarshal([]byte(encoded), &decoded)
	assert.Equal(t, "account_subscribe", decoded.Action)
	assert.Equal(t, "1", decoded.Account)
	assert.Equal(t, "f47ac10b-58cc-0372-8567-0e02b2c3d479", *decoded.Uuid)
}

func TestMapStructureDecodeAccountSubscribeRequest(t *testing.T) {
	id := uuid.MustParse("f47ac10b-58cc-0372-8567-0e02b2c3d479")
	request := map[string]interface{}{
		"action":  "account_subscribe",
		"account": "1",
		"uuid":    id.String(),
	}
	var decoded AccountSubscribe
	mapstructure.Decode(request, &decoded)
	assert.Equal(t, "account_subscribe", decoded.Action)
	assert.Equal(t, "1", decoded.Account)
	// mapstructure can't handle UUIDs so ensure it doesnt get decoded
	assert.Equal(t, id.String(), *decoded.Uuid)
}
