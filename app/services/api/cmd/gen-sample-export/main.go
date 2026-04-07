// gen-sample-export generates a sample FX export Excel file with mock data.
package main

import (
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/xuri/excelize/v2"

	"github.com/kienlongbank/treasury-api/internal/fx"
	"github.com/kienlongbank/treasury-api/internal/model"
	"github.com/kienlongbank/treasury-api/pkg/export"
)

func main() {
	deals := generateMockDeals()

	file := excelize.NewFile()

	params := export.ExportParams{
		User: export.UserInfo{
			ID:       uuid.New(),
			Username: "dealer_01",
			FullName: "Nguyễn Văn An",
			Role:     "Dealer — Phòng Kinh doanh Ngoại tệ",
		},
		DateFrom: time.Date(2026, 3, 1, 0, 0, 0, 0, time.Local),
		DateTo:   time.Date(2026, 3, 31, 0, 0, 0, 0, time.Local),
	}

	exportCode := fmt.Sprintf("EXP-%s-%s", time.Now().Format("20060102-150405"), "A1B2")

	if err := export.BuildDisclaimerPublic(file, params, exportCode); err != nil {
		fmt.Fprintf(os.Stderr, "disclaimer error: %v\n", err)
		os.Exit(1)
	}

	builder := fx.NewFXReportBuilder(deals)
	if err := builder.BuildSheets(file); err != nil {
		fmt.Fprintf(os.Stderr, "build sheets error: %v\n", err)
		os.Exit(1)
	}

	_ = file.SetDocProps(&excelize.DocProperties{
		Creator:     params.User.FullName + " (" + params.User.Username + ")",
		Title:       "Treasury FX Report — 03/2026",
		Subject:     "CONFIDENTIAL — Internal Use Only",
		Description: "Exported from KienlongBank Treasury Management System — Ngân hàng TMCP Kiên Long",
		Category:    "Treasury Report",
	})

	if idx, err := file.GetSheetIndex("Tuyên bố miễn trừ"); err == nil {
		file.SetActiveSheet(idx)
	}

	_ = file.DeleteSheet("Sheet1")

	outPath := "/tmp/Treasury_FX_Report_Sample_202603.xlsx"
	if err := file.SaveAs(outPath); err != nil {
		fmt.Fprintf(os.Stderr, "save error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Sample export saved to: %s\n", outPath)
}

func generateMockDeals() []model.FxDeal {
	counterparties := []struct {
		id   uuid.UUID
		code string
		name string
	}{
		{uuid.New(), "VCB", "Ngân hàng TMCP Ngoại thương Việt Nam"},
		{uuid.New(), "BIDV", "Ngân hàng TMCP Đầu tư và Phát triển VN"},
		{uuid.New(), "TCB", "Ngân hàng TMCP Kỹ Thương Việt Nam"},
		{uuid.New(), "ACB", "Ngân hàng TMCP Á Châu"},
		{uuid.New(), "MBB", "Ngân hàng TMCP Quân Đội"},
		{uuid.New(), "STB", "Ngân hàng TMCP Sài Gòn Thương Tín"},
	}

	pairs := []struct {
		pair    string
		buyCcy  string
		sellCcy string
		rate    float64
	}{
		{"USD/VND", "USD", "VND", 25430},
		{"EUR/VND", "EUR", "VND", 27650},
		{"JPY/VND", "JPY", "VND", 168.5},
		{"GBP/VND", "GBP", "VND", 32100},
		{"USD/EUR", "USD", "EUR", 0.92},
		{"SGD/VND", "SGD", "VND", 19050},
	}

	types := []string{"SPOT", "SPOT", "SPOT", "FORWARD", "FORWARD", "SWAP"}
	directions := []string{"BUY", "SELL", "BUY", "SELL", "BUY_SELL", "SELL_BUY"}
	statuses := []string{"COMPLETED", "COMPLETED", "COMPLETED", "PENDING_SETTLEMENT", "PENDING_L2_APPROVAL", "OPEN"}

	createdBy := uuid.New()
	branchID := uuid.New()

	var deals []model.FxDeal
	for i := 0; i < 25; i++ {
		cp := counterparties[i%len(counterparties)]
		pair := pairs[i%len(pairs)]
		dealType := types[i%len(types)]
		dir := directions[i%len(directions)]
		status := statuses[i%len(statuses)]

		tradeDate := time.Date(2026, 3, 1+i, 9, 30, 0, 0, time.Local)
		valueDate := tradeDate.Add(48 * time.Hour)

		amount := decimal.NewFromFloat(float64((i+1)*50000 + 100000))
		rate := decimal.NewFromFloat(pair.rate * (1 + float64(i%5)*0.001))

		ticket := fmt.Sprintf("FX-%s-%04d", tradeDate.Format("20060102"), i+1)

		deal := model.FxDeal{
			ID:               uuid.New(),
			TicketNumber:     &ticket,
			CounterpartyID:   cp.id,
			CounterpartyCode: cp.code,
			CounterpartyName: cp.name,
			DealType:         dealType,
			Direction:        dir,
			NotionalAmount:   amount,
			CurrencyCode:     pair.buyCcy,
			PairCode:         pair.pair,
			BranchID:         branchID,
			TradeDate:        tradeDate,
			Status:           status,
			CreatedBy:        createdBy,
			CreatedAt:        tradeDate.Add(-time.Hour),
			UpdatedAt:        tradeDate,
			Version:          1,
			Legs: []model.FxDealLeg{
				{
					ID:           uuid.New(),
					LegNumber:    1,
					ValueDate:    valueDate,
					ExchangeRate: rate,
					BuyCurrency:  pair.buyCcy,
					SellCurrency: pair.sellCcy,
					BuyAmount:    amount,
					SellAmount:   amount.Mul(rate),
				},
			},
		}

		deals = append(deals, deal)
	}

	return deals
}
