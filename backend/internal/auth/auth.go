// Package auth handles Google OAuth sign-in and JWT session cookies.
package auth

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"wasooli/internal/httperr"
)

const (
	SessionCookie = "wasooli_session"
	stateCookie   = "wasooli_oauth_state"
	sessionTTL    = 7 * 24 * time.Hour
	ctxUserKey    = "uid"
)

// claims is the JWT payload: the user id (subject) plus email for convenience.
type claims struct {
	Email string `json:"email"`
	jwt.RegisteredClaims
}

// IssueToken signs a session JWT for the given user.
func IssueToken(secret string, userID uuid.UUID, email string) (string, error) {
	c := claims{
		Email: email,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID.String(),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(sessionTTL)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, c).SignedString([]byte(secret))
}

// SetSessionCookie writes the session JWT as an httpOnly cookie.
func SetSessionCookie(c echo.Context, token string, secure bool) {
	c.SetCookie(&http.Cookie{
		Name:     SessionCookie,
		Value:    token,
		Path:     "/",
		MaxAge:   int(sessionTTL.Seconds()),
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}

// ClearSessionCookie expires the session cookie (logout).
func ClearSessionCookie(c echo.Context, secure bool) {
	c.SetCookie(&http.Cookie{
		Name: SessionCookie, Value: "", Path: "/", MaxAge: -1,
		HttpOnly: true, Secure: secure, SameSite: http.SameSiteLaxMode,
	})
}

// Middleware validates the session cookie and stores the user id in the context.
func Middleware(secret string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			cookie, err := c.Cookie(SessionCookie)
			if err != nil || cookie.Value == "" {
				return httperr.Unauthorized("not authenticated")
			}
			var cl claims
			tok, err := jwt.ParseWithClaims(cookie.Value, &cl, func(t *jwt.Token) (any, error) {
				if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, jwt.ErrSignatureInvalid
				}
				return []byte(secret), nil
			})
			if err != nil || !tok.Valid {
				return httperr.Unauthorized("invalid session")
			}
			id, err := uuid.Parse(cl.Subject)
			if err != nil {
				return httperr.Unauthorized("invalid session")
			}
			c.Set(ctxUserKey, id)
			return next(c)
		}
	}
}

// UserID returns the authenticated user's id (only valid behind Middleware).
func UserID(c echo.Context) uuid.UUID {
	id, _ := c.Get(ctxUserKey).(uuid.UUID)
	return id
}

// RandomToken returns a URL-safe random hex string (oauth state, invite tokens).
func RandomToken() string {
	b := make([]byte, 24)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// StateCookieName is exported so handlers can set/clear the oauth state cookie.
const StateCookieName = stateCookie
