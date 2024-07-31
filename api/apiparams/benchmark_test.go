package apiparams_test

import (
	"encoding/json"
	"github.com/lithictech/go-aperitif/v2/api/apiparams"
	"github.com/lithictech/go-aperitif/v2/convext"
	"net/http"
	"strings"
	"testing"
)

type NullAdapter struct {
	request     *http.Request
	paramNames  []string
	paramValues []string
}

func (a NullAdapter) Request([]interface{}) *http.Request {
	return a.request
}
func (a NullAdapter) RouteParamNames([]interface{}) []string {
	return a.paramNames
}
func (a NullAdapter) RouteParamValues([]interface{}) []string {
	return a.paramValues
}

type paramsDef struct {
	S string  `json:"s"`
	I int     `json:"i"`
	B bool    `json:"b"`
	F float64 `json:"f"`
}

const jsonStr = `{"i":10, "s":"abc", "b":true, "f":1.1}`

var paramNames = []string{"i", "s", "b", "f"}
var paramValues = []string{"10", "abc", "true", "1.1"}

// Benchmarks for apiparams.
// Compare its binding performance against path, query, and body params
// to a simple JSON unmarshal.

// Benchmark the speed of a native JSON unmarshal/bind to the same body
func BenchmarkNativeJSONBind(b *testing.B) {
	for n := 0; n < b.N; n++ {
		pd := paramsDef{}
		convext.Must(json.Unmarshal([]byte(jsonStr), &pd))
	}
}

// Benchmark the speed of apiparams's binding to a JSON POST body.
// This should be very similar to the native performance (since it uses that under the hood),
// with the overhead of extra stuff apiparams is doing that isn't part of binding
// (like defaults and validation, not used here).
func BenchmarkAPIParamsJSONBind(b *testing.B) {
	req, err := http.NewRequest("POST", "/foo", strings.NewReader(jsonStr))
	if err != nil {
		panic(err.Error())
	}
	adapter := NullAdapter{request: req}

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		pd := paramsDef{}
		ph := apiparams.New(adapter, &pd)
		convext.Must(ph.BindFromAll())
	}
}

// Benchmark the speed of apiparams's binding to path params.
// This exercises the reflection judo in apiparams.
func BenchmarkAPIParamsParamsBind(b *testing.B) {
	req, err := http.NewRequest("GET", "/foo", nil)
	if err != nil {
		panic(err.Error())
	}
	adapter := NullAdapter{req, paramNames, paramValues}

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		pd := paramsDef{}
		ph := apiparams.New(adapter, &pd)
		convext.Must(ph.BindFromAll())
	}
}

// Benchmark the speed of apiparams's binding to query params.
// This exercises the reflection judo in apiparams.
func BenchmarkAPIParamsQueryParamsBind(b *testing.B) {
	url := "/foo?"
	for i, name := range paramNames {
		val := paramValues[i]
		url += name + "=" + val + "&"
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		panic(err.Error())
	}
	adapter := NullAdapter{request: req}

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		pd := paramsDef{}
		ph := apiparams.New(adapter, &pd)
		convext.Must(ph.BindFromAll())
	}
}
