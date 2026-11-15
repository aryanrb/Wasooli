package handlers

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"wasooli/internal/auth"
	"wasooli/internal/httperr"
	"wasooli/internal/types"
)

func (h *Handler) ListContracts(c echo.Context) error {
	id, err := paramUUID(c, "id")
	if err != nil {
		return err
	}
	cs, err := h.svc.ListContracts(c.Request().Context(), auth.UserID(c), id)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, cs)
}

func (h *Handler) CreateContract(c echo.Context) error {
	id, err := paramUUID(c, "id")
	if err != nil {
		return err
	}
	var in types.CreateContractInput
	if err := c.Bind(&in); err != nil {
		return httperr.BadRequest("invalid request body")
	}
	if err := in.Validate(); err != nil {
		return httperr.BadRequest(err.Error())
	}
	res, err := h.svc.CreateContract(c.Request().Context(), auth.UserID(c), id, in)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, res)
}

func (h *Handler) GetContract(c echo.Context) error {
	id, err := paramUUID(c, "id")
	if err != nil {
		return err
	}
	cv, err := h.svc.GetContract(c.Request().Context(), auth.UserID(c), id)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, cv)
}

func (h *Handler) UpdateContract(c echo.Context) error {
	id, err := paramUUID(c, "id")
	if err != nil {
		return err
	}
	var in types.UpdateContractInput
	if err := c.Bind(&in); err != nil {
		return httperr.BadRequest("invalid request body")
	}
	if err := in.Validate(); err != nil {
		return httperr.BadRequest(err.Error())
	}
	cv, err := h.svc.UpdateContract(c.Request().Context(), auth.UserID(c), id, in)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, cv)
}

func (h *Handler) RaiseRent(c echo.Context) error {
	id, err := paramUUID(c, "id")
	if err != nil {
		return err
	}
	var in types.RaiseRentInput
	if err := c.Bind(&in); err != nil {
		return httperr.BadRequest("invalid request body")
	}
	if err := in.Validate(); err != nil {
		return httperr.BadRequest(err.Error())
	}
	t, err := h.svc.RaiseRent(c.Request().Context(), auth.UserID(c), id, in)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, t)
}

func (h *Handler) EndContract(c echo.Context) error {
	id, err := paramUUID(c, "id")
	if err != nil {
		return err
	}
	cv, err := h.svc.EndContract(c.Request().Context(), auth.UserID(c), id)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, cv)
}

func (h *Handler) ContractPeriods(c echo.Context) error {
	id, err := paramUUID(c, "id")
	if err != nil {
		return err
	}
	ps, err := h.svc.ContractPeriods(c.Request().Context(), auth.UserID(c), id)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, ps)
}
