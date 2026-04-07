package bond

import (
	"fmt"
	"sort"
	"time"

	"github.com/xuri/excelize/v2"

	"github.com/kienlongbank/treasury-api/internal/model"
)

// BondReportBuilder builds Excel sheets for Bond deal export.
type BondReportBuilder struct {
	deals []model.BondDeal
}

// NewBondReportBuilder creates a new BondReportBuilder.
func NewBondReportBuilder(deals []model.BondDeal) *BondReportBuilder {
	return &BondReportBuilder{deals: deals}
}

// Module returns the module name.
func (b *BondReportBuilder) Module() string { return "BOND" }

// ReportType returns the report type.
func (b *BondReportBuilder) ReportType() string { return "BOND_DEALS" }

// RecordCount returns the number of deals.
func (b *BondReportBuilder) RecordCount() int { return len(b.deals) }

// bondDetailSheetName is the canonical name for the detail sheet, used by both detail and pivot.
const bondDetailSheetName = "Chi tiết giao dịch"

// BuildSheets creates the Dashboard, Detail, and Pivot Data sheets.
func (b *BondReportBuilder) BuildSheets(f *excelize.File) error {
	if err := b.buildDetail(f); err != nil {
		return fmt.Errorf("build detail: %w", err)
	}
	if err := b.buildDashboard(f); err != nil {
		return fmt.Errorf("build dashboard: %w", err)
	}
	if err := b.buildPivotData(f); err != nil {
		return fmt.Errorf("build pivot: %w", err)
	}
	return nil
}

// ============================================================
// Dashboard — KPIs + 4 native Excel charts
// ============================================================

func (b *BondReportBuilder) buildDashboard(f *excelize.File) error {
	sheet := "Dashboard"
	if _, err := f.NewSheet(sheet); err != nil {
		return err
	}

	// Styles
	titleStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 14, Color: "#FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"#1F3864"}},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})
	kpiHeaderStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 10, Color: "#FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"#2F5496"}},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "#000000", Style: 1},
			{Type: "right", Color: "#000000", Style: 1},
			{Type: "top", Color: "#000000", Style: 1},
			{Type: "bottom", Color: "#000000", Style: 1},
		},
	})
	kpiValueStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 16, Color: "#2F5496"},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "#2F5496", Style: 1},
			{Type: "right", Color: "#2F5496", Style: 1},
			{Type: "top", Color: "#2F5496", Style: 1},
			{Type: "bottom", Color: "#2F5496", Style: 1},
		},
	})
	sectionStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 11, Color: "#1F3864"},
		Alignment: &excelize.Alignment{Vertical: "center"},
		Border:    []excelize.Border{{Type: "bottom", Color: "#2F5496", Style: 2}},
	})

	_ = f.SetColWidth(sheet, "A", "H", 16)
	_ = f.SetRowHeight(sheet, 1, 30)

	// Row 1: Title
	_ = f.MergeCell(sheet, "A1", "H1")
	_ = f.SetCellValue(sheet, "A1", "TREASURY BOND — BẢNG TỔNG HỢP")
	_ = f.SetCellStyle(sheet, "A1", "H1", titleStyle)

	// Row 3-4: KPI cards — using COUNTIF formulas referencing detail sheet
	lastRow := len(b.deals) + 1
	detailRef := fmt.Sprintf("'%s'", bondDetailSheetName)
	colC := "C" // Loại (BondCategory)
	colD := "D" // Chiều GD (Direction)

	kpis := []struct {
		header  string
		formula string
	}{
		{"Tổng giao dịch", fmt.Sprintf("COUNTA(%s!B2:B%d)", detailRef, lastRow)},
		{"Govi", fmt.Sprintf("COUNTIF(%s!%s2:%s%d,\"GOVERNMENT\")", detailRef, colC, colC, lastRow)},
		{"FI", fmt.Sprintf("COUNTIF(%s!%s2:%s%d,\"FINANCIAL_INSTITUTION\")", detailRef, colC, colC, lastRow)},
		{"CCTG", fmt.Sprintf("COUNTIF(%s!%s2:%s%d,\"CERTIFICATE_OF_DEPOSIT\")", detailRef, colC, colC, lastRow)},
		{"Mua (Buy)", fmt.Sprintf("COUNTIF(%s!%s2:%s%d,\"BUY\")", detailRef, colD, colD, lastRow)},
		{"Bán (Sell)", fmt.Sprintf("COUNTIF(%s!%s2:%s%d,\"SELL\")", detailRef, colD, colD, lastRow)},
	}

	for i, kpi := range kpis {
		col, _ := excelize.CoordinatesToCellName(i+1, 3)
		_ = f.SetCellValue(sheet, col, kpi.header)
		_ = f.SetCellStyle(sheet, col, col, kpiHeaderStyle)

		valCell, _ := excelize.CoordinatesToCellName(i+1, 4)
		_ = f.SetCellFormula(sheet, valCell, kpi.formula)
		_ = f.SetCellStyle(sheet, valCell, valCell, kpiValueStyle)
	}

	// ---- Chart data areas ----

	// Section: Volume by date (row 6+)
	row := 6
	_ = f.SetCellValue(sheet, fmt.Sprintf("A%d", row), "Khối lượng theo ngày giao dịch")
	_ = f.SetCellStyle(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("B%d", row), sectionStyle)
	row++

	dateVolume := b.aggregateByDate()
	dateDataStart := row
	for _, dv := range dateVolume {
		_ = f.SetCellValue(sheet, fmt.Sprintf("A%d", row), dv.label)
		_ = f.SetCellValue(sheet, fmt.Sprintf("B%d", row), dv.count)
		row++
	}
	dateDataEnd := row - 1

	// Chart 1: Volume by Date — Column chart
	if dateDataEnd >= dateDataStart {
		_ = f.AddChart(sheet, fmt.Sprintf("D%d", 6), &excelize.Chart{
			Type: excelize.Col,
			Series: []excelize.ChartSeries{
				{
					Name:       "Số giao dịch",
					Categories: fmt.Sprintf("%s!$A$%d:$A$%d", sheet, dateDataStart, dateDataEnd),
					Values:     fmt.Sprintf("%s!$B$%d:$B$%d", sheet, dateDataStart, dateDataEnd),
					Fill: excelize.Fill{
						Type:  "pattern",
						Color: []string{"#2F5496"},
					},
				},
			},
			Title:     []excelize.RichTextRun{{Text: "Khối lượng giao dịch theo ngày"}},
			Legend:    excelize.ChartLegend{Position: "bottom", ShowLegendKey: false},
			Dimension: excelize.ChartDimension{Width: 480, Height: 290},
			PlotArea:  excelize.ChartPlotArea{ShowVal: true},
		})
	}

	// Section: Distribution by bond category
	row += 1
	catSectionRow := row
	_ = f.SetCellValue(sheet, fmt.Sprintf("A%d", row), "Phân bổ theo loại trái phiếu")
	_ = f.SetCellStyle(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("B%d", row), sectionStyle)
	row++

	catDist := b.aggregateByCategory()
	catDataStart := row
	for _, cd := range catDist {
		_ = f.SetCellValue(sheet, fmt.Sprintf("A%d", row), cd.label)
		_ = f.SetCellValue(sheet, fmt.Sprintf("B%d", row), cd.count)
		row++
	}
	catDataEnd := row - 1

	// Chart 2: Category Distribution — Pie chart
	if catDataEnd >= catDataStart {
		_ = f.AddChart(sheet, fmt.Sprintf("D%d", catSectionRow), &excelize.Chart{
			Type: excelize.Pie,
			Series: []excelize.ChartSeries{
				{
					Name:       "Loại trái phiếu",
					Categories: fmt.Sprintf("%s!$A$%d:$A$%d", sheet, catDataStart, catDataEnd),
					Values:     fmt.Sprintf("%s!$B$%d:$B$%d", sheet, catDataStart, catDataEnd),
				},
			},
			Title:     []excelize.RichTextRun{{Text: "Phân bổ theo loại trái phiếu"}},
			Legend:    excelize.ChartLegend{Position: "right", ShowLegendKey: false},
			Dimension: excelize.ChartDimension{Width: 480, Height: 290},
			PlotArea: excelize.ChartPlotArea{
				ShowPercent: true,
				ShowCatName: true,
			},
		})
	}

	// Section: Portfolio type distribution
	row += 1
	portfolioSectionRow := row
	_ = f.SetCellValue(sheet, fmt.Sprintf("A%d", row), "Phân bổ theo danh mục đầu tư")
	_ = f.SetCellStyle(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("B%d", row), sectionStyle)
	row++

	portfolioDist := b.aggregateByPortfolio()
	portfolioDataStart := row
	for _, pd := range portfolioDist {
		_ = f.SetCellValue(sheet, fmt.Sprintf("A%d", row), pd.label)
		_ = f.SetCellValue(sheet, fmt.Sprintf("B%d", row), pd.count)
		row++
	}
	portfolioDataEnd := row - 1

	// Chart 3: Portfolio Type — Bar chart
	if portfolioDataEnd >= portfolioDataStart {
		_ = f.AddChart(sheet, fmt.Sprintf("D%d", portfolioSectionRow), &excelize.Chart{
			Type: excelize.Bar,
			Series: []excelize.ChartSeries{
				{
					Name:       "Số giao dịch",
					Categories: fmt.Sprintf("%s!$A$%d:$A$%d", sheet, portfolioDataStart, portfolioDataEnd),
					Values:     fmt.Sprintf("%s!$B$%d:$B$%d", sheet, portfolioDataStart, portfolioDataEnd),
					Fill: excelize.Fill{
						Type:  "pattern",
						Color: []string{"#EF7922"},
					},
				},
			},
			Title:     []excelize.RichTextRun{{Text: "Phân bổ theo danh mục đầu tư"}},
			Legend:    excelize.ChartLegend{Position: "bottom", ShowLegendKey: false},
			Dimension: excelize.ChartDimension{Width: 480, Height: 250},
			PlotArea:  excelize.ChartPlotArea{ShowVal: true},
		})
	}

	// Section: Buy vs Sell
	row += 1
	dirSectionRow := row
	_ = f.SetCellValue(sheet, fmt.Sprintf("A%d", row), "Tỷ lệ Mua / Bán")
	_ = f.SetCellStyle(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("B%d", row), sectionStyle)
	row++

	dirDist := b.aggregateByDirection()
	dirDataStart := row
	for _, dd := range dirDist {
		_ = f.SetCellValue(sheet, fmt.Sprintf("A%d", row), dd.label)
		_ = f.SetCellValue(sheet, fmt.Sprintf("B%d", row), dd.count)
		row++
	}
	dirDataEnd := row - 1

	// Chart 4: Buy vs Sell — Doughnut chart
	if dirDataEnd >= dirDataStart {
		_ = f.AddChart(sheet, fmt.Sprintf("D%d", dirSectionRow), &excelize.Chart{
			Type: excelize.Doughnut,
			Series: []excelize.ChartSeries{
				{
					Name:       "Chiều giao dịch",
					Categories: fmt.Sprintf("%s!$A$%d:$A$%d", sheet, dirDataStart, dirDataEnd),
					Values:     fmt.Sprintf("%s!$B$%d:$B$%d", sheet, dirDataStart, dirDataEnd),
				},
			},
			Title:     []excelize.RichTextRun{{Text: "Tỷ lệ Mua / Bán"}},
			Legend:    excelize.ChartLegend{Position: "right", ShowLegendKey: false},
			Dimension: excelize.ChartDimension{Width: 480, Height: 250},
			PlotArea: excelize.ChartPlotArea{
				ShowPercent: true,
				ShowCatName: true,
			},
		})
	}

	return nil
}

// ============================================================
// Detail — raw data (source of truth)
// ============================================================

func (b *BondReportBuilder) buildDetail(f *excelize.File) error {
	sheet := bondDetailSheetName
	if _, err := f.NewSheet(sheet); err != nil {
		return err
	}

	colWidths := map[string]float64{
		"A": 5, "B": 18, "C": 22, "D": 10, "E": 25,
		"F": 16, "G": 16, "H": 20, "I": 14, "J": 14,
		"K": 14, "L": 10, "M": 14, "N": 16, "O": 16,
		"P": 20, "Q": 12, "R": 14, "S": 14, "T": 20,
		"U": 14, "V": 16,
	}
	for col, w := range colWidths {
		_ = f.SetColWidth(sheet, col, col, w)
	}

	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 10, Color: "#FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"#2F5496"}},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true},
		Border: []excelize.Border{
			{Type: "left", Color: "#000000", Style: 1},
			{Type: "right", Color: "#000000", Style: 1},
			{Type: "top", Color: "#000000", Style: 1},
			{Type: "bottom", Color: "#000000", Style: 1},
		},
	})

	evenStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Size: 10},
		Alignment: &excelize.Alignment{Vertical: "center"},
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"#D6E4F0"}},
		Border: []excelize.Border{
			{Type: "left", Color: "#CCCCCC", Style: 1},
			{Type: "right", Color: "#CCCCCC", Style: 1},
			{Type: "bottom", Color: "#CCCCCC", Style: 1},
		},
	})

	oddStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Size: 10},
		Alignment: &excelize.Alignment{Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "#CCCCCC", Style: 1},
			{Type: "right", Color: "#CCCCCC", Style: 1},
			{Type: "bottom", Color: "#CCCCCC", Style: 1},
		},
	})

	headers := []string{
		"STT", "Mã GD", "Loại", "Chiều GD", "Đối tác",
		"Loại GD", "Mã TP", "Tổ chức phát hành", "Lãi suất coupon", "Ngày phát hành",
		"Ngày đáo hạn", "Số lượng", "Mệnh giá", "Giá sạch", "Giá thanh toán",
		"Tổng giá trị", "Danh mục", "Ngày TT", "Kỳ hạn còn lại",
		"Trạng thái", "Ngày GD", "Người tạo",
	}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		_ = f.SetCellValue(sheet, cell, h)
		_ = f.SetCellStyle(sheet, cell, cell, headerStyle)
	}

	for i, deal := range b.deals {
		row := i + 2
		style := oddStyle
		if i%2 == 0 {
			style = evenStyle
		}

		portfolioType := ""
		if deal.PortfolioType != nil {
			portfolioType = *deal.PortfolioType
		}

		issueDateStr := ""
		if deal.IssueDate != nil {
			issueDateStr = deal.IssueDate.Format("02/01/2006")
		}

		values := []interface{}{
			i + 1,
			deal.DealNumber,
			deal.BondCategory,
			deal.Direction,
			deal.CounterpartyName,
			deal.TransactionType,
			deal.BondCode(),
			deal.Issuer,
			deal.CouponRate.InexactFloat64(),
			issueDateStr,
			deal.MaturityDate.Format("02/01/2006"),
			deal.Quantity,
			deal.FaceValue.Round(2).InexactFloat64(),
			deal.CleanPrice.Round(2).InexactFloat64(),
			deal.SettlementPrice.Round(2).InexactFloat64(),
			deal.TotalValue.Round(2).InexactFloat64(),
			portfolioType,
			deal.PaymentDate.Format("02/01/2006"),
			deal.RemainingTenorDays,
			deal.Status,
			deal.TradeDate.Format("02/01/2006"),
			deal.CreatedByName,
		}

		for j, v := range values {
			cell, _ := excelize.CoordinatesToCellName(j+1, row)
			_ = f.SetCellValue(sheet, cell, v)
			_ = f.SetCellStyle(sheet, cell, cell, style)
		}
	}

	// Number format for monetary columns: M=Mệnh giá, N=Giá sạch, O=Giá thanh toán, P=Tổng giá trị
	numStyle, _ := f.NewStyle(&excelize.Style{
		NumFmt:    4, // #,##0.00
		Font:      &excelize.Font{Size: 10},
		Alignment: &excelize.Alignment{Vertical: "center", Horizontal: "right"},
	})
	for i := range b.deals {
		for _, col := range []string{"M", "N", "O", "P"} {
			cell := fmt.Sprintf("%s%d", col, i+2)
			_ = f.SetCellStyle(sheet, cell, cell, numStyle)
		}
	}

	// Auto filter
	if len(b.deals) > 0 {
		lastCell, _ := excelize.CoordinatesToCellName(len(headers), len(b.deals)+1)
		_ = f.AutoFilter(sheet, "A1:"+lastCell, nil)
	}

	// Freeze header row
	_ = f.SetPanes(sheet, &excelize.Panes{
		Freeze:      true,
		Split:       false,
		XSplit:      0,
		YSplit:      1,
		TopLeftCell: "A2",
		ActivePane:  "bottomLeft",
	})

	return nil
}

// ============================================================
// Pivot Data — formulas referencing Detail sheet
// ============================================================

func (b *BondReportBuilder) buildPivotData(f *excelize.File) error {
	sheet := "Pivot Data"
	if _, err := f.NewSheet(sheet); err != nil {
		return err
	}

	pivotHeaderStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 10, Color: "#FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"#548235"}},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "#000000", Style: 1},
			{Type: "right", Color: "#000000", Style: 1},
			{Type: "top", Color: "#000000", Style: 1},
			{Type: "bottom", Color: "#000000", Style: 1},
		},
	})

	headers := []string{
		"Ngày GD", "Loại", "Chiều GD", "Đối tác",
		"Mã TP", "Tổng giá trị", "Danh mục", "Trạng thái",
		"Ngày TT", "Người tạo",
	}

	// Column mapping: pivot col -> detail col letter
	// Detail headers: A=STT, B=Mã GD, C=Loại, D=Chiều GD, E=Đối tác,
	//                 F=Loại GD, G=Mã TP, H=Tổ chức phát hành, I=Lãi suất coupon,
	//                 J=Ngày phát hành, K=Ngày đáo hạn, L=Số lượng, M=Mệnh giá,
	//                 N=Giá sạch, O=Giá thanh toán, P=Tổng giá trị, Q=Danh mục,
	//                 R=Ngày TT, S=Kỳ hạn còn lại, T=Trạng thái, U=Ngày GD, V=Người tạo
	detailCols := []string{"U", "C", "D", "E", "G", "P", "Q", "T", "R", "V"}

	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		_ = f.SetCellValue(sheet, cell, h)
		_ = f.SetCellStyle(sheet, cell, cell, pivotHeaderStyle)
	}

	_ = f.SetColWidth(sheet, "A", "J", 16)

	// Formula rows — each cell references the detail sheet
	for i := 0; i < len(b.deals); i++ {
		dataRow := i + 2
		for j, detailCol := range detailCols {
			pivotCell, _ := excelize.CoordinatesToCellName(j+1, dataRow)
			formula := fmt.Sprintf("'%s'!%s%d", bondDetailSheetName, detailCol, dataRow)
			_ = f.SetCellFormula(sheet, pivotCell, formula)
		}
	}

	// Auto filter on pivot
	if len(b.deals) > 0 {
		lastCell, _ := excelize.CoordinatesToCellName(len(headers), len(b.deals)+1)
		_ = f.AutoFilter(sheet, "A1:"+lastCell, nil)
	}

	return nil
}

// ============================================================
// Aggregation helpers
// ============================================================

type bondAggregate struct {
	label string
	count int
}

func (b *BondReportBuilder) aggregateByDate() []bondAggregate {
	m := make(map[string]int)
	for _, d := range b.deals {
		key := d.TradeDate.Format("02/01/2006")
		m[key]++
	}
	return sortedBondAggregates(m)
}

func (b *BondReportBuilder) aggregateByCategory() []bondAggregate {
	m := make(map[string]int)
	for _, d := range b.deals {
		m[d.BondCategory]++
	}
	return sortedBondAggregates(m)
}

func (b *BondReportBuilder) aggregateByPortfolio() []bondAggregate {
	m := make(map[string]int)
	for _, d := range b.deals {
		pt := "N/A"
		if d.PortfolioType != nil && *d.PortfolioType != "" {
			pt = *d.PortfolioType
		}
		m[pt]++
	}
	return sortedBondAggregates(m)
}

func (b *BondReportBuilder) aggregateByDirection() []bondAggregate {
	m := make(map[string]int)
	for _, d := range b.deals {
		switch d.Direction {
		case "BUY":
			m["Mua (Buy)"]++
		case "SELL":
			m["Bán (Sell)"]++
		default:
			m[d.Direction]++
		}
	}
	return sortedBondAggregates(m)
}

func sortedBondAggregates(m map[string]int) []bondAggregate {
	result := make([]bondAggregate, 0, len(m))
	for k, v := range m {
		result = append(result, bondAggregate{label: k, count: v})
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].count > result[j].count // descending
	})
	return result
}

// Ensure time import is used (date formatting in aggregateByDate).
var _ = time.Now
