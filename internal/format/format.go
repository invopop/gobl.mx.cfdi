// Package format contains helps to help format output.
package format //nolint:revive

import (
	"fmt"

	"github.com/invopop/gobl/num"
)

// OptionalAmount provides empty string for zero amounts.
func OptionalAmount(a num.Amount) string {
	if a.IsZero() {
		return ""
	}

	return a.String()
}

// SchemaLocation provides a string with the namespace and schema location.
func SchemaLocation(namespace, schemaLocation string) string {
	return fmt.Sprintf("%s %s", namespace, schemaLocation)
}

// TaxPercent provides a string with the tax percentage rescaled according to
// CFDI requirements.
func TaxPercent(percent *num.Percentage) string {
	return percent.Base().Rescale(6).String()
}

// TaxRate ensures we add extra precision to the tax rate so that it can be used
// for matching and can be used for CFDI requirements.
func TaxRate(amount num.Amount) string {
	return amount.Rescale(6).String()
}
