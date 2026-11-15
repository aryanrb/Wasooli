// Package services owns business logic, DB access, and error mapping. Handlers stay
// thin. Unlike the per-entity template, there is one Service: authorization and audit
// logging are cross-cutting and shared here.
package services

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"

	"wasooli/internal/db/sqlc"
	"wasooli/internal/httperr"
)

// Service holds the generated queries and a logger.
type Service struct {
	q   *sqlc.Queries
	log *zap.Logger
}

func New(q *sqlc.Queries, log *zap.Logger) *Service {
	return &Service{q: q, log: log}
}

// --- authorization (replaces the RLS a hosted DB would enforce) ---
//
// Access model:
//   - Property-wide (list all contracts, property transactions, add contract/viewer):
//     the REAL owner (owner seat held, owner_vacant=false) or a viewer.
//   - Owner-side of a contract (edit terms, solo recording): whoever holds the owner
//     seat — a real owner OR a tenant's standalone placeholder (owner_vacant=true).
//   - Contract-level view: owner-side, viewer, or the tenant of that contract.
//   A tenant is scoped to their own contract and never sees the property's other contracts.

func (s *Service) propAccess(ctx context.Context, propID, userID uuid.UUID) (sqlc.GetPropertyAccessRow, error) {
	row, err := s.q.GetPropertyAccess(ctx, sqlc.GetPropertyAccessParams{ID: propID, OwnerID: userID})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return row, httperr.NotFound("property not found")
		}
		s.log.Error("access check", zap.Error(err))
		return row, httperr.Internal(err)
	}
	return row, nil
}

func realOwner(a sqlc.GetPropertyAccessRow) bool { return a.IsOwner && !a.OwnerVacant }

// requirePropertyWide: real owner or viewer. Tenants are excluded (they see only their contract).
func (s *Service) requirePropertyWide(ctx context.Context, propID, userID uuid.UUID) error {
	a, err := s.propAccess(ctx, propID, userID)
	if err != nil {
		return err
	}
	if realOwner(a) || a.IsViewer {
		return nil
	}
	return httperr.Forbidden("no access to this property")
}

// requireRealOwner: only a real property owner (manage contracts, viewers).
func (s *Service) requireRealOwner(ctx context.Context, propID, userID uuid.UUID) error {
	a, err := s.propAccess(ctx, propID, userID)
	if err != nil {
		return err
	}
	if realOwner(a) {
		return nil
	}
	return httperr.Forbidden("only the property owner can do this")
}

// requireOwnerSide: holds the owner seat — real owner or tenant placeholder (edit own terms).
func (s *Service) requireOwnerSide(ctx context.Context, propID, userID uuid.UUID) error {
	a, err := s.propAccess(ctx, propID, userID)
	if err != nil {
		return err
	}
	if a.IsOwner {
		return nil
	}
	return httperr.Forbidden("only the owner side can do this")
}

// getContractWithAccess loads a contract and allows owner-side, viewer, or its tenant.
func (s *Service) getContractWithAccess(ctx context.Context, userID, contractID uuid.UUID) (sqlc.Contract, error) {
	c, err := s.q.GetContract(ctx, contractID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return c, httperr.NotFound("contract not found")
		}
		return c, httperr.Internal(err)
	}
	a, err := s.propAccess(ctx, c.PropertyID, userID)
	if err != nil {
		return c, err
	}
	isTenant := c.TenantUserID.Valid && c.TenantUserID.UUID == userID
	if a.IsOwner || a.IsViewer || isTenant {
		return c, nil
	}
	return c, httperr.Forbidden("no access to this contract")
}

// audit writes a log entry; failures are logged, never block the main action.
func (s *Service) audit(ctx context.Context, entityType string, entityID, actor uuid.UUID, action string, details map[string]any) {
	b, err := json.Marshal(details)
	if err != nil || b == nil {
		b = []byte("{}")
	}
	if _, err := s.q.CreateAudit(ctx, sqlc.CreateAuditParams{
		EntityType: entityType, EntityID: entityID, Action: action, Actor: actor, Details: b,
	}); err != nil {
		s.log.Error("audit write", zap.Error(err))
	}
}

// --- small converters between sqlc nullable types and plain Go ---

func nullUUID(id uuid.UUID) uuid.NullUUID { return uuid.NullUUID{UUID: id, Valid: true} }
func textOrNull(s string) pgtype.Text    { return pgtype.Text{String: s, Valid: s != ""} }
func dateStr(t time.Time) string         { return t.Format("2006-01-02") }

func uuidPtr(n uuid.NullUUID) *uuid.UUID {
	if n.Valid {
		id := n.UUID
		return &id
	}
	return nil
}

func rawJSON(b []byte) json.RawMessage {
	if len(b) == 0 {
		return json.RawMessage("[]")
	}
	return json.RawMessage(b)
}
