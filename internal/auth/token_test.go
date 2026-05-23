package auth

import (
	"errors"
	"testing"
	"time"

	"github.com/zalando/go-keyring"
)

func newTestTokenStore() *TokenStore {
	return &TokenStore{service: "courses-mcp-test", user: "token"}
}

func TestTokenValid(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name  string
		token Token
		want  bool
	}{
		{
			name:  "zero value",
			token: Token{},
			want:  false,
		},
		{
			name: "empty access token with future expiry",
			token: Token{
				AccessToken: "",
				ExpiresAt:   now.Add(2 * time.Minute),
			},
			want: false,
		},
		{
			name: "expired token",
			token: Token{
				AccessToken: "tok",
				ExpiresAt:   now.Add(-1 * time.Minute),
			},
			want: false,
		},
		{
			name: "about to expire within buffer (30s left)",
			token: Token{
				AccessToken: "tok",
				ExpiresAt:   now.Add(30 * time.Second),
			},
			want: false,
		},
		{
			name: "exactly at buffer boundary (60s left)",
			token: Token{
				AccessToken: "tok",
				// now.Add(60s) is NOT before ExpiresAt when ExpiresAt == now.Add(60s)
				ExpiresAt: now.Add(60 * time.Second),
			},
			want: false,
		},
		{
			name: "just past buffer (61s left)",
			token: Token{
				AccessToken: "tok",
				ExpiresAt:   now.Add(61 * time.Second),
			},
			want: true,
		},
		{
			name: "well within validity window",
			token: Token{
				AccessToken: "tok",
				ExpiresAt:   now.Add(10 * time.Minute),
			},
			want: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.token.Valid()
			if got != tc.want {
				t.Errorf("Token.Valid() = %v, want %v (token=%+v)", got, tc.want, tc.token)
			}
		})
	}
}

func TestTokenStoreSaveLoad(t *testing.T) {
	store := newTestTokenStore()

	// Probe whether the keychain is available by attempting a set. Skip if not.
	probe := Token{AccessToken: "probe", ExpiresAt: time.Now().Add(5 * time.Minute)}
	if err := store.Save(probe); err != nil {
		t.Skipf("keychain unavailable in this environment, skipping: %v", err)
	}
	t.Cleanup(func() {
		_ = keyring.Delete(store.service, store.user)
	})

	original := Token{
		AccessToken:  "access-abc",
		RefreshToken: "refresh-xyz",
		ExpiresAt:    time.Now().Add(time.Hour).Truncate(time.Millisecond),
	}

	if err := store.Save(original); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	loaded, err := store.Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if loaded.AccessToken != original.AccessToken {
		t.Errorf("AccessToken: got %q, want %q", loaded.AccessToken, original.AccessToken)
	}
	if loaded.RefreshToken != original.RefreshToken {
		t.Errorf("RefreshToken: got %q, want %q", loaded.RefreshToken, original.RefreshToken)
	}
	if !loaded.ExpiresAt.Equal(original.ExpiresAt) {
		t.Errorf("ExpiresAt: got %v, want %v", loaded.ExpiresAt, original.ExpiresAt)
	}
}

func TestTokenStoreLoadNoToken(t *testing.T) {
	store := newTestTokenStore()

	// Ensure the test key doesn't exist from a previous run.
	_ = keyring.Delete(store.service, store.user)

	_, err := store.Load()

	if err == nil {
		t.Error("Load() expected error, got nil")
		return
	}
	if !errors.Is(err, ErrNoToken) {
		t.Skipf("unexpected error (keychain may be unavailable): %v", err)
	}
}
