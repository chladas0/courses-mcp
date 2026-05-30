package tools_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/chladas0/courses-mcp/internal/client"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/chladas0/courses-mcp/internal/tools"
)

// ---------------------------------------------------------------------------
// Helpers to build a minimal MCP server, call a tool, and inspect the result.
// ---------------------------------------------------------------------------

func newServer() *mcp.Server {
	return mcp.NewServer(&mcp.Implementation{Name: "test"}, nil)
}

// callTool invokes the named tool on s with the provided JSON arguments string
// and returns the CallToolResult.
func callTool(t *testing.T, s *mcp.Server, name, argsJSON string) *mcp.CallToolResult {
	t.Helper()
	// Use a direct handler lookup via a session-less call.
	// The easiest approach: spin up a real in-process MCP session.
	ct, cs := mcp.NewInMemoryTransports()

	srv := s
	go func() {
		_ = srv.Run(context.Background(), ct)
	}()

	cli := mcp.NewClient(&mcp.Implementation{Name: "test-client"}, nil)
	sess, err := cli.Connect(context.Background(), cs, nil)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer func() { _ = sess.Close() }()

	var args any
	if argsJSON != "" {
		if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
			t.Fatalf("unmarshal args: %v", err)
		}
	}
	res, err := sess.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      name,
		Arguments: args,
	})
	if err != nil {
		t.Fatalf("CallTool %q: %v", name, err)
	}
	return res
}

// resultText returns the text from the first TextContent block in res.
func resultText(t *testing.T, res *mcp.CallToolResult) string {
	t.Helper()
	if len(res.Content) == 0 {
		t.Fatal("result has no content")
	}
	tc, ok := res.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("first content block is %T, not *mcp.TextContent", res.Content[0])
	}
	return tc.Text
}

// ---------------------------------------------------------------------------
// Static token provider for httptest servers.
// ---------------------------------------------------------------------------

type staticToken string

func (s staticToken) Token(_ context.Context) (string, error)   { return string(s), nil }
func (s staticToken) Refresh(_ context.Context) (string, error) { return string(s), nil }

// ---------------------------------------------------------------------------
// Tests for get_my_info
// ---------------------------------------------------------------------------

func TestGetMyInfo_HappyPath(t *testing.T) {
	want := client.UserInfo{
		Username:       "testuser",
		PersonalNumber: 123456,
		FullName:       "Test User",
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(want)
	}))
	defer srv.Close()

	api := client.NewAPIClientForTest(staticToken("tok"), srv.URL)
	s := newServer()
	tools.RegisterMyInfo(s, api)

	// Verify tool description is non-empty.
	res := callTool(t, s, "get_my_info", "")
	if res.IsError {
		t.Fatalf("unexpected error: %s", resultText(t, res))
	}
	text := resultText(t, res)
	var got client.UserInfo
	if err := json.Unmarshal([]byte(text), &got); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if got.Username != want.Username || got.PersonalNumber != want.PersonalNumber {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestGetMyInfo_Error(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	}))
	defer srv.Close()

	api := client.NewAPIClientForTest(staticToken("tok"), srv.URL)
	s := newServer()
	tools.RegisterMyInfo(s, api)

	res := callTool(t, s, "get_my_info", "")
	if !res.IsError {
		t.Fatal("expected IsError=true, got false")
	}
}

// ---------------------------------------------------------------------------
// Tests for get_my_courses
// ---------------------------------------------------------------------------

func TestGetMyCourses_HappyPath(t *testing.T) {
	want := client.UserCourses{
		Studying: []string{"NI-VCC", "MI-AFP"},
		Teaching: []string{"BI-AG1"},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(want)
	}))
	defer srv.Close()

	api := client.NewAPIClientForTest(staticToken("tok"), srv.URL)
	s := newServer()
	tools.RegisterMyCourses(s, api)

	res := callTool(t, s, "get_my_courses", "")
	if res.IsError {
		t.Fatalf("unexpected error: %s", resultText(t, res))
	}
	text := resultText(t, res)
	var got client.UserCourses
	if err := json.Unmarshal([]byte(text), &got); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if len(got.Studying) != 2 || got.Studying[0] != "NI-VCC" {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestGetMyCourses_Error(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "server error", http.StatusInternalServerError)
	}))
	defer srv.Close()

	api := client.NewAPIClientForTest(staticToken("tok"), srv.URL)
	s := newServer()
	tools.RegisterMyCourses(s, api)

	res := callTool(t, s, "get_my_courses", "")
	if !res.IsError {
		t.Fatal("expected IsError=true, got false")
	}
}

// ---------------------------------------------------------------------------
// Tests for get_course_info
// ---------------------------------------------------------------------------

func TestGetCourseInfo_HappyPath(t *testing.T) {
	want := client.CourseInfo{Code: "NI-VCC", Credits: 5, Completion: "EXAM"}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(want)
	}))
	defer srv.Close()

	api := client.NewAPIClientForTest(staticToken("tok"), srv.URL)
	s := newServer()
	tools.RegisterCourseInfo(s, api)

	res := callTool(t, s, "get_course_info", `{"course_code":"NI-VCC"}`)
	if res.IsError {
		t.Fatalf("unexpected error: %s", resultText(t, res))
	}
	text := resultText(t, res)
	var got client.CourseInfo
	if err := json.Unmarshal([]byte(text), &got); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if got.Code != want.Code || got.Credits != want.Credits {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestGetCourseInfo_MissingParam(t *testing.T) {
	api := client.NewAPIClientForTest(staticToken("tok"), "http://unused")
	s := newServer()
	tools.RegisterCourseInfo(s, api)

	res := callTool(t, s, "get_course_info", `{}`)
	if !res.IsError {
		t.Fatal("expected IsError=true for missing course_code")
	}
	if !strings.Contains(resultText(t, res), "course_code") {
		t.Errorf("expected 'course_code' in error message, got: %s", resultText(t, res))
	}
}

func TestGetCourseInfo_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer srv.Close()

	api := client.NewAPIClientForTest(staticToken("tok"), srv.URL)
	s := newServer()
	tools.RegisterCourseInfo(s, api)

	res := callTool(t, s, "get_course_info", `{"course_code":"XX-FAKE"}`)
	if !res.IsError {
		t.Fatal("expected IsError=true for 404")
	}
}

// ---------------------------------------------------------------------------
// Tests for get_course_page
// ---------------------------------------------------------------------------

func TestGetCoursePage_HappyPath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<html><body><h1>Welcome</h1><p>Hello world</p></body></html>`))
	}))
	defer srv.Close()

	pages := client.NewPageClientForTest(staticToken("tok"), srv.URL)
	s := newServer()
	tools.RegisterCoursePage(s, pages)

	res := callTool(t, s, "get_course_page", `{"course_code":"NI-VCC"}`)
	if res.IsError {
		t.Fatalf("unexpected error: %s", resultText(t, res))
	}
	text := resultText(t, res)
	if text == "" {
		t.Error("expected non-empty markdown output")
	}
}

func TestGetCoursePage_DefaultPath(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<html><body>ok</body></html>`))
	}))
	defer srv.Close()

	pages := client.NewPageClientForTest(staticToken("tok"), srv.URL)
	s := newServer()
	tools.RegisterCoursePage(s, pages)

	// No path argument, should default to "/"
	res := callTool(t, s, "get_course_page", `{"course_code":"NI-VCC"}`)
	if res.IsError {
		t.Fatalf("unexpected error: %s", resultText(t, res))
	}
	// The page client calls srv.URL + "/" + courseCode + path
	// so gotPath = "/NI-VCC/"
	if !strings.HasSuffix(gotPath, "/") {
		t.Errorf("expected path to end with '/', got %q", gotPath)
	}
}

func TestGetCoursePage_MissingCourseCode(t *testing.T) {
	pages := client.NewPageClientForTest(staticToken("tok"), "http://unused")
	s := newServer()
	tools.RegisterCoursePage(s, pages)

	res := callTool(t, s, "get_course_page", `{}`)
	if !res.IsError {
		t.Fatal("expected IsError=true for missing course_code")
	}
}

func TestGetCoursePage_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "forbidden", http.StatusForbidden)
	}))
	defer srv.Close()

	pages := client.NewPageClientForTest(staticToken("tok"), srv.URL)
	s := newServer()
	tools.RegisterCoursePage(s, pages)

	res := callTool(t, s, "get_course_page", `{"course_code":"NI-VCC"}`)
	if !res.IsError {
		t.Fatal("expected IsError=true for 403")
	}
}

// ---------------------------------------------------------------------------
// Tests for get_file_content
// ---------------------------------------------------------------------------

func TestGetFileContent_HappyPath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<html><body><p>Lecture notes</p></body></html>`))
	}))
	defer srv.Close()

	// The PageClient validates the host; we need to use a courses.fit.cvut.cz URL.
	// We override the HTTP transport for testing via NewPageClientForTest.
	pages := client.NewPageClientForTest(staticToken("tok"), srv.URL)
	s := newServer()
	tools.RegisterFileContent(s, pages)

	// Use a courses.fit.cvut.cz URL; the client validates the host.
	res := callTool(t, s, "get_file_content",
		`{"url":"https://courses.fit.cvut.cz/NI-VCC/lectures/01.html"}`)
	if res.IsError {
		t.Fatalf("unexpected error: %s", resultText(t, res))
	}
	if resultText(t, res) == "" {
		t.Error("expected non-empty content")
	}
}

func TestGetFileContent_InvalidDomain(t *testing.T) {
	pages := client.NewPageClientForTest(staticToken("tok"), "http://unused")
	s := newServer()
	tools.RegisterFileContent(s, pages)

	res := callTool(t, s, "get_file_content", `{"url":"https://evil.example.com/file.pdf"}`)
	if !res.IsError {
		t.Fatal("expected IsError=true for non-courses.fit.cvut.cz URL")
	}
}

func TestGetFileContent_MissingURL(t *testing.T) {
	pages := client.NewPageClientForTest(staticToken("tok"), "http://unused")
	s := newServer()
	tools.RegisterFileContent(s, pages)

	res := callTool(t, s, "get_file_content", `{}`)
	if !res.IsError {
		t.Fatal("expected IsError=true for missing url")
	}
}

func TestGetFileContent_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer srv.Close()

	pages := client.NewPageClientForTest(staticToken("tok"), srv.URL)
	s := newServer()
	tools.RegisterFileContent(s, pages)

	res := callTool(t, s, "get_file_content",
		`{"url":"https://courses.fit.cvut.cz/NI-VCC/lectures/missing.html"}`)
	if !res.IsError {
		t.Fatal("expected IsError=true for 404")
	}
}

// ---------------------------------------------------------------------------
// Tests for download_file
// ---------------------------------------------------------------------------

func TestDownloadFile_HappyPath(t *testing.T) {
	content := []byte("%PDF-1.4 fake pdf content")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/pdf")
		_, _ = w.Write(content)
	}))
	defer srv.Close()

	pages := client.NewPageClientForTest(staticToken("tok"), srv.URL)
	s := newServer()
	tools.RegisterDownloadFile(s, pages)

	destDir := t.TempDir()
	res := callTool(t, s, "download_file", fmt.Sprintf(
		`{"url":"https://courses.fit.cvut.cz/NI-VCC/lectures/01.pdf","destination":%q}`, destDir))
	if res.IsError {
		t.Fatalf("unexpected error: %s", resultText(t, res))
	}

	savedPath := destDir + "/01.pdf"
	data, err := os.ReadFile(savedPath)
	if err != nil {
		t.Fatalf("file not saved: %v", err)
	}
	if string(data) != string(content) {
		t.Errorf("file content mismatch: got %q, want %q", data, content)
	}
}

func TestDownloadFile_InvalidDomain(t *testing.T) {
	pages := client.NewPageClientForTest(staticToken("tok"), "http://unused")
	s := newServer()
	tools.RegisterDownloadFile(s, pages)

	res := callTool(t, s, "download_file", `{"url":"https://evil.example.com/file.pdf"}`)
	if !res.IsError {
		t.Fatal("expected IsError=true for non-courses.fit.cvut.cz URL")
	}
}

func TestDownloadFile_MissingURL(t *testing.T) {
	pages := client.NewPageClientForTest(staticToken("tok"), "http://unused")
	s := newServer()
	tools.RegisterDownloadFile(s, pages)

	res := callTool(t, s, "download_file", `{}`)
	if !res.IsError {
		t.Fatal("expected IsError=true for missing url")
	}
}

func TestDownloadFile_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer srv.Close()

	pages := client.NewPageClientForTest(staticToken("tok"), srv.URL)
	s := newServer()
	tools.RegisterDownloadFile(s, pages)

	res := callTool(t, s, "download_file",
		`{"url":"https://courses.fit.cvut.cz/NI-VCC/lectures/missing.pdf"}`)
	if !res.IsError {
		t.Fatal("expected IsError=true for 404")
	}
}

// ---------------------------------------------------------------------------
// Tool description non-empty checks
// ---------------------------------------------------------------------------

func TestToolDescriptionsNonEmpty(t *testing.T) {
	// Stand up a server with all tools registered, then list them via client.
	api := client.NewAPIClientForTest(staticToken("tok"), "http://unused")
	pages := client.NewPageClientForTest(staticToken("tok"), "http://unused")

	s := newServer()
	tools.RegisterMyInfo(s, api)
	tools.RegisterMyCourses(s, api)
	tools.RegisterCourseInfo(s, api)
	tools.RegisterCoursePage(s, pages)
	tools.RegisterFileContent(s, pages)
	tools.RegisterDownloadFile(s, pages)

	ct, cs := mcp.NewInMemoryTransports()
	go func() { _ = s.Run(context.Background(), ct) }()

	cli := mcp.NewClient(&mcp.Implementation{Name: "desc-checker"}, nil)
	sess, err := cli.Connect(context.Background(), cs, nil)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer func() { _ = sess.Close() }()

	res, err := sess.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}
	if len(res.Tools) != 6 {
		t.Fatalf("expected 6 tools, got %d", len(res.Tools))
	}
	for _, tool := range res.Tools {
		if tool.Description == "" {
			t.Errorf("tool %q has empty description", tool.Name)
		}
	}
}
