package models

// Base error model
type badRequestError struct {
	Error string `json:"error"`
	Code  int    `json:"code"`
}

// Specific errors
var INVALID_REQUEST_ERR badRequestError = badRequestError{
	Error: "The request was invalid and not recognized",
	Code:  400,
}
