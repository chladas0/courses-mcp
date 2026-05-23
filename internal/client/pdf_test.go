package client

import (
	"encoding/base64"
	"errors"
	"strings"
	"testing"
)

// minimalTextPDFb64 is a minimal single-page PDF containing the text:
// "This PDF document contains sufficient text to pass the minimum threshold
// of one hundred nonwhitespace characters for extraction."
// (111 non-whitespace characters)
const minimalTextPDFb64 = "JVBERi0xLjQKMSAwIG9iago8PCAvVHlwZSAvQ2F0YWxvZyAvUGFnZXMgMiAwIFIgPj4KZW5kb2JqCjIgMCBvYmoKPDwgL1R5cGUgL1BhZ2VzIC9LaWRzIFszIDAgUl0gL0NvdW50IDEgPj4KZW5kb2JqCjMgMCBvYmoKPDwgL1R5cGUgL1BhZ2UgL1BhcmVudCAyIDAgUiAvTWVkaWFCb3ggWzAgMCA2MTIgNzkyXSAvQ29udGVudHMgNCAwIFIgL1Jlc291cmNlcyA8PCAvRm9udCA8PCAvRjEgNSAwIFIgPj4gPj4gPj4KZW5kb2JqCjQgMCBvYmoKPDwgL0xlbmd0aCAxNjAgPj4Kc3RyZWFtCkJUIC9GMSAxMiBUZiA1MCA3MDAgVGQgKFRoaXMgUERGIGRvY3VtZW50IGNvbnRhaW5zIHN1ZmZpY2llbnQgdGV4dCB0byBwYXNzIHRoZSBtaW5pbXVtIHRocmVzaG9sZCBvZiBvbmUgaHVuZHJlZCBub253aGl0ZXNwYWNlIGNoYXJhY3RlcnMgZm9yIGV4dHJhY3Rpb24uKSBUaiBFVAplbmRzdHJlYW0KZW5kb2JqCjUgMCBvYmoKPDwgL1R5cGUgL0ZvbnQgL1N1YnR5cGUgL1R5cGUxIC9CYXNlRm9udCAvSGVsdmV0aWNhID4+CmVuZG9iagp4cmVmCjAgNgowMDAwMDAwMDAwIDY1NTM1IGYgCjAwMDAwMDAwMDkgMDAwMDAgbiAKMDAwMDAwMDA1OCAwMDAwMCBuIAowMDAwMDAwMTE1IDAwMDAwIG4gCjAwMDAwMDAyNDEgMDAwMDAgbiAKMDAwMDAwMDQ1MSAwMDAwMCBuIAp0cmFpbGVyCjw8IC9TaXplIDYgL1Jvb3QgMSAwIFIgPj4Kc3RhcnR4cmVmCjUyMQolJUVPRgo="

// shortTextPDFb64 is a minimal single-page PDF containing only "Hello World"
// (11 non-whitespace characters, well below the 100 char threshold).
const shortTextPDFb64 = "JVBERi0xLjQKMSAwIG9iago8PCAvVHlwZSAvQ2F0YWxvZyAvUGFnZXMgMiAwIFIgPj4KZW5kb2JqCjIgMCBvYmoKPDwgL1R5cGUgL1BhZ2VzIC9LaWRzIFszIDAgUl0gL0NvdW50IDEgPj4KZW5kb2JqCjMgMCBvYmoKPDwgL1R5cGUgL1BhZ2UgL1BhcmVudCAyIDAgUiAvTWVkaWFCb3ggWzAgMCA2MTIgNzkyXSAvQ29udGVudHMgNCAwIFIgL1Jlc291cmNlcyA8PCAvRm9udCA8PCAvRjEgNSAwIFIgPj4gPj4gPj4KZW5kb2JqCjQgMCBvYmoKPDwgL0xlbmd0aCA0NCA+PgpzdHJlYW0KQlQgL0YxIDEyIFRmIDEwMCA3MDAgVGQgKEhlbGxvIFdvcmxkKSBUaiBFVAplbmRzdHJlYW0KZW5kb2JqCjUgMCBvYmoKPDwgL1R5cGUgL0ZvbnQgL1N1YnR5cGUgL1R5cGUxIC9CYXNlRm9udCAvSGVsdmV0aWNhID4+CmVuZG9iagp4cmVmCjAgNgowMDAwMDAwMDAwIDY1NTM1IGYgCjAwMDAwMDAwMDkgMDAwMDAgbiAKMDAwMDAwMDA1OCAwMDAwMCBuIAowMDAwMDAwMTE1IDAwMDAwIG4gCjAwMDAwMDAyNDEgMDAwMDAgbiAKMDAwMDAwMDMzNCAwMDAwMCBuIAp0cmFpbGVyCjw8IC9TaXplIDYgL1Jvb3QgMSAwIFIgPj4Kc3RhcnR4cmVmCjQwNAolJUVPRgo="

func mustDecodeB64(t *testing.T, s string) []byte {
	t.Helper()
	data, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		t.Fatalf("base64 decode failed: %v", err)
	}
	return data
}

func TestExtractPDFText_WithText(t *testing.T) {
	data := mustDecodeB64(t, minimalTextPDFb64)

	text, err := ExtractPDFText(data)
	if err != nil {
		t.Fatalf("ExtractPDFText returned unexpected error: %v", err)
	}
	if !strings.Contains(text, "nonwhitespace") {
		t.Errorf("expected extracted text to contain 'nonwhitespace', got: %q", text)
	}
}

func TestExtractPDFText_ImageBased(t *testing.T) {
	data := mustDecodeB64(t, shortTextPDFb64)

	text, err := ExtractPDFText(data)
	if !errors.Is(err, ErrImageBasedPDF) {
		t.Fatalf("expected ErrImageBasedPDF, got err=%v, text=%q", err, text)
	}
	if text != "" {
		t.Errorf("expected empty string on ErrImageBasedPDF, got %q", text)
	}
}

func TestExtractPDFText_InvalidData(t *testing.T) {
	_, err := ExtractPDFText([]byte("not a pdf"))
	if err == nil {
		t.Fatal("expected error for invalid PDF data, got nil")
	}
	if errors.Is(err, ErrImageBasedPDF) {
		t.Error("invalid PDF data should not return ErrImageBasedPDF")
	}
}

func TestCountNonWhitespace(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"", 0},
		{"   \t\n", 0},
		{"hello", 5},
		{"hello world", 10},
		{"  hello  world  ", 10},
		{"abc\ndef\tghi", 9},
	}

	for _, tc := range tests {
		got := countNonWhitespace(tc.input)
		if got != tc.want {
			t.Errorf("countNonWhitespace(%q) = %d, want %d", tc.input, got, tc.want)
		}
	}
}
