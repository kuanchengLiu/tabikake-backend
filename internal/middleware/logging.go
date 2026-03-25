package middleware

import (
	"github.com/labstack/echo/v4"
	echomiddleware "github.com/labstack/echo/v4/middleware"
)

// Logger returns an Echo request logging middleware.
func Logger() echo.MiddlewareFunc {
	return echomiddleware.LoggerWithConfig(echomiddleware.LoggerConfig{
		Format: `{"time":"${time_rfc3339}","method":"${method}","uri":"${uri}","status":${status},"latency":"${latency_human}","bytes_out":${bytes_out}}` + "\n",
	})
}

// Recover returns an Echo panic recovery middleware.
func Recover() echo.MiddlewareFunc {
	return echomiddleware.Recover()
}
