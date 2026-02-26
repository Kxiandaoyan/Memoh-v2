package fileparse

import (
	"fmt"

	"github.com/lu4p/cat"
)

// extractDOCX extracts plain text from a DOCX file using lu4p/cat.
func extractDOCX(filePath string) (string, error) {
	text, err := cat.File(filePath)
	if err != nil {
		return "", fmt.Errorf("extract docx: %w", err)
	}
	return text, nil
}
