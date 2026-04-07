package mm

import (
	"fmt"

	"github.com/xuri/excelize/v2"

	"github.com/kienlongbank/treasury-api/pkg/dto"
)

// InterbankReportBuilder builds Excel sheets for MM Interbank deal export.
type InterbankReportBuilder struct {
	deals []dto.MMInterbankResponse
}

// NewInterbankReportBuilder creates a new InterbankReportBuilder.
func NewInterbankReportBuilder(deals []dto.MMInterbankResponse) *InterbankReportBuilder {
	return &InterbankReportBuilder{deals: deals}
}

// Module returns the module name.
func (b *InterbankReportBuilder) Module() string { return "MM_INTERBANK" }

// ReportType returns the report type.
func (b *InterbankReportBuilder) ReportType() string { return "MM_INTERBANK_DEALS" }

// RecordCount returns the number of deals.
func (b *InterbankReportBuilder) RecordCount() int { return len(b.deals) }

// BuildSheets creates the MM Interbank detail sheet.
func (b *InterbankReportBuilder) BuildSheets(f *excelize.File) error {
	sheet := "MM Interbank"
	if _, err := f.NewSheet(sheet); err != nil {
		return fmt.Errorf("create sheet: %w", err)
	}

	// Header style
	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 11, Color: "FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"4472C4"}},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true},
		Border: []excelize.Border{
			{Type: "left", Color: "D9D9D9", Style: 1},
			{Type: "right", Color: "D9D9D9", Style: 1},
			{Type: "top", Color: "D9D9D9", Style: 1},
			{Type: "bottom", Color: "D9D9D9", Style: 1},
		},
	})

	// Data style
	dataStyle, _ := f.NewStyle(&excelize.Style{
		Alignment: &excelize.Alignment{Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "D9D9D9", Style: 1},
			{Type: "right", Color: "D9D9D9", Style: 1},
			{Type: "top", Color: "D9D9D9", Style: 1},
			{Type: "bottom", Color: "D9D9D9", Style: 1},
		},
	})

	// Number style (for amounts)
	numStyle, _ := f.NewStyle(&excelize.Style{
		NumFmt:    4, // #,##0.00
		Alignment: &excelize.Alignment{Vertical: "center", Horizontal: "right"},
		Border: []excelize.Border{
			{Type: "left", Color: "D9D9D9", Style: 1},
			{Type: "right", Color: "D9D9D9", Style: 1},
			{Type: "top", Color: "D9D9D9", Style: 1},
			{Type: "bottom", Color: "D9D9D9", Style: 1},
		},
	})

	// Percentage style
	pctStyle, _ := f.NewStyle(&excelize.Style{
		NumFmt:    10, // 0.00%
		Alignment: &excelize.Alignment{Vertical: "center", Horizontal: "right"},
		Border: []excelize.Border{
			{Type: "left", Color: "D9D9D9", Style: 1},
			{Type: "right", Color: "D9D9D9", Style: 1},
			{Type: "top", Color: "D9D9D9", Style: 1},
			{Type: "bottom", Color: "D9D9D9", Style: 1},
		},
	})

	// Headers
	headers := []string{
		"Mã GD", "Đối tác", "Chiều GD", "Loại tiền", "Số tiền gốc",
		"Lãi suất", "Kỳ hạn", "Ngày GD", "Ngày hiệu lực",
		"Ngày đáo hạn", "Tiền lãi", "Tổng đáo hạn", "Trạng thái",
	}

	// Column widths
	colWidths := map[string]float64{
		"A": 15, "B": 25, "C": 12, "D": 10, "E": 20,
		"F": 10, "G": 10, "H": 14, "I": 14,
		"J": 14, "K": 20, "L": 20, "M": 18,
	}
	for col, w := range colWidths {
		_ = f.SetColWidth(sheet, col, col, w)
	}

	// Write header row
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		_ = f.SetCellValue(sheet, cell, h)
		_ = f.SetCellStyle(sheet, cell, cell, headerStyle)
	}

	// Write data rows
	for rowIdx, deal := range b.deals {
		row := rowIdx + 2

		// Convert rate to float for percentage formatting
		rate, _ := deal.InterestRate.Float64()
		principal, _ := deal.PrincipalAmount.Float64()
		interest, _ := deal.InterestAmount.Float64()
		maturity, _ := deal.MaturityAmount.Float64()

		vals := []any{
			deal.DealNumber,
			deal.CounterpartyName,
			deal.Direction,
			deal.CurrencyCode,
			principal,
			rate / 100, // Convert percentage to decimal for Excel pct format
			deal.TenorDays,
			deal.TradeDate.Format("02/01/2006"),
			deal.EffectiveDate.Format("02/01/2006"),
			deal.MaturityDate.Format("02/01/2006"),
			interest,
			maturity,
			deal.Status,
		}

		for colIdx, v := range vals {
			cell, _ := excelize.CoordinatesToCellName(colIdx+1, row)
			_ = f.SetCellValue(sheet, cell, v)

			// Apply appropriate style
			switch colIdx {
			case 4, 10, 11: // amounts
				_ = f.SetCellStyle(sheet, cell, cell, numStyle)
			case 5: // rate
				_ = f.SetCellStyle(sheet, cell, cell, pctStyle)
			default:
				_ = f.SetCellStyle(sheet, cell, cell, dataStyle)
			}
		}
	}

	return nil
}
