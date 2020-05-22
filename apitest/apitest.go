// Package apitest are functions for testing any API (not just Echo).
package apitest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type RequestOption func(*http.Request)

func JsonReq() RequestOption {
	return SetReqHeader("Content-Type", "application/json")
}

func SetReqHeader(key, value string) RequestOption {
	return func(r *http.Request) {
		r.Header.Set(key, value)
	}
}

func SetQueryParam(key string, value interface{}) RequestOption {
	return SetQueryParams(map[string]interface{}{key: value})
}

func SetQueryParams(values map[string]interface{}) RequestOption {
	return func(r *http.Request) {
		query := r.URL.Query()
		for k, v := range values {
			query.Add(k, fmt.Sprintf("%v", v))
		}
		r.URL.RawQuery = query.Encode()
	}
}

func NewRequest(method, url string, body []byte, opts ...RequestOption) *http.Request {
	req, err := http.NewRequest(method, url, bytes.NewReader(body))
	must(err)
	for _, o := range opts {
		o(req)
	}
	return req
}

func GetRequest(url string, opts ...RequestOption) *http.Request {
	return NewRequest(http.MethodGet, url, nil, opts...)
}

func MustMarshal(o interface{}) []byte {
	b, err := json.MarshalIndent(o, "", "  ")
	must(err)
	return b
}

func MustUnmarshal(s string) interface{} {
	var out interface{}
	err := json.Unmarshal([]byte(s), &out)
	must(err)
	return out
}

func MustUnmarshalFrom(r io.Reader) interface{} {
	var out interface{}
	err := json.NewDecoder(r).Decode(&out)
	must(err)
	return out
}

func must(e error) {
	if e != nil {
		panic(e)
	}
}
