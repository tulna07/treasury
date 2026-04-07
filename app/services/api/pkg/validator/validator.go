// Package validator provides custom validation rules for banking operations.
package validator

import (
	"regexp"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/shopspring/decimal"
)

var currencyRegex = regexp.MustCompile(`^[A-Z]{3}$`)

// zeroCurrencies are currencies with 0 decimal places.
var zeroCurrencies = map[string]bool{
	"VND": true,
	"JPY": true,
	"KRW": true,
}

// New creates a new validator instance with custom banking rules registered.
func New() *validator.Validate {
	v := validator.New()

	// currency — validates ISO 4217 (3 chars uppercase)
	_ = v.RegisterValidation("currency", validateCurrency)

	// deal_type — validates against known deal types
	_ = v.RegisterValidation("deal_type", validateDealType)

	// amount_vnd — validates VND amounts have 0 decimal places
	_ = v.RegisterValidation("amount_vnd", validateAmountVND)

	// amount_fcy — validates foreign currency amounts have max 2 decimal places
	_ = v.RegisterValidation("amount_fcy", validateAmountFCY)

	return v
}

func validateCurrency(fl validator.FieldLevel) bool {
	return currencyRegex.MatchString(fl.Field().String())
}

func validateDealType(fl validator.FieldLevel) bool {
	allowed := map[string]bool{
		"SPOT": true, "FORWARD": true, "SWAP": true,
		"DEPOSIT": true, "LOAN": true, "REPO": true, "REVERSE_REPO": true,
		"GOVERNMENT_BOND": true, "CORPORATE_BOND": true,
		"TREASURY_BILL": true, "CERTIFICATE_OF_DEPOSIT": true,
	}
	return allowed[strings.ToUpper(fl.Field().String())]
}

func validateAmountVND(fl validator.FieldLevel) bool {
	d, ok := fl.Field().Interface().(decimal.Decimal)
	if !ok {
		return false
	}
	// VND must have 0 decimal places
	return d.Equal(d.Truncate(0))
}

func validateAmountFCY(fl validator.FieldLevel) bool {
	d, ok := fl.Field().Interface().(decimal.Decimal)
	if !ok {
		return false
	}
	// FCY must have max 2 decimal places
	return d.Equal(d.Truncate(2))
}

// IsZeroDecimalCurrency checks if a currency uses 0 decimal places.
func IsZeroDecimalCurrency(currency string) bool {
	return zeroCurrencies[strings.ToUpper(currency)]
}

// ValidateStruct validates a struct and returns formatted error messages.
func ValidateStruct(v *validator.Validate, s interface{}) []string {
	err := v.Struct(s)
	if err == nil {
		return nil
	}

	var errs []string
	for _, e := range err.(validator.ValidationErrors) {
		errs = append(errs, formatValidationError(e))
	}
	return errs
}

func formatValidationError(e validator.FieldError) string {
	field := e.Field()
	switch e.Tag() {
	case "required":
		return field + " is required"
	case "min":
		return field + " must be at least " + e.Param()
	case "max":
		return field + " must be at most " + e.Param()
	case "len":
		return field + " must be exactly " + e.Param() + " characters"
	case "oneof":
		return field + " must be one of: " + e.Param()
	case "gt":
		return field + " must be greater than " + e.Param()
	case "gte":
		return field + " must be greater than or equal to " + e.Param()
	case "currency":
		return field + " must be a valid ISO 4217 currency code"
	case "amount_vnd":
		return field + " must have 0 decimal places for VND"
	case "amount_fcy":
		return field + " must have at most 2 decimal places"
	default:
		return field + " failed validation: " + e.Tag()
	}
}
