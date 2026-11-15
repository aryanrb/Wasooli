package services

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"

	"wasooli/internal/db/sqlc"
	"wasooli/internal/httperr"
)

// UpsertGoogleUser creates or updates a user from a Google profile (find-or-create
// by email; links the Google identity).
func (s *Service) UpsertGoogleUser(ctx context.Context, email, name, sub, avatar string) (sqlc.User, error) {
	u, err := s.q.UpsertUserFromGoogle(ctx, sqlc.UpsertUserFromGoogleParams{
		Email:     email,
		Name:      name,
		GoogleSub: textOrNull(sub),
		AvatarUrl: avatar,
	})
	if err != nil {
		s.log.Error("upsert google user", zap.Error(err))
		return u, httperr.Internal(err)
	}
	return u, nil
}

// Me returns the authenticated user's profile.
func (s *Service) Me(ctx context.Context, userID uuid.UUID) (UserView, error) {
	u, err := s.q.GetUser(ctx, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return UserView{}, httperr.Unauthorized("user not found")
		}
		return UserView{}, httperr.Internal(err)
	}
	return userView(u), nil
}
