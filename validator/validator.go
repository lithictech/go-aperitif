package validator

import (
	"fmt"
	"strings"
	"time"

	"github.com/rgalanakis/validator"
)

// ErrorMap is a map which contains all errors from validating a struct.
type ErrorMap map[string]ErrorArray

// ErrorMap implements the Error interface so we can check error against nil.
// The returned error is if existent the first error which was added to the map.
func (err ErrorMap) Error() string {
	lines := make([]string, 0, len(err))
	for k, errs := range err {
		line := fmt.Sprintf("%s: %s", k, errs.Error())
		lines = append(lines, line)
	}
	return strings.Join(lines, " | ")
}

// ErrorArray is a slice of errors returned by the Validate function.
type ErrorArray []error

// ErrorArray implements the Error interface and returns the first error as
// string if existent.
func (err ErrorArray) Error() string {
	errs := make([]string, 0, len(err))
	for _, e := range err {
		errs = append(errs, e.Error())
	}
	return strings.Join(errs, ", ")
}

// Registry is a registry of all available validation functions.
// It must be initialized before using.
// In general, clients should use the global instance available through
// the Validate function; instances are generally only used for testing.
type Registry struct {
	validator *validator.Validator
}

type nowSource func() time.Time

// Init initializes a registry (registers all validators).
func (r *Registry) Init(getNow nowSource) {
	v := validator.NewValidator()
	v.SetValidationFunc("intid", validateIntID)
	v.SetValidationFunc("uuid4", validateUUID4)
	v.SetValidationFunc("url", validateURL)
	v.SetValidationFunc("enum", validateCaseInsensitiveEnum)
	v.SetValidationFunc("cenum", validateCaseSensitiveEnum)
	v.SetValidationFunc("comparenow", makeValidateCompareNow(getNow))
	r.validator = v
}

// Validate validates using all registered validators.
func (r *Registry) Validate(v interface{}) error {
	err := r.validator.Validate(v)
	return coerceValidatorPkgError(err)
}

// NewRegistry returns a new Registry using the given nowSource.
func NewRegistry(getNow nowSource) *Registry {
	r := new(Registry)
	r.Init(getNow)
	return r
}

var globalRegistry *Registry

func init() {
	globalRegistry = NewRegistry(time.Now)
}

// Validate validates the fields of a struct based
// on 'validator' tags and returns errors found indexed
// by the field name.
func Validate(v interface{}) error {
	return globalRegistry.Validate(v)
}

// coerceValidatorPkgError coerces a go-validator/validator error type
// (validator.ErrorArray, validator.ErrorMap, or some unknown type)
// into a common-go/validator error type (ErrorArray, ErrorMap).
// This is done so we are not exposing go-validator types directly.
func coerceValidatorPkgError(err error) error {
	switch realErr := err.(type) {
	case validator.ErrorMap:
		return coerceValidatorPkgErrorMap(realErr)
	case validator.ErrorArray:
		return coerceValidatorPkgErrorArray(realErr)
	default:
		return realErr
	}
}

func coerceValidatorPkgErrorMap(err validator.ErrorMap) ErrorMap {
	result := make(ErrorMap, len(err))
	for k, v := range err {
		result[k] = coerceValidatorPkgErrorArray(v)
	}
	return result
}

func coerceValidatorPkgErrorArray(err validator.ErrorArray) ErrorArray {
	result := make(ErrorArray, 0, len(err))
	for _, e := range err {
		result = append(result, e)
	}
	return result
}
