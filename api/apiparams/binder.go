package apiparams

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
)

// binder handles the binding of a struct to all the defaults and parameters.
type binder struct {
	reflector                        reflector
	req                              *http.Request
	routeParamKeys, routeParamValues []string
	typeDefaulters                   map[reflect.Type]Defaulter
}

func newBinder(r reflector, req *http.Request, routeParamKeys, routeParamValues []string) binder {
	b := binder{
		r,
		req,
		routeParamKeys,
		routeParamValues,
		make(map[reflect.Type]Defaulter),
	}
	return b
}

func (b binder) RegisterDefaulter(t reflect.Type, d Defaulter) {
	b.typeDefaulters[t] = d
}

// Fill in the struct instance from defaults, the JSON body, query params, and path params.
func (b binder) BindFromAll() HTTPError {
	if err := b.setFromDefaults(b.reflector.Underlying()); err != nil {
		return err
	}
	if err := b.setFromHeaders(); err != nil {
		return err
	}
	if err := b.setFromJSONBody(); err != nil {
		return err
	}
	if err := b.setFromForm(); err != nil {
		return err
	}
	if err := b.setFromQueryParams(); err != nil {
		return err
	}
	if err := b.setFromPathParams(); err != nil {
		return err
	}
	return nil
}

// Marshal the body into JSON, bound to the parameter struct.
// Return an error if the content-type is not JSON,
// or any other error occurs (bad unmarshaling).
// Noop if there is no body.
func (b binder) setFromJSONBody() HTTPError {
	if b.req.ContentLength == 0 {
		return nil
	}
	ctype := b.req.Header.Get("Content-Type")
	switch {
	case strings.HasPrefix(ctype, "application/json"):
		body, err := b.requestBody()
		if err != nil {
			return NewHTTPError(http.StatusBadRequest, err.Error())
		}
		return b.decodeJSON(body)
	default:
		return NewHTTPError(http.StatusUnsupportedMediaType, "")
	}
}

func (b binder) decodeJSON(body io.Reader) HTTPError {
	if err := json.NewDecoder(body).Decode(b.reflector.Pointer()); err == nil {
		return nil
	} else if ute, ok := err.(*json.UnmarshalTypeError); ok {
		return NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Unmarshal type error: expected=%v, got=%v, offset=%v", ute.Type, ute.Value, ute.Offset))
	} else if se, ok := err.(*json.SyntaxError); ok {
		return NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Syntax error: offset=%v, error=%v", se.Offset, se.Error()))
	} else {
		return NewHTTPError(http.StatusBadRequest, err.Error())
	}
}

func (b binder) requestBody() (io.Reader, error) {
	if b.req.GetBody == nil {
		return b.req.Body, nil
	}
	return b.req.GetBody()
}

// Walk over the form body, if any, and apply values.
// This is the same as query params, as they're both url.Values.
func (b binder) setFromForm() HTTPError {
	if len(b.req.Form) == 0 {
		return nil
	}
	for k, values := range b.req.Form {
		for _, v := range values {
			if err := b.setField(k, v, ParamSourceForm); err != nil {
				return err
			}
		}
	}
	return nil
}

// Walk over all the fields of a struct,
// setting fields according to any "default" struct tags.
// This function is called recursively if the field of a struct
// is another struct.
// If setting a default fails, return a 500 error, as that is something
// that should never happen (as opposed to, say, a malformed value
// from the client, which is a 400 since it's expected).
func (b binder) setFromDefaults(st reflect.Value) HTTPError {
	underlyingType := st.Type()
	for i := 0; i < underlyingType.NumField(); i++ {
		fieldDef := underlyingType.Field(i)
		if fieldDef.Type.Kind() == reflect.Struct {
			field := st.FieldByName(fieldDef.Name)
			if err := b.setFromDefaults(field); err != nil {
				return err
			}
		}
		defaultValue := fieldDef.Tag.Get("default")
		if defaultValue == "" {
			continue
		}
		if defaulter := b.typeDefaulters[fieldDef.Type]; defaulter != nil {
			defaultValue = defaulter(defaultValue)
		}

		field := st.FieldByName(fieldDef.Name)
		if err := b.reflector.setField(fieldDef, field, defaultValue); err != nil {
			panic("Invalid default value, change the struct def: " + err.Error())
		}
	}
	return nil
}

// Set struct fields from headers.
func (b binder) setFromHeaders() HTTPError {
	for k, values := range b.req.Header {
		k = strings.ToLower(k)
		for _, v := range values {
			if err := b.setField(k, v, ParamSourceHeader); err != nil {
				return err
			}
		}
	}
	return nil
}

// Set struct fields from the URL query parameters.
func (b binder) setFromQueryParams() HTTPError {
	for k, values := range b.req.URL.Query() {
		for _, v := range values {
			if err := b.setField(k, v, ParamSourceQuery); err != nil {
				return err
			}
		}
	}
	return nil
}

// Set struct fields from route/path param values.
func (b binder) setFromPathParams() HTTPError {
	for i, name := range b.routeParamKeys {
		if err := b.setField(name, b.routeParamValues[i], ParamSourcePath); err != nil {
			return err
		}
	}
	return nil
}

// Look up the StructField mapped to paramName
// (iow, look up a field by the json name in its struct tag)
// and set it based on value.
// Return an HTTPError if the field
// cannot be set, usually because it's malformed.
// See reflector.setField for some more info about how fields are set.
func (b binder) setField(paramName, paramValue string, source ParamSource) HTTPError {
	fieldDef, fieldExistsForParam := b.reflector.ParamFieldFor(paramName)
	if !fieldExistsForParam || !fieldDef.CanSetFrom(source) {
		// It's an extra/unbound query or path param.
		// This is unavoidable ("?_=123456"), so no issue.
		return nil
	}
	field := b.reflector.FieldFor(fieldDef.StructField)
	if err := b.reflector.setField(fieldDef.StructField, field, paramValue); err != nil {
		return NewHTTPError(http.StatusBadRequest, err.Error())
	}
	return nil
}
