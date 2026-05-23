package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type fakeStore struct {
	token Token
	err   error
}

func (f *fakeStore) Load() (Token, error) { return f.token, f.err }
func (f *fakeStore) Save(t Token) error   { f.token = t; return nil }

func newProvider(store *fakeStore, portalURL string) *Provider {
	return &Provider{store: store, portalURL: portalURL}
}

func TestRefreshSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{Name: "oauth_access_token", Value: "newtoken", MaxAge: 7200})
	}))
	defer srv.Close()

	store := &fakeStore{token: Token{
		AccessToken:  "oldtoken",
		RefreshToken: "myrefresh",
		ExpiresAt:    time.Now().Add(-1 * time.Minute),
	}}
	got, err := newProvider(store, srv.URL+"/").Token(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "newtoken" {
		t.Errorf("Token() = %q, want %q", got, "newtoken")
	}
}

func TestRefreshFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK) // no cookie
	}))
	defer srv.Close()

	store := &fakeStore{token: Token{
		AccessToken:  "oldtoken",
		RefreshToken: "badrefresh",
		ExpiresAt:    time.Now().Add(-1 * time.Minute),
	}}
	_, err := newProvider(store, srv.URL+"/").Token(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "refresh failed") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidTokenNoHTTP(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))
	defer srv.Close()

	store := &fakeStore{token: Token{
		AccessToken:  "validtoken",
		RefreshToken: "myrefresh",
		ExpiresAt:    time.Now().Add(10 * time.Minute),
	}}
	got, err := newProvider(store, srv.URL+"/").Token(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "validtoken" {
		t.Errorf("Token() = %q, want %q", got, "validtoken")
	}
	if called {
		t.Error("made HTTP request for a valid token")
	}
}

func TestNoTokenSetupHint(t *testing.T) {
	store := &fakeStore{err: ErrNoToken}
	_, err := newProvider(store, DefaultPortalURL).Token(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "setup") {
		t.Errorf("error should mention 'setup', got: %v", err)
	}
}
