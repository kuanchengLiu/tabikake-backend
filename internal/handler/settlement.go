package handler

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/yourname/tabikake/internal/service"
	"github.com/yourname/tabikake/internal/store"
)

// SettlementHandler handles settlement routes.
type SettlementHandler struct {
	settlementSvc *service.SettlementService
}

// NewSettlementHandler creates a new SettlementHandler.
func NewSettlementHandler(settlementSvc *service.SettlementService) *SettlementHandler {
	return &SettlementHandler{settlementSvc: settlementSvc}
}

// Calculate handles GET /trips/:id/settlement — returns settlement without writing to Notion.
func (h *SettlementHandler) Calculate(c echo.Context) error {
	tripID := c.Param("id")
	result, err := h.settlementSvc.Calculate(c.Request().Context(), tripID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "trip not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, result)
}

// Export handles POST /trips/:id/settlement/export — calculates and creates a Notion summary page.
func (h *SettlementHandler) Export(c echo.Context) error {
	tripID := c.Param("id")
	result, err := h.settlementSvc.Export(c.Request().Context(), tripID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "trip not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusCreated, result)
}
