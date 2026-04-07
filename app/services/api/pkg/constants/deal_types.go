package constants

// FX deal types.
const (
	FxTypeSpot    = "SPOT"
	FxTypeForward = "FORWARD"
	FxTypeSwap    = "SWAP"
)

// FX deal directions.
const (
	DirectionSell    = "SELL"
	DirectionBuy     = "BUY"
	DirectionSellBuy = "SELL_BUY"
	DirectionBuySell = "BUY_SELL"
)

// Bond deal categories (matching DB CHECK constraint).
const (
	BondCategoryGovernment            = "GOVERNMENT"
	BondCategoryFinancialInstitution  = "FINANCIAL_INSTITUTION"
	BondCategoryCertificateOfDeposit  = "CERTIFICATE_OF_DEPOSIT"
)

// Bond deal directions.
const (
	BondDirectionBuy  = "BUY"
	BondDirectionSell = "SELL"
)

// Bond transaction types.
const (
	BondTxRepo        = "REPO"
	BondTxReverseRepo = "REVERSE_REPO"
	BondTxOutright    = "OUTRIGHT"
	BondTxOther       = "OTHER"
)

// Bond portfolio types.
const (
	PortfolioHTM = "HTM"
	PortfolioAFS = "AFS"
	PortfolioHFT = "HFT"
)

// Bond confirmation methods.
const (
	ConfirmEmail   = "EMAIL"
	ConfirmReuters = "REUTERS"
	ConfirmOther   = "OTHER"
)

// Bond contract prepared by.
const (
	ContractInternal     = "INTERNAL"
	ContractCounterparty = "COUNTERPARTY"
)

// Money Market deal types.
const (
	MMTypeDeposit     = "DEPOSIT"
	MMTypeLoan        = "LOAN"
	MMTypeRepo        = "REPO"
	MMTypeReverseRepo = "REVERSE_REPO"
)

// Money Market directions.
const (
	MMDirectionPlace  = "PLACE"  // Gửi tiền
	MMDirectionTake   = "TAKE"   // Nhận tiền gửi
	MMDirectionLend   = "LEND"   // Cho vay
	MMDirectionBorrow = "BORROW" // Vay
)

// Money Market subtypes (OMO/Repo).
const (
	MMSubtypeOMO       = "OMO"
	MMSubtypeStateRepo = "STATE_REPO"
)

// Day count conventions.
const (
	DayCountACT365 = "ACT_365"
	DayCountACT360 = "ACT_360"
	DayCountACTACT = "ACT_ACT"
)

// AllFxTypes lists all valid FX deal types.
var AllFxTypes = []string{FxTypeSpot, FxTypeForward, FxTypeSwap}

// AllFxDirections lists all valid FX directions.
var AllFxDirections = []string{DirectionSell, DirectionBuy, DirectionSellBuy, DirectionBuySell}

// AllBondCategories lists all valid bond categories.
var AllBondCategories = []string{BondCategoryGovernment, BondCategoryFinancialInstitution, BondCategoryCertificateOfDeposit}

// AllBondDirections lists all valid bond directions.
var AllBondDirections = []string{BondDirectionBuy, BondDirectionSell}

// AllBondTxTypes lists all valid bond transaction types.
var AllBondTxTypes = []string{BondTxRepo, BondTxReverseRepo, BondTxOutright, BondTxOther}

// AllPortfolioTypes lists all valid portfolio types.
var AllPortfolioTypes = []string{PortfolioHTM, PortfolioAFS, PortfolioHFT}

// AllMMTypes lists all valid MM deal types.
var AllMMTypes = []string{MMTypeDeposit, MMTypeLoan, MMTypeRepo, MMTypeReverseRepo}

// AllMMInterbankDirections lists all valid MM interbank directions.
var AllMMInterbankDirections = []string{MMDirectionPlace, MMDirectionTake, MMDirectionLend, MMDirectionBorrow}

// AllMMSubtypes lists all valid MM OMO/Repo subtypes.
var AllMMSubtypes = []string{MMSubtypeOMO, MMSubtypeStateRepo}

// AllDayCountConventions lists all valid day count conventions.
var AllDayCountConventions = []string{DayCountACT365, DayCountACT360, DayCountACTACT}
