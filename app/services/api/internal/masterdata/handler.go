package masterdata

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/kienlongbank/treasury-api/internal/ctxutil"
	"github.com/kienlongbank/treasury-api/internal/model"
	"github.com/kienlongbank/treasury-api/internal/repository"
	"github.com/kienlongbank/treasury-api/pkg/apperror"
	"github.com/kienlongbank/treasury-api/pkg/audit"
	"github.com/kienlongbank/treasury-api/pkg/dto"
	"github.com/kienlongbank/treasury-api/pkg/httputil"
)

// Handler handles HTTP requests for master data.
type Handler struct {
	cpRepo     repository.CounterpartyRepository
	mdRepo     repository.MasterDataRepository
	auditLog   *audit.Logger
	logger     *zap.Logger
}

// NewHandler creates a new masterdata Handler.
func NewHandler(cpRepo repository.CounterpartyRepository, mdRepo repository.MasterDataRepository, auditLog *audit.Logger, logger *zap.Logger) *Handler {
	return &Handler{cpRepo: cpRepo, mdRepo: mdRepo, auditLog: auditLog, logger: logger}
}

// --- Counterparties ---

// ListCounterparties lists counterparties (with optional search and pagination).
func (h *Handler) ListCounterparties(w http.ResponseWriter, r *http.Request) {
	pag := httputil.ParsePagination(r)

	filter := dto.CounterpartyFilter{}
	if v := r.URL.Query().Get("search"); v != "" {
		filter.Search = &v
	}
	if v := r.URL.Query().Get("is_active"); v == "true" {
		b := true
		filter.IsActive = &b
	}

	cps, total, err := h.cpRepo.List(r.Context(), filter, pag)
	if err != nil {
		httputil.Error(w, r, err)
		return
	}

	items := make([]dto.CounterpartyResponse, 0, len(cps))
	for _, cp := range cps {
		items = append(items, counterpartyToResponse(&cp))
	}

	result := dto.NewPaginationResponse(items, total, pag.Page, pag.PageSize)
	httputil.Success(w, r, result)
}

// GetCounterparty gets a counterparty by ID.
func (h *Handler) GetCounterparty(w http.ResponseWriter, r *http.Request) {
	id, err := httputil.ParseUUID(chi.URLParam(r, "id"))
	if err != nil {
		httputil.Error(w, r, apperror.New(apperror.ErrValidation, "invalid counterparty ID"))
		return
	}

	cp, err := h.cpRepo.GetByID(r.Context(), id)
	if err != nil {
		httputil.Error(w, r, err)
		return
	}

	httputil.Success(w, r, counterpartyToResponse(cp))
}

// CreateCounterparty creates a new counterparty (admin).
func (h *Handler) CreateCounterparty(w http.ResponseWriter, r *http.Request) {
	var req dto.CreateCounterpartyRequest
	if err := httputil.ParseBody(r, &req); err != nil {
		httputil.Error(w, r, apperror.New(apperror.ErrValidation, err.Error()))
		return
	}

	if req.Code == "" || req.FullName == "" || req.CIF == "" {
		httputil.Error(w, r, apperror.New(apperror.ErrValidation, "code, full_name, and cif are required"))
		return
	}

	// Check code uniqueness
	existing, err := h.cpRepo.GetByCode(r.Context(), req.Code)
	if err != nil {
		httputil.Error(w, r, err)
		return
	}
	if existing != nil {
		httputil.Error(w, r, apperror.New(apperror.ErrConflict, "counterparty code already exists"))
		return
	}

	cp := &model.Counterparty{
		Code: req.Code, FullName: req.FullName, ShortName: req.ShortName,
		CIF: req.CIF, SwiftCode: req.SwiftCode, CountryCode: req.CountryCode,
		TaxID: req.TaxID, Address: req.Address, FxUsesLimit: req.FxUsesLimit,
	}
	if err := h.cpRepo.Create(r.Context(), cp); err != nil {
		httputil.Error(w, r, err)
		return
	}

	userID := ctxutil.GetUserUUID(r.Context())
	h.auditLog.Log(r.Context(), audit.Entry{
		UserID: userID, FullName: "admin", Action: "CREATE_COUNTERPARTY",
		DealModule: "SYSTEM", DealID: &cp.ID,
		NewValues: map[string]interface{}{"code": cp.Code, "full_name": cp.FullName},
		IPAddress: audit.ExtractIP(r), UserAgent: r.UserAgent(),
	})

	httputil.Created(w, r, counterpartyToResponse(cp))
}

// UpdateCounterparty updates a counterparty (admin).
func (h *Handler) UpdateCounterparty(w http.ResponseWriter, r *http.Request) {
	id, err := httputil.ParseUUID(chi.URLParam(r, "id"))
	if err != nil {
		httputil.Error(w, r, apperror.New(apperror.ErrValidation, "invalid counterparty ID"))
		return
	}

	var req dto.UpdateCounterpartyRequest
	if err := httputil.ParseBody(r, &req); err != nil {
		httputil.Error(w, r, apperror.New(apperror.ErrValidation, err.Error()))
		return
	}

	cp, err := h.cpRepo.GetByID(r.Context(), id)
	if err != nil {
		httputil.Error(w, r, err)
		return
	}

	oldValues := map[string]interface{}{"full_name": cp.FullName, "swift_code": cp.SwiftCode}

	if req.FullName != nil {
		cp.FullName = *req.FullName
	}
	if req.ShortName != nil {
		cp.ShortName = req.ShortName
	}
	if req.SwiftCode != nil {
		cp.SwiftCode = req.SwiftCode
	}
	if req.CountryCode != nil {
		cp.CountryCode = req.CountryCode
	}
	if req.TaxID != nil {
		cp.TaxID = req.TaxID
	}
	if req.Address != nil {
		cp.Address = req.Address
	}
	if req.FxUsesLimit != nil {
		cp.FxUsesLimit = *req.FxUsesLimit
	}

	if err := h.cpRepo.Update(r.Context(), cp); err != nil {
		httputil.Error(w, r, err)
		return
	}

	userID := ctxutil.GetUserUUID(r.Context())
	h.auditLog.Log(r.Context(), audit.Entry{
		UserID: userID, FullName: "admin", Action: "UPDATE_COUNTERPARTY",
		DealModule: "SYSTEM", DealID: &id,
		OldValues: oldValues, NewValues: req,
		IPAddress: audit.ExtractIP(r), UserAgent: r.UserAgent(),
	})

	// Re-fetch
	updated, err := h.cpRepo.GetByID(r.Context(), id)
	if err != nil {
		httputil.Error(w, r, err)
		return
	}
	httputil.Success(w, r, counterpartyToResponse(updated))
}

// DeleteCounterparty soft-deletes a counterparty (admin).
func (h *Handler) DeleteCounterparty(w http.ResponseWriter, r *http.Request) {
	id, err := httputil.ParseUUID(chi.URLParam(r, "id"))
	if err != nil {
		httputil.Error(w, r, apperror.New(apperror.ErrValidation, "invalid counterparty ID"))
		return
	}

	userID := ctxutil.GetUserUUID(r.Context())
	if err := h.cpRepo.SoftDelete(r.Context(), id, userID); err != nil {
		httputil.Error(w, r, err)
		return
	}

	h.auditLog.Log(r.Context(), audit.Entry{
		UserID: userID, FullName: "admin", Action: "DELETE_COUNTERPARTY",
		DealModule: "SYSTEM", DealID: &id,
		IPAddress: audit.ExtractIP(r), UserAgent: r.UserAgent(),
	})

	httputil.NoContent(w)
}

// --- Currencies ---

// ListCurrencies lists all active currencies.
func (h *Handler) ListCurrencies(w http.ResponseWriter, r *http.Request) {
	currencies, err := h.mdRepo.ListCurrencies(r.Context())
	if err != nil {
		httputil.Error(w, r, err)
		return
	}

	items := make([]dto.CurrencyResponse, 0, len(currencies))
	for _, c := range currencies {
		items = append(items, dto.CurrencyResponse{
			ID: c.ID, Code: c.Code, NumericCode: c.NumericCode,
			Name: c.Name, DecimalPlaces: c.DecimalPlaces, IsActive: c.IsActive,
		})
	}
	httputil.Success(w, r, items)
}

// --- Currency Pairs ---

// ListCurrencyPairs lists all active currency pairs.
func (h *Handler) ListCurrencyPairs(w http.ResponseWriter, r *http.Request) {
	pairs, err := h.mdRepo.ListCurrencyPairs(r.Context())
	if err != nil {
		httputil.Error(w, r, err)
		return
	}

	items := make([]dto.CurrencyPairResponse, 0, len(pairs))
	for _, p := range pairs {
		items = append(items, dto.CurrencyPairResponse{
			ID: p.ID, BaseCurrency: p.BaseCurrency, QuoteCurrency: p.QuoteCurrency,
			PairCode: p.PairCode, RateDecimalPlaces: p.RateDecimalPlaces,
			CalculationRule: p.CalculationRule, ResultCurrency: p.ResultCurrency, IsActive: p.IsActive,
		})
	}
	httputil.Success(w, r, items)
}

// --- Branches ---

// ListBranches lists all active branches.
func (h *Handler) ListBranches(w http.ResponseWriter, r *http.Request) {
	branches, err := h.mdRepo.ListBranches(r.Context())
	if err != nil {
		httputil.Error(w, r, err)
		return
	}

	items := make([]dto.BranchResponse, 0, len(branches))
	for _, b := range branches {
		items = append(items, dto.BranchResponse{
			ID: b.ID, Code: b.Code, Name: b.Name, BranchType: b.BranchType,
			ParentBranchID: b.ParentBranchID, FlexcubeBranch: b.FlexcubeBranch,
			SwiftBranchCode: b.SwiftBranchCode, Address: b.Address, IsActive: b.IsActive,
		})
	}
	httputil.Success(w, r, items)
}

// --- Exchange Rates ---

// ListExchangeRates lists exchange rates with filters.
func (h *Handler) ListExchangeRates(w http.ResponseWriter, r *http.Request) {
	pag := httputil.ParsePagination(r)

	filter := dto.ExchangeRateFilter{}
	if v := r.URL.Query().Get("currency_code"); v != "" {
		filter.CurrencyCode = &v
	}
	if v := r.URL.Query().Get("from_date"); v != "" {
		filter.FromDate = &v
	}
	if v := r.URL.Query().Get("to_date"); v != "" {
		filter.ToDate = &v
	}

	rates, total, err := h.mdRepo.ListExchangeRates(r.Context(), filter, pag)
	if err != nil {
		httputil.Error(w, r, err)
		return
	}

	items := make([]dto.ExchangeRateResponse, 0, len(rates))
	for _, er := range rates {
		items = append(items, dto.ExchangeRateResponse{
			ID: er.ID, CurrencyCode: er.CurrencyCode,
			EffectiveDate: er.EffectiveDate.Format("2006-01-02"),
			BuyTransferRate: er.BuyTransferRate, SellTransferRate: er.SellTransferRate,
			MidRate: er.MidRate, Source: er.Source, CreatedAt: er.CreatedAt,
		})
	}

	result := dto.NewPaginationResponse(items, total, pag.Page, pag.PageSize)
	httputil.Success(w, r, result)
}

// GetLatestRate gets the latest rate for a currency.
func (h *Handler) GetLatestRate(w http.ResponseWriter, r *http.Request) {
	currencyCode := r.URL.Query().Get("currency_code")
	if currencyCode == "" {
		httputil.Error(w, r, apperror.New(apperror.ErrValidation, "currency_code query param required"))
		return
	}

	rate, err := h.mdRepo.GetLatestRate(r.Context(), currencyCode)
	if err != nil {
		httputil.Error(w, r, err)
		return
	}

	httputil.Success(w, r, dto.ExchangeRateResponse{
		ID: rate.ID, CurrencyCode: rate.CurrencyCode,
		EffectiveDate: rate.EffectiveDate.Format("2006-01-02"),
		BuyTransferRate: rate.BuyTransferRate, SellTransferRate: rate.SellTransferRate,
		MidRate: rate.MidRate, Source: rate.Source, CreatedAt: rate.CreatedAt,
	})
}

// --- helpers ---

func counterpartyToResponse(cp *model.Counterparty) dto.CounterpartyResponse {
	return dto.CounterpartyResponse{
		ID: cp.ID, Code: cp.Code, FullName: cp.FullName, ShortName: cp.ShortName,
		CIF: cp.CIF, SwiftCode: cp.SwiftCode, CountryCode: cp.CountryCode,
		TaxID: cp.TaxID, Address: cp.Address, FxUsesLimit: cp.FxUsesLimit,
		IsActive: cp.IsActive, CreatedAt: cp.CreatedAt, UpdatedAt: cp.UpdatedAt,
	}
}
