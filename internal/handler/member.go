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

// MemberHandler handles member and settlement routes.
type MemberHandler struct {
	memberSvc     *service.MemberService
	settlementSvc *service.SettlementService
}

// NewMemberHandler creates a new MemberHandler.
func NewMemberHandler(memberSvc *service.MemberService, settlementSvc *service.SettlementService) *MemberHandler {
	return &MemberHandler{memberSvc: memberSvc, settlementSvc: settlementSvc}
}

// ListMembers handles GET /trips/:id/members
func (h *MemberHandler) ListMembers(c echo.Context) error {
	tripID := c.Param("id")
	members, err := h.memberSvc.ListMembers(c.Request().Context(), tripID)
	if err != nil {
		if errors.Is(err, appdb.ErrNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "trip not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, members)
}

// AddMember handles POST /trips/:id/members
func (h *MemberHandler) AddMember(c echo.Context) error {
	tripID := c.Param("id")

	var req model.CreateMemberRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if req.Name == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "name is required")
	}

	member, err := h.memberSvc.AddMember(c.Request().Context(), tripID, req)
	if err != nil {
		if errors.Is(err, appdb.ErrNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "trip not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusCreated, member)
}

// JoinTrip handles POST /trips/join
func (h *MemberHandler) JoinTrip(c echo.Context) error {
	var req model.JoinTripRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if req.InviteCode == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "invite_code is required")
	}
	if req.Name == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "name is required")
	}

	result, err := h.memberSvc.JoinTrip(c.Request().Context(), req)
	if err != nil {
		if errors.Is(err, appdb.ErrNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "invalid invite code")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusCreated, result)
}

// DeleteMember handles DELETE /trips/:id/members/:member_id
// Requires the requester (X-Member-ID) to be the trip owner.
func (h *MemberHandler) DeleteMember(c echo.Context) error {
	tripID := c.Param("id")
	memberID := c.Param("member_id")

	// Only the owner may remove members.
	requesterID := appmiddleware.GetMemberID(c)
	if requesterID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "X-Member-ID header is required")
	}
	isOwner, err := h.memberSvc.IsOwner(c.Request().Context(), tripID, requesterID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	if !isOwner {
		return echo.NewHTTPError(http.StatusForbidden, "only the trip owner can remove members")
	}

	if err := h.memberSvc.DeleteMember(c.Request().Context(), tripID, memberID); err != nil {
		if errors.Is(err, appdb.ErrNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "member not found")
		}
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	return c.NoContent(http.StatusNoContent)
}

// GetSettlement handles GET /trips/:id/settlement
func (h *MemberHandler) GetSettlement(c echo.Context) error {
	tripID := c.Param("id")
	result, err := h.settlementSvc.GetSettlement(c.Request().Context(), tripID)
	if err != nil {
		if errors.Is(err, appdb.ErrNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "trip not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, result)
}
