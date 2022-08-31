package models

type FcmUpdate struct {
	BaseRequest
	FcmToken string `json:"fcm_token_v2"`
	Account  string `json:"account"`
	Enabled  bool   `json:"enabled"`
}
