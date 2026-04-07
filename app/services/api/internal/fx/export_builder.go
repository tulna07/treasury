package fx

import (
	"fmt"
	"sort"
	"time"

	"github.com/xuri/excelize/v2"

	"github.com/kienlongbank/treasury-api/internal/model"
)

// FXReportBuilder builds Excel sheets for FX deal export.
type FXReportBuilder struct {
	deals []model.FxDeal
}

// NewFXReportBuilder creates a new FXReportBuilder.
func NewFXReportBuilder(deals []model.FxDeal) *FXReportBuilder {
	return &FXReportBuilder{deals: deals}
}

// Module returns the module name.
func (b *FXReportBuilder) Module() string { return "FX" }

// ReportType returns the report type.
func (b *FXReportBuilder) ReportType() string { return "FX_DEALS" }

// RecordCount returns the number of deals.
func (b *FXReportBuilder) RecordCount() int { return len(b.deals) }

// detailSheetName is the canonical name for the detail sheet, used by both detail and pivot.
const detailSheetName = "Chi tiết giao dịch"

// BuildSheets creates the Dashboard, Detail, and Pivot Data sheets.
func (b *FXReportBuilder) BuildSheets(f *excelize.File) error {
	// Detail first — Dashboard charts and Pivot formulas reference this sheet
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

func (b *FXReportBuilder) buildDashboard(f *excelize.File) error {
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
	_ = f.SetCellValue(sheet, "A1", "TREASURY FX — BẢNG TỔNG HỢP")
	_ = f.SetCellStyle(sheet, "A1", "H1", titleStyle)

	// Row 3-4: KPI cards — using COUNTIF formulas referencing detail sheet
	lastRow := len(b.deals) + 1
	detailRef := fmt.Sprintf("'%s'", detailSheetName)
	colC := "C" // Loại (DealType)
	colD := "D" // Chiều (Direction)

	kpis := []struct {
		header  string
		formula string
	}{
		{"Tổng giao dịch", fmt.Sprintf("COUNTA(%s!B2:B%d)", detailRef, lastRow)},
		{"Spot", fmt.Sprintf("COUNTIF(%s!%s2:%s%d,\"SPOT\")", detailRef, colC, colC, lastRow)},
		{"Forward", fmt.Sprintf("COUNTIF(%s!%s2:%s%d,\"FORWARD\")", detailRef, colC, colC, lastRow)},
		{"Swap", fmt.Sprintf("COUNTIF(%s!%s2:%s%d,\"SWAP\")", detailRef, colC, colC, lastRow)},
		{"Mua (Buy)", fmt.Sprintf("COUNTIF(%s!%s2:%s%d,\"BUY\")+COUNTIF(%s!%s2:%s%d,\"BUY_SELL\")", detailRef, colD, colD, lastRow, detailRef, colD, colD, lastRow)},
		{"Bán (Sell)", fmt.Sprintf("COUNTIF(%s!%s2:%s%d,\"SELL\")+COUNTIF(%s!%s2:%s%d,\"SELL_BUY\")", detailRef, colD, colD, lastRow, detailRef, colD, colD, lastRow)},
	}

	for i, kpi := range kpis {
		col, _ := excelize.CoordinatesToCellName(i+1, 3)
		_ = f.SetCellValue(sheet, col, kpi.header)
		_ = f.SetCellStyle(sheet, col, col, kpiHeaderStyle)

		valCell, _ := excelize.CoordinatesToCellName(i+1, 4)
		_ = f.SetCellFormula(sheet, valCell, kpi.formula)
		_ = f.SetCellStyle(sheet, valCell, valCell, kpiValueStyle)
	}

	// ---- Chart data areas (hidden summary tables for charts) ----

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
			Title: []excelize.RichTextRun{{Text: "Khối lượng giao dịch theo ngày"}},
			Legend: excelize.ChartLegend{Position: "bottom", ShowLegendKey: false},
			Dimension: excelize.ChartDimension{Width: 480, Height: 290},
			PlotArea: excelize.ChartPlotArea{
				ShowVal: true,
			},
		})
	}

	// Section: Distribution by pair (below date data)
	row += 1
	pairSectionRow := row
	_ = f.SetCellValue(sheet, fmt.Sprintf("A%d", row), "Phân bổ theo cặp tiền tệ")
	_ = f.SetCellStyle(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("B%d", row), sectionStyle)
	row++

	pairDist := b.aggregateByPair()
	pairDataStart := row
	for _, pd := range pairDist {
		_ = f.SetCellValue(sheet, fmt.Sprintf("A%d", row), pd.label)
		_ = f.SetCellValue(sheet, fmt.Sprintf("B%d", row), pd.count)
		row++
	}
	pairDataEnd := row - 1

	// Chart 2: Pair Distribution — Pie chart
	if pairDataEnd >= pairDataStart {
		_ = f.AddChart(sheet, fmt.Sprintf("D%d", pairSectionRow), &excelize.Chart{
			Type: excelize.Pie,
			Series: []excelize.ChartSeries{
				{
					Name:       "Cặp tiền tệ",
					Categories: fmt.Sprintf("%s!$A$%d:$A$%d", sheet, pairDataStart, pairDataEnd),
					Values:     fmt.Sprintf("%s!$B$%d:$B$%d", sheet, pairDataStart, pairDataEnd),
				},
			},
			Title: []excelize.RichTextRun{{Text: "Phân bổ theo cặp tiền tệ"}},
			Legend: excelize.ChartLegend{Position: "right", ShowLegendKey: false},
			Dimension: excelize.ChartDimension{Width: 480, Height: 290},
			PlotArea: excelize.ChartPlotArea{
				ShowPercent:    true,
				ShowCatName:    true,
				ShowBubbleSize: false,
			},
		})
	}

	// Section: Deal type distribution
	row += 1
	typeSectionRow := row
	_ = f.SetCellValue(sheet, fmt.Sprintf("A%d", row), "Phân bổ theo loại giao dịch")
	_ = f.SetCellStyle(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("B%d", row), sectionStyle)
	row++

	typeDist := b.aggregateByType()
	typeDataStart := row
	for _, td := range typeDist {
		_ = f.SetCellValue(sheet, fmt.Sprintf("A%d", row), td.label)
		_ = f.SetCellValue(sheet, fmt.Sprintf("B%d", row), td.count)
		row++
	}
	typeDataEnd := row - 1

	// Chart 3: Deal Type — Bar chart
	if typeDataEnd >= typeDataStart {
		_ = f.AddChart(sheet, fmt.Sprintf("D%d", typeSectionRow), &excelize.Chart{
			Type: excelize.Bar,
			Series: []excelize.ChartSeries{
				{
					Name:       "Số giao dịch",
					Categories: fmt.Sprintf("%s!$A$%d:$A$%d", sheet, typeDataStart, typeDataEnd),
					Values:     fmt.Sprintf("%s!$B$%d:$B$%d", sheet, typeDataStart, typeDataEnd),
					Fill: excelize.Fill{
						Type:  "pattern",
						Color: []string{"#EF7922"},
					},
				},
			},
			Title:     []excelize.RichTextRun{{Text: "Phân bổ theo loại giao dịch"}},
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

func (b *FXReportBuilder) buildDetail(f *excelize.File) error {
	sheet := detailSheetName
	if _, err := f.NewSheet(sheet); err != nil {
		return err
	}

	colWidths := map[string]float64{
		"A": 5, "B": 18, "C": 10, "D": 10, "E": 14,
		"F": 18, "G": 14, "H": 14, "I": 14,
		"J": 25, "K": 20, "L": 16, "M": 18,
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
		"STT", "Mã GD", "Loại", "Chiều", "Cặp tiền tệ",
		"Số tiền", "Tỷ giá", "Ngày GD", "Ngày giá trị",
		"Đối tác", "Trạng thái", "Người tạo", "Ngày tạo",
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

		pairCode := deal.PairCode
		if pairCode == "" && len(deal.Legs) > 0 {
			pairCode = deal.Legs[0].BuyCurrency + "/" + deal.Legs[0].SellCurrency
		}

		var rate interface{}
		var valueDate string
		if len(deal.Legs) > 0 {
			rate = deal.Legs[0].ExchangeRate.Round(2).InexactFloat64()
			valueDate = deal.Legs[0].ValueDate.Format("02/01/2006")
		}

		ticketNum := ""
		if deal.TicketNumber != nil {
			ticketNum = *deal.TicketNumber
		}

		values := []interface{}{
			i + 1,
			ticketNum,
			deal.DealType,
			deal.Direction,
			pairCode,
			deal.NotionalAmount.Round(2).InexactFloat64(),
			rate,
			deal.TradeDate.Format("02/01/2006"),
			valueDate,
			deal.CounterpartyName,
			deal.Status,
			deal.CreatedBy.String(),
			deal.CreatedAt.Format("02/01/2006 15:04"),
		}

		for j, v := range values {
			cell, _ := excelize.CoordinatesToCellName(j+1, row)
			_ = f.SetCellValue(sheet, cell, v)
			_ = f.SetCellStyle(sheet, cell, cell, style)
		}
	}

	// Number format for Số tiền column (F)
	numStyle, _ := f.NewStyle(&excelize.Style{
		NumFmt:    4, // #,##0.00
		Font:      &excelize.Font{Size: 10},
		Alignment: &excelize.Alignment{Vertical: "center", Horizontal: "right"},
	})
	for i := range b.deals {
		cell := fmt.Sprintf("F%d", i+2)
		_ = f.SetCellStyle(sheet, cell, cell, numStyle)
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

func (b *FXReportBuilder) buildPivotData(f *excelize.File) error {
	sheet := "Pivot Data"
	if _, err := f.NewSheet(sheet); err != nil {
		return err
	}

	// Header style
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
		"Ngày GD", "Loại", "Chiều", "Cặp tiền tệ",
		"Số tiền", "Tỷ giá", "Trạng thái", "Đối tác",
		"Ngày giá trị", "Ngày tạo",
	}

	// Column mapping: pivot col -> detail col letter
	// Detail headers: A=STT, B=Mã GD, C=Loại, D=Chiều, E=Cặp tiền tệ,
	//                 F=Số tiền, G=Tỷ giá, H=Ngày GD, I=Ngày giá trị,
	//                 J=Đối tác, K=Trạng thái, L=Người tạo, M=Ngày tạo
	detailCols := []string{"H", "C", "D", "E", "F", "G", "K", "J", "I", "M"}

	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		_ = f.SetCellValue(sheet, cell, h)
		_ = f.SetCellStyle(sheet, cell, cell, pivotHeaderStyle)
	}

	_ = f.SetColWidth(sheet, "A", "J", 16)

	// Formula rows — each cell references the detail sheet
	for i := 0; i < len(b.deals); i++ {
		dataRow := i + 2 // detail data starts at row 2
		for j, detailCol := range detailCols {
			pivotCell, _ := excelize.CoordinatesToCellName(j+1, dataRow)
			formula := fmt.Sprintf("'%s'!%s%d", detailSheetName, detailCol, dataRow)
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

type aggregate struct {
	label string
	count int
}

func (b *FXReportBuilder) aggregateByDate() []aggregate {
	m := make(map[string]int)
	for _, d := range b.deals {
		key := d.TradeDate.Format("02/01/2006")
		m[key]++
	}
	return sortedAggregates(m)
}

func (b *FXReportBuilder) aggregateByPair() []aggregate {
	m := make(map[string]int)
	for _, d := range b.deals {
		pair := d.PairCode
		if pair == "" && len(d.Legs) > 0 {
			pair = d.Legs[0].BuyCurrency + "/" + d.Legs[0].SellCurrency
		}
		m[pair]++
	}
	return sortedAggregates(m)
}

func (b *FXReportBuilder) aggregateByType() []aggregate {
	m := make(map[string]int)
	for _, d := range b.deals {
		m[d.DealType]++
	}
	return sortedAggregates(m)
}

func (b *FXReportBuilder) aggregateByDirection() []aggregate {
	m := make(map[string]int)
	for _, d := range b.deals {
		switch d.Direction {
		case "BUY", "BUY_SELL":
			m["Mua (Buy)"]++
		case "SELL", "SELL_BUY":
			m["Bán (Sell)"]++
		default:
			m[d.Direction]++
		}
	}
	return sortedAggregates(m)
}

func sortedAggregates(m map[string]int) []aggregate {
	result := make([]aggregate, 0, len(m))
	for k, v := range m {
		result = append(result, aggregate{label: k, count: v})
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].count > result[j].count // descending
	})
	return result
}

// Ensure time import is used (date formatting in aggregateByDate).
var _ = time.Now
