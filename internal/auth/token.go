package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/zalando/go-keyring"
)

const (
	keychainService = "courses-mcp"
	keychainUser    = "token"
)

// ErrNoToken is returned by TokenStore.Load when no token has been persisted yet.
var ErrNoToken = errors.New("no token stored")

// Token holds an OAuth access token and the refresh token used to obtain new ones.
type Token struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// Valid reports whether t can be used without refreshing.
func (t Token) Valid() bool {
	return t.AccessToken != "" && time.Now().Add(60*time.Second).Before(t.ExpiresAt)
}

// TokenStore persists tokens in the OS keychain.
type TokenStore struct {
	service string
	user    string
}

func NewTokenStore() *TokenStore {
	return &TokenStore{service: keychainService, user: keychainUser}
}

// Load returns ErrNoToken if no token has been saved yet.
func (s *TokenStore) Load() (Token, error) {
	secret, err := keyring.Get(s.service, s.user)
	if err != nil {
		if errors.Is(err, keyring.ErrNotFound) {
			return Token{}, ErrNoToken
		}
		return Token{}, fmt.Errorf("keychain get: %w", err)
	}

	var t Token
	if err := json.Unmarshal([]byte(secret), &t); err != nil {
		return Token{}, fmt.Errorf("unmarshal token: %w", err)
	}
	return t, nil
}

func (s *TokenStore) Save(t Token) error {
	data, err := json.Marshal(t)
	if err != nil {
		return fmt.Errorf("marshal token: %w", err)
	}
	if err := keyring.Set(s.service, s.user, string(data)); err != nil {
		return fmt.Errorf("keychain set: %w", err)
	}
	return nil
}
