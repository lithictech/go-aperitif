package apiparams

import (
	"fmt"
	"github.com/lithictech/go-aperitif/v2/validator"
	"net/http"
	"reflect"
	"time"
)

// Adapter is an abstraction for how a web framework handles the HTTP request,
// and request path names and parameters.
// Methods are called with the same arguments as the framework's handler function.
// See package documentation for more information.
type Adapter interface {
	Request(handlerArgs []interface{}) *http.Request
	RouteParamNames(handlerArgs []interface{}) []string
	RouteParamValues(handlerArgs []interface{}) []string
}

// BindAndValidate binds the struct pointed to by paramsStructPr
// to the requests URL, query, and JSON body parameters.
func BindAndValidate(adapter Adapter, paramsStructPtr interface{}, handlerArgs ...interface{}) HTTPError {
	ph := New(adapter, paramsStructPtr, handlerArgs...)
	if err := ph.BindFromAll(); err != nil {
		return err
	}
	if err := ph.Validate(); err != nil {
		return err
	}
	return nil
}

// Handler coordinates the binding and validation of request parameters.
// See package documentation for more info.
type Handler struct {
	reflector reflector
	binder    binder
}

// New returns a new Handler.
// In general, callers should use apiparams.BindAndValidate,
// rather than dealing with Handler explicitly,
// but it is provided here in case callers only want binding or validating for some reason.
func New(adapter Adapter, paramsStructPtr interface{}, handlerArgs ...interface{}) Handler {
	ref := newReflector(paramsStructPtr)
	req := adapter.Request(handlerArgs)
	binder := newBinder(ref, req, adapter.RouteParamNames(handlerArgs), adapter.RouteParamValues(handlerArgs))
	ph := Handler{ref, binder}
	for _, def := range defaultCustomTypes {
		ph.registerCustomType(def)
	}
	return ph
}

// BindFromAll fills in the struct instance from defaults, the JSON body, query params, and path params.
func (ph Handler) BindFromAll() HTTPError {
	return ph.binder.BindFromAll()
}

// Validate calls go-validate.Validate on the (bound) parameter struct,
// and returns an HTTPError if there were validation errors,
// or NoHTTPError if there were none.
func (ph Handler) Validate() HTTPError {
	if err := validator.Validate(ph.reflector.Pointer()); err != nil {
		errMap, ok := err.(validator.ErrorMap)
		if !ok {
			return NewHTTPError(http.StatusUnprocessableEntity, err.Error())
		}
		errs := ph.formatErrors(errMap)
		return httpError{http.StatusUnprocessableEntity, errs}
	}
	return nil
}

// Format a validator.ErrorMap into an array of error strings.
func (ph Handler) formatErrors(errorMap validator.ErrorMap) []string {
	var lines = make([]string, 0, len(errorMap))
	for fieldName, errorArray := range errorMap {
		for _, err := range errorArray {
			line := fmt.Sprintf("%s: %s", ph.reflector.MapFieldNameToParamName(fieldName), err.Error())
			lines = append(lines, line)
		}
	}
	return lines
}

// RegisterCustomType registers a custom type definition onto this handler.
func (ph Handler) RegisterCustomType(def CustomTypeDef) {
	ph.registerCustomType(def.expand())
}

func (ph Handler) registerCustomType(def customTypeDef) {
	ph.reflector.RegisterParser(def.Type, def.Parser)
	ph.binder.RegisterDefaulter(def.Type, def.Defaulter)
}

// Parser accepts a string value and returns a reflect.Value that can be used to set a field of the custom type,
// or an error if the value cannot be parsed.
// If usePtr is true, the parser should return a reflect.Value to a _pointer_ to the type.
// See apiparams package documentation, tests,
// or the built-in time.Time custom type defintion for examples
type Parser func(value string, usePtr bool) (reflect.Value, error)

// Defaulter accepts a string (the value of the "default" struct tag)
// and returns a string that can be parsed in Parser.
// This is often unnecessary-
// it's only really necessary when the default needs out-of-band information, like "now".
type Defaulter func(value string) string

// CustomTypeDef is a description of how to bind a custom type to API parameters.
type CustomTypeDef struct {
	Value     interface{}
	Parser    Parser
	Defaulter Defaulter
}

func (c CustomTypeDef) expand() customTypeDef {
	return customTypeDef{
		Type:      reflect.TypeOf(c.Value),
		Value:     c.Value,
		Parser:    c.Parser,
		Defaulter: c.Defaulter,
	}
}

// customTypeDef expands and caches the simpler interface of CustomTypeDef.
// Use CustomTypeDef#expand to convert.
// We do this, rather than calculate Type as needed,
// because every call to BindAndValidate needs to register custom type defs onto the new Handler.
type customTypeDef struct {
	Type      reflect.Type
	Value     interface{}
	Parser    Parser
	Defaulter Defaulter
}

var defaultCustomTypes = make([]customTypeDef, 0, 2)

// RegisterCustomType registers a custom type definition,
// so that other types can be used in API parameters.
// Using this module-level method makes these custom types available to all Handlers
// (all calls of apiparams.BindAndValidate).
func RegisterCustomType(def CustomTypeDef) {
	defaultCustomTypes = append(defaultCustomTypes, def.expand())
}

func init() {
	RegisterCustomType(CustomTypeDef{
		Value: time.Time{},
		Parser: func(value string, usePtr bool) (reflect.Value, error) {
			v, err := time.Parse(time.RFC3339, value)
			if usePtr {
				return reflect.ValueOf(&v), err
			}
			return reflect.ValueOf(v), err
		}})
}
