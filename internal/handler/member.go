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

// MemberHandler handles member routes.
type MemberHandler struct {
	memberSvc *service.MemberService
	tripSvc   *service.TripService
}

// NewMemberHandler creates a new MemberHandler.
func NewMemberHandler(memberSvc *service.MemberService, tripSvc *service.TripService) *MemberHandler {
	return &MemberHandler{memberSvc: memberSvc, tripSvc: tripSvc}
}

// ListMembers handles GET /trips/:id/members.
func (h *MemberHandler) ListMembers(c echo.Context) error {
	tripID := c.Param("id")
	members, err := h.memberSvc.ListMembers(c.Request().Context(), tripID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, members)
}

// JoinTrip handles POST /trips/join — joins a trip via invite code using the JWT user's identity.
func (h *MemberHandler) JoinTrip(c echo.Context) error {
	claims := appmiddleware.GetUser(c)
	var req model.JoinTripRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if req.InviteCode == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "invite_code is required")
	}
	result, err := h.tripSvc.JoinTrip(c.Request().Context(), claims.UserID, req)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "invalid invite code")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusCreated, result)
}

// DeleteMember handles DELETE /trips/:id/members/:user_id (owner only).
func (h *MemberHandler) DeleteMember(c echo.Context) error {
	claims := appmiddleware.GetUser(c)
	tripID := c.Param("id")
	targetUserID := c.Param("user_id")

	isOwner, err := h.memberSvc.IsOwner(c.Request().Context(), tripID, claims.UserID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	if !isOwner {
		return echo.NewHTTPError(http.StatusForbidden, "only the trip owner can remove members")
	}

	if err := h.memberSvc.RemoveMember(c.Request().Context(), tripID, targetUserID); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "member not found")
		}
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	return c.NoContent(http.StatusNoContent)
}
