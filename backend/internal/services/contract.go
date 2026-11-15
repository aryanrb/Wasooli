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

// CreateContractResult bundles the new contract and, if a tenant email was given,
// the auto-created invitation.
type CreateContractResult struct {
	Contract ContractView    `json:"contract"`
	Invite   *InvitationView `json:"invite,omitempty"`
}

func (s *Service) CreateContract(ctx context.Context, userID, propID uuid.UUID, in types.CreateContractInput) (CreateContractResult, error) {
	var res CreateContractResult
	if err := s.requireRealOwner(ctx, propID, userID); err != nil {
		return res, err
	}
	start, _ := time.Parse(types.DateLayout, in.StartDate)
	anchor := in.Anchor
	if anchor == 0 {
		anchor = int32(start.Day())
	}
	c, err := s.q.CreateContract(ctx, sqlc.CreateContractParams{
		PropertyID: propID, TenantName: in.TenantName,
		StartDate: start, Cycle: in.Cycle, Anchor: anchor,
	})
	if err != nil {
		s.log.Error("create contract", zap.Error(err))
		return res, httperr.Internal(err)
	}
	term, err := s.q.CreateTerm(ctx, sqlc.CreateTermParams{
		ContractID: c.ID, EffectiveFrom: start,
		Components: types.MarshalComponents(in.Components), Total: types.SumComponents(in.Components),
	})
	if err != nil {
		return res, httperr.Internal(err)
	}
	s.audit(ctx, "contract", c.ID, userID, "created", map[string]any{
		"tenant_name": c.TenantName, "cycle": c.Cycle, "total": term.Total,
	})
	res.Contract = contractView(c, &term)

	if in.TenantEmail != "" {
		inv, err := s.inviteToContract(ctx, userID, c.ID, in.TenantEmail)
		if err != nil {
			return res, err
		}
		iv := invitationView(inv)
		res.Invite = &iv
	}
	return res, nil
}

// ListContracts returns all contracts of a property — owners/viewers only (a tenant is
// scoped to their own contract and uses ListTenancies instead).
func (s *Service) ListContracts(ctx context.Context, userID, propID uuid.UUID) ([]ContractView, error) {
	if err := s.requirePropertyWide(ctx, propID, userID); err != nil {
		return nil, err
	}
	cs, err := s.q.ListContractsForProperty(ctx, propID)
	if err != nil {
		return nil, httperr.Internal(err)
	}
	out := make([]ContractView, 0, len(cs))
	for _, c := range cs {
		terms, _ := s.q.ListTermsForContract(ctx, c.ID)
		out = append(out, contractView(c, currentTerm(terms)))
	}
	return out, nil
}

func (s *Service) GetContract(ctx context.Context, userID, contractID uuid.UUID) (ContractView, error) {
	c, err := s.getContractWithAccess(ctx, userID, contractID)
	if err != nil {
		return ContractView{}, err
	}
	terms, _ := s.q.ListTermsForContract(ctx, c.ID)
	return contractView(c, currentTerm(terms)), nil
}

// UpdateContract edits contract basics (e.g. name a tenant later). Owner-side only.
func (s *Service) UpdateContract(ctx context.Context, userID, contractID uuid.UUID, in types.UpdateContractInput) (ContractView, error) {
	c, err := s.q.GetContract(ctx, contractID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ContractView{}, httperr.NotFound("contract not found")
		}
		return ContractView{}, httperr.Internal(err)
	}
	if err := s.requireOwnerSide(ctx, c.PropertyID, userID); err != nil {
		return ContractView{}, err
	}
	tenantName := c.TenantName
	if in.TenantName != "" {
		tenantName = in.TenantName
	}
	cycle := c.Cycle
	if in.Cycle != "" {
		cycle = in.Cycle
	}
	anchor := c.Anchor
	if in.Anchor != 0 {
		anchor = in.Anchor
	}
	start := c.StartDate
	if in.StartDate != "" {
		start, _ = time.Parse(types.DateLayout, in.StartDate)
	}
	updated, err := s.q.UpdateContractDetails(ctx, sqlc.UpdateContractDetailsParams{
		ID: contractID, TenantName: tenantName, Cycle: cycle, Anchor: anchor, StartDate: start,
	})
	if err != nil {
		return ContractView{}, httperr.Internal(err)
	}
	s.audit(ctx, "contract", contractID, userID, "updated", map[string]any{"tenant_name": tenantName})
	terms, _ := s.q.ListTermsForContract(ctx, contractID)
	return contractView(updated, currentTerm(terms)), nil
}

// RaiseRent records a new term (rent change). Future periods use it; past unchanged.
func (s *Service) RaiseRent(ctx context.Context, userID, contractID uuid.UUID, in types.RaiseRentInput) (TermView, error) {
	c, err := s.q.GetContract(ctx, contractID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return TermView{}, httperr.NotFound("contract not found")
		}
		return TermView{}, httperr.Internal(err)
	}
	if err := s.requireOwnerSide(ctx, c.PropertyID, userID); err != nil {
		return TermView{}, err
	}
	eff, _ := time.Parse(types.DateLayout, in.EffectiveFrom)
	term, err := s.q.CreateTerm(ctx, sqlc.CreateTermParams{
		ContractID: contractID, EffectiveFrom: eff,
		Components: types.MarshalComponents(in.Components), Total: types.SumComponents(in.Components),
	})
	if err != nil {
		return TermView{}, httperr.Internal(err)
	}
	s.audit(ctx, "contract", contractID, userID, "rent_changed", map[string]any{
		"effective_from": in.EffectiveFrom, "total": term.Total,
	})
	return termView(term), nil
}

// EndContract archives a tenancy. History stays viewable; the property can take a new contract.
func (s *Service) EndContract(ctx context.Context, userID, contractID uuid.UUID) (ContractView, error) {
	c, err := s.q.GetContract(ctx, contractID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ContractView{}, httperr.NotFound("contract not found")
		}
		return ContractView{}, httperr.Internal(err)
	}
	if err := s.requireOwnerSide(ctx, c.PropertyID, userID); err != nil {
		return ContractView{}, err
	}
	ended, err := s.q.EndContract(ctx, contractID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ContractView{}, httperr.Conflict("contract is already ended")
		}
		return ContractView{}, httperr.Internal(err)
	}
	s.audit(ctx, "contract", contractID, userID, "ended", map[string]any{})
	terms, _ := s.q.ListTermsForContract(ctx, contractID)
	return contractView(ended, currentTerm(terms)), nil
}

// PeriodStatus is one derived billing period with its due/paid/outstanding figures.
type PeriodStatus struct {
	Period      types.Period `json:"period"`
	Due         int64        `json:"due"`
	Paid        int64        `json:"paid"`
	Outstanding int64        `json:"outstanding"`
	Status      string       `json:"status"` // due | partial | paid
}

// ContractPeriods is the rent-due view: each period's due vs approved payments. Most-recent first.
func (s *Service) ContractPeriods(ctx context.Context, userID, contractID uuid.UUID) ([]PeriodStatus, error) {
	c, err := s.getContractWithAccess(ctx, userID, contractID)
	if err != nil {
		return nil, err
	}
	terms, err := s.q.ListTermsForContract(ctx, contractID)
	if err != nil {
		return nil, httperr.Internal(err)
	}
	until := time.Now()
	if c.Status != "active" && c.EndedAt.Valid {
		until = c.EndedAt.Time
	}
	periods := types.GeneratePeriods(c.Cycle, c.StartDate, until)
	out := make([]PeriodStatus, 0, len(periods))
	for _, p := range periods {
		due := termTotalAt(terms, p.Start)
		paid := s.sumApprovedForPeriod(ctx, contractID, p.Key)
		out = append(out, PeriodStatus{
			Period: p, Due: due, Paid: paid,
			Outstanding: max64(due-paid, 0),
			Status:      periodStatus(due, paid),
		})
	}
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out, nil
}

func (s *Service) sumApprovedForPeriod(ctx context.Context, contractID uuid.UUID, key string) int64 {
	txs, err := s.q.ListApprovedIncomeForPeriod(ctx, sqlc.ListApprovedIncomeForPeriodParams{
		ContractID: nullUUID(contractID), PeriodKey: textOrNull(key),
	})
	if err != nil {
		s.log.Error("sum period", zap.Error(err))
		return 0
	}
	var total int64
	for _, t := range txs {
		total += t.Amount
	}
	return total
}

// currentTerm returns the term in effect today (latest effective_from <= now), or nil.
func currentTerm(terms []sqlc.ContractTerm) *sqlc.ContractTerm {
	var cur *sqlc.ContractTerm
	now := time.Now()
	for i := range terms {
		if !terms[i].EffectiveFrom.After(now) {
			cur = &terms[i]
		}
	}
	if cur == nil && len(terms) > 0 {
		cur = &terms[0]
	}
	return cur
}

// termTotalAt returns the due total for a date (term in effect at that date).
func termTotalAt(terms []sqlc.ContractTerm, d time.Time) int64 {
	var total int64
	for _, t := range terms {
		if !t.EffectiveFrom.After(d) {
			total = t.Total
		}
	}
	return total
}

func periodStatus(due, paid int64) string {
	switch {
	case due > 0 && paid >= due:
		return "paid"
	case paid > 0:
		return "partial"
	default:
		return "due"
	}
}

func max64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}
