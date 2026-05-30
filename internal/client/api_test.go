package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type staticToken string

func (s staticToken) Token(_ context.Context) (string, error)   { return string(s), nil }
func (s staticToken) Refresh(_ context.Context) (string, error) { return string(s), nil }

func assertBearer(t *testing.T, r *http.Request) {
	t.Helper()
	if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
		t.Errorf("Authorization = %q, want %q", got, "Bearer test-token")
	}
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func TestUserInfo(t *testing.T) {
	want := UserInfo{
		Username: "novakj", PersonalNumber: 123456,
		FirstName: "Jan", LastName: "Novak", FullName: "Jan Novak",
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearer(t, r)
		writeJSON(w, want)
	}))
	defer srv.Close()

	got, err := NewAPIClientForTest(staticToken("test-token"), srv.URL).UserInfo(context.Background())
	if err != nil {
		t.Fatalf("UserInfo: %v", err)
	}
	if got != want {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestUserCourses(t *testing.T) {
	want := UserCourses{Studying: []string{"NI-VCC", "MI-AFP"}, Teaching: []string{"BI-AG1"}}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearer(t, r)
		writeJSON(w, want)
	}))
	defer srv.Close()

	got, err := NewAPIClientForTest(staticToken("test-token"), srv.URL).UserCourses(context.Background())
	if err != nil {
		t.Fatalf("UserCourses: %v", err)
	}
	if len(got.Studying) != len(want.Studying) || got.Studying[0] != want.Studying[0] {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestCourseInfo(t *testing.T) {
	want := CourseInfo{Code: "NI-VCC", Credits: 5, Completion: "EXAM"}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearer(t, r)
		writeJSON(w, want)
	}))
	defer srv.Close()

	got, err := NewAPIClientForTest(staticToken("test-token"), srv.URL).CourseInfo(context.Background(), "NI-VCC")
	if err != nil {
		t.Fatalf("CourseInfo: %v", err)
	}
	if got != want {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestCourseInfo_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer srv.Close()

	_, err := NewAPIClientForTest(staticToken("test-token"), srv.URL).CourseInfo(context.Background(), "XX-FAKE")
	if err == nil {
		t.Fatal("expected error for 404, got nil")
	}
	if !strings.Contains(err.Error(), "XX-FAKE not found") {
		t.Errorf("unexpected error: %v", err)
	}
}

// refreshableToken is a TokenProvider whose access token can be force-refreshed,
// simulating the real Provider recovering from a server-side token revocation.
type refreshableToken struct {
	current      string
	refreshed    string
	refreshCalls int
}

func (r *refreshableToken) Token(_ context.Context) (string, error) { return r.current, nil }

func (r *refreshableToken) Refresh(_ context.Context) (string, error) {
	r.refreshCalls++
	r.current = r.refreshed
	return r.refreshed, nil
}

func TestRetryOn401WithRefresh(t *testing.T) {
	want := UserInfo{Username: "novakj", PersonalNumber: 123456}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer fresh-token" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		writeJSON(w, want)
	}))
	defer srv.Close()

	tok := &refreshableToken{current: "stale-token", refreshed: "fresh-token"}
	got, err := NewAPIClientForTest(tok, srv.URL).UserInfo(context.Background())
	if err != nil {
		t.Fatalf("UserInfo after 401 retry: %v", err)
	}
	if got != want {
		t.Errorf("got %+v, want %+v", got, want)
	}
	if tok.refreshCalls != 1 {
		t.Errorf("refresh calls = %d, want 1", tok.refreshCalls)
	}
}

func TestNon2xxReturnsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}))
	defer srv.Close()

	_, err := NewAPIClientForTest(staticToken("test-token"), srv.URL).UserInfo(context.Background())
	if err == nil {
		t.Fatal("expected error for 500, got nil")
	}
	if !strings.Contains(err.Error(), "status 500") {
		t.Errorf("expected 'status 500', got: %v", err)
	}
}
