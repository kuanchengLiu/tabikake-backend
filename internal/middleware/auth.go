package middleware

import (
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"

	appdb "github.com/yourname/tabikake/internal/db"
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

const memberIDContextKey = "memberID"

// ValidateTripMember returns middleware that verifies the X-Member-ID header
// belongs to the trip extracted from the route/query params.
// tripParamName: Echo route param name (e.g. "id", "trip_id").
// If empty, falls back to query param "trip_id".
func ValidateTripMember(database *appdb.DB, tripParamName string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			memberID := c.Request().Header.Get("X-Member-ID")
			if memberID == "" {
				return echo.NewHTTPError(http.StatusBadRequest, "X-Member-ID header is required")
			}

			tripID := c.Param(tripParamName)
			if tripID == "" {
				tripID = c.QueryParam("trip_id")
			}
			if tripID == "" {
				return echo.NewHTTPError(http.StatusBadRequest, "trip ID not found in request")
			}

			ok, err := database.IsMember(c.Request().Context(), tripID, memberID)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
			}
			if !ok {
				return echo.NewHTTPError(http.StatusForbidden, "not a member of this trip")
			}

			c.Set(memberIDContextKey, memberID)
			return next(c)
		}
	}
}

// GetMemberID extracts the member ID set by ValidateTripMember, or reads the
// X-Member-ID header directly (for routes where the middleware is optional).
func GetMemberID(c echo.Context) string {
	if v, ok := c.Get(memberIDContextKey).(string); ok && v != "" {
		return v
	}
	return c.Request().Header.Get("X-Member-ID")
}
