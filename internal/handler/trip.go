package handler

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"

	appmiddleware "github.com/yourname/tabikake/internal/middleware"
	"github.com/yourname/tabikake/internal/model"
	"github.com/yourname/tabikake/internal/service"
	"github.com/yourname/tabikake/internal/store"
)

// TripHandler handles trip routes.
type TripHandler struct {
	tripSvc   *service.TripService
	memberSvc *service.MemberService
}

// NewTripHandler creates a new TripHandler.
func NewTripHandler(tripSvc *service.TripService, memberSvc *service.MemberService) *TripHandler {
	return &TripHandler{tripSvc: tripSvc, memberSvc: memberSvc}
}

// ListTrips handles GET /trips — returns trips where the JWT user is a member.
func (h *TripHandler) ListTrips(c echo.Context) error {
	claims := appmiddleware.GetUser(c)
	trips, err := h.tripSvc.ListTrips(c.Request().Context(), claims.UserID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, trips)
}

// CreateTrip handles POST /trips.
func (h *TripHandler) CreateTrip(c echo.Context) error {
	claims := appmiddleware.GetUser(c)
	var req model.CreateTripRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if req.Name == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "name is required")
	}
	trip, err := h.tripSvc.CreateTrip(c.Request().Context(), claims.UserID, req)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusCreated, trip)
}

// GetTrip handles GET /trips/:id — includes is_member for the JWT user.
func (h *TripHandler) GetTrip(c echo.Context) error {
	claims := appmiddleware.GetUser(c)
	tripID := c.Param("id")
	trip, err := h.tripSvc.GetTrip(c.Request().Context(), tripID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "trip not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	isMember, _ := h.memberSvc.IsMember(c.Request().Context(), tripID, claims.UserID)
	return c.JSON(http.StatusOK, map[string]interface{}{
		"trip":      trip,
		"is_member": isMember,
	})
}

// UpdateTrip handles PATCH /trips/:id (owner only).
func (h *TripHandler) UpdateTrip(c echo.Context) error {
	claims := appmiddleware.GetUser(c)
	tripID := c.Param("id")
	isOwner, err := h.memberSvc.IsOwner(c.Request().Context(), tripID, claims.UserID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	if !isOwner {
		return echo.NewHTTPError(http.StatusForbidden, "only the trip owner can update this trip")
	}
	var req model.UpdateTripRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	trip, err := h.tripSvc.UpdateTrip(c.Request().Context(), tripID, req)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "trip not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, trip)
}

// DeleteTrip handles DELETE /trips/:id (owner only).
func (h *TripHandler) DeleteTrip(c echo.Context) error {
	claims := appmiddleware.GetUser(c)
	tripID := c.Param("id")
	isOwner, err := h.memberSvc.IsOwner(c.Request().Context(), tripID, claims.UserID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	if !isOwner {
		return echo.NewHTTPError(http.StatusForbidden, "only the trip owner can delete this trip")
	}
	if err := h.tripSvc.DeleteTrip(c.Request().Context(), tripID); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "trip not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.NoContent(http.StatusNoContent)
}

// GetJoinInfo handles GET /trips/join-info?code= (public, no auth).
func (h *TripHandler) GetJoinInfo(c echo.Context) error {
	code := c.QueryParam("code")
	if code == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "code is required")
	}
	info, err := h.tripSvc.GetJoinInfo(c.Request().Context(), code)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "invalid invite code")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, info)
}
