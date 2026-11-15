package services

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"wasooli/internal/db/sqlc"
	"wasooli/internal/httperr"
)

type AuditView struct {
	ID        uuid.UUID       `json:"id"`
	Action    string          `json:"action"`
	Actor     uuid.UUID       `json:"actor"`
	ActorName string          `json:"actor_name"`
	Details   json.RawMessage `json:"details"`
	CreatedAt time.Time       `json:"created_at"`
}

// ListAudit returns the timeline for a property or contract the user can access.
func (s *Service) ListAudit(ctx context.Context, userID uuid.UUID, entityType string, entityID uuid.UUID) ([]AuditView, error) {
	switch entityType {
	case "property":
		if err := s.requirePropertyWide(ctx, entityID, userID); err != nil {
			return nil, err
		}
	case "contract":
		if _, err := s.getContractWithAccess(ctx, userID, entityID); err != nil {
			return nil, err
		}
	default:
		return nil, httperr.BadRequest("entity_type must be property or contract")
	}
	rows, err := s.q.ListAuditForEntity(ctx, sqlc.ListAuditForEntityParams{EntityType: entityType, EntityID: entityID})
	if err != nil {
		return nil, httperr.Internal(err)
	}
	out := make([]AuditView, 0, len(rows))
	for _, r := range rows {
		out = append(out, AuditView{
			ID: r.ID, Action: r.Action, Actor: r.Actor, ActorName: r.ActorName,
			Details: rawJSON(r.Details), CreatedAt: r.CreatedAt,
		})
	}
	return out, nil
}
