package models

type BaseRequest struct {
	Action string `json:"action" mapstructure:"action"`
}
