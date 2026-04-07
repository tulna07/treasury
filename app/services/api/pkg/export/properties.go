package export

import (
	"fmt"

	"github.com/xuri/excelize/v2"
)

// setDocProperties sets the document properties for the Excel file.
func setDocProperties(f *excelize.File, params ExportParams, builder ReportBuilder) error {
	return f.SetDocProps(&excelize.DocProperties{
		Creator:     fmt.Sprintf("%s (%s)", params.User.FullName, params.User.Username),
		Title:       fmt.Sprintf("Treasury Report — %s %s", builder.Module(), builder.ReportType()),
		Subject:     "CONFIDENTIAL — Ngân hàng TMCP Kiên Long",
		Description: fmt.Sprintf("Exported from KienlongBank Treasury System. Module: %s, Type: %s", builder.Module(), builder.ReportType()),
		Category:    "Financial Report",
	})
}
