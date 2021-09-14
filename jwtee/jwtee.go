// Package jwtee wraps github.com/dgrijalva/jwt-go
// with some tooling that makes it easier to use
// in most practical usage.
package jwtee

import (
	"crypto/subtle"
	"errors"
	"github.com/dgrijalva/jwt-go"
	"time"
)

type Error struct {
	msg string
}

func (e Error) Error() string {
	return e.msg
}

type Jwtee struct {
	Secret []byte
	Aud    string
	Iss    string
	Alg    jwt.SigningMethod
}

type Input struct {
	Secret string
	Aud    string
	Iss    string
	Alg    string
}

func New(input Input) (Jwtee, error) {
	j := Jwtee{
		Secret: []byte(input.Secret),
		Aud:    input.Aud,
		Iss:    input.Iss,
		Alg:    jwt.GetSigningMethod(input.Alg),
	}
	if len(j.Secret) == 0 {
		return j, errors.New("secret is required")
	}
	if j.Aud == "" {
		return j, errors.New("aud is required")
	}
	if j.Iss == "" {
		return j, errors.New("iss is required")
	}
	if j.Alg == nil {
		return j, errors.New("valid alg is required")
	}
	return j, nil
}

func (j Jwtee) Parse(tokenString string) (*jwt.Token, error) {
	return jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if token.Method != j.Alg {
			return token, Error{msg: "invalid alg"}
		}
		checkAud := verifyArrayAudience(token.Claims.(jwt.MapClaims), j.Aud, true)
		if !checkAud {
			return token, Error{msg: "invalid aud"}
		}
		checkIss := token.Claims.(jwt.MapClaims).VerifyIssuer(j.Iss, true)
		if !checkIss {
			return token, Error{msg: "invalid iss"}
		}
		return j.Secret, nil
	})
}

func (j Jwtee) ParseMapClaims(tokenString string) (jwt.MapClaims, error) {
	tok, err := j.Parse(tokenString)
	if tok == nil {
		panic("token should never be nil")
	}
	return tok.Claims.(jwt.MapClaims), err
}

func (j Jwtee) BuildTtl(ttl time.Duration, moreClaims map[string]interface{}) (string, error) {
	tok := jwt.New(j.Alg)
	mc := tok.Claims.(jwt.MapClaims)
	mc["iss"] = j.Iss
	mc["aud"] = j.Aud
	mc["exp"] = jwt.TimeFunc().Add(ttl).Unix()
	for k, v := range moreClaims {
		mc[k] = v
	}
	return tok.SignedString(j.Secret)
}

func (j Jwtee) Dup(input Input) Jwtee {
	if len(input.Secret) > 0 {
		j.Secret = []byte(input.Secret)
	}
	if input.Aud != "" {
		j.Aud = input.Aud
	}
	if input.Iss != "" {
		j.Iss = input.Iss
	}
	if input.Alg != "" {
		j.Alg = jwt.GetSigningMethod(input.Alg)
	}
	return j
}

// See https://github.com/dgrijalva/jwt-go/pull/308
// These two methods are straight copy paste
func verifyArrayAudience(m jwt.MapClaims, cmp string, req bool) bool {
	switch m["aud"].(type) {
	case string:
		aud := m["aud"].(string)
		return verifyAudHelper(aud, cmp, req)
	default:
		auds := m["aud"].([]interface{})
		for _, aud := range auds {
			if verifyAudHelper(aud.(string), cmp, req) {
				return true
			}
		}
		return false
	}
}

func verifyAudHelper(aud string, cmp string, required bool) bool {
	if aud == "" {
		return !required
	}
	if subtle.ConstantTimeCompare([]byte(aud), []byte(cmp)) != 0 {
		return true
	} else {
		return false
	}
}

func StringClaim(claims jwt.MapClaims, key string) (string, bool) {
	v, ok := claims[key]
	if !ok {
		return "", false
	}
	s, ok := v.(string)
	if !ok {
		return "", false
	}
	return s, len(s) > 0
}
