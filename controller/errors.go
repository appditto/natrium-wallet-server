package controller

import (
	"net/http"

	"github.com/go-chi/render"
)

type ErrorResponse struct {
	Error string `json:"error"`
}

var InvalidRequestError = ErrorResponse{
	Error: "The request was invalid and not recognized",
}

func ErrInvalidRequest(w http.ResponseWriter, r *http.Request) {
	render.Status(r, http.StatusBadRequest)
	render.JSON(w, r, &InvalidRequestError)
}

var UnsupportedActionError = ErrorResponse{
	Error: "The requested action is not supported in this API",
}

func ErrUnsupportedAction(w http.ResponseWriter, r *http.Request) {
	// We return a 200 since it's what the old API did, it maintains compatibility
	render.Status(r, http.StatusOK)
	render.JSON(w, r, &UnsupportedActionError)
}

func ErrBadrequest(w http.ResponseWriter, r *http.Request, errorText string) {
	render.Status(r, http.StatusBadRequest)
	render.JSON(w, r, &ErrorResponse{
		Error: errorText,
	})
}

func ErrInternalServerError(w http.ResponseWriter, r *http.Request, errorText string) {
	render.Status(r, http.StatusInternalServerError)
	render.JSON(w, r, &ErrorResponse{
		Error: errorText,
	})
}
