package handlers

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"wasooli/internal/auth"
	"wasooli/internal/httperr"
	"wasooli/internal/types"
)

func (h *Handler) ListProperties(c echo.Context) error {
	ps, err := h.svc.ListProperties(c.Request().Context(), auth.UserID(c))
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, ps)
}

// ListTenancies returns the caller's own contracts (their tenant-side entry point).
func (h *Handler) ListTenancies(c echo.Context) error {
	ts, err := h.svc.ListTenancies(c.Request().Context(), auth.UserID(c))
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, ts)
}

func (h *Handler) CreateProperty(c echo.Context) error {
	var in types.CreatePropertyInput
	if err := c.Bind(&in); err != nil {
		return httperr.BadRequest("invalid request body")
	}
	if err := in.Validate(); err != nil {
		return httperr.BadRequest(err.Error())
	}
	p, err := h.svc.CreateProperty(c.Request().Context(), auth.UserID(c), in)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, p)
}

func (h *Handler) GetProperty(c echo.Context) error {
	id, err := paramUUID(c, "id")
	if err != nil {
		return err
	}
	p, err := h.svc.GetProperty(c.Request().Context(), auth.UserID(c), id)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, p)
}

func (h *Handler) AddViewer(c echo.Context) error {
	id, err := paramUUID(c, "id")
	if err != nil {
		return err
	}
	var in struct {
		Email string `json:"email"`
	}
	if err := c.Bind(&in); err != nil || in.Email == "" {
		return httperr.BadRequest("email is required")
	}
	if err := h.svc.AddViewer(c.Request().Context(), auth.UserID(c), id, in.Email); err != nil {
		return err
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}
