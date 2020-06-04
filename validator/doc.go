/*
Package validator provides a number of validations useful to building APIs.
It is built on https://github.com/go-validator/validator,
though it does not expose it. See that package for more info
and on building custom validators.

Available validators include:

	len
		For numeric numbers, len will simply make sure that the
		value is equal to the parameter given. For strings, it
		checks that the string length is exactly that number of
		characters. For slices,	arrays, and maps, validates the
		number of items. (Usage: len=10)

	max
		For numeric numbers, max will simply make sure that the
		value is lesser or equal to the parameter given. For strings,
		it checks that the string length is at most that number of
		characters. For slices,	arrays, and maps, validates the
		number of items. (Usage: max=10)

	min
		For numeric numbers, min will simply make sure that the value
		is greater or equal to the parameter given. For strings, it
		checks that the string length is at least that number of
		characters. For slices, arrays, and maps, validates the
		number of items. (Usage: min=10)

	nonzero
		This validates that the value is not zero. The appropriate
		zero value is given by the Go spec (e.g. for int it's 0, for
		string it's "", for pointers is nil, etc.) For structs, it
		will not check to see if the struct itself has all zero
		values, instead use a pointer or put nonzero on the struct's
		keys that you care about. (Usage: nonzero)

	regexp
		Only valid for string types, it will validator that the
		value matches the regular expression provided as parameter.
		(Usage: regexp=^a.*b$)

	intid
		For string types, validate that the string must be an integer
		0 or greater, and not begin with 0's which can lead to
		ambiguous base parsing.
		If "opt" is specified, an empty string is accepted.
		(Usage: intid intid=opt)

	uuid4
		For string types, validate that the string confirms to a (simple)
		UUID version 4 format, with or without dashes.
		If "opt" is specified, an empty string is accepted.
		(validation will only be done if a value is provided).
		(Usage: uuid4 uuid4=opt)

	url
		For string types, validate that the string is parseable as
		a request URI via net/url.ParseRequestURI.
		It assumes that the value is an absolute URI or an absolute path.
		The url is assumed not to have a #fragment suffix.
		If "opt" is specified, an empty string is accepted.
		(Usage: url url=opt)

	enum
		For string types, validate that the string is one of the specified choices.
		Choices should be pipe-delimited. Matching is case-insensitive.
		If "|opt" is the trailing argument, treat the value as optional
		(an empty string is valid).

		For string slices, validate that each member is one of the specified choices.
		"|opt" cannot be used for string slices, since it is ambiguous in two ways.
		First, an empty slice is valid because it does not contain any invalid elements;
		use min=1 to require at least one element.
		Second, an empty string is generally an invalid element value;
		if this is not desired, use an empty enum (enum=a||b).
		(Usage: enum=bird|shark|whale enum=bird|shark|whale|opt)

	cenum
		Same as enum validator, but comparison is case-sensitive.
		(Usage: cenum=bird|shark|whale cenum=bird|shark|whale|opt)

	comparenow
		For time.Time types, validate the time relative to
		the time unit the current moment is in.
		Specify the unit, and whether the field must be after now ("gt"),
		now or after ("gte"), now or before ("lte"),
		or before now ("lt"). Now is calculated in the local timezone
		(using time.Now()) and truncated according to the unit.
		Provide a trailing "|opt" if the value is optional
		(validation will only be done if a value is provided).
		(Usage: comparenow=hour|gte comparenow=day|lt|opt)

Optional validations

Most validators support a way to specify they are optional.
Usually that is something like providing "opt" as a value, like `intid=opt`,
or specifying "|opt" as a trailing value, like `enum=a|b|c|opt`.
See example usages for details.

Nil pointers are generally considered valid. See Pointers section for more details.

Pointers

If validator is validating a pointer field, it will generally validate the underlying type the same
as non-pointer fields. The only real difference is that a nil pointer will be considered valid,
because pointer fields generally specify a value is optional.

If a nil pointer isn't valid for a pointer field, you can use the "nonzero" validation.
For example, a nil pointer is acceptable here, even though there is no trailing "|opt" flag:

    type d struct {
        D *time.Time `json:"d" validate:"comparenow=lte|day"`
    }

However, a nil pointer is not acceptable here, because of the "nonzero" validation:

    type d struct {
        D *time.Time `json:"d" validate:"comparenow=lte|day,nonzero"`
    }

*/
package validator
