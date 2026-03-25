package middleware

import (
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"

	"github.com/yourname/tabikake/internal/model"
)

const userContextKey = "user"

// JWTAuth returns an Echo middleware that validates Bearer JWT tokens.
// On success, it stores *model.JWTClaims in the context under key "user".
func JWTAuth(secret string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			authHeader := c.Request().Header.Get("Authorization")
			if authHeader == "" {
				return echo.NewHTTPError(http.StatusUnauthorized, "missing Authorization header")
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
				return echo.NewHTTPError(http.StatusUnauthorized, "invalid Authorization header format")
			}

			tokenStr := parts[1]
			claims := &model.JWTClaims{}

			token, err := jwt.ParseWithClaims(tokenStr, &jwtClaimsAdapter{claims}, func(t *jwt.Token) (interface{}, error) {
				if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, echo.NewHTTPError(http.StatusUnauthorized, "unexpected signing method")
				}
				return []byte(secret), nil
			})
			if err != nil || !token.Valid {
				return echo.NewHTTPError(http.StatusUnauthorized, "invalid or expired token")
			}

			c.Set(userContextKey, claims)
			return next(c)
		}
	}
}

// GetUser extracts the authenticated user claims from the Echo context.
func GetUser(c echo.Context) *model.JWTClaims {
	val := c.Get(userContextKey)
	if val == nil {
		return nil
	}
	claims, _ := val.(*model.JWTClaims)
	return claims
}

// jwtClaimsAdapter bridges model.JWTClaims with jwt.Claims interface.
type jwtClaimsAdapter struct {
	*model.JWTClaims
}

func (j *jwtClaimsAdapter) GetExpirationTime() (*jwt.NumericDate, error) { return nil, nil }
func (j *jwtClaimsAdapter) GetIssuedAt() (*jwt.NumericDate, error)       { return nil, nil }
func (j *jwtClaimsAdapter) GetNotBefore() (*jwt.NumericDate, error)      { return nil, nil }
func (j *jwtClaimsAdapter) GetIssuer() (string, error)                   { return "", nil }
func (j *jwtClaimsAdapter) GetSubject() (string, error)                  { return j.UserID, nil }
func (j *jwtClaimsAdapter) GetAudience() (jwt.ClaimStrings, error)       { return nil, nil }
