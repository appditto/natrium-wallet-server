package models

// account_subscribe request
type AccountSubscribe struct {
	BaseRequest
	Account             string `json:"account"`
	Currency            string `json:"currency"`
	FcmToken            string `json:"fcm_token_v2"`
	NotificationEnabled bool   `json:"notification_enabled"`
}
