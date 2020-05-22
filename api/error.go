package api

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type Error struct {
	HTTPStatus int
	ErrorCode  string
	Message    string
	Original   error
}

func (e Error) Error() string {
	s := fmt.Sprintf("%s: [%d] %s", e.ErrorCode, e.HTTPStatus, e.Message)
	if e.Original != nil {
		s += " (Original: " + e.Original.Error() + ")"
	}
	return s
}

func (e Error) ToMap() map[string]interface{} {
	m := map[string]interface{}{
		"http_status": e.HTTPStatus,
		"error_code":  e.ErrorCode,
		"message":     e.Message,
	}
	if e.Original != nil {
		m["original"] = e.Original.Error()
	}
	return m
}

func (e Error) MarshalJSON() ([]byte, error) {
	return json.Marshal(e.ToMap())
}

func NewError(httpStatus int, errorCode string, original ...error) Error {
	e := Error{
		ErrorCode:  errorCode,
		HTTPStatus: httpStatus,
		Message:    http.StatusText(httpStatus),
	}
	if len(original) > 0 {
		e.Original = original[0]
	}
	return e
}

func NewInternalError(original ...error) Error {
	return NewError(500, "internal_error", original...)
}
