package apiparams

import (
	"bytes"
	"fmt"
	"reflect"
	"strconv"
)

var (
	typeOfStringSlice = reflect.TypeOf([]string{})
	typeOfIntSlice    = reflect.TypeOf([]int{})
)

// reflector holds as much of the reflection code as possible, because reflection is hard.
type reflector struct {
	pointerValue, underlyingValue reflect.Value
	paramFieldsByJsonName         map[string]paramField
	jsonNamesByFieldName          map[string]string
	typeParsers                   map[reflect.Type]Parser
}

func newReflector(paramsStructPtr interface{}) reflector {
	pointerValue := reflect.ValueOf(paramsStructPtr)
	r := reflector{
		pointerValue,
		pointerValue.Elem(),
		make(map[string]paramField),
		make(map[string]string),
		make(map[reflect.Type]Parser),
	}
	r.parseStructTags(r.underlyingValue.Type())
	return r
}

func (r reflector) RegisterParser(t reflect.Type, p Parser) {
	r.typeParsers[t] = p
}

// Underlying returns the reflect.Value for the actual struct
// (what the pointer points to).
func (r reflector) Underlying() reflect.Value {
	return r.underlyingValue
}

// Pointer returns the actual pointer to the struct (the thing passed into BindAndValudate).
func (r reflector) Pointer() interface{} {
	return r.pointerValue.Interface()
}

// ParamFieldFor returns the StructField for a parameter/json name.
// This is only valid for top-level parameter struct fields.
func (r reflector) ParamFieldFor(jsonName string) (paramField, bool) {
	val, found := r.paramFieldsByJsonName[jsonName]
	return val, found
}

// FieldFor returns the reflect.Value for the parameter struct instance
// for a StructField definition.
func (r reflector) FieldFor(fd reflect.StructField) reflect.Value {
	return r.underlyingValue.FieldByName(fd.Name)
}

// MapFieldNameToParamName convert a field name string ("Foo") or path ("Foo.Bar" or "Foo[0].Bar")
// to a parameter name string ("foo", "foo.bar", "foo[0].bar",
// whatever was set up in struct tags).
func (r reflector) MapFieldNameToParamName(fieldName string) string {
	fm := fieldMapper{
		r.jsonNamesByFieldName,
		bytes.NewBuffer(nil),
		make([]byte, 0),
	}
	return fm.Map(fieldName)
}

type fieldMapper struct {
	lookup map[string]string
	buffer *bytes.Buffer
	run    []byte
}

func (f *fieldMapper) Map(fieldName string) string {
	var isInIndexRun = false
	for _, b := range []byte(fieldName) {
		if b == '.' { // path separator, mapAndFlushRun what we have and start a new path
			f.mapAndFlushRun()
			f.buffer.WriteByte(b)
		} else if b == '[' { // mapAndFlushRun the name we have, then write until the closing brace
			f.mapAndFlushRun()
			isInIndexRun = true
			f.buffer.WriteByte(b)
		} else if b == ']' { // close the index run
			isInIndexRun = false
			f.buffer.WriteByte(b)
		} else if isInIndexRun { // write directly, no need to map this thing
			f.buffer.WriteByte(b)
		} else { // we're in a string name run, we need to map it
			f.run = append(f.run, b)
		}
	}
	f.mapAndFlushRun()
	return f.buffer.String()
}
func (f *fieldMapper) mapAndFlushRun() {
	if len(f.run) == 0 {
		return
	}
	mapped := f.lookup[string(f.run)]
	f.buffer.WriteString(mapped)
	f.run = make([]byte, 0)
}

// Parse the fields on the parameter struct type recursively,
// mapping the reflect.StructField to the name we should expect
// it to be called in parameters. In other words, this struct:
//
//	type Params struct {
//		Foo string `json:"foo"`
//	}
//
// would set a map of
// {"foo": <reflect.StructField>} for paramFieldsByJsonName and
// {"Foo": "foo"} for jsonNamesByFieldName.
// We use this to easily look up the field as we iterate through
// path and query params.
//
// For nested params, the mapping is still flat. For example, this struct:
//
//	type Params struct {
//		[]struct {
//			A string `json:"a"`
//		}
//		Nest struct {
//			B int `json:"b"`
//		} `json:"nest"`
//
// would set a map of
// {"a": <reflect.StructField>, "nest":<StructField>, "b":<StructField>} for paramFieldsByJsonName and
// {"A": "a", "Nest": "nest", "B": "b"} for jsonNamesByFieldName.
// The two use cases for these maps are:
//
//   - Mapping JSON names to field defs (for setting fields from path and query params).
//     Since path/query params are a flat list of key-value pairs, we don't need
//     deep parameters from the struct.
//   - Mapping validation field errors (like "Foo" or "Nest[0].B" to JSON names.
//     The only alternative is to map names back after the fact,
//     or write yet-another-validator that is consistent with the way we parse names
//     from struct tags.
//     See the MapFieldNameToParamName method doc for more details on how this works.
func (r reflector) parseStructTags(underlyingType reflect.Type) {
	for i := 0; i < underlyingType.NumField(); i++ {
		fieldDef := underlyingType.Field(i)
		if fieldDef.Anonymous {
			r.parseStructTags(fieldDef.Type)
		}
		paramField, ok := parseToParamField(fieldDef)
		if !ok {
			continue
		}
		r.paramFieldsByJsonName[paramField.Name] = paramField
		r.jsonNamesByFieldName[fieldDef.Name] = paramField.Name

		switch fieldDef.Type.Kind() {
		case reflect.Struct:
			r.parseStructTags(fieldDef.Type)
		case reflect.Slice:
			sliceElementType := fieldDef.Type.Elem()
			if sliceElementType.Kind() == reflect.Struct {
				r.parseStructTags(sliceElementType)
			}
		}
	}
}

// Set a struct field, parsing/coercing value into the right type.
// value can parse into a basic type (int, float, string, bool),
// a simple slice type, or a supported struct type like time.Time.
// Return an error if the parse/set fails.
// More details about parsing mechanics are in parseValue.
//
// This code will panic for programmer errors, like if the struct
// field can't be set (usually because a pointer wasn't passed in properly),
// or because a field is being set of a type that isn't supported.
// For the latter case, imagine a struct with:
//
//	D time.Time `json:"d"`
//	Foo MyFooType `json:"foo"`
//
// time.Time is a supported type, so would be fine, but MyFooType
// is not a supported type so this code would panic.
func (r reflector) setField(fieldDef reflect.StructField, field reflect.Value, value string) error {
	if !field.CanSet() {
		panic(fmt.Sprintf("cannot set field %s, some reflection/pointer programming stuff probably", fieldDef.Name))
	}
	v, err := r.parseValue(fieldDef.Type, field, value)
	if err != nil {
		return err
	}
	field.Set(v)
	return nil
}

// parseValue parses a string value into a reflect.Value that can be set via reflection.
//
//   - t is the reflect.Type of the field that the value will be parsed into,
//     such as a basic type like string or int, a slice type like []string or []int, or a struct type.
//   - field is the reflect.Value of the existing struct field-
//     this is only used for slice types, which need to append to the field.
//   - value is the string value to parse.
//
// This is verbose, if generally straightforward.
// If t is not a pointer type, the reflect.Value returned points to the new field value.
// However, if t is a pointer type, the reflect.Value returned points to a _pointer_ to the new field value.
// This introduces some verbosity, because we need this if statement for every type/kind.
//
// Finally, note also that this code does not have to work recursively/totally flexibly.
// apiparams only sets "simple" fields: those that can be expressed in a path,
// query param, or string default. Ie, we do not need to support slices of arbitrary structs!
// That is an exercise for bodies, using Go's json lib.
func (r reflector) parseValue(t reflect.Type, field reflect.Value, value string) (reflect.Value, error) {
	var fieldValueType = t
	var isPtr = false
	if fieldValueType.Kind() == reflect.Ptr {
		fieldValueType = t.Elem()
		isPtr = true
	}
	if p := r.typeParsers[fieldValueType]; p != nil {
		return p(value, isPtr)
	}

	fieldValueKind := fieldValueType.Kind()

	switch fieldValueKind {
	case reflect.Int:
		temp, err := strconv.ParseInt(value, 10, 64)
		v := int(temp)
		if isPtr {
			return reflect.ValueOf(&v), err
		}
		return reflect.ValueOf(v), err

	case reflect.Int64:
		temp, err := strconv.ParseInt(value, 10, 64)
		v := temp
		if isPtr {
			return reflect.ValueOf(&v), err
		}
		return reflect.ValueOf(temp), err

	case reflect.Int32:
		temp, err := strconv.ParseInt(value, 10, 32)
		v := int32(temp)
		if isPtr {
			return reflect.ValueOf(&v), err
		}
		return reflect.ValueOf(v), err

	case reflect.String:
		if isPtr {
			return reflect.ValueOf(&value), nil
		}
		return reflect.ValueOf(value), nil

	case reflect.Float64:
		v, err := strconv.ParseFloat(value, 64)
		if isPtr {
			return reflect.ValueOf(&v), err
		}
		return reflect.ValueOf(v), err

	case reflect.Float32:
		temp, err := strconv.ParseFloat(value, 32)
		v := float32(temp)
		if isPtr {
			return reflect.ValueOf(&v), err
		}
		return reflect.ValueOf(v), err

	case reflect.Bool:
		v, err := strconv.ParseBool(value)
		if isPtr {
			return reflect.ValueOf(&v), err
		}
		return reflect.ValueOf(v), err

	case reflect.Slice:
		var sliceVal = field
		if isPtr {
			sliceVal = field.Elem()
		}
		// If the original field was a pointer, it's possible sliceVal is nil.
		// We need to create a new slice if that's the case.
		// This provides a better API than forcing the caller to initialize a slice/ptr,
		// which would introduce a significant amount of bookkeeping for this common case.
		if field.IsNil() {
			sliceVal = reflect.MakeSlice(fieldValueType, 0, 1)
		}
		// Call this function recursively to parse the string value into the slice's underlying type.
		elementVal, err := r.parseValue(fieldValueType.Elem(), sliceVal, value)
		if err != nil {
			return elementVal, err
		}

		// This would fail if sliceVal is nil; see comment above about why we initialize it.
		newSliceVal := reflect.Append(sliceVal, elementVal)

		// Now we're back to the verbose "if ptr" duplication.
		switch fieldValueType {
		case typeOfStringSlice:
			if isPtr {
				i := newSliceVal.Interface().([]string)
				return reflect.ValueOf(&i), nil
			}
			return newSliceVal, nil
		case typeOfIntSlice:
			if isPtr {
				i := newSliceVal.Interface().([]int)
				return reflect.ValueOf(&i), nil
			}
			return newSliceVal, nil
		}
	}

	panicUnsupportedType(t)
	panic("unreachable")
}

func panicUnsupportedType(t reflect.Type) {
	panic(fmt.Sprintf(
		"parameter struct has parsed field with type %v, kind %v; "+
			"support must be added, or the type must change",
		t, t.Kind()))
}
