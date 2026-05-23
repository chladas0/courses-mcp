package auth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"
)

// DefaultPortalURL is used to exchange a refresh token for a new access token.
// Hack: nginx only sets oauth_access_token when serving a real .html course page,
// not the SPA root. BI-PA1 is the immortal first-year required course; it will
// outlive us all.
const DefaultPortalURL = "https://courses.fit.cvut.cz/BI-PA1/index.html"

// refreshClient has no client-level timeout; the per-request context timeout in
// refresh() caps each call at 15 seconds.
var refreshClient = &http.Client{}

type tokenStorage interface {
	Load() (Token, error)
	Save(Token) error
}

// Provider manages OAuth tokens for the courses portal.
// It loads tokens from the OS keychain and transparently refreshes the access
// token via the portal's cookie mechanism when it expires.
type Provider struct {
	store     tokenStorage
	portalURL string
}

func NewProvider(store *TokenStore) *Provider {
	return &Provider{store: store, portalURL: DefaultPortalURL}
}

// Token returns a valid access token, refreshing if needed.
func (p *Provider) Token(ctx context.Context) (string, error) {
	token, err := p.store.Load()
	if err != nil {
		if errors.Is(err, ErrNoToken) {
			return "", fmt.Errorf("no token stored: run --setup to authenticate")
		}
		return "", err
	}

	if !token.Valid() {
		if token.RefreshToken == "" {
			return "", fmt.Errorf("access token expired and no refresh token available: run --setup to authenticate")
		}

		token, err = refresh(ctx, token.RefreshToken, p.portalURL)
		if err != nil {
			return "", err
		}

		// Non-fatal: we still have a valid token in memory.
		_ = p.store.Save(token)
	}

	return token.AccessToken, nil
}

// refresh obtains a new access token from the courses portal by presenting the
// refresh token as a cookie. The portal responds with a Set-Cookie header that
// contains oauth_access_token and its Max-Age.
func refresh(ctx context.Context, refreshToken, portalURL string) (Token, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, portalURL, nil)
	if err != nil {
		return Token{}, fmt.Errorf("refresh: build request: %w", err)
	}
	req.Header.Set("Cookie", "oauth_refresh_token="+refreshToken)

	resp, err := refreshClient.Do(req)
	if err != nil {
		return Token{}, fmt.Errorf("refresh: GET %s: %w", portalURL, err)
	}
	defer func() { _ = resp.Body.Close() }()

	var accessToken string
	var maxAge int

	for _, c := range resp.Cookies() {
		if c.Name == "oauth_access_token" {
			accessToken = c.Value
			maxAge = c.MaxAge
			break
		}
	}

	if accessToken == "" {
		return Token{}, errors.New("refresh failed: portal did not return a new access token (refresh token may be expired)")
	}

	if maxAge <= 0 {
		maxAge = 2 * 60 * 60 // portal typically issues ~2h tokens; use as fallback
	}
	return Token{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    time.Now().Add(time.Duration(maxAge) * time.Second),
	}, nil
}
