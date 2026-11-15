package types

import (
	"errors"
	"strings"
	"time"
)

// CreatePropertyInput is the body for POST /properties. The creator declares whether
// they are the owner (a managed property) or the tenant (a standalone tenancy record).
type CreatePropertyInput struct {
	Role     string `json:"role"` // owner | tenant
	Name     string `json:"name"`
	Address  string `json:"address"`
	Currency string `json:"currency"`
	// tenant-only: the tenancy's contract terms (the tenant records what they pay).
	TenantName string      `json:"tenant_name"`
	StartDate  string      `json:"start_date"` // YYYY-MM-DD
	Cycle      string      `json:"cycle"`
	Anchor     int32       `json:"anchor"`
	Components  []Component `json:"components"`
}

func (in CreatePropertyInput) Validate() error {
	if in.Role != "owner" && in.Role != "tenant" {
		return errors.New("role must be owner or tenant")
	}
	if strings.TrimSpace(in.Name) == "" {
		return errors.New("name is required")
	}
	if in.Role == "tenant" {
		if !ValidCycle(in.Cycle) {
			return errors.New("cycle must be weekly, monthly, or yearly")
		}
		if _, err := time.Parse(DateLayout, in.StartDate); err != nil {
			return errors.New("start_date must be YYYY-MM-DD")
		}
		if len(in.Components) == 0 {
			return errors.New("at least one rent component is required")
		}
	}
	return nil
}

// CreateContractInput is the body for POST /properties/:id/contracts (owner adds a tenancy).
type CreateContractInput struct {
	TenantName  string      `json:"tenant_name"`
	TenantEmail string      `json:"tenant_email"` // optional: auto-send a tenant invite
	StartDate   string      `json:"start_date"`
	Cycle       string      `json:"cycle"`
	Anchor      int32       `json:"anchor"`
	Components   []Component `json:"components"`
}

func (in CreateContractInput) Validate() error {
	if !ValidCycle(in.Cycle) {
		return errors.New("cycle must be weekly, monthly, or yearly")
	}
	if _, err := time.Parse(DateLayout, in.StartDate); err != nil {
		return errors.New("start_date must be YYYY-MM-DD")
	}
	if len(in.Components) == 0 {
		return errors.New("at least one rent component is required")
	}
	return nil
}

// UpdateContractInput edits contract basics (e.g. set a tenant's display name). Empty
// fields are left unchanged. Rent changes go through RaiseRent, not here.
type UpdateContractInput struct {
	TenantName string `json:"tenant_name"`
	Cycle      string `json:"cycle"`
	Anchor     int32  `json:"anchor"`
	StartDate  string `json:"start_date"`
}

func (in UpdateContractInput) Validate() error {
	if in.Cycle != "" && !ValidCycle(in.Cycle) {
		return errors.New("cycle must be weekly, monthly, or yearly")
	}
	if in.StartDate != "" {
		if _, err := time.Parse(DateLayout, in.StartDate); err != nil {
			return errors.New("start_date must be YYYY-MM-DD")
		}
	}
	return nil
}

// RaiseRentInput is the body for POST /contracts/:id/terms (a rent change).
type RaiseRentInput struct {
	EffectiveFrom string      `json:"effective_from"`
	Components     []Component `json:"components"`
}

func (in RaiseRentInput) Validate() error {
	if _, err := time.Parse(DateLayout, in.EffectiveFrom); err != nil {
		return errors.New("effective_from must be YYYY-MM-DD")
	}
	if len(in.Components) == 0 {
		return errors.New("at least one component is required")
	}
	return nil
}

// RecordTransactionInput is the body for POST /transactions.
type RecordTransactionInput struct {
	PropertyID string      `json:"property_id"`
	ContractID string      `json:"contract_id"`
	PeriodKey  string      `json:"period_key"`
	Direction  string      `json:"direction"` // income | expense
	Amount     int64       `json:"amount"`
	Components   []Component `json:"components"`
	Note       string      `json:"note"`
	PaidAt     string      `json:"paid_at"`
}

func (in RecordTransactionInput) Validate() error {
	if in.Direction != "income" && in.Direction != "expense" {
		return errors.New("direction must be income or expense")
	}
	if in.Amount <= 0 {
		return errors.New("amount must be greater than 0")
	}
	if _, err := time.Parse(DateLayout, in.PaidAt); err != nil {
		return errors.New("paid_at must be YYYY-MM-DD")
	}
	return nil
}

// CreateInvitationInput sends a tenant, owner, or viewer invitation.
//   - tenant: owner invites a tenant to a contract (contract_id required).
//   - owner:  a standalone tenant invites the real owner to their tenancy (contract_id required).
//   - viewer: owner grants read-only access to a property (property_id required).
type CreateInvitationInput struct {
	ToEmail    string `json:"to_email"`
	Role       string `json:"role"`
	PropertyID string `json:"property_id"`
	ContractID string `json:"contract_id"`
}

func (in CreateInvitationInput) Validate() error {
	if strings.TrimSpace(in.ToEmail) == "" {
		return errors.New("to_email is required")
	}
	switch in.Role {
	case "tenant", "owner":
		if in.ContractID == "" {
			return errors.New("contract_id is required for a " + in.Role + " invite")
		}
	case "viewer":
		if in.PropertyID == "" {
			return errors.New("property_id is required for a viewer invite")
		}
	default:
		return errors.New("role must be tenant, owner, or viewer")
	}
	return nil
}

// CounterInvitationInput proposes revised terms in a negotiation.
type CounterInvitationInput struct {
	Token    string        `json:"token"`
	Proposed ProposedTerms `json:"proposed"`
}

func (in CounterInvitationInput) Validate() error {
	if in.Token == "" {
		return errors.New("token is required")
	}
	if !ValidCycle(in.Proposed.Cycle) {
		return errors.New("cycle must be weekly, monthly, or yearly")
	}
	if _, err := time.Parse(DateLayout, in.Proposed.StartDate); err != nil {
		return errors.New("start_date must be YYYY-MM-DD")
	}
	if len(in.Proposed.Components) == 0 {
		return errors.New("at least one component is required")
	}
	return nil
}

// AcceptInvitationInput finalizes a link. Keep/MergeContractID apply to an owner
// accepting a tenant's record (#4); RetireContractID lets a tenant archive their own
// standalone when joining an owner's contract.
type AcceptInvitationInput struct {
	Token            string `json:"token"`
	Keep             string `json:"keep"`               // "owner" | "tenant" (owner-invite only)
	MergeContractID  string `json:"merge_contract_id"`  // the accepter's own standalone to fold in
	RetireContractID string `json:"retire_contract_id"` // a tenant's standalone to archive on join
}
