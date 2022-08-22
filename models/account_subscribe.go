package models

import "github.com/google/uuid"

// account_subscribe request
type AccountSubscribe struct {
	BaseRequest
	Uuid                *uuid.UUID `json:"uuid"`
	Account             string     `json:"account"`
	Currency            string     `json:"currency"`
	FcmToken            string     `json:"fcm_token_v2"`
	NotificationEnabled bool       `json:"notification_enabled"`
}
