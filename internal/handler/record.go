package handler

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"

	appdb "github.com/yourname/tabikake/internal/db"
	appmiddleware "github.com/yourname/tabikake/internal/middleware"
	"github.com/yourname/tabikake/internal/model"
	"github.com/yourname/tabikake/internal/service"
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
		if errors.Is(err, appdb.ErrNotFound) {
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
	var req model.CreateRecordRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	if req.TripID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "trip_id is required")
	}
	if req.Store == "" || req.Date == "" || req.AmountJPY == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "store, date, and amount_jpy are required")
	}

	user := appmiddleware.GetUser(c)
	if user == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "not authenticated")
	}
	if req.PaidBy == "" {
		req.PaidBy = user.UserID
	}
	if req.PaidByName == "" {
		req.PaidByName = user.UserName
	}

	record, err := h.recordSvc.CreateRecord(c.Request().Context(), req)
	if err != nil {
		if errors.Is(err, appdb.ErrNotFound) {
			return echo.NewHTTPError(http.StatusBadRequest, "trip_id not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusCreated, record)
}
