package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/yourname/tabikake/internal/service"
)

// DashboardHandler handles dashboard aggregation routes.
type DashboardHandler struct {
	dashboardSvc *service.DashboardService
}

// NewDashboardHandler creates a new DashboardHandler.
func NewDashboardHandler(dashboardSvc *service.DashboardService) *DashboardHandler {
	return &DashboardHandler{dashboardSvc: dashboardSvc}
}

// GetDashboard handles GET /dashboard/:trip_id
func (h *DashboardHandler) GetDashboard(c echo.Context) error {
	tripID := c.Param("trip_id")
	if tripID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "trip_id is required")
	}

	dashboard, err := h.dashboardSvc.GetDashboard(c.Request().Context(), tripID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, dashboard)
}
