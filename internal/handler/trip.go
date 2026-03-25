package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/yourname/tabikake/internal/model"
	"github.com/yourname/tabikake/internal/service"
)

// TripHandler handles trip routes.
type TripHandler struct {
	tripSvc *service.TripService
}

// NewTripHandler creates a new TripHandler.
func NewTripHandler(tripSvc *service.TripService) *TripHandler {
	return &TripHandler{tripSvc: tripSvc}
}

// ListTrips handles GET /trips
func (h *TripHandler) ListTrips(c echo.Context) error {
	trips, err := h.tripSvc.ListTrips(c.Request().Context())
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	if trips == nil {
		trips = []model.Trip{}
	}
	return c.JSON(http.StatusOK, trips)
}

// CreateTrip handles POST /trips
// Creates a Notion page + Records database, then saves to SQLite.
func (h *TripHandler) CreateTrip(c echo.Context) error {
	var req model.CreateTripRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if req.Name == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "name is required")
	}

	trip, err := h.tripSvc.CreateTrip(c.Request().Context(), req)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusCreated, trip)
}

// GetTrip handles GET /trips/:id
func (h *TripHandler) GetTrip(c echo.Context) error {
	tripID := c.Param("id")
	trip, err := h.tripSvc.GetTrip(c.Request().Context(), tripID)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, err.Error())
	}
	return c.JSON(http.StatusOK, trip)
}
