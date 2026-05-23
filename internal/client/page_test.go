package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFetchPage_ReturnsMarkdown(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(`<html><body><h1>Hello Course</h1><p>Welcome.</p></body></html>`))
	}))
	defer srv.Close()

	c := NewPageClientForTest(staticToken("tok"), srv.URL)
	md, err := c.FetchPage(context.Background(), "NI-VCC", "/")
	if err != nil {
		t.Fatalf("FetchPage: %v", err)
	}
	if !strings.Contains(md, "Hello Course") {
		t.Errorf("expected 'Hello Course' in markdown, got: %s", md)
	}
}

func TestFetchPage_Non2xxReturnsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	c := NewPageClientForTest(staticToken("tok"), srv.URL)
	_, err := c.FetchPage(context.Background(), "NI-VCC", "/lectures/")
	if err == nil {
		t.Fatal("expected error for 404, got nil")
	}
	if !strings.Contains(err.Error(), "status 404") {
		t.Errorf("expected 'status 404', got: %v", err)
	}
}

func TestFetchPage_InvalidCourseCodeReturnsError(t *testing.T) {
	c := NewPageClientForTest(staticToken("tok"), "http://unused")
	for _, bad := range []string{"NI-VCC/../etc", "NI/VCC", "../etc"} {
		_, err := c.FetchPage(context.Background(), bad, "/")
		if err == nil {
			t.Errorf("expected error for course code %q, got nil", bad)
		}
	}
}

func TestFetchPage_InvalidPathReturnsError(t *testing.T) {
	c := NewPageClientForTest(staticToken("tok"), "http://unused")
	_, err := c.FetchPage(context.Background(), "NI-VCC", "/../etc")
	if err == nil {
		t.Fatal("expected error for path with .., got nil")
	}
}

func TestFetchFile_HTMLReturnsMarkdown(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(`<html><body><h1>File Page</h1></body></html>`))
	}))
	defer srv.Close()

	c := NewPageClientForTest(staticToken("tok"), srv.URL)
	md, err := c.FetchFile(context.Background(), "https://courses.fit.cvut.cz/some/page.html")
	if err != nil {
		t.Fatalf("FetchFile: %v", err)
	}
	if !strings.Contains(md, "File Page") {
		t.Errorf("expected 'File Page' in markdown, got: %s", md)
	}
}

func TestFetchFile_InvalidHostReturnsError(t *testing.T) {
	c := NewPageClientForTest(staticToken("tok"), "http://unused")
	_, err := c.FetchFile(context.Background(), "https://evil.example.com/path")
	if err == nil {
		t.Fatal("expected error for invalid host, got nil")
	}
	if !strings.Contains(err.Error(), "only courses.fit.cvut.cz URLs are supported") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestFetchFile_OversizedResponseReturnsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/pdf")
		chunk := make([]byte, 4096)
		remaining := 20*1024*1024 + 2
		for remaining > 0 {
			n := remaining
			if n > len(chunk) {
				n = len(chunk)
			}
			_, _ = w.Write(chunk[:n])
			remaining -= n
		}
	}))
	defer srv.Close()

	c := NewPageClientForTest(staticToken("tok"), srv.URL)
	_, err := c.FetchFile(context.Background(), "https://courses.fit.cvut.cz/large.pdf")
	if err == nil {
		t.Fatal("expected 'file too large' error, got nil")
	}
	if !strings.Contains(err.Error(), "file too large") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestFetchFile_UnsupportedContentTypeReturnsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write([]byte{0x89, 0x50, 0x4E, 0x47})
	}))
	defer srv.Close()

	c := NewPageClientForTest(staticToken("tok"), srv.URL)
	_, err := c.FetchFile(context.Background(), "https://courses.fit.cvut.cz/image.png")
	if err == nil {
		t.Fatal("expected error for unsupported content type, got nil")
	}
	if !strings.Contains(err.Error(), "unsupported content type") {
		t.Errorf("unexpected error: %v", err)
	}
}
