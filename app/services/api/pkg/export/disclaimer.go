package export

import (
	"fmt"
	"time"

	"github.com/xuri/excelize/v2"
)

// BuildDisclaimerPublic is the exported version for external callers (e.g. sample generator).
func BuildDisclaimerPublic(f *excelize.File, params ExportParams, exportCode string) error {
	return buildDisclaimer(f, params, exportCode)
}

// buildDisclaimer creates the "Tuyên bố miễn trừ" (Disclaimer) sheet in the workbook.
func buildDisclaimer(f *excelize.File, params ExportParams, exportCode string) error {
	sheet := "Tuyên bố miễn trừ"
	idx, err := f.NewSheet(sheet)
	if err != nil {
		return err
	}
	_ = idx

	// Column widths
	_ = f.SetColWidth(sheet, "A", "A", 5)
	_ = f.SetColWidth(sheet, "B", "B", 60)

	// Title style
	titleStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 16, Color: "#C00000"},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})

	// Header style
	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 11},
		Alignment: &excelize.Alignment{Vertical: "center"},
		Border: []excelize.Border{
			{Type: "bottom", Color: "#000000", Style: 1},
		},
	})

	// Value style
	valueStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Size: 11},
		Alignment: &excelize.Alignment{Vertical: "center"},
		Border: []excelize.Border{
			{Type: "bottom", Color: "#CCCCCC", Style: 1},
		},
	})

	// Rule style
	ruleStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Size: 10},
		Alignment: &excelize.Alignment{WrapText: true, Vertical: "top"},
	})

	row := 1

	// Title
	_ = f.MergeCell(sheet, "A1", "B1")
	_ = f.SetCellValue(sheet, "A1", "KIENLONGBANK - TÀI LIỆU MẬT")
	_ = f.SetCellStyle(sheet, "A1", "B1", titleStyle)
	_ = f.SetRowHeight(sheet, 1, 30)
	row = 3

	// Export info
	info := []struct {
		label string
		value string
	}{
		{"Người xuất", fmt.Sprintf("%s (%s)", params.User.FullName, params.User.Username)},
		{"Chức danh", params.User.Role},
		{"Thời điểm", time.Now().Format("02/01/2006 15:04:05")},
		{"Khoảng dữ liệu", fmt.Sprintf("%s — %s", params.DateFrom.Format("02/01/2006"), params.DateTo.Format("02/01/2006"))},
		{"Mã xuất", exportCode},
	}

	for _, item := range info {
		_ = f.SetCellValue(sheet, fmt.Sprintf("A%d", row), item.label)
		_ = f.SetCellStyle(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("A%d", row), headerStyle)
		_ = f.SetCellValue(sheet, fmt.Sprintf("B%d", row), item.value)
		_ = f.SetCellStyle(sheet, fmt.Sprintf("B%d", row), fmt.Sprintf("B%d", row), valueStyle)
		row++
	}

	row += 1

	// Rules
	rules := []string{
		"1. Tài liệu này là tài sản thuộc sở hữu của Ngân hàng TMCP Kiên Long (KienlongBank). Mọi hành vi sao chép, phân phối, hoặc tiết lộ nội dung mà không có sự chấp thuận bằng văn bản đều bị nghiêm cấm.",
		"2. Chỉ những người được ủy quyền mới có quyền truy cập tài liệu này. Người nhận có trách nhiệm bảo mật thông tin theo quy định nội bộ của Ngân hàng.",
		"3. Dữ liệu trong báo cáo này được trích xuất từ hệ thống Treasury tại thời điểm xuất. KienlongBank không chịu trách nhiệm cho bất kỳ sai lệch nào phát sinh do thay đổi dữ liệu sau thời điểm xuất.",
		"4. Mọi hoạt động xuất dữ liệu đều được ghi nhận trong hệ thống audit. Vi phạm quy định bảo mật sẽ bị xử lý theo quy chế của Ngân hàng và pháp luật hiện hành.",
		"5. Nếu bạn nhận được tài liệu này do nhầm lẫn, vui lòng thông báo ngay cho bộ phận IT và tiêu hủy tất cả các bản sao.",
	}

	_ = f.SetCellValue(sheet, fmt.Sprintf("A%d", row), "Quy định bảo mật:")
	_ = f.SetCellStyle(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("A%d", row), headerStyle)
	row++

	for _, rule := range rules {
		_ = f.MergeCell(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("B%d", row))
		_ = f.SetCellValue(sheet, fmt.Sprintf("A%d", row), rule)
		_ = f.SetCellStyle(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("B%d", row), ruleStyle)
		_ = f.SetRowHeight(sheet, row, 40)
		row++
	}

	return nil
}
