package services

import (
	"context"

	"github.com/google/uuid"

	"wasooli/internal/httperr"
)

// YearlySummary is approved income vs expense for one year across the user's properties.
type YearlySummary struct {
	Year    int32 `json:"year"`
	Income  int64 `json:"income"`
	Expense int64 `json:"expense"`
	Net     int64 `json:"net"`
}

// Analytics returns per-year approved totals across every property the user can see.
func (s *Service) Analytics(ctx context.Context, userID uuid.UUID) ([]YearlySummary, error) {
	// Analytics spans properties the user owns/views plus their own tenancies.
	seen := map[uuid.UUID]bool{}
	var ids []uuid.UUID
	props, err := s.q.ListPropertiesForUser(ctx, userID)
	if err != nil {
		return nil, httperr.Internal(err)
	}
	for _, p := range props {
		if !seen[p.ID] {
			seen[p.ID] = true
			ids = append(ids, p.ID)
		}
	}
	tenancies, err := s.q.ListTenanciesForUser(ctx, nullUUID(userID))
	if err != nil {
		return nil, httperr.Internal(err)
	}
	for _, t := range tenancies {
		if !seen[t.PropertyID] {
			seen[t.PropertyID] = true
			ids = append(ids, t.PropertyID)
		}
	}
	if len(ids) == 0 {
		return []YearlySummary{}, nil
	}
	rows, err := s.q.SumApprovedByYear(ctx, ids)
	if err != nil {
		return nil, httperr.Internal(err)
	}

	byYear := map[int32]*YearlySummary{}
	order := []int32{}
	for _, r := range rows {
		y := byYear[r.Year]
		if y == nil {
			y = &YearlySummary{Year: r.Year}
			byYear[r.Year] = y
			order = append(order, r.Year)
		}
		if r.Direction == "income" {
			y.Income += r.Total
		} else {
			y.Expense += r.Total
		}
	}
	out := make([]YearlySummary, 0, len(order))
	for _, y := range order {
		s := byYear[y]
		s.Net = s.Income - s.Expense
		out = append(out, *s)
	}
	return out, nil
}
