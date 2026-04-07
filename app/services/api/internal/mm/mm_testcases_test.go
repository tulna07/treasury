package mm

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/kienlongbank/treasury-api/pkg/apperror"
	"github.com/kienlongbank/treasury-api/pkg/constants"
	"github.com/kienlongbank/treasury-api/pkg/dto"
)

// ============================================================================
// MM-LN: INTERBANK TABLE-DRIVEN TEST CASES (matching Excel BRD v3)
// ============================================================================

func TestMMLN_Create(t *testing.T) {
	type tc struct {
		name      string
		modify    func(*dto.CreateMMInterbankRequest)
		wantErr   bool
		errCode   string
		checkResp func(*testing.T, *dto.MMInterbankResponse)
	}

	cases := []tc{
		{
			name:   "LN-01: Create PLACE VND ACT_365",
			modify: func(r *dto.CreateMMInterbankRequest) {},
			checkResp: func(t *testing.T, r *dto.MMInterbankResponse) {
				if r.Direction != constants.MMDirectionPlace {
					t.Errorf("expected PLACE, got %s", r.Direction)
				}
				if r.CurrencyCode != "VND" {
					t.Errorf("expected VND, got %s", r.CurrencyCode)
				}
			},
		},
		{
			name: "LN-02: Create TAKE VND ACT_360",
			modify: func(r *dto.CreateMMInterbankRequest) {
				r.Direction = constants.MMDirectionTake
				r.DayCountConvention = constants.DayCountACT360
			},
			checkResp: func(t *testing.T, r *dto.MMInterbankResponse) {
				if r.Direction != constants.MMDirectionTake {
					t.Errorf("expected TAKE, got %s", r.Direction)
				}
			},
		},
		{
			name: "LN-03: Create LEND USD ACT_365",
			modify: func(r *dto.CreateMMInterbankRequest) {
				r.Direction = constants.MMDirectionLend
				r.CurrencyCode = "USD"
				r.PrincipalAmount = decimal.NewFromInt(1_000_000)
			},
			checkResp: func(t *testing.T, r *dto.MMInterbankResponse) {
				if r.Direction != constants.MMDirectionLend {
					t.Errorf("expected LEND, got %s", r.Direction)
				}
				if r.CurrencyCode != "USD" {
					t.Errorf("expected USD, got %s", r.CurrencyCode)
				}
			},
		},
		{
			name: "LN-04: Create BORROW EUR ACT_ACT",
			modify: func(r *dto.CreateMMInterbankRequest) {
				r.Direction = constants.MMDirectionBorrow
				r.CurrencyCode = "EUR"
				r.DayCountConvention = constants.DayCountACTACT
				r.PrincipalAmount = decimal.NewFromInt(500_000)
			},
			checkResp: func(t *testing.T, r *dto.MMInterbankResponse) {
				if r.Direction != constants.MMDirectionBorrow {
					t.Errorf("expected BORROW, got %s", r.Direction)
				}
			},
		},
		{
			name: "LN-05: Create with collateral",
			modify: func(r *dto.CreateMMInterbankRequest) {
				r.HasCollateral = true
				r.CollateralCurrency = ptrString("VND")
				r.CollateralDescription = ptrString("Trái phiếu chính phủ TD2125068")
			},
			checkResp: func(t *testing.T, r *dto.MMInterbankResponse) {
				if !r.HasCollateral {
					t.Error("expected has_collateral true")
				}
			},
		},
		{
			name: "LN-06: Create with international settlement",
			modify: func(r *dto.CreateMMInterbankRequest) {
				r.CurrencyCode = "USD"
				r.PrincipalAmount = decimal.NewFromInt(2_000_000)
				r.RequiresInternationalSettlement = true
			},
			checkResp: func(t *testing.T, r *dto.MMInterbankResponse) {
				if !r.RequiresInternationalSettlement {
					t.Error("expected requires_international_settlement true")
				}
			},
		},
		{
			name: "LN-07: Create with ticket number",
			modify: func(r *dto.CreateMMInterbankRequest) {
				r.TicketNumber = ptrString("TKT-20260405-001")
			},
			checkResp: func(t *testing.T, r *dto.MMInterbankResponse) {
				if r.TicketNumber == nil || *r.TicketNumber != "TKT-20260405-001" {
					t.Error("expected ticket number TKT-20260405-001")
				}
			},
		},
		{
			name: "LN-08: Short tenor (1 day overnight)",
			modify: func(r *dto.CreateMMInterbankRequest) {
				r.MaturityDate = futureDate(1)
			},
			checkResp: func(t *testing.T, r *dto.MMInterbankResponse) {
				if r.TenorDays != 1 {
					t.Errorf("expected tenor 1, got %d", r.TenorDays)
				}
			},
		},
		{
			name: "LN-09: Long tenor (365 days)",
			modify: func(r *dto.CreateMMInterbankRequest) {
				r.MaturityDate = futureDate(365)
			},
			checkResp: func(t *testing.T, r *dto.MMInterbankResponse) {
				if r.TenorDays != 365 {
					t.Errorf("expected tenor 365, got %d", r.TenorDays)
				}
			},
		},
	}

	ctx := makeAuthContext(t, dealerUserID, []string{constants.RoleDealer})
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			req := makeInterbankRequest()
			c.modify(&req)

			resp, err := testInterbankService.CreateDeal(ctx, req, "127.0.0.1", "test")
			if c.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if c.errCode != "" && !apperror.Is(err, apperror.ErrorCode(c.errCode)) {
					t.Errorf("expected error code %s, got %v", c.errCode, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if resp.Status != constants.StatusOpen {
				t.Errorf("expected OPEN, got %s", resp.Status)
			}
			if c.checkResp != nil {
				c.checkResp(t, resp)
			}
		})
	}
}

func TestMMLN_Validation(t *testing.T) {
	type tc struct {
		name    string
		modify  func(*dto.CreateMMInterbankRequest)
		errCode string
	}

	cases := []tc{
		{
			name:    "LN-V01: Zero principal",
			modify:  func(r *dto.CreateMMInterbankRequest) { r.PrincipalAmount = decimal.Zero },
			errCode: string(apperror.ErrValidation),
		},
		{
			name:    "LN-V02: Negative principal",
			modify:  func(r *dto.CreateMMInterbankRequest) { r.PrincipalAmount = decimal.NewFromInt(-1000) },
			errCode: string(apperror.ErrValidation),
		},
		{
			name:    "LN-V03: Zero interest rate",
			modify:  func(r *dto.CreateMMInterbankRequest) { r.InterestRate = decimal.Zero },
			errCode: string(apperror.ErrValidation),
		},
		{
			name:    "LN-V04: Negative interest rate",
			modify:  func(r *dto.CreateMMInterbankRequest) { r.InterestRate = decimal.NewFromFloat(-2.5) },
			errCode: string(apperror.ErrValidation),
		},
		{
			name: "LN-V05: Maturity before effective",
			modify: func(r *dto.CreateMMInterbankRequest) {
				r.MaturityDate = r.EffectiveDate.Add(-24 * time.Hour)
			},
			errCode: string(apperror.ErrValidation),
		},
		{
			name: "LN-V06: Maturity equals effective",
			modify: func(r *dto.CreateMMInterbankRequest) {
				r.MaturityDate = r.EffectiveDate
			},
			errCode: string(apperror.ErrValidation),
		},
		{
			name:    "LN-V07: Invalid direction",
			modify:  func(r *dto.CreateMMInterbankRequest) { r.Direction = "INVALID" },
			errCode: string(apperror.ErrValidation),
		},
		{
			name:    "LN-V08: Empty direction",
			modify:  func(r *dto.CreateMMInterbankRequest) { r.Direction = "" },
			errCode: string(apperror.ErrValidation),
		},
		{
			name:    "LN-V09: Invalid day count convention",
			modify:  func(r *dto.CreateMMInterbankRequest) { r.DayCountConvention = "30/360" },
			errCode: string(apperror.ErrValidation),
		},
		{
			name:    "LN-V10: Missing counterparty",
			modify:  func(r *dto.CreateMMInterbankRequest) { r.CounterpartyID = uuid.Nil },
			errCode: string(apperror.ErrValidation),
		},
	}

	ctx := makeAuthContext(t, dealerUserID, []string{constants.RoleDealer})
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			req := makeInterbankRequest()
			c.modify(&req)

			_, err := testInterbankService.CreateDeal(ctx, req, "127.0.0.1", "test")
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !apperror.Is(err, apperror.ErrorCode(c.errCode)) {
				t.Errorf("expected %s, got %v", c.errCode, err)
			}
		})
	}
}

func TestMMLN_InterestFormula(t *testing.T) {
	type tc struct {
		name      string
		principal int64
		rate      float64
		tenorDays int
		dayCount  string
		currency  string
		// For exact validation we verify interest > 0 and correct rounding
	}

	cases := []tc{
		{
			name:      "LN-F01: VND ACT_365 90d 6.5%",
			principal: 10_000_000_000,
			rate:      6.5,
			tenorDays: 90,
			dayCount:  constants.DayCountACT365,
			currency:  "VND",
		},
		{
			name:      "LN-F02: VND ACT_360 90d 6.5%",
			principal: 10_000_000_000,
			rate:      6.5,
			tenorDays: 90,
			dayCount:  constants.DayCountACT360,
			currency:  "VND",
		},
		{
			name:      "LN-F03: VND ACT_ACT 90d 6.5%",
			principal: 10_000_000_000,
			rate:      6.5,
			tenorDays: 90,
			dayCount:  constants.DayCountACTACT,
			currency:  "VND",
		},
		{
			name:      "LN-F04: USD ACT_365 180d 5.25%",
			principal: 5_000_000,
			rate:      5.25,
			tenorDays: 180,
			dayCount:  constants.DayCountACT365,
			currency:  "USD",
		},
		{
			name:      "LN-F05: USD ACT_360 30d 4.0%",
			principal: 1_000_000,
			rate:      4.0,
			tenorDays: 30,
			dayCount:  constants.DayCountACT360,
			currency:  "USD",
		},
		{
			name:      "LN-F06: VND ACT_365 1d overnight 7.0%",
			principal: 100_000_000_000, // 100 billion
			rate:      7.0,
			tenorDays: 1,
			dayCount:  constants.DayCountACT365,
			currency:  "VND",
		},
		{
			name:      "LN-F07: VND ACT_365 365d 8.0%",
			principal: 10_000_000_000,
			rate:      8.0,
			tenorDays: 365,
			dayCount:  constants.DayCountACT365,
			currency:  "VND",
		},
		{
			name:      "LN-F08: EUR ACT_ACT 60d 3.5%",
			principal: 2_000_000,
			rate:      3.5,
			tenorDays: 60,
			dayCount:  constants.DayCountACTACT,
			currency:  "EUR",
		},
	}

	ctx := makeAuthContext(t, dealerUserID, []string{constants.RoleDealer})
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			req := makeInterbankRequest()
			req.CurrencyCode = c.currency
			req.PrincipalAmount = decimal.NewFromInt(c.principal)
			req.InterestRate = decimal.NewFromFloat(c.rate)
			req.DayCountConvention = c.dayCount
			req.MaturityDate = futureDate(c.tenorDays)

			resp, err := testInterbankService.CreateDeal(ctx, req, "127.0.0.1", "test")
			if err != nil {
				t.Fatalf("CreateDeal: %v", err)
			}

			// Verify interest calculated
			if resp.InterestAmount.IsZero() {
				t.Error("expected non-zero interest")
			}

			// Verify maturity = principal + interest
			expectedMaturity := resp.PrincipalAmount.Add(resp.InterestAmount)
			if !resp.MaturityAmount.Equal(expectedMaturity) {
				t.Errorf("maturity %s != principal %s + interest %s = %s",
					resp.MaturityAmount, resp.PrincipalAmount, resp.InterestAmount, expectedMaturity)
			}

			// Verify VND has no decimals
			if c.currency == "VND" && resp.InterestAmount.Exponent() < 0 {
				t.Errorf("VND interest should be whole number, got %s", resp.InterestAmount)
			}

			// Verify USD has max 2 decimals
			if c.currency == "USD" && resp.InterestAmount.Exponent() < -2 {
				t.Errorf("USD interest should have ≤2 decimals, got %s", resp.InterestAmount)
			}

			// Verify tenor days
			if resp.TenorDays != c.tenorDays {
				t.Errorf("expected tenor %d, got %d", c.tenorDays, resp.TenorDays)
			}
		})
	}
}

func TestMMLN_Workflow(t *testing.T) {
	type tc struct {
		name       string
		setup      func(t *testing.T) uuid.UUID // returns deal ID
		action     func(t *testing.T, dealID uuid.UUID) error
		wantStatus string
		wantErr    bool
		errCode    string
	}

	cases := []tc{
		{
			name:  "LN-W01: DeskHead approve OPEN → PENDING_TP_REVIEW",
			setup: func(t *testing.T) uuid.UUID { return createInterbankInStatus(t, constants.StatusOpen) },
			action: func(t *testing.T, id uuid.UUID) error {
				ctx := makeAuthContext(t, deskHeadUserID, []string{constants.RoleDeskHead})
				return testInterbankService.ApproveDeal(ctx, id, dto.ApprovalRequest{Action: "APPROVE"}, "127.0.0.1", "test")
			},
			wantStatus: constants.StatusPendingTPReview,
		},
		{
			name:  "LN-W02: DeskHead approve TP_REVIEW → PENDING_L2",
			setup: func(t *testing.T) uuid.UUID { return createInterbankInStatus(t, constants.StatusPendingTPReview) },
			action: func(t *testing.T, id uuid.UUID) error {
				ctx := makeAuthContext(t, deskHeadUserID, []string{constants.RoleDeskHead})
				return testInterbankService.ApproveDeal(ctx, id, dto.ApprovalRequest{Action: "APPROVE"}, "127.0.0.1", "test")
			},
			wantStatus: constants.StatusPendingL2Approval,
		},
		{
			name:  "LN-W03: DeskHead reject TP_REVIEW �� OPEN",
			setup: func(t *testing.T) uuid.UUID { return createInterbankInStatus(t, constants.StatusPendingTPReview) },
			action: func(t *testing.T, id uuid.UUID) error {
				ctx := makeAuthContext(t, deskHeadUserID, []string{constants.RoleDeskHead})
				return testInterbankService.ApproveDeal(ctx, id, dto.ApprovalRequest{Action: "REJECT"}, "127.0.0.1", "test")
			},
			wantStatus: constants.StatusOpen,
		},
		{
			name:  "LN-W04: Director approve → PENDING_RISK",
			setup: func(t *testing.T) uuid.UUID { return createInterbankInStatus(t, constants.StatusPendingL2Approval) },
			action: func(t *testing.T, id uuid.UUID) error {
				ctx := makeAuthContext(t, directorUserID, []string{constants.RoleDivisionHead})
				return testInterbankService.ApproveDeal(ctx, id, dto.ApprovalRequest{Action: "APPROVE"}, "127.0.0.1", "test")
			},
			wantStatus: constants.StatusPendingRiskApproval,
		},
		{
			name:  "LN-W05: Director reject → REJECTED",
			setup: func(t *testing.T) uuid.UUID { return createInterbankInStatus(t, constants.StatusPendingL2Approval) },
			action: func(t *testing.T, id uuid.UUID) error {
				ctx := makeAuthContext(t, directorUserID, []string{constants.RoleDivisionHead})
				return testInterbankService.ApproveDeal(ctx, id, dto.ApprovalRequest{Action: "REJECT"}, "127.0.0.1", "test")
			},
			wantStatus: constants.StatusRejected,
		},
		{
			name: "LN-W06: RiskOfficer approve → PENDING_BOOKING",
			setup: func(t *testing.T) uuid.UUID {
				return createInterbankInStatus(t, constants.StatusPendingRiskApproval)
			},
			action: func(t *testing.T, id uuid.UUID) error {
				riskID := createTestUser(t, testPool, constants.RoleRiskOfficer)
				ctx := makeAuthContext(t, riskID, []string{constants.RoleRiskOfficer})
				return testInterbankService.ApproveDeal(ctx, id, dto.ApprovalRequest{Action: "APPROVE"}, "127.0.0.1", "test")
			},
			wantStatus: constants.StatusPendingBooking,
		},
		{
			name: "LN-W07: RiskOfficer reject → VOIDED_BY_RISK",
			setup: func(t *testing.T) uuid.UUID {
				return createInterbankInStatus(t, constants.StatusPendingRiskApproval)
			},
			action: func(t *testing.T, id uuid.UUID) error {
				riskID := createTestUser(t, testPool, constants.RoleRiskOfficer)
				ctx := makeAuthContext(t, riskID, []string{constants.RoleRiskOfficer})
				return testInterbankService.ApproveDeal(ctx, id, dto.ApprovalRequest{Action: "REJECT"}, "127.0.0.1", "test")
			},
			wantStatus: constants.StatusVoidedByRisk,
		},
		{
			name: "LN-W08: Accountant approve → PENDING_CHIEF_ACCOUNTANT",
			setup: func(t *testing.T) uuid.UUID {
				return createInterbankInStatus(t, constants.StatusPendingBooking)
			},
			action: func(t *testing.T, id uuid.UUID) error {
				accID := createTestUser(t, testPool, constants.RoleAccountant)
				ctx := makeAuthContext(t, accID, []string{constants.RoleAccountant})
				return testInterbankService.ApproveDeal(ctx, id, dto.ApprovalRequest{Action: "APPROVE"}, "127.0.0.1", "test")
			},
			wantStatus: constants.StatusPendingChiefAccountant,
		},
		{
			name: "LN-W09: Accountant reject → VOIDED_BY_ACCOUNTING",
			setup: func(t *testing.T) uuid.UUID {
				return createInterbankInStatus(t, constants.StatusPendingBooking)
			},
			action: func(t *testing.T, id uuid.UUID) error {
				accID := createTestUser(t, testPool, constants.RoleAccountant)
				ctx := makeAuthContext(t, accID, []string{constants.RoleAccountant})
				return testInterbankService.ApproveDeal(ctx, id, dto.ApprovalRequest{Action: "REJECT"}, "127.0.0.1", "test")
			},
			wantStatus: constants.StatusVoidedByAccounting,
		},
		{
			name: "LN-W10: ChiefAccountant approve (no TTQT) → COMPLETED",
			setup: func(t *testing.T) uuid.UUID {
				return createInterbankInStatus(t, constants.StatusPendingChiefAccountant)
			},
			action: func(t *testing.T, id uuid.UUID) error {
				caID := createTestUser(t, testPool, constants.RoleChiefAccountant)
				ctx := makeAuthContext(t, caID, []string{constants.RoleChiefAccountant})
				return testInterbankService.ApproveDeal(ctx, id, dto.ApprovalRequest{Action: "APPROVE"}, "127.0.0.1", "test")
			},
			wantStatus: constants.StatusCompleted,
		},
		{
			name: "LN-W11: ChiefAccountant reject → VOIDED_BY_ACCOUNTING",
			setup: func(t *testing.T) uuid.UUID {
				return createInterbankInStatus(t, constants.StatusPendingChiefAccountant)
			},
			action: func(t *testing.T, id uuid.UUID) error {
				caID := createTestUser(t, testPool, constants.RoleChiefAccountant)
				ctx := makeAuthContext(t, caID, []string{constants.RoleChiefAccountant})
				return testInterbankService.ApproveDeal(ctx, id, dto.ApprovalRequest{Action: "REJECT"}, "127.0.0.1", "test")
			},
			wantStatus: constants.StatusVoidedByAccounting,
		},
		{
			name: "LN-W12: Settlement approve → COMPLETED",
			setup: func(t *testing.T) uuid.UUID {
				return createInterbankInStatus(t, constants.StatusPendingSettlement)
			},
			action: func(t *testing.T, id uuid.UUID) error {
				sID := createTestUser(t, testPool, constants.RoleSettlementOfficer)
				ctx := makeAuthContext(t, sID, []string{constants.RoleSettlementOfficer})
				return testInterbankService.ApproveDeal(ctx, id, dto.ApprovalRequest{Action: "APPROVE"}, "127.0.0.1", "test")
			},
			wantStatus: constants.StatusCompleted,
		},
		{
			name: "LN-W13: Settlement reject → VOIDED_BY_SETTLEMENT",
			setup: func(t *testing.T) uuid.UUID {
				return createInterbankInStatus(t, constants.StatusPendingSettlement)
			},
			action: func(t *testing.T, id uuid.UUID) error {
				sID := createTestUser(t, testPool, constants.RoleSettlementOfficer)
				ctx := makeAuthContext(t, sID, []string{constants.RoleSettlementOfficer})
				return testInterbankService.ApproveDeal(ctx, id, dto.ApprovalRequest{Action: "REJECT"}, "127.0.0.1", "test")
			},
			wantStatus: constants.StatusVoidedBySettlement,
		},
		{
			name:  "LN-W14: Self-approval blocked",
			setup: func(t *testing.T) uuid.UUID { return createInterbankInStatus(t, constants.StatusOpen) },
			action: func(t *testing.T, id uuid.UUID) error {
				ctx := makeAuthContext(t, dealerUserID, []string{constants.RoleDealer})
				return testInterbankService.ApproveDeal(ctx, id, dto.ApprovalRequest{Action: "APPROVE"}, "127.0.0.1", "test")
			},
			wantErr: true,
			errCode: string(apperror.ErrSelfApproval),
		},
		{
			name:  "LN-W15: Cannot approve COMPLETED deal",
			setup: func(t *testing.T) uuid.UUID { return createInterbankInStatus(t, constants.StatusCompleted) },
			action: func(t *testing.T, id uuid.UUID) error {
				ctx := makeAuthContext(t, deskHeadUserID, []string{constants.RoleDeskHead})
				return testInterbankService.ApproveDeal(ctx, id, dto.ApprovalRequest{Action: "APPROVE"}, "127.0.0.1", "test")
			},
			wantErr: true,
			errCode: string(apperror.ErrInvalidTransition),
		},
		{
			name:  "LN-W16: Cannot cancel OPEN deal",
			setup: func(t *testing.T) uuid.UUID { return createInterbankInStatus(t, constants.StatusOpen) },
			action: func(t *testing.T, id uuid.UUID) error {
				ctx := makeAuthContext(t, dealerUserID, []string{constants.RoleDealer})
				return testInterbankService.CancelDeal(ctx, id, "reason", "127.0.0.1", "test")
			},
			wantErr: true,
			errCode: string(apperror.ErrInvalidTransition),
		},
		{
			name:  "LN-W17: Cannot recall COMPLETED deal",
			setup: func(t *testing.T) uuid.UUID { return createInterbankInStatus(t, constants.StatusCompleted) },
			action: func(t *testing.T, id uuid.UUID) error {
				ctx := makeAuthContext(t, dealerUserID, []string{constants.RoleDealer})
				return testInterbankService.RecallDeal(ctx, id, "reason", "127.0.0.1", "test")
			},
			wantErr: true,
			errCode: string(apperror.ErrInvalidTransition),
		},
		{
			name:  "LN-W18: Cannot recall OPEN deal",
			setup: func(t *testing.T) uuid.UUID { return createInterbankInStatus(t, constants.StatusOpen) },
			action: func(t *testing.T, id uuid.UUID) error {
				ctx := makeAuthContext(t, dealerUserID, []string{constants.RoleDealer})
				return testInterbankService.RecallDeal(ctx, id, "reason", "127.0.0.1", "test")
			},
			wantErr: true,
			errCode: string(apperror.ErrInvalidTransition),
		},
		{
			name:  "LN-W19: Cannot clone OPEN deal",
			setup: func(t *testing.T) uuid.UUID { return createInterbankInStatus(t, constants.StatusOpen) },
			action: func(t *testing.T, id uuid.UUID) error {
				ctx := makeAuthContext(t, dealerUserID, []string{constants.RoleDealer})
				_, err := testInterbankService.CloneDeal(ctx, id, "127.0.0.1", "test")
				return err
			},
			wantErr: true,
			errCode: string(apperror.ErrInvalidTransition),
		},
		{
			name:  "LN-W20: Recall requires reason",
			setup: func(t *testing.T) uuid.UUID { return createInterbankInStatus(t, constants.StatusPendingL2Approval) },
			action: func(t *testing.T, id uuid.UUID) error {
				ctx := makeAuthContext(t, dealerUserID, []string{constants.RoleDealer})
				return testInterbankService.RecallDeal(ctx, id, "", "127.0.0.1", "test")
			},
			wantErr: true,
			errCode: string(apperror.ErrValidation),
		},
		{
			name:  "LN-W21: Cancel requires reason",
			setup: func(t *testing.T) uuid.UUID { return createInterbankInStatus(t, constants.StatusCompleted) },
			action: func(t *testing.T, id uuid.UUID) error {
				ctx := makeAuthContext(t, dealerUserID, []string{constants.RoleDealer})
				return testInterbankService.CancelDeal(ctx, id, "", "127.0.0.1", "test")
			},
			wantErr: true,
			errCode: string(apperror.ErrValidation),
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			dealID := c.setup(t)
			err := c.action(t, dealID)

			if c.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if c.errCode != "" && !apperror.Is(err, apperror.ErrorCode(c.errCode)) {
					t.Errorf("expected %s, got %v", c.errCode, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			ctx := makeAuthContext(t, dealerUserID, []string{constants.RoleDealer})
			resp, err := testInterbankService.GetDeal(ctx, dealID)
			if err != nil {
				t.Fatalf("GetDeal: %v", err)
			}
			if resp.Status != c.wantStatus {
				t.Errorf("expected status %s, got %s", c.wantStatus, resp.Status)
			}
		})
	}
}

// ============================================================================
// MM-OMO: OMO TABLE-DRIVEN TEST CASES (matching Excel BRD v3)
// ============================================================================

func TestMMOMO_Create(t *testing.T) {
	type tc struct {
		name      string
		modify    func(*dto.CreateMMOMORepoRequest)
		wantErr   bool
		errCode   string
		checkResp func(*testing.T, *dto.MMOMORepoResponse)
	}

	cases := []tc{
		{
			name:   "OMO-01: Create OMO deal",
			modify: func(r *dto.CreateMMOMORepoRequest) {},
			checkResp: func(t *testing.T, r *dto.MMOMORepoResponse) {
				if r.DealSubtype != constants.MMSubtypeOMO {
					t.Errorf("expected OMO, got %s", r.DealSubtype)
				}
			},
		},
		{
			name: "OMO-02: Create with zero haircut",
			modify: func(r *dto.CreateMMOMORepoRequest) {
				r.HaircutPct = decimal.Zero
			},
			checkResp: func(t *testing.T, r *dto.MMOMORepoResponse) {
				if !r.HaircutPct.IsZero() {
					t.Errorf("expected zero haircut, got %s", r.HaircutPct)
				}
			},
		},
		{
			name: "OMO-03: Create with high haircut",
			modify: func(r *dto.CreateMMOMORepoRequest) {
				r.HaircutPct = decimal.NewFromFloat(15.0)
			},
			checkResp: func(t *testing.T, r *dto.MMOMORepoResponse) {
				if !r.HaircutPct.Equal(decimal.NewFromFloat(15.0)) {
					t.Errorf("expected 15%% haircut, got %s", r.HaircutPct)
				}
			},
		},
		{
			name: "OMO-04: Create with short tenor (1 day)",
			modify: func(r *dto.CreateMMOMORepoRequest) {
				r.TenorDays = 1
				r.SettlementDate2 = futureDate(1)
			},
		},
		{
			name: "OMO-05: Create with long tenor (91 days)",
			modify: func(r *dto.CreateMMOMORepoRequest) {
				r.TenorDays = 91
				r.SettlementDate2 = futureDate(91)
			},
			checkResp: func(t *testing.T, r *dto.MMOMORepoResponse) {
				if r.TenorDays != 91 {
					t.Errorf("expected tenor 91, got %d", r.TenorDays)
				}
			},
		},
		{
			name: "OMO-06: Create with note",
			modify: func(r *dto.CreateMMOMORepoRequest) {
				r.Note = ptrString("Phiên OMO đặc biệt cuối quý")
			},
			checkResp: func(t *testing.T, r *dto.MMOMORepoResponse) {
				if r.Note == nil || *r.Note != "Phiên OMO đặc biệt cuối quý" {
					t.Error("expected note preserved")
				}
			},
		},
	}

	ctx := makeAuthContext(t, dealerUserID, []string{constants.RoleDealer})
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			req := makeOMORequest()
			c.modify(&req)

			resp, err := testOMORepoService.CreateDeal(ctx, req, "127.0.0.1", "test")
			if c.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if c.errCode != "" && !apperror.Is(err, apperror.ErrorCode(c.errCode)) {
					t.Errorf("expected %s, got %v", c.errCode, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if resp.Status != constants.StatusOpen {
				t.Errorf("expected OPEN, got %s", resp.Status)
			}
			if c.checkResp != nil {
				c.checkResp(t, resp)
			}
		})
	}
}

func TestMMOMO_Validation(t *testing.T) {
	type tc struct {
		name    string
		modify  func(*dto.CreateMMOMORepoRequest)
		errCode string
	}

	cases := []tc{
		{
			name:    "OMO-V01: Zero notional",
			modify:  func(r *dto.CreateMMOMORepoRequest) { r.NotionalAmount = decimal.Zero },
			errCode: string(apperror.ErrValidation),
		},
		{
			name:    "OMO-V02: Negative notional",
			modify:  func(r *dto.CreateMMOMORepoRequest) { r.NotionalAmount = decimal.NewFromInt(-1000) },
			errCode: string(apperror.ErrValidation),
		},
		{
			name:    "OMO-V03: Zero tenor",
			modify:  func(r *dto.CreateMMOMORepoRequest) { r.TenorDays = 0 },
			errCode: string(apperror.ErrValidation),
		},
		{
			name:    "OMO-V04: Negative tenor",
			modify:  func(r *dto.CreateMMOMORepoRequest) { r.TenorDays = -1 },
			errCode: string(apperror.ErrValidation),
		},
		{
			name: "OMO-V05: Settlement date 2 before date 1",
			modify: func(r *dto.CreateMMOMORepoRequest) {
				r.SettlementDate2 = r.SettlementDate1.Add(-24 * time.Hour)
			},
			errCode: string(apperror.ErrValidation),
		},
		{
			name: "OMO-V06: Settlement dates equal",
			modify: func(r *dto.CreateMMOMORepoRequest) {
				r.SettlementDate2 = r.SettlementDate1
			},
			errCode: string(apperror.ErrValidation),
		},
		{
			name:    "OMO-V07: Empty session name",
			modify:  func(r *dto.CreateMMOMORepoRequest) { r.SessionName = "" },
			errCode: string(apperror.ErrValidation),
		},
		{
			name:    "OMO-V08: Zero winning rate",
			modify:  func(r *dto.CreateMMOMORepoRequest) { r.WinningRate = decimal.Zero },
			errCode: string(apperror.ErrValidation),
		},
		{
			name:    "OMO-V09: Missing counterparty",
			modify:  func(r *dto.CreateMMOMORepoRequest) { r.CounterpartyID = uuid.Nil },
			errCode: string(apperror.ErrValidation),
		},
		{
			name:    "OMO-V10: Missing bond catalog",
			modify:  func(r *dto.CreateMMOMORepoRequest) { r.BondCatalogID = uuid.Nil },
			errCode: string(apperror.ErrValidation),
		},
	}

	ctx := makeAuthContext(t, dealerUserID, []string{constants.RoleDealer})
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			req := makeOMORequest()
			c.modify(&req)

			_, err := testOMORepoService.CreateDeal(ctx, req, "127.0.0.1", "test")
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !apperror.Is(err, apperror.ErrorCode(c.errCode)) {
				t.Errorf("expected %s, got %v", c.errCode, err)
			}
		})
	}
}

func TestMMOMO_Workflow(t *testing.T) {
	type tc struct {
		name       string
		setup      func(t *testing.T) uuid.UUID
		action     func(t *testing.T, dealID uuid.UUID) error
		wantStatus string
		wantErr    bool
		errCode    string
	}

	cases := []tc{
		{
			name:  "OMO-W01: DeskHead approve OPEN → PENDING_L2",
			setup: func(t *testing.T) uuid.UUID { return createOMOInStatus(t, constants.StatusOpen) },
			action: func(t *testing.T, id uuid.UUID) error {
				ctx := makeAuthContext(t, deskHeadUserID, []string{constants.RoleDeskHead})
				return testOMORepoService.ApproveDeal(ctx, id, dto.ApprovalRequest{Action: "APPROVE"}, "127.0.0.1", "test")
			},
			wantStatus: constants.StatusPendingL2Approval,
		},
		{
			name:  "OMO-W02: Director approve → PENDING_BOOKING (no risk)",
			setup: func(t *testing.T) uuid.UUID { return createOMOInStatus(t, constants.StatusPendingL2Approval) },
			action: func(t *testing.T, id uuid.UUID) error {
				ctx := makeAuthContext(t, directorUserID, []string{constants.RoleDivisionHead})
				return testOMORepoService.ApproveDeal(ctx, id, dto.ApprovalRequest{Action: "APPROVE"}, "127.0.0.1", "test")
			},
			wantStatus: constants.StatusPendingBooking,
		},
		{
			name:  "OMO-W03: Director reject → REJECTED",
			setup: func(t *testing.T) uuid.UUID { return createOMOInStatus(t, constants.StatusPendingL2Approval) },
			action: func(t *testing.T, id uuid.UUID) error {
				ctx := makeAuthContext(t, directorUserID, []string{constants.RoleDivisionHead})
				return testOMORepoService.ApproveDeal(ctx, id, dto.ApprovalRequest{Action: "REJECT"}, "127.0.0.1", "test")
			},
			wantStatus: constants.StatusRejected,
		},
		{
			name: "OMO-W04: Accountant approve → PENDING_CHIEF_ACCOUNTANT",
			setup: func(t *testing.T) uuid.UUID {
				return createOMOInStatus(t, constants.StatusPendingBooking)
			},
			action: func(t *testing.T, id uuid.UUID) error {
				accID := createTestUser(t, testPool, constants.RoleAccountant)
				ctx := makeAuthContext(t, accID, []string{constants.RoleAccountant})
				return testOMORepoService.ApproveDeal(ctx, id, dto.ApprovalRequest{Action: "APPROVE"}, "127.0.0.1", "test")
			},
			wantStatus: constants.StatusPendingChiefAccountant,
		},
		{
			name: "OMO-W05: Accountant reject → VOIDED_BY_ACCOUNTING",
			setup: func(t *testing.T) uuid.UUID {
				return createOMOInStatus(t, constants.StatusPendingBooking)
			},
			action: func(t *testing.T, id uuid.UUID) error {
				accID := createTestUser(t, testPool, constants.RoleAccountant)
				ctx := makeAuthContext(t, accID, []string{constants.RoleAccountant})
				return testOMORepoService.ApproveDeal(ctx, id, dto.ApprovalRequest{Action: "REJECT"}, "127.0.0.1", "test")
			},
			wantStatus: constants.StatusVoidedByAccounting,
		},
		{
			name: "OMO-W06: ChiefAccountant approve → COMPLETED",
			setup: func(t *testing.T) uuid.UUID {
				return createOMOInStatus(t, constants.StatusPendingChiefAccountant)
			},
			action: func(t *testing.T, id uuid.UUID) error {
				caID := createTestUser(t, testPool, constants.RoleChiefAccountant)
				ctx := makeAuthContext(t, caID, []string{constants.RoleChiefAccountant})
				return testOMORepoService.ApproveDeal(ctx, id, dto.ApprovalRequest{Action: "APPROVE"}, "127.0.0.1", "test")
			},
			wantStatus: constants.StatusCompleted,
		},
		{
			name: "OMO-W07: ChiefAccountant reject → VOIDED_BY_ACCOUNTING",
			setup: func(t *testing.T) uuid.UUID {
				return createOMOInStatus(t, constants.StatusPendingChiefAccountant)
			},
			action: func(t *testing.T, id uuid.UUID) error {
				caID := createTestUser(t, testPool, constants.RoleChiefAccountant)
				ctx := makeAuthContext(t, caID, []string{constants.RoleChiefAccountant})
				return testOMORepoService.ApproveDeal(ctx, id, dto.ApprovalRequest{Action: "REJECT"}, "127.0.0.1", "test")
			},
			wantStatus: constants.StatusVoidedByAccounting,
		},
		{
			name:  "OMO-W08: Self-approval blocked",
			setup: func(t *testing.T) uuid.UUID { return createOMOInStatus(t, constants.StatusOpen) },
			action: func(t *testing.T, id uuid.UUID) error {
				ctx := makeAuthContext(t, dealerUserID, []string{constants.RoleDealer})
				return testOMORepoService.ApproveDeal(ctx, id, dto.ApprovalRequest{Action: "APPROVE"}, "127.0.0.1", "test")
			},
			wantErr: true,
			errCode: string(apperror.ErrSelfApproval),
		},
		{
			name:  "OMO-W09: Cannot approve COMPLETED",
			setup: func(t *testing.T) uuid.UUID { return createOMOInStatus(t, constants.StatusCompleted) },
			action: func(t *testing.T, id uuid.UUID) error {
				ctx := makeAuthContext(t, deskHeadUserID, []string{constants.RoleDeskHead})
				return testOMORepoService.ApproveDeal(ctx, id, dto.ApprovalRequest{Action: "APPROVE"}, "127.0.0.1", "test")
			},
			wantErr: true,
			errCode: string(apperror.ErrInvalidTransition),
		},
		{
			name:  "OMO-W10: Cannot cancel OPEN",
			setup: func(t *testing.T) uuid.UUID { return createOMOInStatus(t, constants.StatusOpen) },
			action: func(t *testing.T, id uuid.UUID) error {
				ctx := makeAuthContext(t, dealerUserID, []string{constants.RoleDealer})
				return testOMORepoService.CancelDeal(ctx, id, "reason", "127.0.0.1", "test")
			},
			wantErr: true,
			errCode: string(apperror.ErrInvalidTransition),
		},
		{
			name:  "OMO-W11: Cannot recall COMPLETED",
			setup: func(t *testing.T) uuid.UUID { return createOMOInStatus(t, constants.StatusCompleted) },
			action: func(t *testing.T, id uuid.UUID) error {
				ctx := makeAuthContext(t, dealerUserID, []string{constants.RoleDealer})
				return testOMORepoService.RecallDeal(ctx, id, "reason", "127.0.0.1", "test")
			},
			wantErr: true,
			errCode: string(apperror.ErrInvalidTransition),
		},
		{
			name:  "OMO-W12: Cannot clone OPEN",
			setup: func(t *testing.T) uuid.UUID { return createOMOInStatus(t, constants.StatusOpen) },
			action: func(t *testing.T, id uuid.UUID) error {
				ctx := makeAuthContext(t, dealerUserID, []string{constants.RoleDealer})
				_, err := testOMORepoService.CloneDeal(ctx, id, "127.0.0.1", "test")
				return err
			},
			wantErr: true,
			errCode: string(apperror.ErrInvalidTransition),
		},
		{
			name:  "OMO-W13: Recall requires reason",
			setup: func(t *testing.T) uuid.UUID { return createOMOInStatus(t, constants.StatusPendingL2Approval) },
			action: func(t *testing.T, id uuid.UUID) error {
				ctx := makeAuthContext(t, dealerUserID, []string{constants.RoleDealer})
				return testOMORepoService.RecallDeal(ctx, id, "", "127.0.0.1", "test")
			},
			wantErr: true,
			errCode: string(apperror.ErrValidation),
		},
		{
			name:  "OMO-W14: Cancel requires reason",
			setup: func(t *testing.T) uuid.UUID { return createOMOInStatus(t, constants.StatusCompleted) },
			action: func(t *testing.T, id uuid.UUID) error {
				ctx := makeAuthContext(t, dealerUserID, []string{constants.RoleDealer})
				return testOMORepoService.CancelDeal(ctx, id, "", "127.0.0.1", "test")
			},
			wantErr: true,
			errCode: string(apperror.ErrValidation),
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			dealID := c.setup(t)
			err := c.action(t, dealID)

			if c.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if c.errCode != "" && !apperror.Is(err, apperror.ErrorCode(c.errCode)) {
					t.Errorf("expected %s, got %v", c.errCode, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			ctx := makeAuthContext(t, dealerUserID, []string{constants.RoleDealer})
			resp, err := testOMORepoService.GetDeal(ctx, dealID)
			if err != nil {
				t.Fatalf("GetDeal: %v", err)
			}
			if resp.Status != c.wantStatus {
				t.Errorf("expected status %s, got %s", c.wantStatus, resp.Status)
			}
		})
	}
}

// ============================================================================
// MM-RK: REPO KBNN TABLE-DRIVEN TEST CASES (matching Excel BRD v3)
// ============================================================================

func TestMMRK_Create(t *testing.T) {
	type tc struct {
		name      string
		modify    func(*dto.CreateMMOMORepoRequest)
		checkResp func(*testing.T, *dto.MMOMORepoResponse)
	}

	cases := []tc{
		{
			name:   "RK-01: Create Repo KBNN deal",
			modify: func(r *dto.CreateMMOMORepoRequest) {},
			checkResp: func(t *testing.T, r *dto.MMOMORepoResponse) {
				if r.DealSubtype != constants.MMSubtypeStateRepo {
					t.Errorf("expected STATE_REPO, got %s", r.DealSubtype)
				}
				if len(r.DealNumber) < 3 || r.DealNumber[:3] != "RK-" {
					t.Errorf("expected deal number starting with RK-, got %s", r.DealNumber)
				}
			},
		},
		{
			name: "RK-02: Create with different tenor",
			modify: func(r *dto.CreateMMOMORepoRequest) {
				r.TenorDays = 28
				r.SettlementDate2 = futureDate(28)
			},
			checkResp: func(t *testing.T, r *dto.MMOMORepoResponse) {
				if r.TenorDays != 28 {
					t.Errorf("expected tenor 28, got %d", r.TenorDays)
				}
			},
		},
		{
			name: "RK-03: Create with different winning rate",
			modify: func(r *dto.CreateMMOMORepoRequest) {
				r.WinningRate = decimal.NewFromFloat(3.25)
			},
			checkResp: func(t *testing.T, r *dto.MMOMORepoResponse) {
				if !r.WinningRate.Equal(decimal.NewFromFloat(3.25)) {
					t.Errorf("expected winning rate 3.25, got %s", r.WinningRate)
				}
			},
		},
		{
			name: "RK-04: Create with high notional",
			modify: func(r *dto.CreateMMOMORepoRequest) {
				r.NotionalAmount = decimal.NewFromInt(500_000_000_000) // 500B
			},
			checkResp: func(t *testing.T, r *dto.MMOMORepoResponse) {
				if !r.NotionalAmount.Equal(decimal.NewFromInt(500_000_000_000)) {
					t.Errorf("expected 500B, got %s", r.NotionalAmount)
				}
			},
		},
	}

	ctx := makeAuthContext(t, dealerUserID, []string{constants.RoleDealer})
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			req := makeRepoKBNNRequest()
			c.modify(&req)

			resp, err := testOMORepoService.CreateDeal(ctx, req, "127.0.0.1", "test")
			if err != nil {
				t.Fatalf("CreateDeal: %v", err)
			}
			if resp.Status != constants.StatusOpen {
				t.Errorf("expected OPEN, got %s", resp.Status)
			}
			if c.checkResp != nil {
				c.checkResp(t, resp)
			}
		})
	}
}

func TestMMRK_FullApprovalWorkflow(t *testing.T) {
	ctx := makeAuthContext(t, dealerUserID, []string{constants.RoleDealer})
	req := makeRepoKBNNRequest()
	resp, err := testOMORepoService.CreateDeal(ctx, req, "127.0.0.1", "test")
	if err != nil {
		t.Fatalf("CreateDeal Repo KBNN: %v", err)
	}

	approveOMOFullChain(t, resp.ID)

	deal, _ := testOMORepoService.GetDeal(ctx, resp.ID)
	if deal.Status != constants.StatusCompleted {
		t.Errorf("expected COMPLETED, got %s", deal.Status)
	}
}

func TestMMRK_CancelFlow(t *testing.T) {
	ctx := makeAuthContext(t, dealerUserID, []string{constants.RoleDealer})
	req := makeRepoKBNNRequest()
	resp, err := testOMORepoService.CreateDeal(ctx, req, "127.0.0.1", "test")
	if err != nil {
		t.Fatalf("CreateDeal: %v", err)
	}

	// Advance to COMPLETED
	advanceOMOStatus(t, resp.ID, constants.StatusCompleted)

	// Request cancel
	err = testOMORepoService.CancelDeal(ctx, resp.ID, "session error", "127.0.0.1", "test")
	if err != nil {
		t.Fatalf("CancelDeal: %v", err)
	}

	deal, _ := testOMORepoService.GetDeal(ctx, resp.ID)
	if deal.Status != constants.StatusPendingCancelL1 {
		t.Errorf("expected PENDING_CANCEL_L1, got %s", deal.Status)
	}

	// L1 + L2 approve cancel
	dhCtx := makeAuthContext(t, deskHeadUserID, []string{constants.RoleDeskHead})
	testOMORepoService.ApproveCancelDeal(dhCtx, resp.ID, dto.ApprovalRequest{Action: "APPROVE"}, "127.0.0.1", "test")

	dirCtx := makeAuthContext(t, directorUserID, []string{constants.RoleDivisionHead})
	testOMORepoService.ApproveCancelDeal(dirCtx, resp.ID, dto.ApprovalRequest{Action: "APPROVE"}, "127.0.0.1", "test")

	deal, _ = testOMORepoService.GetDeal(ctx, resp.ID)
	if deal.Status != constants.StatusCancelled {
		t.Errorf("expected CANCELLED, got %s", deal.Status)
	}
}

func TestMMRK_CloneRejected(t *testing.T) {
	ctx := makeAuthContext(t, dealerUserID, []string{constants.RoleDealer})
	req := makeRepoKBNNRequest()
	resp, err := testOMORepoService.CreateDeal(ctx, req, "127.0.0.1", "test")
	if err != nil {
		t.Fatalf("CreateDeal: %v", err)
	}

	advanceOMOStatus(t, resp.ID, constants.StatusRejected)

	clone, err := testOMORepoService.CloneDeal(ctx, resp.ID, "127.0.0.1", "test")
	if err != nil {
		t.Fatalf("CloneDeal: %v", err)
	}
	if clone.Status != constants.StatusOpen {
		t.Errorf("expected OPEN, got %s", clone.Status)
	}
	if clone.DealSubtype != constants.MMSubtypeStateRepo {
		t.Errorf("expected STATE_REPO subtype preserved, got %s", clone.DealSubtype)
	}
}
