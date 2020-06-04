package validator_test

import (
	"errors"
	"github.com/lithictech/go-aperitif/validator"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"testing"
	"time"
)

func TestValidator(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Validator Suite")
}

var _ = Describe("Validator", func() {

	var registry *validator.Registry

	BeforeEach(func() {
		registry = validator.NewRegistry(time.Now)
	})

	expectInvalid := func(st interface{}, field, error string) {
		errs := registry.Validate(st)
		Expect(errs).To(HaveOccurred())

		errMap, ok := errs.(validator.ErrorMap)
		if !ok {
			panic("validator.Validate didn't return ErrorMap, what?")
		}

		errsForField := errMap[field]
		Expect(errsForField).To(HaveLen(1))

		Expect(errsForField[0].Error()).To(Equal(error))
	}

	expectValid := func(st interface{}) {
		errs := registry.Validate(st)
		Expect(errs).To(Not(HaveOccurred()))
	}

	It("uses go-validator under the hood", func() {
		type t struct {
			I int `validate:"min=5"`
		}
		expectValid(t{5})
		expectValid(t{6})
		expectInvalid(t{4}, "I", "less than min")
	})

	Describe("ErrorMap", func() {

		It("renders all errors in its Error()", func() {
			e := validator.ErrorMap{
				"Abc": validator.ErrorArray{errors.New("err1"), errors.New("err2")},
				"Xyz": validator.ErrorArray{errors.New("err3")},
			}
			possibilities := []string{
				"Abc: err1, err2 | Xyz: err3",
				"Xyz: err3 | Abc: err1, err2",
			}
			Expect(possibilities).To(ContainElement(e.Error()))
		})
	})

	Describe("ErrorArray", func() {

		It("renders all errors in its Error()", func() {
			e := validator.ErrorArray{errors.New("err1"), errors.New("err2")}
			Expect(e.Error()).To(Equal("err1, err2"))
		})

	})

	Describe("comparenow", func() {

		now := time.Date(2012, 11, 22, 6, 38, 12, 120, time.Local)
		zeroDay := time.Time{}
		earlierDay := now.Add(-50 * time.Hour)
		today := now
		laterDay := now.Add(50 * time.Hour)

		JustBeforeEach(func() {
			registry = validator.NewRegistry(func() time.Time { return now })
		})

		It("can specify gte now", func() {
			type d struct {
				D time.Time `json:"d" validate:"comparenow=gte"`
			}
			expectInvalid(d{zeroDay}, "D", "before now")
			expectInvalid(d{earlierDay}, "D", "before now")
			expectValid(d{today})
			expectValid(d{laterDay})
		})

		It("can specify gt today", func() {
			type d struct {
				D time.Time `json:"d" validate:"comparenow=gt"`
			}
			expectInvalid(d{zeroDay}, "D", "before or at now")
			expectInvalid(d{earlierDay}, "D", "before or at now")
			expectInvalid(d{today}, "D", "before or at now")
			expectValid(d{laterDay})
		})

		It("can specify lte today", func() {
			type d struct {
				D time.Time `json:"d" validate:"comparenow=lte"`
			}
			expectValid(d{zeroDay})
			expectValid(d{earlierDay})
			expectValid(d{today})
			expectInvalid(d{laterDay}, "D", "after now")
		})

		It("can specify lt today", func() {
			type d struct {
				D time.Time `json:"d" validate:"comparenow=lt"`
			}
			expectValid(d{zeroDay})
			expectValid(d{earlierDay})
			expectInvalid(d{today}, "D", "after or at now")
			expectInvalid(d{laterDay}, "D", "after or at now")
		})

		It("can be optional", func() {
			type d struct {
				D time.Time `json:"d" validate:"comparenow=gt|opt"`
			}
			expectValid(d{zeroDay})
			expectInvalid(d{earlierDay}, "D", "before or at now")
			expectInvalid(d{today}, "D", "before or at now")
			expectValid(d{laterDay})
		})

		It("can validate pointer fields", func() {
			type d struct {
				D *time.Time `json:"d" validate:"comparenow=lte"`
			}
			expectValid(d{nil})
			expectValid(d{&today})
			expectInvalid(d{&laterDay}, "D", "after now")
		})
	})

	Describe("intid", func() {
		It("requires an integer-like string (0 or greater)", func() {
			type s struct {
				ID string `json:"id" validate:"intid "`
			}
			expectInvalid(s{"1.1"}, "ID", "not an integer string")
			expectInvalid(s{"-1"}, "ID", "not an integer string")
			expectInvalid(s{"1a"}, "ID", "not an integer string")
			expectInvalid(s{""}, "ID", "not an integer string")
			expectValid(s{"1"})
			expectValid(s{"0"})
		})

		It("can specify it is optional (empty string is valid)", func() {
			type s struct {
				ID string `json:"id" validate:"intid=opt"`
			}
			expectInvalid(s{"1.1"}, "ID", "not an integer string")
			expectInvalid(s{"-1"}, "ID", "not an integer string")
			expectInvalid(s{"1a"}, "ID", "not an integer string")
			expectValid(s{""})
			expectValid(s{"1"})
			expectValid(s{"0"})
		})

		It("can validate pointer fields", func() {
			type s struct {
				ID *string `json:"id" validate:"intid"`
			}
			expectValid(s{nil})
			valid := "123"
			expectValid(s{&valid})
			invalid := "abc"
			expectInvalid(s{&invalid}, "ID", "not an integer string")
		})

	})

	Describe("uuid", func() {
		It("requires a uuid4 formatted string", func() {
			type d struct {
				V string `json:"v" validate:"uuid4"`
			}
			expectValid(d{"feff425d-b10e-4b50-93c3-4a0124481da4"})
			expectValid(d{"feff425db10e4b5093c34a0124481da4"})
			expectInvalid(d{"zeff425db10e4b5093c34a0124481da4"}, "V", "not a uuid4 string")
			expectInvalid(d{"feff"}, "V", "not a uuid4 string")
			expectInvalid(d{""}, "V", "not a uuid4 string")
		})

		It("can be optional", func() {
			type d struct {
				V string `json:"v" validate:"uuid4=opt"`
			}
			expectValid(d{"feff425d-b10e-4b50-93c3-4a0124481da4"})
			expectValid(d{"feff425db10e4b5093c34a0124481da4"})
			expectInvalid(d{"zeff425db10e4b5093c34a0124481da4"}, "V", "not a uuid4 string")
			expectInvalid(d{"feff"}, "V", "not a uuid4 string")
			expectValid(d{""})
		})

		It("can validate pointer fields", func() {
			type d struct {
				V *string `json:"v" validate:"uuid4"`
			}
			expectValid(d{nil})
			valid := "feff425db10e4b5093c34a0124481da4"
			expectValid(d{&valid})
			invalid := "abc"
			expectInvalid(d{&invalid}, "V", "not a uuid4 string")
		})
	})

	Describe("enum", func() {
		It("requires a case-insensitive choice from a list of strings", func() {
			type d struct {
				V string `json:"v" validate:"enum=a|opt|c"`
			}
			expectValid(d{"A"})
			expectValid(d{"opt"})
			expectValid(d{"c"})
			expectInvalid(d{"d"}, "V", "is not one of a|opt|c")
			expectInvalid(d{"feff"}, "V", "is not one of a|opt|c")
			expectInvalid(d{""}, "V", "empty string")
		})

		It("can validate a string slice", func() {
			type d struct {
				V []string `json:"v" validate:"enum=a|b"`
			}
			expectValid(d{[]string{}})
			expectValid(d{[]string{"A", "b"}})
			expectInvalid(d{[]string{"A", "b", "c"}}, "V", "element not one of a|b")
			expectInvalid(d{[]string{""}}, "V", "element not one of a|b")
		})

		It("is unsupported if optional is used for a string slice field", func() {
			type d struct {
				V []string `json:"v" validate:"enum=a|opt"`
			}
			expectInvalid(d{[]string{}}, "V", "bad parameter")
		})

		It("can be optional", func() {
			type d struct {
				V string `json:"v" validate:"enum=a|b|c|opt"`
			}
			expectValid(d{"A"})
			expectValid(d{"b"})
			expectValid(d{"c"})
			expectInvalid(d{"d"}, "V", "is not one of a|b|c")
			expectInvalid(d{"feff"}, "V", "is not one of a|b|c")
			expectInvalid(d{"opt"}, "V", "is not one of a|b|c")
			expectValid(d{""})
		})

		It("can validate pointer fields", func() {
			type d struct {
				V *string `json:"v" validate:"enum=a|b|c"`
			}
			expectValid(d{nil})
			valid := "b"
			expectValid(d{&valid})
			invalid := "d"
			expectInvalid(d{&invalid}, "V", "is not one of a|b|c")
		})

		It("can validate pointer slice fields", func() {
			type d struct {
				V *[]string `json:"v" validate:"enum=a|b|c"`
			}
			expectValid(d{nil})
			valid := []string{"b"}
			expectValid(d{&valid})
			invalid := []string{"d"}
			expectInvalid(d{&invalid}, "V", "element not one of a|b|c")
		})
	})

	Describe("cenum", func() {
		It("requires a case-sensitive choice from a list of strings", func() {
			type d struct {
				V string `json:"v" validate:"cenum=A|opt|c"`
			}
			expectInvalid(d{"a"}, "V", "is not one of A|opt|c")
			expectValid(d{"opt"})
			expectValid(d{"c"})
			expectInvalid(d{"d"}, "V", "is not one of A|opt|c")
			expectInvalid(d{"feff"}, "V", "is not one of A|opt|c")
			expectInvalid(d{""}, "V", "empty string")
		})

		It("can validate a string slice", func() {
			type d struct {
				V []string `json:"v" validate:"cenum=a|b"`
			}
			expectValid(d{[]string{"a"}})
			expectInvalid(d{[]string{"A", "b"}}, "V", "element not one of a|b")
		})

		It("is unsupported if optional is used for a string slice field", func() {
			type d struct {
				V []string `json:"v" validate:"cenum=a|opt"`
			}
			expectInvalid(d{[]string{}}, "V", "bad parameter")
		})

		It("can be optional", func() {
			type d struct {
				V string `json:"v" validate:"cenum=A|b|c|opt"`
			}
			expectValid(d{"A"})
			expectValid(d{"b"})
			expectValid(d{"c"})
			expectInvalid(d{"d"}, "V", "is not one of A|b|c")
			expectInvalid(d{"feff"}, "V", "is not one of A|b|c")
			expectInvalid(d{"opt"}, "V", "is not one of A|b|c")
			expectValid(d{""})
		})

		It("can validate pointer fields", func() {
			type d struct {
				V *string `json:"v" validate:"cenum=a|b|c"`
			}
			expectValid(d{nil})
			valid := "b"
			expectValid(d{&valid})
			invalid := "d"
			expectInvalid(d{&invalid}, "V", "is not one of a|b|c")
		})

		It("can validate pointerslice  fields", func() {
			type d struct {
				V *[]string `json:"v" validate:"cenum=a|b|c"`
			}
			expectValid(d{nil})
			valid := []string{"b"}
			expectValid(d{&valid})
			invalid := []string{"d"}
			expectInvalid(d{&invalid}, "V", "element not one of a|b|c")
		})
	})

	Describe("url", func() {
		It("requires a parse-able URL", func() {
			type s struct {
				URL string `json:"url" validate:"url"`
			}
			expectInvalid(s{"foo.com"}, "URL", "not a valid url")
			expectInvalid(s{""}, "URL", "not a valid url")
			expectValid(s{"http://foo.com"})
			expectValid(s{"/go/lang"})
		})

		It("can specify it is optional (empty string is valid)", func() {
			type s struct {
				URL string `json:"url" validate:"url=opt"`
			}
			expectInvalid(s{"foo.com"}, "URL", "not a valid url")
			expectValid(s{""})
			expectValid(s{"http://foo.com"})
		})

		It("can validate pointer fields", func() {
			type s struct {
				URL *string `json:"url" validate:"url"`
			}
			expectValid(s{nil})
			invalid := "foo.com"
			expectInvalid(s{&invalid}, "URL", "not a valid url")
			empty := ""
			expectInvalid(s{&empty}, "URL", "not a valid url")
			valid := "http://foo.com"
			expectValid(s{&valid})
		})
	})
})
