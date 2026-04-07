package fx

import (
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xuri/excelize/v2"

	"github.com/kienlongbank/treasury-api/internal/model"
)

func makeSampleDeals() []model.FxDeal {
	return []model.FxDeal{
		{
			ID:               uuid.New(),
			TicketNumber:     ptrString("FX-001"),
			CounterpartyName: "ACB",
			DealType:         "SPOT",
			Direction:        "BUY",
			NotionalAmount:   decimal.NewFromInt(1000000),
			CurrencyCode:     "USD",
			PairCode:         "USD/VND",
			TradeDate:        time.Now(),
			Status:           "COMPLETED",
			CreatedBy:        uuid.New(),
			CreatedAt:        time.Now(),
			Legs: []model.FxDealLeg{
				{
					LegNumber:    1,
					ValueDate:    time.Now().AddDate(0, 0, 2),
					ExchangeRate: decimal.NewFromFloat(24500),
					BuyCurrency:  "USD",
					SellCurrency: "VND",
					BuyAmount:    decimal.NewFromInt(1000000),
					SellAmount:   decimal.NewFromFloat(24500000000),
				},
			},
		},
		{
			ID:               uuid.New(),
			TicketNumber:     ptrString("FX-002"),
			CounterpartyName: "BIDV",
			DealType:         "FORWARD",
			Direction:        "SELL",
			NotionalAmount:   decimal.NewFromInt(500000),
			CurrencyCode:     "EUR",
			PairCode:         "EUR/VND",
			TradeDate:        time.Now(),
			Status:           "PENDING_L2_APPROVAL",
			CreatedBy:        uuid.New(),
			CreatedAt:        time.Now(),
			Legs: []model.FxDealLeg{
				{
					LegNumber:    1,
					ValueDate:    time.Now().AddDate(0, 1, 0),
					ExchangeRate: decimal.NewFromFloat(27000),
					BuyCurrency:  "EUR",
					SellCurrency: "VND",
					BuyAmount:    decimal.NewFromInt(500000),
					SellAmount:   decimal.NewFromFloat(13500000000),
				},
			},
		},
		{
			ID:               uuid.New(),
			CounterpartyName: "Vietcombank",
			DealType:         "SWAP",
			Direction:        "SELL_BUY",
			NotionalAmount:   decimal.NewFromInt(2000000),
			CurrencyCode:     "USD",
			PairCode:         "USD/VND",
			TradeDate:        time.Now(),
			Status:           "OPEN",
			CreatedBy:        uuid.New(),
			CreatedAt:        time.Now(),
			Legs: []model.FxDealLeg{
				{LegNumber: 1, ValueDate: time.Now().AddDate(0, 0, 2), ExchangeRate: decimal.NewFromFloat(24500), BuyCurrency: "USD", SellCurrency: "VND"},
				{LegNumber: 2, ValueDate: time.Now().AddDate(0, 1, 0), ExchangeRate: decimal.NewFromFloat(24600), BuyCurrency: "VND", SellCurrency: "USD"},
			},
		},
	}
}

func TestFXReportBuilder_Module(t *testing.T) {
	b := NewFXReportBuilder(nil)
	assert.Equal(t, "FX", b.Module())
	assert.Equal(t, "FX_DEALS", b.ReportType())
}

func TestFXReportBuilder_BuildSheets_Empty(t *testing.T) {
	b := NewFXReportBuilder(nil)
	assert.Equal(t, 0, b.RecordCount())

	f := excelize.NewFile()
	defer f.Close()

	err := b.BuildSheets(f)
	require.NoError(t, err)

	sheets := f.GetSheetList()
	assert.Contains(t, sheets, "Dashboard")
	assert.Contains(t, sheets, "Chi tiết giao dịch")
	assert.Contains(t, sheets, "Pivot Data")
}

func TestFXReportBuilder_BuildSheets_WithDeals(t *testing.T) {
	deals := makeSampleDeals()
	b := NewFXReportBuilder(deals)
	assert.Equal(t, 3, b.RecordCount())

	f := excelize.NewFile()
	defer f.Close()

	err := b.BuildSheets(f)
	require.NoError(t, err)

	// Detail sheet: header
	val, err := f.GetCellValue("Chi tiết giao dịch", "A1")
	require.NoError(t, err)
	assert.Equal(t, "STT", val)

	// Row 2 = first deal
	val, err = f.GetCellValue("Chi tiết giao dịch", "A2")
	require.NoError(t, err)
	assert.Equal(t, "1", val)

	// Deal type column C
	val, err = f.GetCellValue("Chi tiết giao dịch", "C2")
	require.NoError(t, err)
	assert.Equal(t, "SPOT", val)

	// Row 4 = third deal
	val, err = f.GetCellValue("Chi tiết giao dịch", "A4")
	require.NoError(t, err)
	assert.Equal(t, "3", val)

	// Pivot Data headers (Vietnamese)
	val, err = f.GetCellValue("Pivot Data", "A1")
	require.NoError(t, err)
	assert.Equal(t, "Ngày GD", val)

	val, err = f.GetCellValue("Pivot Data", "B1")
	require.NoError(t, err)
	assert.Equal(t, "Loại", val)

	// Pivot Data row 2 should have formula (not direct value)
	// excelize GetCellFormula returns the formula string
	formula, err := f.GetCellFormula("Pivot Data", "A2")
	require.NoError(t, err)
	assert.Contains(t, formula, "Chi tiết giao dịch", "pivot cell should reference detail sheet")
}

func TestFXReportBuilder_Dashboard_KPIs(t *testing.T) {
	deals := makeSampleDeals()
	b := NewFXReportBuilder(deals)

	f := excelize.NewFile()
	defer f.Close()

	err := b.BuildSheets(f)
	require.NoError(t, err)

	// Title row 1
	val, err := f.GetCellValue("Dashboard", "A1")
	require.NoError(t, err)
	assert.Equal(t, "TREASURY FX — BẢNG TỔNG HỢP", val)

	// KPI headers at row 3
	val, err = f.GetCellValue("Dashboard", "A3")
	require.NoError(t, err)
	assert.Equal(t, "Tổng giao dịch", val)

	val, err = f.GetCellValue("Dashboard", "B3")
	require.NoError(t, err)
	assert.Equal(t, "Spot", val)

	// KPI values at row 4 should be formulas (COUNTA/COUNTIF)
	formula, err := f.GetCellFormula("Dashboard", "A4")
	require.NoError(t, err)
	assert.Contains(t, formula, "COUNTA", "total deals should use COUNTA formula")

	formula, err = f.GetCellFormula("Dashboard", "B4")
	require.NoError(t, err)
	assert.Contains(t, formula, "COUNTIF", "spot count should use COUNTIF formula")
}

func TestFXReportBuilder_Dashboard_Charts(t *testing.T) {
	deals := makeSampleDeals()
	b := NewFXReportBuilder(deals)

	f := excelize.NewFile()
	defer f.Close()

	err := b.BuildSheets(f)
	require.NoError(t, err)

	// Verify chart data sections exist
	val, err := f.GetCellValue("Dashboard", "A6")
	require.NoError(t, err)
	assert.Equal(t, "Khối lượng theo ngày giao dịch", val)

	// Verify pair distribution section exists (dynamic row, search for it)
	found := false
	for row := 7; row < 50; row++ {
		val, _ = f.GetCellValue("Dashboard", fmt.Sprintf("A%d", row))
		if val == "Phân bổ theo cặp tiền tệ" {
			found = true
			break
		}
	}
	assert.True(t, found, "pair distribution section should exist")
}

func TestFXReportBuilder_Detail_Headers(t *testing.T) {
	b := NewFXReportBuilder(nil)

	f := excelize.NewFile()
	defer f.Close()

	err := b.BuildSheets(f)
	require.NoError(t, err)

	expectedHeaders := []string{
		"STT", "Mã GD", "Loại", "Chiều", "Cặp tiền tệ",
		"Số tiền", "Tỷ giá", "Ngày GD", "Ngày giá trị",
		"Đối tác", "Trạng thái", "Người tạo", "Ngày tạo",
	}

	for i, expected := range expectedHeaders {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		val, err := f.GetCellValue("Chi tiết giao dịch", cell)
		require.NoError(t, err)
		assert.Equal(t, expected, val, "header at column %d should be %q", i+1, expected)
	}
}

func TestFXReportBuilder_Detail_NumberFormat(t *testing.T) {
	deals := makeSampleDeals()
	b := NewFXReportBuilder(deals)

	f := excelize.NewFile()
	defer f.Close()

	err := b.BuildSheets(f)
	require.NoError(t, err)

	// Số tiền (column F) should have numeric value
	val, err := f.GetCellValue("Chi tiết giao dịch", "F2")
	require.NoError(t, err)
	assert.NotEmpty(t, val, "amount should be set")
}
