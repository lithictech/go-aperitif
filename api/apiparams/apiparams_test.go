package apiparams_test

import (
	"fmt"
	"github.com/labstack/echo/v4"
	"github.com/lithictech/go-aperitif/v2/api/apiparams"
	. "github.com/lithictech/go-aperitif/v2/api/echoapitest"
	. "github.com/lithictech/go-aperitif/v2/apitest"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/rgalanakis/golangal"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestApiParams(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "apiparams package Suite")
}

type EchoAdapter struct{}

func (EchoAdapter) Request(handlerArgs []interface{}) *http.Request {
	return handlerArgs[0].(echo.Context).Request()
}
func (EchoAdapter) RouteParamNames(handlerArgs []interface{}) []string {
	return handlerArgs[0].(echo.Context).ParamNames()
}
func (EchoAdapter) RouteParamValues(handlerArgs []interface{}) []string {
	return handlerArgs[0].(echo.Context).ParamValues()
}

type StdlibAdapter struct {
	ParamNames  []string
	ParamValues []string
}

func (a StdlibAdapter) Request(handlerArgs []interface{}) *http.Request {
	return handlerArgs[1].(*http.Request)
}
func (a StdlibAdapter) RouteParamNames([]interface{}) []string {
	return a.ParamNames
}
func (a StdlibAdapter) RouteParamValues([]interface{}) []string {
	return a.ParamValues
}

var _ = Describe("apiparams package", func() {

	var (
		e     *echo.Echo
		group *echo.Group
		ad    *EchoAdapter
	)

	BeforeEach(func() {
		e = echo.New()
		group = e.Group("")
		ad = &EchoAdapter{}
	})

	type emptyHandlerParams struct{}

	shouldFailHandler := func(paramsPtr interface{}) echo.HandlerFunc {
		return func(c echo.Context) error {
			if err := apiparams.BindAndValidate(ad, paramsPtr, c); err != nil {
				return echo.NewHTTPError(err.Code(), err.Error())
			}
			fmt.Println("Unreachable handler was reached...")
			panic("this code should not be reached")
		}
	}

	It("returns a 415 for requests with a body but non-JSON Content-Type", func() {
		group.POST("/foo", shouldFailHandler(&emptyHandlerParams{}))
		resp := Serve(e, NewRequest("POST", "/foo", []byte(`{}`), func(r *http.Request) {
			r.Header.Add("Content-Type", "application/xml")
		}))
		Expect(resp).To(HaveResponseCode(415))
	})

	Context("binds the parameter struct", func() {

		It("to query parameters", func() {
			type handlerParams struct {
				Set      string `json:"set"`
				ThisName string `json:","`
				Dash     string `json:"-,"`
				Ignore   string `json:"-"`
			}
			group.GET(
				"/foo",
				func(c echo.Context) error {
					hp := handlerParams{}
					Expect(apiparams.BindAndValidate(ad, &hp, c)).To(Succeed())
					Expect(hp.Set).To(Equal("1"))
					Expect(hp.ThisName).To(Equal("2"))
					Expect(hp.Dash).To(Equal("3"))
					Expect(hp.Ignore).To(Equal(""))
					return c.JSON(http.StatusOK, 1)
				},
			)
			resp := Serve(e, GetRequest("/foo?set=1&ThisName=2&-=3&Ignore=4"))
			Expect(resp).To(HaveResponseCode(200))
		})

		It("to array query parameters", func() {
			type handlerParams struct {
				Strings []string `json:"strings"`
				Ints    []int    `json:"ints"`
			}
			group.GET(
				"/foo",
				func(c echo.Context) error {
					hp := handlerParams{}
					Expect(apiparams.BindAndValidate(ad, &hp, c)).To(Succeed())
					Expect(hp.Strings).To(BeEquivalentTo([]string{"x", "y"}))
					Expect(hp.Ints).To(BeEquivalentTo([]int{1, 2}))
					return c.JSON(http.StatusOK, 1)
				},
			)
			resp := Serve(e, GetRequest("/foo?strings[]=x&strings[]=y&ints[]=1&ints[]=2"))
			Expect(resp).To(HaveResponseCode(200))
		})

		It("to multiple occurances of the same query parameter", func() {
			type handlerParams struct {
				Tags []string `json:"tag"`
			}
			group.GET(
				"/foo",
				func(c echo.Context) error {
					hp := handlerParams{}
					Expect(apiparams.BindAndValidate(ad, &hp, c)).To(Succeed())
					Expect(hp.Tags).To(Equal([]string{"c", "a", "b"}))
					return c.JSON(http.StatusOK, 1)
				},
			)
			resp := Serve(e, GetRequest("/foo?tag=c&tag=a&tag=b"))
			Expect(resp).To(HaveResponseCode(200))
		})

		It("to path parameters", func() {
			type handlerParams struct {
				Set      string `json:"set"`
				ThisName string `json:","`
			}
			group.GET(
				"/foo/:set/:ThisName",
				func(c echo.Context) error {
					hp := handlerParams{}
					Expect(apiparams.BindAndValidate(ad, &hp, c)).To(Succeed())
					Expect(hp.Set).To(Equal("abc"))
					Expect(hp.ThisName).To(Equal("xyz"))
					return c.JSON(http.StatusOK, 1)
				},
			)
			resp := Serve(e, GetRequest("/foo/abc/xyz"))
			Expect(resp).To(HaveResponseCode(200))
		})

		It("to the JSON form body", func() {
			type handlerParams struct {
				Set      int `json:"set"`
				ThisName int `json:","`
				Dash     int `json:"-,"`
			}
			group.POST(
				"/foo",
				func(c echo.Context) error {
					hp := handlerParams{}
					Expect(apiparams.BindAndValidate(ad, &hp, c)).To(Succeed())
					Expect(hp.Set).To(Equal(1))
					Expect(hp.ThisName).To(Equal(2))
					Expect(hp.Dash).To(Equal(3))
					return c.JSON(http.StatusOK, 1)
				},
			)
			resp := Serve(e, NewRequest("POST", "/foo", []byte(`{"set":1,"ThisName":2,"-":3}`), JsonReq()))
			Expect(resp).To(HaveResponseCode(200))
		})

		It("to nested JSON form body params", func() {
			type handlerParams struct {
				A struct {
					AA string `json:"aa"`
				} `json:"a"`
			}
			group.POST(
				"/foo",
				func(c echo.Context) error {
					hp := handlerParams{}
					Expect(apiparams.BindAndValidate(ad, &hp, c)).To(Succeed())
					Expect(hp.A.AA).To(Equal("bb"))
					return c.JSON(http.StatusOK, 1)
				},
			)
			resp := Serve(e, NewRequest("POST", "/foo", []byte(`{"a":{"aa":"bb"}}`), JsonReq()))
			Expect(resp).To(HaveResponseCode(200))
		})

		It("can bind to pointer fields", func() {
			type handlerParams struct {
				I1 *int    `json:"i1"`
				I2 *int    `json:"i2"`
				S1 *string `json:"s1"`
				S2 *string `json:"s2"`
			}
			group.POST(
				"/foo",
				func(c echo.Context) error {
					hp := handlerParams{}
					Expect(apiparams.BindAndValidate(ad, &hp, c)).To(Succeed())
					Expect(*hp.I1).To(Equal(10))
					Expect(hp.I2).To(BeNil())
					Expect(*hp.S1).To(Equal("abc"))
					Expect(hp.S2).To(BeNil())
					return c.JSON(http.StatusOK, 1)
				},
			)
			resp := Serve(e, NewRequest("POST", "/foo?i1=10&s1=abc", []byte(""), JsonReq()))
			Expect(resp).To(HaveResponseCode(200))
		})

		Describe("to all supported field types", func() {
			qparams := strings.Join([]string{
				"s=a",
				"sptr=a",
				"strslice=a",
				"strslice=b",
				"strsliceptr=a",
				"strsliceptr=b",
				"i=1",
				"i32=1",
				"i64=1",
				"intslice=1",
				"intslice=2",
				"f32=1",
				"f64=1",
				"b=true",
				"ut=2012-01",
				"t=2000-02-02T02:02:02.00001-08:00",
			}, "&")

			It("when they are not pointers", func() {
				type handlerParams struct {
					S        string    `json:"s"`
					StrSlice []string  `json:"strslice"`
					I        int       `json:"i"`
					I64      int64     `json:"i64"`
					I32      int32     `json:"i32"`
					IntSlice []int     `json:"intslice"`
					F64      float64   `json:"f64"`
					F32      float32   `json:"f32"`
					B        bool      `json:"b"`
					T        time.Time `json:"t"`
				}
				group.GET(
					"/foo",
					func(c echo.Context) error {
						hp := handlerParams{}
						Expect(apiparams.BindAndValidate(ad, &hp, c)).To(Succeed())
						Expect(hp.S).To(Equal("a"))
						Expect(hp.StrSlice).To(Equal([]string{"a", "b"}))
						Expect(hp.I).To(Equal(1))
						Expect(hp.I64).To(Equal(int64(1)))
						Expect(hp.I32).To(Equal(int32(1)))
						Expect(hp.IntSlice).To(Equal([]int{1, 2}))
						Expect(hp.F64).To(Equal(float64(1)))
						Expect(hp.F32).To(Equal(float32(1)))
						Expect(hp.B).To(BeTrue())
						Expect(hp.T.IsZero()).To(BeFalse())
						return c.JSON(http.StatusOK, 1)
					},
				)
				resp := Serve(e, GetRequest("/foo?"+qparams))
				Expect(resp).To(HaveResponseCode(200))
			})

			It("when they are pointers", func() {
				type handlerParams struct {
					S        *string    `json:"s"`
					StrSlice *[]string  `json:"strslice"`
					I        *int       `json:"i"`
					I64      *int64     `json:"i64"`
					I32      *int32     `json:"i32"`
					IntSlice *[]int     `json:"intslice"`
					F64      *float64   `json:"f64"`
					F32      *float32   `json:"f32"`
					B        *bool      `json:"b"`
					T        *time.Time `json:"t"`
				}
				group.GET(
					"/foo",
					func(c echo.Context) error {
						hp := handlerParams{}
						//ss := make([]string, 0)
						//hp.StrSlice = &ss
						//is := make([]int, 0)
						//hp.IntSlice = &is

						Expect(apiparams.BindAndValidate(ad, &hp, c)).To(Succeed())
						Expect(*hp.S).To(Equal("a"))
						Expect(*hp.StrSlice).To(Equal([]string{"a", "b"}))
						Expect(*hp.I).To(Equal(1))
						Expect(*hp.I64).To(Equal(int64(1)))
						Expect(*hp.I32).To(Equal(int32(1)))
						Expect(*hp.IntSlice).To(Equal([]int{1, 2}))
						Expect(*hp.F64).To(Equal(float64(1)))
						Expect(*hp.F32).To(Equal(float32(1)))
						Expect(*hp.B).To(BeTrue())
						t := *hp.T
						Expect(t.IsZero()).To(BeFalse())
						return c.JSON(http.StatusOK, 1)
					},
				)
				resp := Serve(e, GetRequest("/foo?"+qparams))
				Expect(resp).To(HaveResponseCode(200))
			})

		})

		It("parses fields based on their path/query/header struct tag, rather than json, if provided", func() {
			type handlerParams struct {
				Header string `header:"fieldh"`
				Path   string `path:"fieldp"`
				Query  string `query:"fieldq"`
				Body   string `json:"fieldb"`
			}
			group.POST(
				"/foo/:fieldp",
				func(c echo.Context) error {
					hp := handlerParams{}
					Expect(apiparams.BindAndValidate(ad, &hp, c)).To(Succeed())
					Expect(hp.Header).To(Equal("headerset"))
					Expect(hp.Path).To(Equal("pathset"))
					Expect(hp.Query).To(Equal("queryset"))
					Expect(hp.Body).To(Equal("bodyset"))
					return c.JSON(http.StatusOK, 1)
				},
			)
			resp := Serve(e,
				NewRequest(
					"POST",
					"/foo/pathset?fieldq=queryset",
					[]byte(`{"fieldb":"bodyset"}`),
					func(request *http.Request) {
						request.Header.Add("Content-Type", "application/json")
						request.Header.Set("fieldh", "headerset")
					}))
			Expect(resp).To(HaveResponseCode(200))
		})

		It("parses form fields from form or json struct tags", func() {
			type handlerParams struct {
				FormTag int    `form:"formTag"`
				JSONTag string `json:"jsonTag"`
			}
			group.POST(
				"/foo",
				func(c echo.Context) error {
					hp := handlerParams{}
					Expect(apiparams.BindAndValidate(ad, &hp, c)).To(Succeed())
					Expect(hp.FormTag).To(Equal(2))
					Expect(hp.JSONTag).To(Equal("xyz"))
					return c.JSON(http.StatusOK, 1)
				},
			)
			resp := Serve(e,
				NewRequest("POST",
					"/foo",
					[]byte(""),
					func(request *http.Request) {
						request.Form = url.Values{}
						request.Form.Set("formTag", "2")
						request.Form.Set("jsonTag", "xyz")
					}))
			Expect(resp).To(HaveResponseCode(200))
		})

		It("parses the form", func() {
			type handlerParams struct {
				FormTag int `form:"formTag"`
				JSONTag int `json:"jsonTag"`
			}
			group.POST(
				"/foo",
				func(c echo.Context) error {
					hp := handlerParams{}
					Expect(apiparams.BindAndValidate(ad, &hp, c)).To(Succeed())
					Expect(hp.FormTag).To(BeEquivalentTo(123))
					Expect(hp.JSONTag).To(BeEquivalentTo(456))
					return c.JSON(http.StatusOK, 1)
				},
			)
			resp := Serve(e,
				NewRequest("POST",
					"/foo",
					[]byte("formTag=123&jsonTag=456"),
					SetReqHeader("Content-Type", "application/x-www-form-urlencoded")))
			Expect(resp).To(HaveResponseCode(200))
		})
	})

	Describe("defaults", func() {

		It("can be set for query params", func() {
			type handlerParams struct {
				S string `json:"s" default:"hi"`
				I int    `json:"i" default:"5"`
			}
			group.GET(
				"/foo",
				func(c echo.Context) error {
					hp := handlerParams{}
					Expect(apiparams.BindAndValidate(ad, &hp, c)).To(Succeed())
					Expect(hp.S).To(Equal("hi"))
					Expect(hp.I).To(Equal(5))
					return c.JSON(http.StatusOK, 1)
				},
			)
			resp := Serve(e, GetRequest("/foo"))
			Expect(resp).To(HaveResponseCode(200))
		})

		It("can be set for form params", func() {
			type handlerParams struct {
				S string `json:"s" default:"hi"`
				I int    `json:"i" default:"5"`
			}
			group.POST(
				"/foo",
				func(c echo.Context) error {
					hp := handlerParams{}
					Expect(apiparams.BindAndValidate(ad, &hp, c)).To(Succeed())
					Expect(hp.S).To(Equal("hi"))
					Expect(hp.I).To(Equal(5))
					return c.JSON(http.StatusOK, 1)
				},
			)
			resp := Serve(e, NewRequest("POST", "/foo", []byte("{}"), JsonReq()))
			Expect(resp).To(HaveResponseCode(200))
		})

		It("defaults nested structs", func() {
			type handlerParams struct {
				A struct {
					AA string `default:"eggs"`
				}
			}
			group.POST(
				"/foo",
				func(c echo.Context) error {
					hp := handlerParams{}
					Expect(apiparams.BindAndValidate(ad, &hp, c)).To(Succeed())
					Expect(hp.A.AA).To(Equal("eggs"))
					return c.JSON(http.StatusOK, 1)
				},
			)
			resp := Serve(e, NewRequest("POST", "/foo", []byte("{}"), JsonReq()))
			Expect(resp).To(HaveResponseCode(200))
		})

		It("panics if an invalid default is specified", func() {
			type handlerParams struct {
				A int `default:"abc"`
			}
			group.GET(
				"/foo",
				shouldFailHandler(&handlerParams{}),
			)
			Expect(func() {
				Serve(e, GetRequest("/foo"))
			}).To(Panic())
		})

		It("panics if a pointer isn't passed", func() {
			type handlerParams struct {
				A int `default:"abc"`
			}
			group.GET(
				"/foo",
				shouldFailHandler(handlerParams{}),
			)
			Expect(func() {
				Serve(e, GetRequest("/foo"))
			}).To(Panic())
		})

		It("can default pointer fields", func() {
			type handlerParams struct {
				I *int    `default:"10"`
				S *string `default:"abc"`
			}
			group.POST(
				"/foo",
				func(c echo.Context) error {
					hp := handlerParams{}
					Expect(apiparams.BindAndValidate(ad, &hp, c)).To(Succeed())
					Expect(*hp.I).To(Equal(10))
					Expect(*hp.S).To(Equal("abc"))
					return c.JSON(http.StatusOK, 1)
				},
			)
			resp := Serve(e, NewRequest("POST", "/foo", []byte("{}"), JsonReq()))
			Expect(resp).To(HaveResponseCode(200))
		})
	})

	Describe("coerces", func() {

		It("basic types in query parameters", func() {
			type handlerParams struct {
				A string  `json:"a"`
				B int     `json:"b"`
				C float64 `json:"c"`
				D bool    `json:"d"`
				E string  `json:"e"`

				F int64 `json:"f"`
			}
			group.GET(
				"/foo",
				func(c echo.Context) error {
					hp := handlerParams{}
					Expect(apiparams.BindAndValidate(ad, &hp, c)).To(Succeed())
					Expect(hp.A).To(Equal("abc"))
					Expect(hp.B).To(Equal(2))
					Expect(hp.C).To(Equal(0.1))
					Expect(hp.D).To(Equal(true))
					Expect(hp.E).To(Equal(""))
					return c.JSON(http.StatusOK, 1)
				},
			)
			resp := Serve(e, GetRequest("/foo?a=abc&b=2&c=0.1&d=true&e=&f=1"))
			Expect(resp).To(HaveResponseCode(200))
		})

		It("times in query and body parameters", func() {
			type handlerParams struct {
				A time.Time `json:"a"`
				B time.Time `json:"b"`
			}
			t := time.Now()
			group.POST(
				"/foo",
				func(c echo.Context) error {
					hp := handlerParams{}
					Expect(apiparams.BindAndValidate(ad, &hp, c)).To(Succeed())
					Expect(hp.A).To(BeTemporally("~", t, time.Second))
					Expect(hp.B).To(BeTemporally("~", t, time.Second))
					return c.JSON(http.StatusOK, 1)
				},
			)
			path := fmt.Sprintf("/foo?a=%v", t.Format(time.RFC3339))
			body := fmt.Sprintf(`{"b":"%v"}`, t.Format(time.RFC3339))
			resp := Serve(e, NewRequest("POST", path, []byte(body), JsonReq()))
			Expect(resp).To(HaveResponseCode(200))
		})
	})

	It("ignores query and path parameters not found in the parameter struct", func() {
		group.GET(
			"/foo/:a",
			func(c echo.Context) error {
				return c.JSON(http.StatusOK, 1)
			},
		)
		resp := Serve(e, GetRequest("/foo/abc?b=xyz"))
		Expect(resp).To(HaveResponseCode(200))
	})

	It("returns a 400 if path or query parameters cannot be parsed into the proper type", func() {
		type handlerParams struct {
			A int `json:"a"`
		}
		group.GET(
			"/foo",
			shouldFailHandler(&handlerParams{}),
		)
		resp := Serve(e, GetRequest("/foo?a=abc"))
		Expect(resp).To(HaveResponseCode(400))
	})

	It("returns a 400 if form parameters are the wrong type", func() {
		type handlerParams struct {
			A int `json:"a"`
		}
		group.POST(
			"/foo",
			shouldFailHandler(&handlerParams{}),
		)
		resp := Serve(e, NewRequest("POST", "/foo", []byte(`{"a":"abc"}`), JsonReq()))
		Expect(resp).To(HaveResponseCode(400))
	})

	It("binds/walks embedded fields in the parameter struct", func() {
		type baseUserParams struct {
			ID    int    `path:"id" validate:"min=1"`
			Email string `json:"email" validate:"min=1"`
		}
		type userParams struct {
			baseUserParams
			Name string `json:"name" validate:"min=1"`
			Age  int    `json:"age" default:"10"`
		}
		group.POST(
			"/users/:id",
			func(c echo.Context) error {
				hp := userParams{}
				if err := apiparams.BindAndValidate(ad, &hp, c); err != nil {
					return echo.NewHTTPError(err.Code(), err.Error())
				}
				Expect(hp.ID).To(Equal(123))
				Expect(hp.Email).To(Equal("a@b.c"))
				Expect(hp.Name).To(Equal("jane"))
				Expect(hp.Age).To(Equal(10))
				return c.JSON(http.StatusOK, nil)
			},
		)
		resp := Serve(e, NewRequest("POST", "/users/123", []byte(`{"email":"a@b.c","name":"jane"}`), JsonReq()))
		Expect(resp).To(HaveResponseCode(200))
	})

	Describe("validation", func() {

		type handlerParams struct {
			S string `json:"s" validate:"len=2"`
		}

		It("422s for invalid path params", func() {
			group.GET(
				"/foo/:s",
				shouldFailHandler(&handlerParams{}),
			)
			resp := Serve(e, GetRequest("/foo/abcdefg"))
			Expect(resp).To(HaveResponseCode(422))
			Expect(resp.Body.String()).To(ContainSubstring("s: invalid length"))
		})

		It("422s for invalid query params", func() {
			group.GET(
				"/foo",
				shouldFailHandler(&handlerParams{}),
			)
			resp := Serve(e, GetRequest("/foo?s=abc"))
			Expect(resp).To(HaveResponseCode(422))
			Expect(resp.Body.String()).To(ContainSubstring("s: invalid length"))
		})

		It("422s for invalid form params", func() {
			group.POST(
				"/foo",
				shouldFailHandler(&handlerParams{}),
			)
			resp := Serve(e, NewRequest("POST", "/foo", []byte(`{"s":"a"}`), JsonReq()))
			Expect(resp).To(HaveResponseCode(422))
			Expect(resp.Body.String()).To(ContainSubstring("s: invalid length"))
		})

		It("validates pointer fields", func() {
			type handlerParams struct {
				I *int    `json:"i" validate:"len=2"`
				S *string `json:"s" validate:"len=2"`
			}
			group.GET(
				"/foo/:s/:i",
				shouldFailHandler(&handlerParams{}),
			)
			resp := Serve(e, GetRequest("/foo/abcdefg/1"))
			Expect(resp).To(HaveResponseCode(422))
			Expect(resp.Body.String()).To(ContainSubstring("s: invalid length"))
			Expect(resp.Body.String()).To(ContainSubstring("i: invalid length"))
		})

		It("maps field names to JSON names", func() {
			group.GET(
				"/foo/:s",
				shouldFailHandler(&handlerParams{}),
			)
			resp := Serve(e, GetRequest("/foo/abcdefg"))
			Expect(resp).To(HaveResponseCode(422))
			Expect(resp.Body.String()).To(HavePrefix(`{"message":"s: invalid length"}`))
		})

		It("maps nested field names to JSON names", func() {
			type handlerParams struct {
				Nested struct {
					S string `json:"s" validate:"len=2"`
				} `json:"nested"`
				Slice []struct {
					I int `json:"i" validate:"min=1"`
				} `json:"slice"`
			}
			group.POST(
				"/foo",
				shouldFailHandler(&handlerParams{}),
			)
			resp := Serve(e, NewRequest("POST", "/foo", []byte(`{"nested":{"s": "a"},"slice":[{"I":1},{"I":0}]}`), JsonReq()))
			Expect(resp).To(HaveResponseCode(422))
			Expect(resp.Body.String()).To(ContainSubstring(`nested.s: invalid length`))
			Expect(resp.Body.String()).To(ContainSubstring(`slice[1].i: less than min`))
		})
	})

	It("passes the full feature test from the example", func() {
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
	})

	Describe("StdlibAdapter", func() {
		It("can be used for success", func() {
			type noteParams struct {
				ID     int  `json:"id" validate:"min=1"`
				Pretty bool `json:"pretty"`
				Note   struct {
					Content string `json:"content" default:"hello" validate:"max=256"`
				} `json:"note"`
			}
			handler := func(resp http.ResponseWriter, req *http.Request) {
				idParam := strings.Split(req.URL.Path, "/")[2]
				ad := StdlibAdapter{[]string{"id"}, []string{idParam}}

				hp := noteParams{}
				Expect(apiparams.BindAndValidate(ad, &hp, resp, req)).To(Succeed())
				Expect(hp.ID).To(Equal(123))
				Expect(hp.Pretty).To(BeTrue())
				Expect(hp.Note.Content).To(Equal("hello"))
			}
			resp := httptest.NewRecorder()
			handler(resp, NewRequest("POST", "/notes/123?pretty=true", []byte("{}"), JsonReq()))
			Expect(resp).To(HaveResponseCode(200))
		})

		It("can be used for errors", func() {
			handler := func(resp http.ResponseWriter, req *http.Request) {
				if err := apiparams.BindAndValidate(StdlibAdapter{}, &emptyHandlerParams{}, resp, req); err != nil {
					resp.WriteHeader(err.Code())
					resp.Write([]byte(err.Error()))
					return
				}
				panic("should not reach here")
			}
			resp := httptest.NewRecorder()
			handler(resp, NewRequest("POST", "/foo", []byte(`123abc`), JsonReq()))
			Expect(resp).To(HaveResponseCode(400))
			Expect(resp.Body.String()).To(ContainSubstring("Unmarshal type error: expected"))
		})
	})

	Describe("custom types", func() {
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
			Defaulter: func(value string) string {
				i, err := strconv.Atoi(value)
				if err != nil {
					panic("Could not parse " + value)
				}
				return strconv.Itoa(i * 2)
			},
		})

		type IntOrString struct {
			Int int
			Str string
		}

		apiparams.RegisterCustomType(apiparams.CustomTypeDef{
			Value: IntOrString{},
			Parser: func(value string, usePtr bool) (reflect.Value, error) {
				v := IntOrString{}
				if i, err := strconv.Atoi(value); err == nil {
					v.Int = i
				} else {
					v.Str = value
				}
				if usePtr {
					return reflect.ValueOf(&v), nil
				}
				return reflect.ValueOf(v), nil
			},
		})

		type MyString string

		apiparams.RegisterCustomType(apiparams.CustomTypeDef{
			Value: MyString(""),
			Parser: func(v string, usePtr bool) (reflect.Value, error) {
				s := MyString(v)
				if usePtr {
					return reflect.ValueOf(&s), nil
				}
				return reflect.ValueOf(s), nil
			},
		})

		It("can get defaults", func() {
			type handlerParams struct {
				UnixTime          UnixTime    `default:"20"`
				IntOrStringInt    IntOrString `default:"20"`
				IntOrStringString IntOrString `default:"abc"`
				MyString          MyString    `default:"abc"`
			}

			group.POST(
				"/foo",
				func(c echo.Context) error {
					hp := handlerParams{}
					Expect(apiparams.BindAndValidate(ad, &hp, c)).To(Succeed())
					Expect(time.Time(hp.UnixTime)).To(Equal(time.Unix(40, 0)))
					Expect(hp.IntOrStringInt.Int).To(Equal(20))
					Expect(hp.IntOrStringString.Str).To(Equal("abc"))
					Expect(hp.MyString).To(Equal(MyString("abc")))
					return c.JSON(http.StatusOK, 1)
				},
			)
			resp := Serve(e, NewRequest("POST", "/foo", []byte("{}"), JsonReq()))
			Expect(resp).To(HaveResponseCode(200))
		})

		It("can be bound", func() {
			type handlerParams struct {
				UnixTime       UnixTime     `query:"unixTime"`
				UnixTimePtr    *UnixTime    `query:"unixTimePtr"`
				IntOrString    IntOrString  `query:"intOrStr"`
				IntOrStringPtr *IntOrString `query:"intOrStrPtr"`
				MyString       MyString     `query:"myStr"`
				MyStringPtr    *MyString    `query:"myStrPtr"`
			}

			group.POST(
				"/foo",
				func(c echo.Context) error {
					hp := handlerParams{}
					Expect(apiparams.BindAndValidate(ad, &hp, c)).To(Succeed())

					Expect(time.Time(hp.UnixTime)).To(Equal(time.Unix(50, 0)))
					Expect(hp.UnixTimePtr).To(Not(BeNil()))
					Expect(time.Time(*hp.UnixTimePtr)).To(Equal(time.Unix(60, 0)))

					Expect(hp.IntOrString.Str).To(Equal("hi"))
					Expect(hp.IntOrStringPtr).To(Not(BeNil()))
					Expect(hp.IntOrStringPtr.Int).To(Equal(20))

					Expect(hp.MyString).To(Equal(MyString("x")))
					Expect(*hp.MyStringPtr).To(Equal(MyString("y")))
					return c.JSON(http.StatusOK, 1)
				},
			)
			query := "unixTime=50&unixTimePtr=60&intOrStr=hi&intOrStrPtr=20&myStr=x&myStrPtr=y"
			resp := Serve(e, NewRequest("POST", "/foo?"+query, []byte("{}"), JsonReq()))
			Expect(resp).To(HaveResponseCode(200))
		})
	})

	Describe("using apiparams multiple times for the same request", func() {
		type handlerParams struct {
			Field string `json:"field"`
		}

		It("succeeds (older versions of Go would fail)", func() {
			type handlerParams2 struct {
				Field string `json:"field2"`
			}
			group.POST(
				"/foo",
				func(c echo.Context) error {
					hp1 := handlerParams{}
					hp2 := handlerParams2{}
					Expect(apiparams.BindAndValidate(ad, &hp1, c)).To(Succeed())
					Expect(apiparams.BindAndValidate(ad, &hp2, c)).To(Succeed())
					Expect(hp1.Field).To(Equal("1"))
					Expect(hp2.Field).To(Equal("2"))
					return c.NoContent(204)
				},
			)
			resp := Serve(e, NewRequest("POST", "/foo", []byte(`{"field":"1", "field2":"2"}`), JsonReq()))
			Expect(resp).To(HaveResponseCode(204))
		})
	})
})
