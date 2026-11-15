package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"

	"wasooli/config"
)

// GoogleUser is the subset of the Google userinfo response we use.
type GoogleUser struct {
	Sub     string `json:"id"`
	Email   string `json:"email"`
	Name    string `json:"name"`
	Picture string `json:"picture"`
}

// GoogleConfig builds the OAuth2 config for Google sign-in.
func GoogleConfig(cfg *config.Config) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     cfg.GoogleClientID,
		ClientSecret: cfg.GoogleClientSecret,
		RedirectURL:  cfg.GoogleRedirectURL,
		Scopes:       []string{"openid", "email", "profile"},
		Endpoint:     google.Endpoint,
	}
}

// FetchGoogleUser exchanges an auth code and loads the user's Google profile.
func FetchGoogleUser(ctx context.Context, oc *oauth2.Config, code string) (GoogleUser, error) {
	var u GoogleUser
	tok, err := oc.Exchange(ctx, code)
	if err != nil {
		return u, fmt.Errorf("oauth exchange: %w", err)
	}
	resp, err := oc.Client(ctx, tok).Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		return u, fmt.Errorf("userinfo: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return u, fmt.Errorf("userinfo status %d: %s", resp.StatusCode, body)
	}
	if err := json.NewDecoder(resp.Body).Decode(&u); err != nil {
		return u, fmt.Errorf("decode userinfo: %w", err)
	}
	return u, nil
}
