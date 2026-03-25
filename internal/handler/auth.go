package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"

	appmiddleware "github.com/yourname/tabikake/internal/middleware"
	"github.com/yourname/tabikake/internal/model"
	"github.com/yourname/tabikake/internal/service"
)

// AuthHandler handles authentication routes.
type AuthHandler struct {
	authSvc *service.AuthService
}

// NewAuthHandler creates a new AuthHandler.
func NewAuthHandler(authSvc *service.AuthService) *AuthHandler {
	return &AuthHandler{authSvc: authSvc}
}

// NotionCallback handles POST /auth/notion/callback
// Accepts either:
//   - JSON body:              {"code": "..."}
//   - Form-encoded body:      code=...  (used by Postman OAuth 2.0 flow)
//
// Always responds with {"access_token": "<jwt>", "token_type": "Bearer", "user": {...}}
// so Postman can use it directly as an OAuth 2.0 Access Token URL.
func (h *AuthHandler) NotionCallback(c echo.Context) error {
	// Try form value first (Postman OAuth 2.0 sends application/x-www-form-urlencoded)
	code := c.FormValue("code")

	// Fallback to JSON body
	if code == "" {
		var req struct {
			Code string `json:"code"`
		}
		if err := c.Bind(&req); err == nil {
			code = req.Code
		}
	}

	if code == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "code is required")
	}

	resp, err := h.authSvc.ExchangeCode(c.Request().Context(), code)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}

	// Return in OAuth 2.0 token response format so Postman auto-uses the JWT
	return c.JSON(http.StatusOK, map[string]interface{}{
		"access_token": resp.Token,
		"token_type":   "Bearer",
		"user":         resp.User,
	})
}

// Me handles GET /auth/me
// Returns the currently authenticated user info from the JWT.
func (h *AuthHandler) Me(c echo.Context) error {
	user := appmiddleware.GetUser(c)
	if user == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "not authenticated")
	}
	return c.JSON(http.StatusOK, model.NotionUser{
		ID:   user.UserID,
		Name: user.UserName,
	})
}
