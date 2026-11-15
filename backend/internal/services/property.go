package services

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"

	"wasooli/internal/db/sqlc"
	"wasooli/internal/httperr"
	"wasooli/internal/types"
)

// CreatePropertyResult returns the property and, when created as a tenant, the
// standalone tenancy contract it generated.
type CreatePropertyResult struct {
	Property sqlc.Property `json:"property"`
	Contract *ContractView `json:"contract,omitempty"`
}

func (s *Service) CreateProperty(ctx context.Context, userID uuid.UUID, in types.CreatePropertyInput) (CreatePropertyResult, error) {
	var res CreatePropertyResult
	currency := in.Currency
	if currency == "" {
		currency = "NPR"
	}
	p, err := s.q.CreateProperty(ctx, sqlc.CreatePropertyParams{
		OwnerID: userID, Name: in.Name, Address: in.Address, Currency: currency,
		OwnerVacant: in.Role == "tenant",
	})
	if err != nil {
		s.log.Error("create property", zap.Error(err))
		return res, httperr.Internal(err)
	}
	s.audit(ctx, "property", p.ID, userID, "created", map[string]any{"name": p.Name, "role": in.Role})
	res.Property = p

	// A tenant's standalone record is a property + one contract where they are the tenant.
	if in.Role == "tenant" {
		start, _ := time.Parse(types.DateLayout, in.StartDate)
		anchor := in.Anchor
		if anchor == 0 {
			anchor = int32(start.Day())
		}
		c, err := s.q.CreateContract(ctx, sqlc.CreateContractParams{
			PropertyID: p.ID, TenantUserID: nullUUID(userID), TenantName: in.TenantName,
			StartDate: start, Cycle: in.Cycle, Anchor: anchor,
		})
		if err != nil {
			return res, httperr.Internal(err)
		}
		term, err := s.q.CreateTerm(ctx, sqlc.CreateTermParams{
			ContractID: c.ID, EffectiveFrom: start,
			Components: types.MarshalComponents(in.Components), Total: types.SumComponents(in.Components),
		})
		if err != nil {
			return res, httperr.Internal(err)
		}
		s.audit(ctx, "contract", c.ID, userID, "created", map[string]any{"standalone": true, "total": term.Total})
		cv := contractView(c, &term)
		res.Contract = &cv
	}
	return res, nil
}

// ListProperties returns properties the user owns or can view (not tenancies).
func (s *Service) ListProperties(ctx context.Context, userID uuid.UUID) ([]sqlc.Property, error) {
	ps, err := s.q.ListPropertiesForUser(ctx, userID)
	if err != nil {
		s.log.Error("list properties", zap.Error(err))
		return nil, httperr.Internal(err)
	}
	return ps, nil
}

// ListTenancies returns the user's own contracts (tenant side) — their entry point as a tenant.
func (s *Service) ListTenancies(ctx context.Context, userID uuid.UUID) ([]TenancyView, error) {
	rows, err := s.q.ListTenanciesForUser(ctx, nullUUID(userID))
	if err != nil {
		return nil, httperr.Internal(err)
	}
	out := make([]TenancyView, 0, len(rows))
	for _, r := range rows {
		terms, _ := s.q.ListTermsForContract(ctx, r.ID)
		out = append(out, tenancyView(r, currentTerm(terms)))
	}
	return out, nil
}

func (s *Service) GetProperty(ctx context.Context, userID, propID uuid.UUID) (sqlc.Property, error) {
	if err := s.requirePropertyWide(ctx, propID, userID); err != nil {
		return sqlc.Property{}, err
	}
	p, err := s.q.GetProperty(ctx, propID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return p, httperr.NotFound("property not found")
		}
		return p, httperr.Internal(err)
	}
	return p, nil
}

// AddViewer grants read-only access to an already-registered user.
func (s *Service) AddViewer(ctx context.Context, ownerID, propID uuid.UUID, email string) error {
	if err := s.requireRealOwner(ctx, propID, ownerID); err != nil {
		return err
	}
	u, err := s.q.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return httperr.NotFound("no user with that email; send an invitation instead")
		}
		return httperr.Internal(err)
	}
	if _, err := s.q.CreateGrant(ctx, sqlc.CreateGrantParams{
		UserID: u.ID, PropertyID: propID, Role: "viewer", GrantedBy: ownerID,
	}); err != nil {
		return httperr.Internal(err)
	}
	s.audit(ctx, "property", propID, ownerID, "viewer_added", map[string]any{"email": email})
	return nil
}
