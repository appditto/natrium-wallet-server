package models

import "github.com/google/uuid"

// account_subscribe request
type AccountSubscribe struct {
	Action              string     `json:"action" mapstructure:"action"`
	Uuid                *uuid.UUID `json:"uuid,omitempty" mapstructure:"uuid,omitempty"`
	Account             string     `json:"account" mapstructure:"account"`
	Currency            *string    `json:"currency,omitempty" mapstructure:"currency,omitempty"`
	FcmToken            string     `json:"fcm_token_v2" mapstructure:"fcm_token_v2"`
	NotificationEnabled bool       `json:"notification_enabled" mapstructure:"notification_enabled"`
}
