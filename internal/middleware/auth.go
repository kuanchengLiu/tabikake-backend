package middleware

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/yourname/tabikake/internal/model"
	"github.com/yourname/tabikake/internal/service"
)

const userContextKey = "user"

// JWTAuth returns middleware that validates the auth_token httpOnly cookie via authSvc.
func JWTAuth(authSvc *service.AuthService) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			cookie, err := c.Cookie("auth_token")
			if err != nil || cookie.Value == "" {
				return echo.NewHTTPError(http.StatusUnauthorized, "not authenticated")
			}
			claims, err := authSvc.ValidateSession(c.Request().Context(), cookie.Value)
			if err != nil {
				return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
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
