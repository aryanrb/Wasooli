// Package handlers is the thin HTTP layer: parse the request, call a service,
// write the response. No business logic or DB access lives here.
package handlers

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
	"golang.org/x/oauth2"

	"wasooli/config"
	"wasooli/internal/auth"
	"wasooli/internal/httperr"
	"wasooli/internal/services"
)

type Handler struct {
	svc   *services.Service
	cfg   *config.Config
	oauth *oauth2.Config
	log   *zap.Logger
}

func New(svc *services.Service, cfg *config.Config, log *zap.Logger) *Handler {
	return &Handler{svc: svc, cfg: cfg, oauth: auth.GoogleConfig(cfg), log: log}
}

// Health is a liveness probe.
func (h *Handler) Health(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

// RegisterRoutes mounts all API routes under the given group.
func (h *Handler) RegisterRoutes(g *echo.Group) {
	// Public auth endpoints.
	g.GET("/auth/google/login", h.GoogleLogin)
	g.GET("/auth/google/callback", h.GoogleCallback)
	g.POST("/auth/logout", h.Logout)

	// Everything else requires a valid session.
	a := g.Group("", auth.Middleware(h.cfg.JWTSecret))
	a.GET("/me", h.Me)

	a.GET("/properties", h.ListProperties)
	a.POST("/properties", h.CreateProperty)
	a.GET("/properties/:id", h.GetProperty)
	a.POST("/properties/:id/viewers", h.AddViewer)
	a.GET("/properties/:id/contracts", h.ListContracts)
	a.POST("/properties/:id/contracts", h.CreateContract)
	a.GET("/properties/:id/transactions", h.ListPropertyTransactions)
	a.GET("/properties/:id/audit", h.PropertyAudit)

	a.GET("/tenancies", h.ListTenancies)

	a.GET("/contracts/:id", h.GetContract)
	a.PATCH("/contracts/:id", h.UpdateContract)
	a.POST("/contracts/:id/terms", h.RaiseRent)
	a.POST("/contracts/:id/end", h.EndContract)
	a.GET("/contracts/:id/periods", h.ContractPeriods)
	a.GET("/contracts/:id/transactions", h.ListContractTransactions)
	a.GET("/contracts/:id/audit", h.ContractAudit)

	a.POST("/transactions", h.RecordTransaction)
	a.GET("/transactions/pending", h.ListPending)
	a.POST("/transactions/:id/approve", h.ApproveTransaction)
	a.POST("/transactions/:id/reject", h.RejectTransaction)

	a.POST("/invitations", h.CreateInvitation)
	a.GET("/invitations", h.ListMyInvitations)
	a.POST("/invitations/counter", h.CounterInvitation)
	a.POST("/invitations/accept", h.AcceptInvitation)
	a.POST("/invitations/decline", h.DeclineInvitation)

	a.GET("/analytics", h.Analytics)
}

// paramUUID parses a path parameter as a UUID.
func paramUUID(c echo.Context, name string) (uuid.UUID, error) {
	id, err := uuid.Parse(c.Param(name))
	if err != nil {
		return uuid.Nil, httperr.BadRequest("invalid " + name)
	}
	return id, nil
}
