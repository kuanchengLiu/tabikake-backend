package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/yourname/tabikake/internal/service"
)

// SplitHandler handles settlement export routes.
type SplitHandler struct {
	splitSvc *service.SplitService
}

// NewSplitHandler creates a new SplitHandler.
func NewSplitHandler(splitSvc *service.SplitService) *SplitHandler {
	return &SplitHandler{splitSvc: splitSvc}
}

// ExportSettlement handles POST /split/export/:trip_id
// Calculates the settlement and creates a summary page in the trip's Notion parent page.
func (h *SplitHandler) ExportSettlement(c echo.Context) error {
	tripID := c.Param("trip_id")
	if tripID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "trip_id is required")
	}

	resp, err := h.splitSvc.ExportSettlement(c.Request().Context(), tripID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusCreated, resp)
}
