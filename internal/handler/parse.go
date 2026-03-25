package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/yourname/tabikake/internal/service"
)

// ParseHandler handles receipt OCR parsing.
type ParseHandler struct {
	parseSvc *service.ParseService
}

// NewParseHandler creates a new ParseHandler.
func NewParseHandler(parseSvc *service.ParseService) *ParseHandler {
	return &ParseHandler{parseSvc: parseSvc}
}

// ParseReceipt handles POST /parse
// Accepts a multipart form with an "image" field, or a JSON body with "image_base64".
// Returns the structured receipt JSON without writing to Notion.
func (h *ParseHandler) ParseReceipt(c echo.Context) error {
	ctx := c.Request().Context()

	// Try multipart file first
	file, header, err := c.Request().FormFile("image")
	if err == nil {
		defer file.Close()
		result, err := h.parseSvc.ParseReceiptFile(ctx, file, header)
		if err != nil {
			return echo.NewHTTPError(http.StatusUnprocessableEntity, err.Error())
		}
		return c.JSON(http.StatusOK, result)
	}

	// Fallback: JSON body with base64 image
	var req struct {
		ImageBase64 string `json:"image_base64"`
	}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request: provide multipart 'image' or JSON 'image_base64'")
	}
	if req.ImageBase64 == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "image is required (multipart 'image' field or 'image_base64')")
	}

	result, err := h.parseSvc.ParseReceiptBase64(ctx, req.ImageBase64)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnprocessableEntity, err.Error())
	}
	return c.JSON(http.StatusOK, result)
}
