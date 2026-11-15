package handlers

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"wasooli/internal/auth"
)

func (h *Handler) Analytics(c echo.Context) error {
	a, err := h.svc.Analytics(c.Request().Context(), auth.UserID(c))
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, a)
}

func (h *Handler) PropertyAudit(c echo.Context) error {
	id, err := paramUUID(c, "id")
	if err != nil {
		return err
	}
	rows, err := h.svc.ListAudit(c.Request().Context(), auth.UserID(c), "property", id)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, rows)
}

func (h *Handler) ContractAudit(c echo.Context) error {
	id, err := paramUUID(c, "id")
	if err != nil {
		return err
	}
	rows, err := h.svc.ListAudit(c.Request().Context(), auth.UserID(c), "contract", id)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, rows)
}
