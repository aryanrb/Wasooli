package handlers

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"

	"wasooli/internal/auth"
	"wasooli/internal/httperr"
)

// GoogleLogin redirects to Google's consent screen with a CSRF state cookie.
func (h *Handler) GoogleLogin(c echo.Context) error {
	if h.oauth.ClientID == "" {
		return httperr.New(http.StatusServiceUnavailable, "Google sign-in is not configured")
	}
	state := auth.RandomToken()
	c.SetCookie(&http.Cookie{
		Name: auth.StateCookieName, Value: state, Path: "/", MaxAge: 600,
		HttpOnly: true, Secure: h.cfg.SecureCookies(), SameSite: http.SameSiteLaxMode,
	})
	return c.Redirect(http.StatusTemporaryRedirect, h.oauth.AuthCodeURL(state))
}

// GoogleCallback verifies state, exchanges the code, upserts the user, sets the
// session cookie, and redirects to the SPA.
func (h *Handler) GoogleCallback(c echo.Context) error {
	ctx := c.Request().Context()
	state, err := c.Cookie(auth.StateCookieName)
	if err != nil || state.Value == "" || c.QueryParam("state") != state.Value {
		return httperr.BadRequest("invalid oauth state")
	}
	code := c.QueryParam("code")
	if code == "" {
		return httperr.BadRequest("missing authorization code")
	}
	gu, err := auth.FetchGoogleUser(ctx, h.oauth, code)
	if err != nil {
		h.log.Error("google sign-in", zap.Error(err))
		return httperr.Unauthorized("google sign-in failed")
	}
	if gu.Email == "" {
		return httperr.Unauthorized("google account has no email")
	}
	u, err := h.svc.UpsertGoogleUser(ctx, gu.Email, gu.Name, gu.Sub, gu.Picture)
	if err != nil {
		return err
	}
	tok, err := auth.IssueToken(h.cfg.JWTSecret, u.ID, u.Email)
	if err != nil {
		return httperr.Internal(err)
	}
	auth.SetSessionCookie(c, tok, h.cfg.SecureCookies())
	return c.Redirect(http.StatusTemporaryRedirect, h.cfg.FrontendURL)
}

// Logout clears the session cookie.
func (h *Handler) Logout(c echo.Context) error {
	auth.ClearSessionCookie(c, h.cfg.SecureCookies())
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

// Me returns the authenticated user's profile.
func (h *Handler) Me(c echo.Context) error {
	u, err := h.svc.Me(c.Request().Context(), auth.UserID(c))
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, u)
}
