package validator

import (
	"errors"
	"github.com/lithictech/go-aperitif/kronos"
	"github.com/lithictech/go-aperitif/stringutil"
	"github.com/rgalanakis/validator"
	"net/url"
	"regexp"
	"strings"
	"time"
)

func newError(s string) validator.TextErr {
	return validator.TextErr{Err: errors.New(s)}
}

var (
	// ErrInvalidIntID is the error returned when a string is not a valid integer ID.
	ErrInvalidIntID = newError("not an integer string")
	// ErrInvalidURL is the error returned when a string cannot be parsed as a request URI.
	ErrInvalidURL = newError("not a valid url")
	// ErrInvalidUUID4 is the error returned when a string cannot be parsed as a UUID4.
	ErrInvalidUUID4 = newError("not a uuid4 string")
)

const optional = "opt"

// Split the param string on |,
// and return a type of (other args, if param ends in |opt, error in the case of empty args).
// Examples:
//
//	"a|b" -> (["a", "b"], false, nil)
//	"a|opt" -> (["a"], true, nil)
//	"|opt" -> ([], false, <error>)
func splitOptionalVal(param string) ([]string, bool, error) {
	params := strings.Split(param, "|")
	if len(params) == 0 {
		return nil, false, validator.ErrBadParameter
	}
	optional := params[len(params)-1] == optional
	if optional {
		params = params[:len(params)-1]
	}
	if len(params) == 0 {
		return nil, false, validator.ErrBadParameter
	}
	return params, optional, nil
}

// NOTE ON POINTER FIELDS
// go-validator seems to coaelesce a pointer field into a non-pointer field if it's not empty.
// So instead of having to do something like:
//
//     s, ok := v.(string)
//     if !ok {
//         sptr, ptrok := v.(*string)
//         if !ptrok {
//             return ErrUnsupported
//         } else if sptr == nil {
//             return nil
//         } else {
//             s = *sptr
//         }
//     }
//
// The final "else" is dereferencing the pointer into the string value,
// but we don't need to do that, since go-validator does it for us.
// Ie, that "else" code can never be hit. So we only worry about the former two cases,
// and end up with code like:
//
//     s, ok := v.(string)
//     if !ok {
//         if ptr, ok := v.(*string); ok && ptr == nil {
//             return nil
//         }
//         return ErrUnsupported
//     }
//

func validateCaseInsensitiveEnum(v interface{}, param string) error {
	return validateEnumImpl(v, param, strings.ToLower)
}

func validateCaseSensitiveEnum(v interface{}, param string) error {
	return validateEnumImpl(v, param, nil)
}

func validateEnumImpl(v interface{}, param string, mapper func(string) string) error {
	choices, optional, err := splitOptionalVal(param)
	if err != nil {
		return err
	}
	if mapper != nil {
		choices = stringutil.Map(choices, mapper)
	}

	if s, ok := v.(string); ok {
		if mapper != nil {
			s = mapper(s)
		}
		return validateEnumImplStr(s, choices, optional)
	}
	if ptr, ok := v.(*string); ok && ptr == nil {
		return nil
	}

	if ss, ok := v.([]string); ok {
		if optional {
			return validator.ErrBadParameter
		}
		if mapper != nil {
			ss = stringutil.Map(ss, mapper)
		}
		return validateEnumImplSlice(ss, choices)
	}

	if ptr, ok := v.(*[]string); ok && ptr == nil {
		return nil
	}

	return validator.ErrUnsupported
}

func validateEnumImplStr(s string, choices []string, optional bool) error {
	if s == "" {
		if optional {
			return nil
		}
		return newError("empty string")
	}
	for _, choice := range choices {
		if choice == s {
			return nil
		}
	}
	return newError("is not one of " + strings.Join(choices, "|"))
}

func validateEnumImplSlice(ss []string, choices []string) error {
	for _, s := range ss {
		if !stringutil.Contains(choices, s) {
			return newError("element not one of " + strings.Join(choices, "|"))
		}
	}
	return nil
}

func makeStringValidator(malformed error, validate func(string) bool) validator.ValidationFunc {
	return func(v interface{}, param string) error {
		s, ok := v.(string)
		if !ok {
			if ptr, ok := v.(*string); ok && ptr == nil {
				return nil
			}
			return validator.ErrUnsupported
		}
		if s == "" {
			if param == optional {
				return nil
			}
			return malformed
		}
		if !validate(s) {
			return malformed
		}
		return nil
	}
}

// Don't allow a leading 0 which is ambiguous (can indicate hex/octal value when parsing)
var intIDRegexp = regexp.MustCompile("^[1-9][0-9]*$")

var validateIntID = makeStringValidator(ErrInvalidIntID, func(s string) bool {
	if s == "0" {
		return true
	}
	return intIDRegexp.MatchString(s)
})

var uuid4Regexp = regexp.MustCompile("^[0-9a-fA-F-]{32}")

var validateUUID4 = makeStringValidator(ErrInvalidUUID4, uuid4Regexp.MatchString)

var validateURL = makeStringValidator(ErrInvalidURL, func(s string) bool {
	// using url.Parse is worthless, it treats almost anything as valid
	_, err := url.ParseRequestURI(s)
	return err == nil
})

func makeValidateCompareNow(getNow nowSource) validator.ValidationFunc {
	return func(v interface{}, param string) error {
		validating, ok := v.(time.Time)
		if !ok {
			if ptr, ok := v.(*time.Time); ok && ptr == nil {
				return nil
			}
			return validator.ErrUnsupported
		}
		params, optional, err := splitOptionalVal(param)
		if err != nil {
			return err
		}
		if len(params) < 1 {
			return validator.ErrBadParameter
		}

		var msg = ""
		c := kronos.Compare(validating, getNow())
		switch params[0] {
		case "gte":
			if c < 0 {
				msg = "before"
			}
		case "gt":
			if c <= 0 {
				msg = "before or at"
			}
		case "lte":
			if c > 0 {
				msg = "after"
			}
		case "lt":
			if c >= 0 {
				msg = "after or at"
			}
		default:
			return validator.ErrBadParameter
		}
		if msg == "" {
			return nil
		}
		if optional && validating.IsZero() {
			return nil
		}
		return newError(msg + " now")
	}
}
