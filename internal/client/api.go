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
type TokenProvider interface {
	Token(ctx context.Context) (string, error)
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
	token, err := c.tokens.Token(ctx)
	if err != nil {
		return fmt.Errorf("acquire token: %w", err)
	}

	url := c.apiBase() + path
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("build request %s: %w", path, err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := httpClient.Do(req)
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
