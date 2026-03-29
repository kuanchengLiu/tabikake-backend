package middleware

import (
	"github.com/labstack/echo/v4"
	echomiddleware "github.com/labstack/echo/v4/middleware"
)

// CORS returns an Echo CORS middleware configured for the frontend origin.
func CORS(frontendURL string) echo.MiddlewareFunc {
	origins := []string{"http://localhost:3000"}
	if frontendURL != "" {
		origins = append(origins, frontendURL)
	}

	return echomiddleware.CORSWithConfig(echomiddleware.CORSConfig{
		AllowOrigins:     origins,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Content-Type"},
		AllowCredentials: true,
	})
}
