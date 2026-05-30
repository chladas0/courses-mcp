package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const apiBase = "https://courses.fit.cvut.cz/api/v2"

// intentionally not http.DefaultClient
var httpClient = &http.Client{Timeout: 30 * time.Second}

// TokenProvider supplies a valid OAuth access token, refreshing it if needed.
// Refresh forces a new access token regardless of local expiry; it is used to
// recover when the server rejects a locally-unexpired token (e.g. revoked
// out-of-band) with 401.
type TokenProvider interface {
	Token(ctx context.Context) (string, error)
	Refresh(ctx context.Context) (string, error)
}

// doWithRefresh sends an authenticated GET request and, if the server responds
// with 401, force-refreshes the access token and retries once. setAuth applies
// the token to the request (Bearer header for the API, cookie for pages), so the
// retry mechanism is shared across both clients.
func doWithRefresh(ctx context.Context, hc *http.Client, tokens TokenProvider, url string, setAuth func(*http.Request, string)) (*http.Response, error) {
	send := func(token string) (*http.Response, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return nil, fmt.Errorf("build request %s: %w", url, err)
		}
		setAuth(req, token)
		return hc.Do(req)
	}

	token, err := tokens.Token(ctx)
	if err != nil {
		return nil, fmt.Errorf("acquire token: %w", err)
	}
	resp, err := send(token)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusUnauthorized {
		return resp, nil
	}

	// Token rejected despite being locally valid: force a refresh and retry once.
	_ = resp.Body.Close()
	token, err = tokens.Refresh(ctx)
	if err != nil {
		return nil, fmt.Errorf("refresh after 401: %w", err)
	}
	return send(token)
}

// UserInfo holds the authenticated user's profile as returned by the API.
type UserInfo struct {
	Username       string `json:"username"`
	PersonalNumber int    `json:"personalNumber"`
	FirstName      string `json:"firstName"`
	LastName       string `json:"lastName"`
	FullName       string `json:"fullName"`
	VocativeName   string `json:"vocativeName"`
}

// UserCourses holds the sets of course codes the user is enrolled in or teaching.
type UserCourses struct {
	Studying []string `json:"studying"`
	Teaching []string `json:"teaching"`
}

// CourseInfo holds KOS metadata for a single course.
type CourseInfo struct {
	Code       string `json:"code"`
	Credits    int    `json:"credits"`
	Completion string `json:"completion"`
}

type statusError struct {
	method string
	path   string
	code   int
}

func (e *statusError) Error() string {
	return fmt.Sprintf("courses api %s %s: status %d", e.method, e.path, e.code)
}

// APIClient calls the courses.fit.cvut.cz REST API.
type APIClient struct {
	tokens TokenProvider
	base   string // overrides apiBase in tests
}

func NewAPIClient(tokens TokenProvider) *APIClient {
	return &APIClient{tokens: tokens}
}

func (c *APIClient) apiBase() string {
	if c.base != "" {
		return c.base
	}
	return apiBase
}

func (c *APIClient) get(ctx context.Context, path string, dst any) error {
	url := c.apiBase() + path
	setAuth := func(req *http.Request, token string) {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := doWithRefresh(ctx, httpClient, c.tokens, url, setAuth)
	if err != nil {
		return fmt.Errorf("courses api GET %s: %w", path, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &statusError{method: http.MethodGet, path: path, code: resp.StatusCode}
	}

	if err := json.NewDecoder(resp.Body).Decode(dst); err != nil {
		return fmt.Errorf("decode response %s: %w", path, err)
	}
	return nil
}

func (c *APIClient) UserInfo(ctx context.Context) (UserInfo, error) {
	var info UserInfo
	if err := c.get(ctx, "/users/me", &info); err != nil {
		return UserInfo{}, err
	}
	return info, nil
}

func (c *APIClient) UserCourses(ctx context.Context) (UserCourses, error) {
	var courses UserCourses
	if err := c.get(ctx, "/users/me/courses", &courses); err != nil {
		return UserCourses{}, err
	}
	return courses, nil
}

func (c *APIClient) CourseInfo(ctx context.Context, code string) (CourseInfo, error) {
	if strings.Contains(code, "/") || strings.Contains(code, "..") {
		return CourseInfo{}, fmt.Errorf("invalid course code: %q", code)
	}
	var info CourseInfo
	path := "/courses/" + code
	err := c.get(ctx, path, &info)
	if err != nil {
		if se, ok := err.(*statusError); ok && se.code == http.StatusNotFound {
			return CourseInfo{}, fmt.Errorf("course %s not found", code)
		}
		return CourseInfo{}, err
	}
	return info, nil
}
