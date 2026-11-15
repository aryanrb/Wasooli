package handlers

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"wasooli/internal/auth"
	"wasooli/internal/httperr"
	"wasooli/internal/types"
)

func (h *Handler) RecordTransaction(c echo.Context) error {
	var in types.RecordTransactionInput
	if err := c.Bind(&in); err != nil {
		return httperr.BadRequest("invalid request body")
	}
	if err := in.Validate(); err != nil {
		return httperr.BadRequest(err.Error())
	}
	tx, err := h.svc.RecordTransaction(c.Request().Context(), auth.UserID(c), in)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, tx)
}

func (h *Handler) ListPending(c echo.Context) error {
	txs, err := h.svc.ListPending(c.Request().Context(), auth.UserID(c))
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, txs)
}

func (h *Handler) ApproveTransaction(c echo.Context) error { return h.decide(c, true) }
func (h *Handler) RejectTransaction(c echo.Context) error  { return h.decide(c, false) }

func (h *Handler) decide(c echo.Context, approve bool) error {
	id, err := paramUUID(c, "id")
	if err != nil {
		return err
	}
	tx, err := h.svc.SetApproval(c.Request().Context(), auth.UserID(c), id, approve)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, tx)
}

func (h *Handler) ListContractTransactions(c echo.Context) error {
	id, err := paramUUID(c, "id")
	if err != nil {
		return err
	}
	txs, err := h.svc.ListContractTransactions(c.Request().Context(), auth.UserID(c), id)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, txs)
}

func (h *Handler) ListPropertyTransactions(c echo.Context) error {
	id, err := paramUUID(c, "id")
	if err != nil {
		return err
	}
	txs, err := h.svc.ListPropertyTransactions(c.Request().Context(), auth.UserID(c), id)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, txs)
}
