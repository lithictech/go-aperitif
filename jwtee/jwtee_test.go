package jwtee_test

import (
	"github.com/dgrijalva/jwt-go"
	"github.com/lithictech/go-aperitif/jwtee"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/rgalanakis/golangal"
	"testing"
	"time"
)

func TestJwtee(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "jwtee package Suite")
}

var _ = Describe("jwtee", func() {
	secret := "xyz"
	aud := "hi"
	iss := "there"
	alg := "HS256"

	validInput := func() jwtee.Input {
		return jwtee.Input{
			Secret: secret,
			Aud:    aud,
			Iss:    iss,
			Alg:    alg,
		}
	}

	newJwtee := func() jwtee.Jwtee {
		j, err := jwtee.New(validInput())
		Expect(err).ToNot(HaveOccurred())
		return j
	}

	It("requires secret, aud, iss, and alg", func() {
		var err error
		var jw jwtee.Jwtee
		jw, err = jwtee.New(validInput())
		Expect(err).ToNot(HaveOccurred())
		Expect(jw).To(And(
			MatchField("Aud", aud),
			MatchField("Iss", iss),
			MatchField("Alg", jwt.SigningMethodHS256),
		))
		_, err = jwtee.New(jwtee.Input{
			Secret: "",
			Aud:    aud,
			Iss:    iss,
			Alg:    alg,
		})
		Expect(err).To(MatchError(ContainSubstring("secret is required")))
		_, err = jwtee.New(jwtee.Input{
			Secret: secret,
			Aud:    "",
			Iss:    iss,
			Alg:    alg,
		})
		Expect(err).To(MatchError(ContainSubstring("aud is required")))
		_, err = jwtee.New(jwtee.Input{
			Secret: secret,
			Aud:    aud,
			Iss:    "",
			Alg:    alg,
		})
		Expect(err).To(MatchError(ContainSubstring("iss is required")))
		_, err = jwtee.New(jwtee.Input{
			Secret: secret,
			Aud:    aud,
			Iss:    iss,
			Alg:    "",
		})
		Expect(err).To(MatchError(ContainSubstring("alg is required")))
	})
	It("can dup itself with non-empty input values", func() {
		jw := newJwtee()
		jw2 := jw.Dup(jwtee.Input{Aud: "another"})
		Expect(jw.Aud).To(Equal(aud))
		Expect(jw.Iss).To(Equal(iss))
		Expect(jw2.Aud).To(Equal("another"))
		Expect(jw2.Iss).To(Equal(iss))
	})
	Describe("parsing", func() {
		It("can verify with a string aud claim", func() {
			jw := newJwtee()
			s := `eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJhdWQiOiJoaSIsImlzcyI6InRoZXJlIiwiaWF0IjoxNTE2MjM5MDIyfQ.kTgZa43Zq9LrjDAEerD8feT2_TrIhzCPO1UC4bBXzgQ`
			cl, err := jw.ParseMapClaims(s)
			Expect(err).ToNot(HaveOccurred())
			Expect(cl["aud"]).To(Equal("hi"))
		})

		It("can fail with an invalid aud claim", func() {
			jw := newJwtee()
			s := `eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJhdWQiOiJ5byIsImlzcyI6InRoZXJlIiwiaWF0IjoxNTE2MjM5MDIyfQ.BG7D0kCIcdgTfhOFNxArgubEL_2_WQmxE4vpnOv_AlU`
			cl, err := jw.ParseMapClaims(s)
			Expect(err).To(MatchError("invalid aud"))
			Expect(cl["aud"]).To(Equal("yo"))
		})
		It("can verify with an array aud claim", func() {
			jw := newJwtee()
			s := `eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJhdWQiOlsiaGkiLCJoZWxsbyJdLCJpc3MiOiJ0aGVyZSIsImlhdCI6MTUxNjIzOTAyMn0.37-1H6f20flFs2vjJ6u2nzh7BQ51kyQyELEX0y_xE3c`
			cl, err := jw.ParseMapClaims(s)
			Expect(err).ToNot(HaveOccurred())
			Expect(cl["aud"]).To(BeEquivalentTo([]interface{}{"hi", "hello"}))
		})
		It("can fail if aud not in array aud claim", func() {
			jw := newJwtee()
			s := `eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJhdWQiOlsieW8iXSwiaXNzIjoidGhlcmUiLCJpYXQiOjE1MTYyMzkwMjJ9.u-WkwjTF4kxdGB2wtinAtC1usOnIqeTPnDKg2HQ2gJw`
			cl, err := jw.ParseMapClaims(s)
			Expect(err).To(MatchError("invalid aud"))
			Expect(cl["aud"]).To(BeEquivalentTo([]interface{}{"yo"}))
		})
		It("can verify against an issuer", func() {
			jw := newJwtee()
			s := `eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJhdWQiOiJoaSIsImlzcyI6InRoZXJlIiwiaWF0IjoxNTE2MjM5MDIyfQ.kTgZa43Zq9LrjDAEerD8feT2_TrIhzCPO1UC4bBXzgQ`
			cl, err := jw.ParseMapClaims(s)
			Expect(err).ToNot(HaveOccurred())
			Expect(cl["iss"]).To(Equal("there"))
		})
		It("can fail with an invalid issuer", func() {
			jw := newJwtee()
			s := `eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJhdWQiOiJoaSIsImlzcyI6InlvbmRlciIsImlhdCI6MTUxNjIzOTAyMn0.Wo0zf5P9H4HAnOWTgdUKNN0W-jTTJot0lEl5kE1r3YY`
			cl, err := jw.ParseMapClaims(s)
			Expect(err).To(MatchError("invalid iss"))
			Expect(cl["iss"]).To(Equal("yonder"))
		})
		It("validates against the signing method", func() {
			jw := newJwtee()
			s := `eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJhdWQiOiJoaSIsImlzcyI6InRoZXJlIiwiaWF0IjoxNTE2MjM5MDIyfQ.kTgZa43Zq9LrjDAEerD8feT2_TrIhzCPO1UC4bBXzgQ`
			tok, err := jw.Parse(s)
			Expect(err).ToNot(HaveOccurred())
			Expect(tok.Header["alg"]).To(Equal("HS256"))
		})
		It("can fail with an unexpected signing method", func() {
			jw := newJwtee()
			s := `eyJhbGciOiJIUzUxMiIsInR5cCI6IkpXVCJ9.eyJhdWQiOiJoaSIsImlzcyI6InlvbmRlciIsImlhdCI6MTUxNjIzOTAyMn0.7q_DMegJbTO9uxWPy7n2mfDrBgAO3xBSpVmGjqHG6-ubve8QH2Y1d2noYWMk-wjSwkfbVB1K98FCfVDvZxhfGA`
			tok, err := jw.Parse(s)
			Expect(err).To(MatchError("invalid alg"))
			Expect(tok.Header["alg"]).To(Equal("HS512"))
		})
		It("can verify an unexpired exp", func() {
			jw := newJwtee()
			s := `eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJhdWQiOiJoaSIsImlzcyI6InRoZXJlIiwiaWF0IjoxNTE2MjM5MDIyLCJleHAiOjE1OTc5ODkyMzM1MTR9.bpLset-GCKbOlish900xGamrRCqQzmX06A2e2BtAdJE`
			cl, err := jw.ParseMapClaims(s)
			Expect(err).ToNot(HaveOccurred())
			Expect(cl["exp"]).To(BeEquivalentTo(1597989233514))
		})
		It("fails expired exp", func() {
			jw := newJwtee()
			s := `eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJhdWQiOiJoaSIsImlzcyI6InRoZXJlIiwiaWF0IjoxNTE2MjM5MDIyLCJleHAiOjV9.XsqW1BORkEZBwYXzDVgPJmSV-6wDzkFaZ7NacIfDjNY`
			cl, err := jw.ParseMapClaims(s)
			Expect(err).To(MatchError("Token is expired"))
			Expect(cl["exp"]).To(BeEquivalentTo(5))
		})
	})
	Describe("building", func() {
		origTime := jwt.TimeFunc
		AfterEach(func() {
			jwt.TimeFunc = origTime
		})
		It("builds a token with the default fields and additional fields", func() {
			jwt.TimeFunc = func() time.Time {
				return time.Unix(10000, 0)
			}
			jw := newJwtee()
			js, err := jw.BuildTtl(654*time.Second, map[string]interface{}{"sub": 1234})
			Expect(err).ToNot(HaveOccurred())
			expected := `eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJhdWQiOiJoaSIsImV4cCI6MTA2NTQsImlzcyI6InRoZXJlIiwic3ViIjoxMjM0fQ.OgPwnSrNaEpCgSMcILAdATor2NGlupnt7ggbqr32NL0`
			Expect(js).To(Equal(expected))
		})
	})
	Describe("StringClaim", func() {
		It("extracts a non-empty string claim", func() {
			c := jwt.MapClaims{"s": "", "s2": "a", "i": 1}
			var s string
			var ok bool
			s, ok = jwtee.StringClaim(c, "s2")
			Expect(ok).To(BeTrue())
			Expect(s).To(Equal("a"))

			s, ok = jwtee.StringClaim(c, "s")
			Expect(ok).To(BeFalse())
			Expect(s).To(BeEmpty())

			s, ok = jwtee.StringClaim(c, "x")
			Expect(ok).To(BeFalse())
			Expect(s).To(BeEmpty())

			s, ok = jwtee.StringClaim(c, "i")
			Expect(ok).To(BeFalse())
			Expect(s).To(BeEmpty())
		})
	})
})
