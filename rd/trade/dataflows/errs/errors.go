package errs

import (
	"fmt"
)

type VendorError struct {
	Symbol  string
	Detail  string
	Message string
}

func (e *VendorError) Error() string {
	return e.Message
}

type NoMarketDataError struct {
	VendorError
	Canonical string
}

func NewNoMarketDataError(symbol string, canonical string, detail string) *NoMarketDataError {
	msg := fmt.Sprintf("No market data for %q", symbol)
	if canonical != "" && canonical != symbol {
		msg += fmt.Sprintf(" (queried as %q)", canonical)
	}
	if detail != "" {
		msg += fmt.Sprintf(": %s", detail)
	}
	return &NoMarketDataError{
		VendorError: VendorError{
			Symbol:  symbol,
			Detail:  detail,
			Message: msg,
		},
		Canonical: canonical,
	}
}

type VendorRateLimitError struct {
	VendorError
	Vendor string
}

func NewVendorRateLimitError(vendor, symbol string) *VendorRateLimitError {
	return &VendorRateLimitError{
		VendorError: VendorError{
			Symbol:  symbol,
			Message: fmt.Sprintf("Vendor %s rate limit exceeded for %s", vendor, symbol),
		},
		Vendor: vendor,
	}
}

type VendorNotConfiguredError struct {
	VendorError
	Vendor string
}

func NewVendorNotConfiguredError(vendor, message string) *VendorNotConfiguredError {
	return &VendorNotConfiguredError{
		VendorError: VendorError{
			Message: message,
		},
		Vendor: vendor,
	}
}
