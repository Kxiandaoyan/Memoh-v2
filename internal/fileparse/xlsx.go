package fileparse

import (
	"fmt"
	"strings"

	"github.com/xuri/excelize/v2"
)

// extractXLSX extracts cell text from all sheets of an XLSX/XLS file.
func extractXLSX(filePath string) (string, error) {
	f, err := excelize.OpenFile(filePath)
	if err != nil {
		return "", fmt.Errorf("open xlsx: %w", err)
	}
	defer f.Close()

	var buf strings.Builder
	for _, sheet := range f.GetSheetList() {
		buf.WriteString(fmt.Sprintf("## Sheet: %s\n", sheet))
		rows, err := f.GetRows(sheet)
		if err != nil {
			continue
		}
		for _, row := range rows {
			buf.WriteString(strings.Join(row, "\t"))
			buf.WriteString("\n")
		}
		buf.WriteString("\n")
	}
	return buf.String(), nil
}
