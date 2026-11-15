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

// RecordTransaction writes a ledger entry. Approval rules:
//   - income on a LINKED contract (real owner + tenant both present): either party may
//     record; the entry is PENDING until the other approves.
//   - income on a solo/standalone contract, or any expense: the owner-side records it
//     auto-approved. A standalone tenant IS the owner-side of their own record, so they
//     record freely; on a linked contract they are NOT, so their entries need approval.
func (s *Service) RecordTransaction(ctx context.Context, userID uuid.UUID, in types.RecordTransactionInput) (TransactionView, error) {
	propID, err := uuid.Parse(in.PropertyID)
	if err != nil {
		return TransactionView{}, httperr.BadRequest("invalid property_id")
	}
	a, err := s.propAccess(ctx, propID, userID)
	if err != nil {
		return TransactionView{}, err
	}

	paidAt, _ := time.Parse(types.DateLayout, in.PaidAt)
	params := sqlc.CreateTransactionParams{
		PropertyID: propID,
		RecordedBy: userID,
		Direction:  in.Direction,
		Amount:     in.Amount,
		Components: types.MarshalComponents(in.Components),
		Note:       in.Note,
		PaidAt:     paidAt,
		PeriodKey:  textOrNull(in.PeriodKey),
	}

	if in.ContractID != "" {
		cid, err := uuid.Parse(in.ContractID)
		if err != nil {
			return TransactionView{}, httperr.BadRequest("invalid contract_id")
		}
		c, err := s.q.GetContract(ctx, cid)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return TransactionView{}, httperr.NotFound("contract not found")
			}
			return TransactionView{}, httperr.Internal(err)
		}
		if c.PropertyID != propID {
			return TransactionView{}, httperr.BadRequest("contract does not belong to that property")
		}
		params.ContractID = nullUUID(cid)
		isTenant := c.TenantUserID.Valid && c.TenantUserID.UUID == userID
		linked := !a.OwnerVacant && c.TenantUserID.Valid

		if in.Direction == "income" && linked {
			if !(a.IsOwner || isTenant) {
				return TransactionView{}, httperr.Forbidden("only the owner or tenant can record on this contract")
			}
			params.ApprovalStatus = "pending"
		} else {
			// Solo income or any expense: the owner-side records it, auto-approved.
			if !a.IsOwner {
				return TransactionView{}, httperr.Forbidden("only the owner side can record this entry")
			}
			params.ApprovalStatus = "approved"
			params.ApprovedBy = nullUUID(userID)
		}
	} else {
		// Property-level entry (typically an expense): real owner only.
		if !realOwner(a) {
			return TransactionView{}, httperr.Forbidden("only the owner can record property-level entries")
		}
		params.ApprovalStatus = "approved"
		params.ApprovedBy = nullUUID(userID)
	}

	tx, err := s.q.CreateTransaction(ctx, params)
	if err != nil {
		s.log.Error("create transaction", zap.Error(err))
		return TransactionView{}, httperr.Internal(err)
	}
	s.audit(ctx, "transaction", tx.ID, userID, "recorded", map[string]any{
		"direction": tx.Direction, "amount": tx.Amount, "approval": tx.ApprovalStatus,
	})
	return txView(tx), nil
}

// SetApproval approves or rejects a pending transaction. Only the counterparty
// (the owner or tenant who did NOT record it) may decide.
func (s *Service) SetApproval(ctx context.Context, userID, txID uuid.UUID, approve bool) (TransactionView, error) {
	tx, err := s.q.GetTransaction(ctx, txID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return TransactionView{}, httperr.NotFound("transaction not found")
		}
		return TransactionView{}, httperr.Internal(err)
	}
	if tx.ApprovalStatus != "pending" {
		return TransactionView{}, httperr.Conflict("transaction is not pending")
	}
	if tx.RecordedBy == userID {
		return TransactionView{}, httperr.Forbidden("you cannot approve your own entry")
	}
	if !tx.ContractID.Valid {
		return TransactionView{}, httperr.BadRequest("not an approvable transaction")
	}
	c, err := s.q.GetContract(ctx, tx.ContractID.UUID)
	if err != nil {
		return TransactionView{}, httperr.Internal(err)
	}
	a, err := s.propAccess(ctx, c.PropertyID, userID)
	if err != nil {
		return TransactionView{}, err
	}
	isTenant := c.TenantUserID.Valid && c.TenantUserID.UUID == userID
	if !(a.IsOwner || isTenant) {
		return TransactionView{}, httperr.Forbidden("only the owner or tenant can decide")
	}

	status := "approved"
	if !approve {
		status = "rejected"
	}
	updated, err := s.q.SetTransactionApproval(ctx, sqlc.SetTransactionApprovalParams{
		ID: txID, ApprovalStatus: status, ApprovedBy: nullUUID(userID),
	})
	if err != nil {
		return TransactionView{}, httperr.Internal(err)
	}
	s.audit(ctx, "transaction", txID, userID, status, map[string]any{})
	return txView(updated), nil
}

// ListPending returns the user's approval inbox.
func (s *Service) ListPending(ctx context.Context, userID uuid.UUID) ([]TransactionView, error) {
	txs, err := s.q.ListPendingForUser(ctx, userID)
	if err != nil {
		return nil, httperr.Internal(err)
	}
	return txViews(txs), nil
}

func (s *Service) ListContractTransactions(ctx context.Context, userID, contractID uuid.UUID) ([]TransactionView, error) {
	if _, err := s.getContractWithAccess(ctx, userID, contractID); err != nil {
		return nil, err
	}
	txs, err := s.q.ListTransactionsForContract(ctx, nullUUID(contractID))
	if err != nil {
		return nil, httperr.Internal(err)
	}
	return txViews(txs), nil
}

func (s *Service) ListPropertyTransactions(ctx context.Context, userID, propID uuid.UUID) ([]TransactionView, error) {
	if err := s.requirePropertyWide(ctx, propID, userID); err != nil {
		return nil, err
	}
	txs, err := s.q.ListTransactionsForProperty(ctx, propID)
	if err != nil {
		return nil, httperr.Internal(err)
	}
	return txViews(txs), nil
}
