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
// Creates a Notion page + Records database, saves to SQLite, and auto-creates owner member.
func (h *TripHandler) CreateTrip(c echo.Context) error {
	var req model.CreateTripRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if req.Name == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "name is required")
	}
	if req.OwnerName == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "owner_name is required")
	}
	if req.OwnerAvatarColor == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "owner_avatar_color is required")
	}

	resp, err := h.tripSvc.CreateTrip(c.Request().Context(), req)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusCreated, resp)
}

// GetTrip handles GET /trips/:id
// Accepts optional X-Member-ID header to populate is_member field.
func (h *TripHandler) GetTrip(c echo.Context) error {
	tripID := c.Param("id")
	trip, err := h.tripSvc.GetTrip(c.Request().Context(), tripID)
	if err != nil {
		if errors.Is(err, appdb.ErrNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "trip not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	memberID := appmiddleware.GetMemberID(c)
	isMember := false
	if memberID != "" {
		isMember, _ = h.tripSvc.IsMember(c.Request().Context(), tripID, memberID)
	}

	return c.JSON(http.StatusOK, model.TripDetailResponse{Trip: *trip, IsMember: isMember})
}

// GetJoinInfo handles GET /trips/join-info?code= (public, no auth required)
func (h *TripHandler) GetJoinInfo(c echo.Context) error {
	code := c.QueryParam("code")
	if code == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "code is required")
	}

	info, err := h.tripSvc.GetJoinInfo(c.Request().Context(), code)
	if err != nil {
		if errors.Is(err, appdb.ErrNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "invalid invite code")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, info)
}
