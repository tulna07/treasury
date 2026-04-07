package bond

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

// ptrString defined in integration_test.go

func makeSampleBondDeals() []model.BondDeal {
	now := time.Now()
	return []model.BondDeal{
		{
			ID:               uuid.New(),
			DealNumber:       "G-20260405-0001",
			BondCategory:     "GOVERNMENT",
			TradeDate:        now,
			Direction:        "BUY",
			CounterpartyName: "ACB",
			TransactionType:  "OUTRIGHT",
			BondCodeDisplay:  "VGB5Y2026",
			Issuer:           "Kho bạc Nhà nước",
			CouponRate:       decimal.NewFromFloat(5.5),
			IssueDate:        &now,
			MaturityDate:     now.AddDate(5, 0, 0),
			Quantity:         1000,
			FaceValue:        decimal.NewFromInt(100000),
			CleanPrice:       decimal.NewFromFloat(99500),
			SettlementPrice:  decimal.NewFromFloat(100200),
			TotalValue:       decimal.NewFromFloat(100200000),
			PortfolioType:    ptrString("HTM"),
			PaymentDate:      now.AddDate(0, 0, 2),
			RemainingTenorDays: 1825,
			Status:           "COMPLETED",
			CreatedBy:        uuid.New(),
			CreatedByName:    "Nguyễn Văn A",
			CreatedAt:        now,
		},
		{
			ID:               uuid.New(),
			DealNumber:       "F-20260405-0001",
			BondCategory:     "FINANCIAL_INSTITUTION",
			TradeDate:        now,
			Direction:        "SELL",
			CounterpartyName: "BIDV",
			TransactionType:  "REPO",
			BondCodeDisplay:  "TCB2027",
			Issuer:           "Techcombank",
			CouponRate:       decimal.NewFromFloat(7.2),
			MaturityDate:     now.AddDate(1, 0, 0),
			Quantity:         500,
			FaceValue:        decimal.NewFromInt(100000),
			CleanPrice:       decimal.NewFromFloat(98000),
			SettlementPrice:  decimal.NewFromFloat(98500),
			TotalValue:       decimal.NewFromFloat(49250000),
			PortfolioType:    ptrString("AFS"),
			PaymentDate:      now.AddDate(0, 0, 1),
			RemainingTenorDays: 365,
			Status:           "PENDING_L2_APPROVAL",
			CreatedBy:        uuid.New(),
			CreatedByName:    "Trần Thị B",
			CreatedAt:        now,
		},
		{
			ID:               uuid.New(),
			DealNumber:       "G-20260405-0002",
			BondCategory:     "CERTIFICATE_OF_DEPOSIT",
			TradeDate:        now,
			Direction:        "BUY",
			CounterpartyName: "Vietcombank",
			TransactionType:  "OUTRIGHT",
			BondCodeManual:   ptrString("CD-VCB-6M"),
			Issuer:           "Vietcombank",
			CouponRate:       decimal.NewFromFloat(6.0),
			MaturityDate:     now.AddDate(0, 6, 0),
			Quantity:         2000,
			FaceValue:        decimal.NewFromInt(100000),
			CleanPrice:       decimal.NewFromFloat(100000),
			SettlementPrice:  decimal.NewFromFloat(100000),
			TotalValue:       decimal.NewFromFloat(200000000),
			PortfolioType:    ptrString("HFT"),
			PaymentDate:      now,
			RemainingTenorDays: 180,
			Status:           "OPEN",
			CreatedBy:        uuid.New(),
			CreatedByName:    "Lê Văn C",
			CreatedAt:        now,
		},
	}
}

func TestBondReportBuilder_Module(t *testing.T) {
	b := NewBondReportBuilder(nil)
	assert.Equal(t, "BOND", b.Module())
	assert.Equal(t, "BOND_DEALS", b.ReportType())
}

func TestBondReportBuilder_BuildSheets_Empty(t *testing.T) {
	b := NewBondReportBuilder(nil)
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

func TestBondReportBuilder_BuildSheets_WithDeals(t *testing.T) {
	deals := makeSampleBondDeals()
	b := NewBondReportBuilder(deals)
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

	// Bond category column C
	val, err = f.GetCellValue("Chi tiết giao dịch", "C2")
	require.NoError(t, err)
	assert.Equal(t, "GOVERNMENT", val)

	// Row 4 = third deal
	val, err = f.GetCellValue("Chi tiết giao dịch", "A4")
	require.NoError(t, err)
	assert.Equal(t, "3", val)

	// Pivot Data headers
	val, err = f.GetCellValue("Pivot Data", "A1")
	require.NoError(t, err)
	assert.Equal(t, "Ngày GD", val)

	val, err = f.GetCellValue("Pivot Data", "B1")
	require.NoError(t, err)
	assert.Equal(t, "Loại", val)

	// Pivot Data row 2 should have formula referencing detail sheet
	formula, err := f.GetCellFormula("Pivot Data", "A2")
	require.NoError(t, err)
	assert.Contains(t, formula, "Chi tiết giao dịch", "pivot cell should reference detail sheet")
}

func TestBondReportBuilder_Dashboard_KPIs(t *testing.T) {
	deals := makeSampleBondDeals()
	b := NewBondReportBuilder(deals)

	f := excelize.NewFile()
	defer f.Close()

	err := b.BuildSheets(f)
	require.NoError(t, err)

	// Title row 1
	val, err := f.GetCellValue("Dashboard", "A1")
	require.NoError(t, err)
	assert.Equal(t, "TREASURY BOND — BẢNG TỔNG HỢP", val)

	// KPI headers at row 3
	val, err = f.GetCellValue("Dashboard", "A3")
	require.NoError(t, err)
	assert.Equal(t, "Tổng giao dịch", val)

	val, err = f.GetCellValue("Dashboard", "B3")
	require.NoError(t, err)
	assert.Equal(t, "Govi", val)

	// KPI values at row 4 should be formulas
	formula, err := f.GetCellFormula("Dashboard", "A4")
	require.NoError(t, err)
	assert.Contains(t, formula, "COUNTA", "total deals should use COUNTA formula")

	formula, err = f.GetCellFormula("Dashboard", "B4")
	require.NoError(t, err)
	assert.Contains(t, formula, "COUNTIF", "govi count should use COUNTIF formula")
}

func TestBondReportBuilder_Dashboard_Charts(t *testing.T) {
	deals := makeSampleBondDeals()
	b := NewBondReportBuilder(deals)

	f := excelize.NewFile()
	defer f.Close()

	err := b.BuildSheets(f)
	require.NoError(t, err)

	// Verify chart data sections exist
	val, err := f.GetCellValue("Dashboard", "A6")
	require.NoError(t, err)
	assert.Equal(t, "Khối lượng theo ngày giao dịch", val)

	// Verify category distribution section exists
	found := false
	for row := 7; row < 50; row++ {
		val, _ = f.GetCellValue("Dashboard", fmt.Sprintf("A%d", row))
		if val == "Phân bổ theo loại trái phiếu" {
			found = true
			break
		}
	}
	assert.True(t, found, "category distribution section should exist")
}

func TestBondReportBuilder_Detail_Headers(t *testing.T) {
	b := NewBondReportBuilder(nil)

	f := excelize.NewFile()
	defer f.Close()

	err := b.BuildSheets(f)
	require.NoError(t, err)

	expectedHeaders := []string{
		"STT", "Mã GD", "Loại", "Chiều GD", "Đối tác",
		"Loại GD", "Mã TP", "Tổ chức phát hành", "Lãi suất coupon", "Ngày phát hành",
		"Ngày đáo hạn", "Số lượng", "Mệnh giá", "Giá sạch", "Giá thanh toán",
		"Tổng giá trị", "Danh mục", "Ngày TT", "Kỳ hạn còn lại",
		"Trạng thái", "Ngày GD", "Người tạo",
	}

	for i, expected := range expectedHeaders {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		val, err := f.GetCellValue("Chi tiết giao dịch", cell)
		require.NoError(t, err)
		assert.Equal(t, expected, val, "header at column %d should be %q", i+1, expected)
	}
}

func TestBondReportBuilder_Detail_NumberFormat(t *testing.T) {
	deals := makeSampleBondDeals()
	b := NewBondReportBuilder(deals)

	f := excelize.NewFile()
	defer f.Close()

	err := b.BuildSheets(f)
	require.NoError(t, err)

	// Tổng giá trị (column P) should have numeric value
	val, err := f.GetCellValue("Chi tiết giao dịch", "P2")
	require.NoError(t, err)
	assert.NotEmpty(t, val, "total value should be set")
}
