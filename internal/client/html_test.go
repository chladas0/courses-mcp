package client

import (
	"strings"
	"testing"
)

// TestHTMLToMarkdown_NavStripped verifies that <nav> elements and their
// contents are removed from the output.
func TestHTMLToMarkdown_NavStripped(t *testing.T) {
	input := `<html><body>
		<nav><a href="/home">Home</a></nav>
		<main><p>Main content here.</p></main>
	</body></html>`

	got, err := HTMLToMarkdown(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(got, "Home") {
		t.Errorf("nav content should be stripped, but got:\n%s", got)
	}
	if !strings.Contains(got, "Main content here") {
		t.Errorf("main content should be preserved, but got:\n%s", got)
	}
}

// TestHTMLToMarkdown_HeaderFooterStripped verifies that <header> and <footer>
// elements are removed from the output.
func TestHTMLToMarkdown_HeaderFooterStripped(t *testing.T) {
	input := `<html><body>
		<header><h1>Site Title</h1></header>
		<p>Body text.</p>
		<footer><p>Copyright 2024</p></footer>
	</body></html>`

	got, err := HTMLToMarkdown(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(got, "Site Title") {
		t.Errorf("header content should be stripped, but got:\n%s", got)
	}
	if strings.Contains(got, "Copyright") {
		t.Errorf("footer content should be stripped, but got:\n%s", got)
	}
	if !strings.Contains(got, "Body text") {
		t.Errorf("body content should be preserved, but got:\n%s", got)
	}
}

// TestHTMLToMarkdown_SidebarMenubarStripped verifies that elements with class
// "menubar" or "sidebar" are removed.
func TestHTMLToMarkdown_SidebarMenubarStripped(t *testing.T) {
	input := `<html><body>
		<div class="menubar">Menu item one</div>
		<div class="sidebar extra">Sidebar content</div>
		<p>Real content.</p>
	</body></html>`

	got, err := HTMLToMarkdown(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(got, "Menu item one") {
		t.Errorf("menubar content should be stripped, but got:\n%s", got)
	}
	if strings.Contains(got, "Sidebar content") {
		t.Errorf("sidebar content should be stripped, but got:\n%s", got)
	}
	if !strings.Contains(got, "Real content") {
		t.Errorf("main content should be preserved, but got:\n%s", got)
	}
}

// TestHTMLToMarkdown_LinksPreserved verifies that <a href> links are converted
// to Markdown link syntax.
func TestHTMLToMarkdown_LinksPreserved(t *testing.T) {
	input := `<html><body>
		<p>Visit <a href="https://example.com">Example</a> for details.</p>
	</body></html>`

	got, err := HTMLToMarkdown(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(got, "[Example](https://example.com)") {
		t.Errorf("expected Markdown link [Example](https://example.com), but got:\n%s", got)
	}
}

// TestHTMLToMarkdown_ImageReplaced verifies that inline image markdown is
// replaced with a linked reference: ![alt](url) → [image: alt](url).
func TestHTMLToMarkdown_ImageReplaced(t *testing.T) {
	input := `<html><body>
		<p><img src="https://example.com/photo.png" alt="a cat"/></p>
	</body></html>`

	got, err := HTMLToMarkdown(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(got, "![") {
		t.Errorf("inline image markdown should be replaced, but got:\n%s", got)
	}
	if !strings.Contains(got, "[image: a cat](https://example.com/photo.png)") {
		t.Errorf("expected [image: a cat](...), but got:\n%s", got)
	}
}

// TestHTMLToMarkdown_TableConverted verifies that HTML tables are converted to
// Markdown table syntax.
func TestHTMLToMarkdown_TableConverted(t *testing.T) {
	input := `<html><body>
		<table>
			<thead><tr><th>Name</th><th>Value</th></tr></thead>
			<tbody><tr><td>Alpha</td><td>1</td></tr></tbody>
		</table>
	</body></html>`

	got, err := HTMLToMarkdown(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Markdown tables use | as column separators.
	if !strings.Contains(got, "|") {
		t.Errorf("expected Markdown table with | separators, but got:\n%s", got)
	}
	if !strings.Contains(got, "Name") || !strings.Contains(got, "Alpha") {
		t.Errorf("table cell contents should be preserved, but got:\n%s", got)
	}
}

// TestHTMLToMarkdown_CodeBlock verifies that <code> elements are converted to
// Markdown code formatting.
func TestHTMLToMarkdown_CodeBlock(t *testing.T) {
	input := `<html><body>
		<pre><code>func main() {
    fmt.Println("hello")
}</code></pre>
	</body></html>`

	got, err := HTMLToMarkdown(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Markdown fenced code blocks use backticks.
	if !strings.Contains(got, "`") {
		t.Errorf("expected backtick code block, but got:\n%s", got)
	}
	if !strings.Contains(got, "fmt.Println") {
		t.Errorf("code content should be preserved, but got:\n%s", got)
	}
}
