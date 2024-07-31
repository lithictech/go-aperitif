// Package auth0jwt is a modification of the Auth0 provided Go tutorial:
// https://auth0.com/docs/quickstart/backend/golang
// As you may guess that has several issues, but a lot of what's here has been taken verbatim.
package auth0jwt

import (
	"crypto/subtle"
	"encoding/json"
	"errors"
	"github.com/auth0/go-jwt-middleware"
	"github.com/dgrijalva/jwt-go"
	"github.com/labstack/echo/v4"
	"net/http"
)

type Jwks struct {
	Keys []JSONWebKeys `json:"keys"`
}

type JSONWebKeys struct {
	Kty string   `json:"kty"`
	Kid string   `json:"kid"`
	Use string   `json:"use"`
	N   string   `json:"n"`
	E   string   `json:"e"`
	X5c []string `json:"x5c"`
}

type Config struct {
	// Aud is used to veirfy the 'aud' claim. It's the identifier of the API in Auth0.
	Aud string
	// Iss is used to verify the 'iss' claim.
	Iss string
	// JwksPath is the path to the file like "https://my-application.auth0.com/.well-known/jwks.json".
	// See https://auth0.com/docs/tokens/concepts/jwks
	JwksPath string
}

func NewMiddleware(cfg Config) echo.MiddlewareFunc {
	mw := jwtmiddleware.New(jwtmiddleware.Options{
		ValidationKeyGetter: func(token *jwt.Token) (interface{}, error) {
			checkAud := verifyArrayAudience(token.Claims.(jwt.MapClaims), cfg.Aud, true)
			if !checkAud {
				return token, echo.NewHTTPError(401, "invalid audience")
			}
			checkIss := token.Claims.(jwt.MapClaims).VerifyIssuer(cfg.Iss, true)
			if !checkIss {
				return token, echo.NewHTTPError(401, "invalid issuer")
			}

			cert, err := getPemCert(cfg.JwksPath, token)
			if err != nil {
				return nil, err
			}

			result, _ := jwt.ParseRSAPublicKeyFromPEM([]byte(cert))
			return result, nil
		},
		UserProperty:        "user",
		CredentialsOptional: false,
		Debug:               false,
	})

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			req := c.Request()
			// req gets stomped by CheckJWT, to have a new context
			err := mw.CheckJWT(c.Response().Writer, req)
			if err != nil {
				return err
			}
			user := req.Context().Value(mw.Options.UserProperty)
			if user == nil {
				panic("why is 'user' nil in jwt context!?")
			}
			c.Set(mw.Options.UserProperty, user)
			return next(c)
		}
	}
}

func getPemCert(jwksPath string, token *jwt.Token) (string, error) {
	cert := ""
	resp, err := http.Get(jwksPath)

	if err != nil {
		return cert, err
	}
	defer resp.Body.Close()

	var jwks = Jwks{}
	err = json.NewDecoder(resp.Body).Decode(&jwks)

	if err != nil {
		return cert, err
	}

	for k := range jwks.Keys {
		if token.Header["kid"] == jwks.Keys[k].Kid {
			cert = "-----BEGIN CERTIFICATE-----\n" + jwks.Keys[k].X5c[0] + "\n-----END CERTIFICATE-----"
		}
	}

	if cert == "" {
		err := errors.New("unable to find appropriate key")
		return cert, err
	}

	return cert, nil
}

// Seehttps://github.com/dgrijalva/jwt-go/pull/308
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
