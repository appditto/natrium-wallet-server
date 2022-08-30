package dbmodels

// Store FCM tokens in database for push notifications
type FcmToken struct {
	Base
	FcmToken string `json:"fcm_token" gorm:"index:fcm_token_index,unique"`
	Account  string `json:"account" gorm:"index:fcm_token_index,unique"`
}
