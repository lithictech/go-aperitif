package apiparams

import (
	"net/http"
	"strings"
)

// HTTPError is an interface for errors returned from request pre-handling.
type HTTPError interface {
	// Code returns the HTTP status code for the error.
	Code() int
	// Messages returns a slice of error strings.
	// If there is only one error, this should contain the same as Message.
	Messages() []string
	// Error fulfills the error interface. Returns Messages, joined with a comma.
	Error() string
}

type httpError struct {
	code     int
	messages []string
}

func (e httpError) Code() int {
	return e.code
}

func (e httpError) Messages() []string {
	return e.messages
}

func (e httpError) Error() string {
	return strings.Join(e.Messages(), ", ")
}

func NewHTTPError(code int, message string) HTTPError {
	if message == "" {
		message = http.StatusText(code)
	}
	return httpError{code, []string{message}}
}
