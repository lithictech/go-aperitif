package apiparams

import (
	"reflect"
	"strings"
)

// ParamSource is a struct tag name that can define where a field is set by.
// For example, a field of:
//     Wibble string `path:"wibble"`
// would be said to have a Source of "path".
// In general, fields can only be set from their parameter source,
// so that the Wibble field can only be set from the path and not a query parameter.
// The exception would be the JSON param source, which can be set by any param sources.
//
// Possible param sources are json, path, query, and header.
type ParamSource string

const (
	ParamSourceJSON   = ParamSource("json")
	ParamSourceForm   = ParamSource("form")
	ParamSourcePath   = ParamSource("path")
	ParamSourceQuery  = ParamSource("query")
	ParamSourceHeader = ParamSource("header")
)

var AllParamSources = []ParamSource{
	ParamSourceJSON,
	ParamSourceForm,
	ParamSourcePath,
	ParamSourceQuery,
	ParamSourceHeader,
}

// paramField is a container for a StructField that has some sort of parameter exposure,
// whether via query, path, header, or json/body parameters.
// For a struct field of:
//
//     Field string `header:"x-my-field"`
//
// - Name is "x-my-field"
// - Source is "header"
// - StructField is the reflect.StructField for Field
type paramField struct {
	Name        string
	Source      ParamSource
	StructField reflect.StructField
}

// parseToParamField parses the struct tags from a StructField into a paramField
// that indicates how the parameter is supposed to be set: its Source (header, query, path, json)
// the Name used to set the parameter, and a reference back to the parsed StructField.
// This means parsing the struct field:
//     Field string `query:"pretty"`
// would return a paramField with a Source of "query" and Name of "pretty".
// This also resolves json field naming rules (like `query:"-"` indicating not to set the field).
// If no paramField can be parsed (it has no tags, or the tags indicate not to export the field),
// found is false.
func parseToParamField(fieldDef reflect.StructField) (pf paramField, found bool) {
	pf.StructField = fieldDef
	for _, src := range AllParamSources {
		tag, ok := fieldDef.Tag.Lookup(string(src))
		if !ok || tag == "-" {
			continue
		}
		parts := strings.Split(tag, ",")
		if len(parts) > 1 && parts[0] == "" {
			pf.Name = fieldDef.Name
		} else {
			pf.Name = parts[0]
		}
		pf.Source = src
		found = true
		break
	}
	return pf, found
}

// CanSetFrom returns true if a parameter from source ps can be set by this paramField.
// A parameter from ps can be set by the receiver's parameter is the sources are the same
// ([Field string `path:"foo"`] and ps is "header"), or the paramField's source is "json",
// which is used as a super-source (anything can bind to it).
func (p paramField) CanSetFrom(ps ParamSource) bool {
	return p.Source == ParamSourceJSON || p.Source == ps
}
