package handlers

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"wasooli/internal/auth"
	"wasooli/internal/httperr"
	"wasooli/internal/types"
)

func (h *Handler) CreateInvitation(c echo.Context) error {
	var in types.CreateInvitationInput
	if err := c.Bind(&in); err != nil {
		return httperr.BadRequest("invalid request body")
	}
	if err := in.Validate(); err != nil {
		return httperr.BadRequest(err.Error())
	}
	inv, err := h.svc.CreateInvitation(c.Request().Context(), auth.UserID(c), in)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, inv)
}

func (h *Handler) ListMyInvitations(c echo.Context) error {
	invs, err := h.svc.ListMyInvitations(c.Request().Context(), auth.UserID(c))
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, invs)
}

func (h *Handler) CounterInvitation(c echo.Context) error {
	var in types.CounterInvitationInput
	if err := c.Bind(&in); err != nil {
		return httperr.BadRequest("invalid request body")
	}
	if err := in.Validate(); err != nil {
		return httperr.BadRequest(err.Error())
	}
	inv, err := h.svc.CounterInvitation(c.Request().Context(), auth.UserID(c), in)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, inv)
}

func (h *Handler) AcceptInvitation(c echo.Context) error {
	var in types.AcceptInvitationInput
	if err := c.Bind(&in); err != nil || in.Token == "" {
		return httperr.BadRequest("token is required")
	}
	inv, err := h.svc.AcceptInvitation(c.Request().Context(), auth.UserID(c), in)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, inv)
}

func (h *Handler) DeclineInvitation(c echo.Context) error {
	var in struct {
		Token string `json:"token"`
	}
	if err := c.Bind(&in); err != nil || in.Token == "" {
		return httperr.BadRequest("token is required")
	}
	if err := h.svc.DeclineInvitation(c.Request().Context(), auth.UserID(c), in.Token); err != nil {
		return err
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}
