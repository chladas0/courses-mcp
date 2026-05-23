package client

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const pageBase = "https://courses.fit.cvut.cz"

// 60-second timeout to accommodate larger file downloads.
var pageHTTPClient = &http.Client{Timeout: 60 * time.Second}

// PageClient fetches pages and files from courses.fit.cvut.cz.
// Authentication uses the OAuth access token as a browser cookie.
type PageClient struct {
	tokens TokenProvider
	base   string // overrides pageBase in tests
}

// NewPageClient creates a new PageClient that authenticates using the provided TokenProvider.
func NewPageClient(tokens TokenProvider) *PageClient {
	return &PageClient{tokens: tokens}
}

func (c *PageClient) baseURL() string {
	if c.base != "" {
		return c.base
	}
	return pageBase
}

// FetchPage fetches a course page and returns its content as Markdown.
// path must start with "/" (e.g. "/" for homepage, "/lectures/").
func (c *PageClient) FetchPage(ctx context.Context, courseCode, path string) (string, error) {
	if strings.Contains(courseCode, "/") || strings.Contains(courseCode, "..") {
		return "", fmt.Errorf("invalid course code: %q", courseCode)
	}
	if strings.Contains(path, "..") {
		return "", fmt.Errorf("invalid path: %q", path)
	}

	token, err := c.tokens.Token(ctx)
	if err != nil {
		return "", fmt.Errorf("acquire token: %w", err)
	}

	rawURL := c.baseURL() + "/" + courseCode + path
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return "", fmt.Errorf("build request for %s%s: %w", courseCode, path, err)
	}
	req.Header.Set("Cookie", "oauth_access_token="+token)

	resp, err := pageHTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetch page %s%s: %w", courseCode, path, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("fetch page %s%s: status %d", courseCode, path, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read page %s%s: %w", courseCode, path, err)
	}

	md, err := HTMLToMarkdown(string(body))
	if err != nil {
		return "", fmt.Errorf("convert page %s%s to markdown: %w", courseCode, path, err)
	}
	return md, nil
}

// FetchFile downloads a file from a courses.fit.cvut.cz URL and returns its text content.
// Supports HTML (→ Markdown) and PDF (→ extracted text).
func (c *PageClient) FetchFile(ctx context.Context, rawURL string) (string, error) {
	data, contentType, err := c.FetchRaw(ctx, rawURL)
	if err != nil {
		return "", err
	}

	switch {
	case strings.Contains(contentType, "text/html"):
		md, err := HTMLToMarkdown(string(data))
		if err != nil {
			return "", fmt.Errorf("convert file %s to markdown: %w", rawURL, err)
		}
		return md, nil
	case strings.Contains(contentType, "application/pdf"):
		text, err := ExtractPDFText(data)
		if err != nil {
			return "", fmt.Errorf("extract PDF text from %s: %w", rawURL, err)
		}
		return text, nil
	default:
		return "", fmt.Errorf("unsupported content type: %s", contentType)
	}
}

// FetchRaw downloads a file from a courses.fit.cvut.cz URL and returns its raw
// bytes and Content-Type. Returns an error for URLs outside courses.fit.cvut.cz
// or files larger than 20 MB.
func (c *PageClient) FetchRaw(ctx context.Context, rawURL string) ([]byte, string, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Host != "courses.fit.cvut.cz" {
		return nil, "", errors.New("only courses.fit.cvut.cz URLs are supported")
	}

	fetchURL := rawURL
	if c.base != "" {
		if base, err := url.Parse(c.base); err == nil {
			parsed.Scheme = base.Scheme
			parsed.Host = base.Host
			fetchURL = parsed.String()
		}
	}

	token, err := c.tokens.Token(ctx)
	if err != nil {
		return nil, "", fmt.Errorf("acquire token: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fetchURL, nil)
	if err != nil {
		return nil, "", fmt.Errorf("build request for %s: %w", rawURL, err)
	}
	req.Header.Set("Cookie", "oauth_access_token="+token)

	resp, err := pageHTTPClient.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("fetch file %s: %w", rawURL, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, "", fmt.Errorf("fetch file %s: status %d", rawURL, resp.StatusCode)
	}

	contentType := resp.Header.Get("Content-Type")

	const maxSize = 20 * 1024 * 1024
	limited := io.LimitReader(resp.Body, int64(maxSize)+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return nil, "", fmt.Errorf("read file %s: %w", rawURL, err)
	}
	if len(data) == maxSize+1 {
		return nil, "", errors.New("file too large (>20 MB)")
	}
	return data, contentType, nil
}
