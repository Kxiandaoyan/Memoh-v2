// Package fileparse extracts plain text from uploaded files so that
// the content can be injected into the LLM prompt regardless of model.
package fileparse

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// maxTextBytes is the hard cap on extracted text (100 KB).
const maxTextBytes = 100 * 1024

// ExtractText reads filePath and returns its plain-text representation.
// mimeType is used as the primary dispatch key; file extension is the fallback.
func ExtractText(filePath string, mimeType string) (string, error) {
	mime := strings.ToLower(strings.TrimSpace(mimeType))
	ext := strings.ToLower(filepath.Ext(filePath))

	var (
		text string
		err  error
	)

	switch {
	case mime == "application/pdf":
		text, err = extractPDF(filePath)
	case mime == "application/vnd.openxmlformats-officedocument.wordprocessingml.document" || ext == ".docx":
		text, err = extractDOCX(filePath)
	case mime == "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet" ||
		mime == "application/vnd.ms-excel" || ext == ".xlsx" || ext == ".xls":
		text, err = extractXLSX(filePath)
	case isPlainText(mime, ext):
		text, err = extractPlainText(filePath)
	default:
		return "", fmt.Errorf("unsupported file type: %s (%s)", mime, ext)
	}

	if err != nil {
		return "", err
	}
	return truncate(text), nil
}

// isPlainText returns true for MIME types / extensions that can be read as-is.
func isPlainText(mime, ext string) bool {
	if strings.HasPrefix(mime, "text/") {
		return true
	}
	switch mime {
	case "application/json", "application/xml", "application/csv":
		return true
	}
	switch ext {
	case ".txt", ".md", ".csv", ".json", ".xml", ".html", ".htm",
		".yaml", ".yml", ".log", ".ini", ".cfg", ".conf":
		return true
	}
	return false
}

// truncate caps text at maxTextBytes and appends a marker.
func truncate(s string) string {
	if len(s) <= maxTextBytes {
		return s
	}
	return s[:maxTextBytes] + "\n[... truncated]"
}

// extractPlainText reads the entire file as UTF-8 text.
func extractPlainText(filePath string) (string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("read file: %w", err)
	}
	return string(data), nil
}
