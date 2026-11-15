package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"

	"wasooli/internal/db/sqlc"
	"wasooli/internal/httperr"
	"wasooli/internal/types"
)

func randomToken() string {
	b := make([]byte, 24)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// buildProposed snapshots a contract's current terms as the starting negotiation proposal.
func buildProposed(c sqlc.Contract, term *sqlc.ContractTerm) types.ProposedTerms {
	p := types.ProposedTerms{
		TenantName: c.TenantName,
		StartDate:  dateStr(c.StartDate),
		Cycle:      c.Cycle,
		Anchor:     c.Anchor,
	}
	if term != nil {
		p.Components = types.ParseComponents(term.Components)
	}
	return p
}

// applyProposed writes agreed terms onto a contract when a link is finalized.
func (s *Service) applyProposed(ctx context.Context, contractID uuid.UUID, p types.ProposedTerms) error {
	c, err := s.q.GetContract(ctx, contractID)
	if err != nil {
		return httperr.Internal(err)
	}
	start := c.StartDate
	if p.StartDate != "" {
		if t, e := time.Parse(types.DateLayout, p.StartDate); e == nil {
			start = t
		}
	}
	cycle := c.Cycle
	if types.ValidCycle(p.Cycle) {
		cycle = p.Cycle
	}
	anchor := c.Anchor
	if p.Anchor != 0 {
		anchor = p.Anchor
	}
	tenantName := c.TenantName
	if p.TenantName != "" {
		tenantName = p.TenantName
	}
	if _, err := s.q.UpdateContractDetails(ctx, sqlc.UpdateContractDetailsParams{
		ID: contractID, TenantName: tenantName, Cycle: cycle, Anchor: anchor, StartDate: start,
	}); err != nil {
		return httperr.Internal(err)
	}
	if len(p.Components) > 0 {
		if _, err := s.q.CreateTerm(ctx, sqlc.CreateTermParams{
			ContractID: contractID, EffectiveFrom: start,
			Components: types.MarshalComponents(p.Components), Total: types.SumComponents(p.Components),
		}); err != nil {
			return httperr.Internal(err)
		}
	}
	return nil
}

// CreateInvitation sends a tenant, owner, or viewer invitation with a terms snapshot.
func (s *Service) CreateInvitation(ctx context.Context, userID uuid.UUID, in types.CreateInvitationInput) (InvitationView, error) {
	switch in.Role {
	case "tenant":
		cid, err := uuid.Parse(in.ContractID)
		if err != nil {
			return InvitationView{}, httperr.BadRequest("invalid contract_id")
		}
		inv, err := s.inviteToContract(ctx, userID, cid, in.ToEmail)
		if err != nil {
			return InvitationView{}, err
		}
		return invitationView(inv), nil

	case "owner":
		cid, err := uuid.Parse(in.ContractID)
		if err != nil {
			return InvitationView{}, httperr.BadRequest("invalid contract_id")
		}
		c, err := s.q.GetContract(ctx, cid)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return InvitationView{}, httperr.NotFound("contract not found")
			}
			return InvitationView{}, httperr.Internal(err)
		}
		a, err := s.propAccess(ctx, c.PropertyID, userID)
		if err != nil {
			return InvitationView{}, err
		}
		if !a.IsOwner || !a.OwnerVacant {
			return InvitationView{}, httperr.Forbidden("only a standalone tenancy can invite an owner")
		}
		terms, _ := s.q.ListTermsForContract(ctx, cid)
		prop := buildProposed(c, currentTerm(terms))
		inv, err := s.q.CreateInvitation(ctx, sqlc.CreateInvitationParams{
			FromUser: userID, ToEmail: in.ToEmail,
			PropertyID: nullUUID(c.PropertyID), ContractID: nullUUID(cid),
			Role: "owner", Token: randomToken(), Proposed: prop.Marshal(), LastActor: nullUUID(userID),
		})
		if err != nil {
			return InvitationView{}, httperr.Internal(err)
		}
		s.audit(ctx, "invitation", inv.ID, userID, "sent", map[string]any{"role": "owner", "to": in.ToEmail})
		return invitationView(inv), nil

	case "viewer":
		pid, err := uuid.Parse(in.PropertyID)
		if err != nil {
			return InvitationView{}, httperr.BadRequest("invalid property_id")
		}
		if err := s.requireRealOwner(ctx, pid, userID); err != nil {
			return InvitationView{}, err
		}
		inv, err := s.q.CreateInvitation(ctx, sqlc.CreateInvitationParams{
			FromUser: userID, ToEmail: in.ToEmail, PropertyID: nullUUID(pid),
			Role: "viewer", Token: randomToken(), Proposed: []byte("{}"), LastActor: nullUUID(userID),
		})
		if err != nil {
			return InvitationView{}, httperr.Internal(err)
		}
		s.audit(ctx, "invitation", inv.ID, userID, "sent", map[string]any{"role": "viewer", "to": in.ToEmail})
		return invitationView(inv), nil

	default:
		return InvitationView{}, httperr.BadRequest("role must be tenant, owner, or viewer")
	}
}

func (s *Service) inviteToContract(ctx context.Context, fromUser, contractID uuid.UUID, email string) (sqlc.Invitation, error) {
	c, err := s.q.GetContract(ctx, contractID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return sqlc.Invitation{}, httperr.NotFound("contract not found")
		}
		return sqlc.Invitation{}, httperr.Internal(err)
	}
	if err := s.requireRealOwner(ctx, c.PropertyID, fromUser); err != nil {
		return sqlc.Invitation{}, err
	}
	terms, _ := s.q.ListTermsForContract(ctx, contractID)
	prop := buildProposed(c, currentTerm(terms))
	inv, err := s.q.CreateInvitation(ctx, sqlc.CreateInvitationParams{
		FromUser: fromUser, ToEmail: email,
		PropertyID: nullUUID(c.PropertyID), ContractID: nullUUID(contractID),
		Role: "tenant", Token: randomToken(), Proposed: prop.Marshal(), LastActor: nullUUID(fromUser),
	})
	if err != nil {
		return sqlc.Invitation{}, httperr.Internal(err)
	}
	s.audit(ctx, "invitation", inv.ID, fromUser, "sent", map[string]any{"role": "tenant", "to": email})
	return inv, nil
}

// invParticipant reports whether the user is the recipient; errors if not a participant.
func (s *Service) invParticipant(ctx context.Context, inv sqlc.Invitation, userID uuid.UUID) (bool, error) {
	u, err := s.q.GetUser(ctx, userID)
	if err != nil {
		return false, httperr.Internal(err)
	}
	isRecipient := strings.EqualFold(u.Email, inv.ToEmail)
	if !isRecipient && inv.FromUser != userID {
		return false, httperr.Forbidden("not a participant in this invitation")
	}
	return isRecipient, nil
}

func invOpen(status string) bool { return status == "pending" || status == "countered" }

// CounterInvitation proposes revised terms. Only the party who is not the last actor may counter.
func (s *Service) CounterInvitation(ctx context.Context, userID uuid.UUID, in types.CounterInvitationInput) (InvitationView, error) {
	inv, err := s.q.GetInvitationByToken(ctx, in.Token)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return InvitationView{}, httperr.NotFound("invitation not found")
		}
		return InvitationView{}, httperr.Internal(err)
	}
	if !invOpen(inv.Status) {
		return InvitationView{}, httperr.Conflict("invitation is " + inv.Status)
	}
	if inv.Role == "viewer" {
		return InvitationView{}, httperr.BadRequest("viewer invitations have no terms to negotiate")
	}
	if _, err := s.invParticipant(ctx, inv, userID); err != nil {
		return InvitationView{}, err
	}
	if inv.LastActor.Valid && inv.LastActor.UUID == userID {
		return InvitationView{}, httperr.Forbidden("waiting for the other party to respond")
	}
	updated, err := s.q.CounterInvitation(ctx, sqlc.CounterInvitationParams{
		ID: inv.ID, Proposed: in.Proposed.Marshal(), LastActor: nullUUID(userID),
	})
	if err != nil {
		return InvitationView{}, httperr.Internal(err)
	}
	s.audit(ctx, "invitation", inv.ID, userID, "countered", map[string]any{})
	return invitationView(updated), nil
}

// AcceptInvitation finalizes a link. See AcceptInvitationInput for the merge options.
func (s *Service) AcceptInvitation(ctx context.Context, userID uuid.UUID, in types.AcceptInvitationInput) (InvitationView, error) {
	inv, err := s.q.GetInvitationByToken(ctx, in.Token)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return InvitationView{}, httperr.NotFound("invitation not found")
		}
		return InvitationView{}, httperr.Internal(err)
	}
	if !invOpen(inv.Status) {
		return InvitationView{}, httperr.Conflict("invitation already " + inv.Status)
	}
	isRecipient, err := s.invParticipant(ctx, inv, userID)
	if err != nil {
		return InvitationView{}, err
	}
	proposed := types.ParseProposed(inv.Proposed)

	switch inv.Role {
	case "viewer":
		if !isRecipient {
			return InvitationView{}, httperr.Forbidden("only the invited user can accept")
		}
		if _, err := s.q.CreateGrant(ctx, sqlc.CreateGrantParams{
			UserID: userID, PropertyID: inv.PropertyID.UUID, Role: "viewer", GrantedBy: inv.FromUser,
		}); err != nil {
			return InvitationView{}, httperr.Internal(err)
		}
		s.audit(ctx, "property", inv.PropertyID.UUID, userID, "viewer_added", map[string]any{})

	case "tenant":
		if inv.LastActor.Valid && inv.LastActor.UUID == userID {
			return InvitationView{}, httperr.Forbidden("waiting for the other party to respond")
		}
		tenantUser, err := s.q.GetUserByEmail(ctx, inv.ToEmail)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return InvitationView{}, httperr.BadRequest("the invited tenant has not signed up yet")
			}
			return InvitationView{}, httperr.Internal(err)
		}
		cid := inv.ContractID.UUID
		if err := s.applyProposed(ctx, cid, proposed); err != nil {
			return InvitationView{}, err
		}
		if _, err := s.q.SetContractTenant(ctx, sqlc.SetContractTenantParams{
			ID: cid, TenantUserID: nullUUID(tenantUser.ID),
		}); err != nil {
			return InvitationView{}, httperr.Internal(err)
		}
		s.audit(ctx, "contract", cid, userID, "tenant_joined", map[string]any{})
		if isRecipient && in.RetireContractID != "" {
			s.archiveStandalone(ctx, userID, in.RetireContractID, cid)
		}

	case "owner":
		if !isRecipient {
			return InvitationView{}, httperr.Forbidden("only the invited owner can accept and link")
		}
		ct, err := s.q.GetContract(ctx, inv.ContractID.UUID) // tenant's standalone contract
		if err != nil {
			return InvitationView{}, httperr.Internal(err)
		}
		if in.Keep == "owner" {
			co, err := s.ownerContractToMerge(ctx, userID, in.MergeContractID)
			if err != nil {
				return InvitationView{}, err
			}
			if err := s.applyProposed(ctx, co.ID, proposed); err != nil {
				return InvitationView{}, err
			}
			if _, err := s.q.SetContractTenant(ctx, sqlc.SetContractTenantParams{
				ID: co.ID, TenantUserID: ct.TenantUserID,
			}); err != nil {
				return InvitationView{}, httperr.Internal(err)
			}
			if _, err := s.q.ArchiveContract(ctx, sqlc.ArchiveContractParams{ID: ct.ID, ArchivedInto: nullUUID(co.ID)}); err != nil {
				return InvitationView{}, httperr.Internal(err)
			}
			s.audit(ctx, "contract", co.ID, userID, "linked", map[string]any{"kept": "owner"})
			s.audit(ctx, "contract", ct.ID, userID, "archived", map[string]any{"into": co.ID.String()})
		} else {
			// keep tenant's record: owner takes over the tenant's property
			if _, err := s.q.ClaimPropertyOwner(ctx, sqlc.ClaimPropertyOwnerParams{
				ID: inv.PropertyID.UUID, OwnerID: userID,
			}); err != nil {
				return InvitationView{}, httperr.Internal(err)
			}
			if err := s.applyProposed(ctx, ct.ID, proposed); err != nil {
				return InvitationView{}, err
			}
			if in.MergeContractID != "" {
				co, err := s.ownerContractToMerge(ctx, userID, in.MergeContractID)
				if err != nil {
					return InvitationView{}, err
				}
				if _, err := s.q.ArchiveContract(ctx, sqlc.ArchiveContractParams{ID: co.ID, ArchivedInto: nullUUID(ct.ID)}); err != nil {
					return InvitationView{}, httperr.Internal(err)
				}
				s.audit(ctx, "contract", co.ID, userID, "archived", map[string]any{"into": ct.ID.String()})
			}
			s.audit(ctx, "property", inv.PropertyID.UUID, userID, "owner_claimed", map[string]any{"kept": "tenant"})
		}
	}

	updated, err := s.q.SetInvitationStatus(ctx, sqlc.SetInvitationStatusParams{ID: inv.ID, Status: "accepted"})
	if err != nil {
		return InvitationView{}, httperr.Internal(err)
	}
	return invitationView(updated), nil
}

// ownerContractToMerge loads a contract the caller owns (a real owner), to fold into a link.
func (s *Service) ownerContractToMerge(ctx context.Context, userID uuid.UUID, idStr string) (sqlc.Contract, error) {
	if idStr == "" {
		return sqlc.Contract{}, httperr.BadRequest("merge_contract_id is required to keep your own records")
	}
	id, err := uuid.Parse(idStr)
	if err != nil {
		return sqlc.Contract{}, httperr.BadRequest("invalid merge_contract_id")
	}
	c, err := s.q.GetContract(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return c, httperr.NotFound("contract to merge not found")
		}
		return c, httperr.Internal(err)
	}
	if err := s.requireRealOwner(ctx, c.PropertyID, userID); err != nil {
		return c, err
	}
	return c, nil
}

// archiveStandalone archives a tenant's own standalone contract when they join an owner's.
func (s *Service) archiveStandalone(ctx context.Context, userID uuid.UUID, idStr string, into uuid.UUID) {
	id, err := uuid.Parse(idStr)
	if err != nil {
		return
	}
	c, err := s.q.GetContract(ctx, id)
	if err != nil || !c.TenantUserID.Valid || c.TenantUserID.UUID != userID {
		return
	}
	if _, err := s.q.ArchiveContract(ctx, sqlc.ArchiveContractParams{ID: id, ArchivedInto: nullUUID(into)}); err != nil {
		s.log.Error("archive standalone", zap.Error(err))
		return
	}
	s.audit(ctx, "contract", id, userID, "archived", map[string]any{"into": into.String()})
}

// DeclineInvitation marks an invite declined (either participant).
func (s *Service) DeclineInvitation(ctx context.Context, userID uuid.UUID, token string) error {
	inv, err := s.q.GetInvitationByToken(ctx, token)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return httperr.NotFound("invitation not found")
		}
		return httperr.Internal(err)
	}
	if _, err := s.invParticipant(ctx, inv, userID); err != nil {
		return err
	}
	if _, err := s.q.SetInvitationStatus(ctx, sqlc.SetInvitationStatusParams{ID: inv.ID, Status: "declined"}); err != nil {
		return httperr.Internal(err)
	}
	return nil
}

// ListMyInvitations returns open invites the user is involved in, with whose-turn flags.
func (s *Service) ListMyInvitations(ctx context.Context, userID uuid.UUID) ([]InvitationView, error) {
	u, err := s.q.GetUser(ctx, userID)
	if err != nil {
		return nil, httperr.Internal(err)
	}
	invs, err := s.q.ListInvitationsForUser(ctx, sqlc.ListInvitationsForUserParams{ToEmail: u.Email, FromUser: userID})
	if err != nil {
		return nil, httperr.Internal(err)
	}
	out := make([]InvitationView, 0, len(invs))
	for _, i := range invs {
		v := invitationView(i)
		v.Mine = i.FromUser == userID
		v.YourTurn = !(i.LastActor.Valid && i.LastActor.UUID == userID)
		out = append(out, v)
	}
	return out, nil
}
