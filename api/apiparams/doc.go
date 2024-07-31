/*
Package apiparams provides a framework-agnostic method
for declarative parameter declaration and validation
that should be used at the start of route handlers.
It provides support for binding a struct to route,
query, and JSON body parameters.

For example, consider the following test:

	type noteParams struct {
		ID     int  `json:"id" validate:"min=1"`
		Pretty bool `json:"pretty"`
		Note   struct {
			Content string `json:"content" default:"hello" validate:"max=256"`
		} `json:"note"`
		At time.Time `json:"at" validate:"comparenow=gt"`
	}
	group.POST(
		"/notes/:id",
		func(c echo.Context) error {
			hp := noteParams{}
			if err := apiparams.BindAndValidate(ad, &hp, c); err != nil {
				return echo.NewHTTPError(err.Code(), err.Error())
			}
			Expect(hp.ID).To(Equal(123))
			Expect(hp.Pretty).To(BeTrue())
			Expect(hp.Note.Content).To(Equal("hello"))
			Expect(hp.At.Year()).To(Equal(2050))
			return c.JSON(http.StatusOK, nil)
		},
	)
	resp := Serve(e, NewRequest("POST", "/notes/123?pretty=true", []byte(`{"at":"2050-06-04T05:48:36Z"}`), JsonReq()))
	Expect(resp).To(HaveResponseCode(200))

Note all the benefits:

  - Data is pulled from path parameters, query parameters, any JSON body,
    and defaults defined in struct tags. The variable names used for values
    is specified via the appropriate struct tag.
    See ParamSource for more details, but possible tags are "path", "query", "header", "form", and "json".
    The "json" tag will bind from any source, not just a JSON request body.
    This makes it clear at the endpoint and model definitions where data comes from and
    how an endpoint is supposed to be called.
  - Path and query param coercion is done from the basic JSON types,
    depending on the struct field type (int/float, string, bool).
  - Validation is done using the validator package.
    Custom validators can be registered as we need to express more
    sophisticated validations.

# Validations

See validator for a list of available validators and usage examples.

# Adapters

The only non-obvious prerequisite to using apiparams.BindAndValidate is
to create a apiparams.Adapter for your HTTP framework of choice.

The adapters are necessary so that apiparams has a consistent interface
into how to get an *http.Request, and the names and values of path parameters.
These should be logic-free types that are usually stateless,
so very lightweight and easy to copy into repos as needed.

Here's an example of an Echo (labstack/echo) adapter:

	type EchoAdapter struct {}
	func (EchoAdapter) Request(handlerArgs []interface{}) *http.Request {
		return handlerArgs[0].(echo.Context).Request()
	}
	func (EchoAdapter) RouteParamNames(handlerArgs []interface{}) []string {
		return handlerArgs[0].(echo.Context).ParamNames()
	}
	func (EchoAdapter) RouteParamValues(handlerArgs []interface{}) []string {
		return handlerArgs[0].(echo.Context).ParamValues()
	}

The signature for echo.HandlerFunc is func(echo.Context) error,
so we know that handlerArgs[0] is always going to be an echo.Context.
We can use that context to look up the http.Request,
and path param names and values.

Here's an example of a standard library (net/http) adapter:

	type StdlibAdapter struct {
		ParamNames []string
		ParamValues []string
	}
	func (a StdlibAdapter) Request(handlerArgs []interface{}) *http.Request {
		return handlerArgs[1].(*http.Request)
	}
	func (a StdlibAdapter) RouteParamNames(handlerArgs []interface{}) []string {
		return a.ParamNames
	}
	func (a StdlibAdapter) RouteParamValues(handlerArgs []interface{}) []string {
		return a.ParamValues
	}

The signature for an http.HandlerFunc is func(http.ResponseWriter, *http.Request),
so we know that handlerArgs[1] is an *http.Request.
Note that the standard library has no concept of path/route parameters,
so RouteParamNames and RouteParamValues return some adapter state.

Finally, here is an example of a chi (chi-go/chi) adapter:

	type ChiAdapter struct {}
	func (ChiAdapter) Request(handlerArgs []interface{}) *http.Request {
		return handlerArgs[1].(*http.Request)
	}
	func (c ChiAdapter) RouteParamNames(handlerArgs []interface{}) []string {
		if rctx := RouteContext(c.Request(handlerArgs).Context()); rctx != nil {
			return rctx.URLParams.Keys
		}
		return make([]string, 0)
	}
	func (c ChiAdapter) RouteParamValues(handlerArgs []interface{}) []string {
		if rctx := RouteContext(c.Request(handlerArgs).Context()); rctx != nil {
			return rctx.URLParams.Values
		}
		return make([]string, 0)
	}

chi handlers are the same as http.HandlerFunc, but store state in the http.Request#Context.
chi pulls data out of there to figure out a URL Param, like when chi.URLParam is used.

Note again that in general only one of these need to be defined and once per-project
(or you can put them into a library, whatever floats your boat).

# Errors

apiparams.BindAndValidate returns a apiparams.HTTPError. Nil result means no error.
The HTTPError can be one of various error codes (415, 422, 400, 500)
for reasons like an incorrect Content-Type (a body with any type but "application/json"),
unparseable value (like "abc" for an integer field),
parseable-but-invalid value (like a too-high number), or malformed JSON.

Callers should wrap the result in the appropriate error for their framework,
or can write the Code and Message to the HTTP response.

# Custom Types

Custom types can be used in an API by providing a CustomTypeDef and passing it to RegisterCustomType.
A CustomTypeDef consists of a _defaulter_ and a _parser_.

Note that a custom type is automatically registered for time.Time,
as shown in this documentation.

The _parser_ takes a string and returns a reflect.Value that can be used to set a field
of the custom type.

For example, perhaps we have an API that passes in an integer for a Unix timestamp,
but we want to work with it as a time.Time.
We can use the following type and parser:

	type UnixTime time.Time

	apiparams.RegisterCustomType(apiparams.CustomTypeDef{
		Value: UnixTime{},
		Parser: func(value string, usePtr bool) (reflect.Value, error) {
			i, err := strconv.Atoi(value)
			if err != nil {
				return reflect.Value{}, err
			}
			v := UnixTime(time.Unix(int64(i), 0))
			if usePtr {
				return reflect.ValueOf(&v), nil
			}
			return reflect.ValueOf(v), nil
		},
	})

Note that if usePtr is true, the reflect.Value must be a value for a _pointer_,
not the raw value.

We can also provide a _defaulter_ as part of our custom type.
The defaulter is a function that takes a string, and returns a new string used for a default.
This allows, for example, defaulting a UnixTime to the current time, based on a `default`
tag value of "now":

	type MyParams struct {
		T UnixTime `default:"now"`
	}

	apiparams.RegisterCustomType(apiparams.CustomTypeDef{
		Value: UnixTime{},
		Parser: parser,
		Defaulter: func(value string) string {
			if value == "now" {
				return strconv.Itoa(time.Now().Unix())
			}
			panic("Invalid default value " + value)
		},
	})

Note also the defaulting behavior for a Time demonstrated in previous sections.

The custom defaulter methods may want to panic if the value is invalid-
the value is read from the struct tags, so is known at compile time and will never change.
Thus it shouldn't be considered an input error, but a programming error, like invalid syntax-
however, it can also return an empty string, which will hit the Parser which can treat it as a normal error.
*/
package apiparams
