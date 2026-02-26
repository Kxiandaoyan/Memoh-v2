package fileparse

import (
	"fmt"
	"strings"

	"github.com/ledongthuc/pdf"
)

// extractPDF extracts plain text from a PDF file using ledongthuc/pdf.
func extractPDF(filePath string) (string, error) {
	f, r, err := pdf.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("open pdf: %w", err)
	}
	defer f.Close()

	var buf strings.Builder
	numPages := r.NumPage()
	for i := 1; i <= numPages; i++ {
		p := r.Page(i)
		if p.V.IsNull() {
			continue
		}
		text, err := p.GetPlainText(nil)
		if err != nil {
			continue
		}
		buf.WriteString(text)
		if i < numPages {
			buf.WriteString("\n")
		}
	}
	return buf.String(), nil
}
