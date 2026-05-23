package client

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"unicode"

	"github.com/ledongthuc/pdf"
)

// ErrImageBasedPDF is returned when a PDF appears to contain no extractable text.
// This typically occurs with scanned documents where pages are stored as images.
var ErrImageBasedPDF = errors.New("PDF appears to be image-based; text extraction not supported")

// ExtractPDFText extracts plain text from PDF bytes.
// Pages are joined with "\n---\n".
// Returns ErrImageBasedPDF if the extracted text is too short to be useful
// (< 100 non-whitespace characters), indicating a scanned/image-based PDF.
func ExtractPDFText(data []byte) (string, error) {
	r, err := pdf.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", fmt.Errorf("opening PDF: %w", err)
	}

	numPages := r.NumPage()
	pages := make([]string, 0, numPages)

	for i := 0; i < numPages; i++ {
		pageText, err := r.Page(i + 1).GetPlainText(nil)
		if err != nil {
			return "", fmt.Errorf("extracting text from page %d: %w", i+1, err)
		}
		pages = append(pages, pageText)
	}

	result := strings.Join(pages, "\n---\n")

	if countNonWhitespace(result) < 100 {
		return "", ErrImageBasedPDF
	}

	return result, nil
}

func countNonWhitespace(s string) int {
	count := 0
	for _, r := range s {
		if !unicode.IsSpace(r) {
			count++
		}
	}
	return count
}
