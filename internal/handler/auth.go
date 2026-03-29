package handler

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"

	appmiddleware "github.com/yourname/tabikake/internal/middleware"
	"github.com/yourname/tabikake/internal/service"
)

// AuthHandler handles authentication routes.
type AuthHandler struct {
	authSvc     *service.AuthService
	frontendURL string
}

// NewAuthHandler creates a new AuthHandler.
func NewAuthHandler(authSvc *service.AuthService, frontendURL string) *AuthHandler {
	return &AuthHandler{authSvc: authSvc, frontendURL: frontendURL}
}

// OAuthURL handles GET /auth/notion/url — returns the Notion OAuth authorization URL.
func (h *AuthHandler) OAuthURL(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"url": h.authSvc.OAuthURL()})
}

// NotionCallback handles GET /auth/notion/callback?code=
// Called when Notion redirects the browser directly to the backend.
// Sets an httpOnly auth cookie and redirects to the frontend.
func (h *AuthHandler) NotionCallback(c echo.Context) error {
	code := c.QueryParam("code")
	if code == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "code is required")
	}

	tokenStr, user, err := h.authSvc.HandleCallback(c.Request().Context(), code)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}

	h.setAuthCookie(c, tokenStr)
	_ = user
	return c.Redirect(http.StatusFound, h.frontendURL)
}

// NotionCallbackPost handles POST /auth/notion/callback
// Called by the frontend after it receives the OAuth code from Notion's redirect.
// Sets an httpOnly auth cookie and returns the user as JSON.
func (h *AuthHandler) NotionCallbackPost(c echo.Context) error {
	var req struct {
		Code string `json:"code" form:"code"`
	}
	if err := c.Bind(&req); err != nil || req.Code == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "code is required")
	}

	tokenStr, user, err := h.authSvc.HandleCallback(c.Request().Context(), req.Code)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}

	h.setAuthCookie(c, tokenStr)
	return c.JSON(http.StatusOK, map[string]interface{}{
		"token": tokenStr,
		"user":  user,
	})
}

func (h *AuthHandler) setAuthCookie(c echo.Context, tokenStr string) {
	cookie := new(http.Cookie)
	cookie.Name = "auth_token"
	cookie.Value = tokenStr
	cookie.HttpOnly = true
	cookie.SameSite = http.SameSiteLaxMode
	cookie.Path = "/"
	cookie.Expires = time.Now().Add(30 * 24 * time.Hour)
	c.SetCookie(cookie)
}

// Me handles GET /auth/me — returns the current user's profile.
func (h *AuthHandler) Me(c echo.Context) error {
	claims := appmiddleware.GetUser(c)
	user, err := h.authSvc.GetUser(c.Request().Context(), claims.UserID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, user)
}

// Logout handles POST /auth/logout — deletes the session and clears the cookie.
func (h *AuthHandler) Logout(c echo.Context) error {
	claims := appmiddleware.GetUser(c)
	_ = h.authSvc.Logout(c.Request().Context(), claims.SessionID)

	cookie := new(http.Cookie)
	cookie.Name = "auth_token"
	cookie.Value = ""
	cookie.HttpOnly = true
	cookie.Path = "/"
	cookie.MaxAge = -1
	c.SetCookie(cookie)

	return c.JSON(http.StatusOK, map[string]bool{"success": true})
}
