package models

type FcmUpdate struct {
	Action   string `json:"action" mapstructure:"action"`
	FcmToken string `json:"fcm_token_v2" mapstructure:"fcm_token_v2"`
	Account  string `json:"account" mapstructure:"account"`
	Enabled  bool   `json:"enabled" mapstructure:"enabled"`
}
