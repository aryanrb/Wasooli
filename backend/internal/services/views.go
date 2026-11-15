package services

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"wasooli/internal/db/sqlc"
)

// View structs are the JSON shapes returned to the SPA. They render jsonb (stored as
// []byte, which would otherwise base64-encode) as raw JSON and flatten sqlc nullables.

type UserView struct {
	ID        uuid.UUID `json:"id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	AvatarURL string    `json:"avatar_url"`
}

func userView(u sqlc.User) UserView {
	return UserView{ID: u.ID, Email: u.Email, Name: u.Name, AvatarURL: u.AvatarUrl}
}

type ContractView struct {
	ID                uuid.UUID       `json:"id"`
	PropertyID        uuid.UUID       `json:"property_id"`
	TenantUserID      *uuid.UUID      `json:"tenant_user_id"`
	TenantName        string          `json:"tenant_name"`
	Linked            bool            `json:"linked"` // a real owner AND a tenant are both present
	StartDate         string          `json:"start_date"`
	Cycle             string          `json:"cycle"`
	Anchor            int32           `json:"anchor"`
	Status            string          `json:"status"`
	CreatedAt         time.Time       `json:"created_at"`
	CurrentComponents json.RawMessage `json:"current_components"`
	CurrentTotal      int64           `json:"current_total"`
}

func contractView(c sqlc.Contract, currentTerm *sqlc.ContractTerm) ContractView {
	v := ContractView{
		ID:           c.ID,
		PropertyID:   c.PropertyID,
		TenantUserID: uuidPtr(c.TenantUserID),
		TenantName:   c.TenantName,
		Linked:       c.TenantUserID.Valid,
		StartDate:    dateStr(c.StartDate),
		Cycle:        c.Cycle,
		Anchor:       c.Anchor,
		Status:       c.Status,
		CreatedAt:    c.CreatedAt,
		CurrentComponents: json.RawMessage("[]"),
	}
	if currentTerm != nil {
		v.CurrentComponents = rawJSON(currentTerm.Components)
		v.CurrentTotal = currentTerm.Total
	}
	return v
}

// TenancyView is a tenant's own contract plus the property context they're allowed to see
// (name + currency only — not the property's other contracts). owner_vacant=true means
// it's still a standalone record with no linked owner.
type TenancyView struct {
	ContractView
	PropertyName string `json:"property_name"`
	Currency     string `json:"currency"`
	OwnerVacant  bool   `json:"owner_vacant"`
}

func tenancyView(r sqlc.ListTenanciesForUserRow, currentTerm *sqlc.ContractTerm) TenancyView {
	cv := ContractView{
		ID:                r.ID,
		PropertyID:        r.PropertyID,
		TenantUserID:      uuidPtr(r.TenantUserID),
		TenantName:        r.TenantName,
		Linked:            !r.OwnerVacant && r.TenantUserID.Valid,
		StartDate:         dateStr(r.StartDate),
		Cycle:             r.Cycle,
		Anchor:            r.Anchor,
		Status:            r.Status,
		CreatedAt:         r.CreatedAt,
		CurrentComponents: json.RawMessage("[]"),
	}
	if currentTerm != nil {
		cv.CurrentComponents = rawJSON(currentTerm.Components)
		cv.CurrentTotal = currentTerm.Total
	}
	return TenancyView{ContractView: cv, PropertyName: r.PropertyName, Currency: r.Currency, OwnerVacant: r.OwnerVacant}
}

type TermView struct {
	ID            uuid.UUID       `json:"id"`
	EffectiveFrom string          `json:"effective_from"`
	Components    json.RawMessage `json:"components"`
	Total         int64           `json:"total"`
}

func termView(t sqlc.ContractTerm) TermView {
	return TermView{ID: t.ID, EffectiveFrom: dateStr(t.EffectiveFrom), Components: rawJSON(t.Components), Total: t.Total}
}

type TransactionView struct {
	ID             uuid.UUID       `json:"id"`
	PropertyID     uuid.UUID       `json:"property_id"`
	ContractID     *uuid.UUID      `json:"contract_id"`
	PeriodKey      string          `json:"period_key"`
	RecordedBy     uuid.UUID       `json:"recorded_by"`
	Direction      string          `json:"direction"`
	Amount         int64           `json:"amount"`
	Components     json.RawMessage `json:"components"`
	Note           string          `json:"note"`
	PaidAt         string          `json:"paid_at"`
	ApprovalStatus string          `json:"approval_status"`
	CreatedAt      time.Time       `json:"created_at"`
}

func txView(t sqlc.Transaction) TransactionView {
	return TransactionView{
		ID:             t.ID,
		PropertyID:     t.PropertyID,
		ContractID:     uuidPtr(t.ContractID),
		PeriodKey:      t.PeriodKey.String,
		RecordedBy:     t.RecordedBy,
		Direction:      t.Direction,
		Amount:         t.Amount,
		Components:     rawJSON(t.Components),
		Note:           t.Note,
		PaidAt:         dateStr(t.PaidAt),
		ApprovalStatus: t.ApprovalStatus,
		CreatedAt:      t.CreatedAt,
	}
}

func txViews(ts []sqlc.Transaction) []TransactionView {
	out := make([]TransactionView, 0, len(ts))
	for _, t := range ts {
		out = append(out, txView(t))
	}
	return out
}

type InvitationView struct {
	ID         uuid.UUID       `json:"id"`
	FromUser   uuid.UUID       `json:"from_user"`
	ToEmail    string          `json:"to_email"`
	PropertyID *uuid.UUID      `json:"property_id"`
	ContractID *uuid.UUID      `json:"contract_id"`
	Role       string          `json:"role"`
	Token      string          `json:"token"`
	Status     string          `json:"status"`
	Proposed   json.RawMessage `json:"proposed"`
	Mine       bool            `json:"mine"`      // true if the caller sent it
	YourTurn   bool            `json:"your_turn"` // the caller may accept/counter now
	CreatedAt  time.Time       `json:"created_at"`
}

func invitationView(i sqlc.Invitation) InvitationView {
	return InvitationView{
		ID: i.ID, FromUser: i.FromUser, ToEmail: i.ToEmail,
		PropertyID: uuidPtr(i.PropertyID), ContractID: uuidPtr(i.ContractID),
		Role: i.Role, Token: i.Token, Status: i.Status,
		Proposed:  rawJSONObj(i.Proposed),
		CreatedAt: i.CreatedAt,
	}
}

func rawJSONObj(b []byte) json.RawMessage {
	if len(b) == 0 {
		return json.RawMessage("{}")
	}
	return json.RawMessage(b)
}
