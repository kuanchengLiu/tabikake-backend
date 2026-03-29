package handler

import (
	"encoding/base64"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"

	appmiddleware "github.com/yourname/tabikake/internal/middleware"
	"github.com/yourname/tabikake/internal/model"
	"github.com/yourname/tabikake/internal/service"
	"github.com/yourname/tabikake/internal/store"
)

// RecordHandler handles expense record routes.
type RecordHandler struct {
	recordSvc *service.RecordService
}

// NewRecordHandler creates a new RecordHandler.
func NewRecordHandler(recordSvc *service.RecordService) *RecordHandler {
	return &RecordHandler{recordSvc: recordSvc}
}

// ListRecords handles GET /records?trip_id=
func (h *RecordHandler) ListRecords(c echo.Context) error {
	tripID := c.QueryParam("trip_id")
	if tripID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "trip_id is required")
	}
	records, err := h.recordSvc.ListRecords(c.Request().Context(), tripID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return echo.NewHTTPError(http.StatusBadRequest, "trip not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	if records == nil {
		records = []model.Record{}
	}
	return c.JSON(http.StatusOK, records)
}

// CreateRecord handles POST /records
func (h *RecordHandler) CreateRecord(c echo.Context) error {
	claims := appmiddleware.GetUser(c)
	var req model.CreateRecordRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if req.TripID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "trip_id is required")
	}
	if req.StoreNameZH == "" || req.Date == "" || req.AmountJPY == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "store_name_zh, date, and amount_jpy are required")
	}
	if req.PaidByUserID == "" {
		req.PaidByUserID = claims.UserID
	}
	record, err := h.recordSvc.CreateRecord(c.Request().Context(), req)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return echo.NewHTTPError(http.StatusBadRequest, "trip_id not found")
		}
		if service.IsValidationError(err) {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusCreated, record)
}

// UpdateRecord handles PATCH /records/:id
func (h *RecordHandler) UpdateRecord(c echo.Context) error {
	pageID := c.Param("id")
	var req model.UpdateRecordRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	record, err := h.recordSvc.UpdateRecord(c.Request().Context(), pageID, req)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, record)
}

// DeleteRecord handles DELETE /records/:id
func (h *RecordHandler) DeleteRecord(c echo.Context) error {
	pageID := c.Param("id")
	if err := h.recordSvc.DeleteRecord(c.Request().Context(), pageID); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, map[string]bool{"success": true})
}

// ParseReceipt handles POST /parse
// Accepts multipart form with "image" field, or JSON body with "image_base64" (data URI or raw base64).
func (h *RecordHandler) ParseReceipt(c echo.Context) error {
	ctx := c.Request().Context()

	// Try multipart file first.
	file, header, err := c.Request().FormFile("image")
	if err == nil {
		defer file.Close()
		data, err := io.ReadAll(file)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "read file: "+err.Error())
		}
		mediaType := http.DetectContentType(data)
		if mediaType == "application/octet-stream" {
			mediaType = extensionMediaType(header.Filename)
		}
		imageBase64 := base64.StdEncoding.EncodeToString(data)
		result, err := h.recordSvc.ParseReceipt(ctx, imageBase64, mediaType)
		if err != nil {
			return echo.NewHTTPError(http.StatusUnprocessableEntity, err.Error())
		}
		return c.JSON(http.StatusOK, result)
	}

	// Fallback: JSON body with base64 image.
	var req struct {
		ImageBase64 string `json:"image_base64"`
	}
	if err := c.Bind(&req); err != nil || req.ImageBase64 == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "provide multipart 'image' field or JSON 'image_base64'")
	}

	imageBase64, mediaType := stripDataURI(req.ImageBase64)
	result, err := h.recordSvc.ParseReceipt(ctx, imageBase64, mediaType)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnprocessableEntity, err.Error())
	}
	return c.JSON(http.StatusOK, result)
}

// extensionMediaType guesses a media type from the file extension.
func extensionMediaType(filename string) string {
	lower := strings.ToLower(filename)
	switch {
	case strings.HasSuffix(lower, ".jpg") || strings.HasSuffix(lower, ".jpeg"):
		return "image/jpeg"
	case strings.HasSuffix(lower, ".png"):
		return "image/png"
	case strings.HasSuffix(lower, ".webp"):
		return "image/webp"
	default:
		return "image/jpeg"
	}
}

// stripDataURI parses "data:<type>;base64,<data>" or returns input as-is with jpeg assumed.
func stripDataURI(input string) (imageBase64, mediaType string) {
	if strings.HasPrefix(input, "data:") {
		rest := input[5:]
		if semi := strings.IndexByte(rest, ';'); semi > 0 {
			mt := rest[:semi]
			after := rest[semi+1:]
			if strings.HasPrefix(after, "base64,") {
				return after[7:], mt
			}
		}
	}
	return input, "image/jpeg"
}
