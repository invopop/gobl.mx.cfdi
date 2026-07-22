// Package format contains helps to help format output.
package format //nolint:revive

import (
	"fmt"

	"github.com/invopop/gobl/cal"
	"github.com/invopop/gobl/num"
)

// OptionalAmount provides empty string for zero amounts.
func OptionalAmount(a num.Amount) string {
	if a.IsZero() {
		return ""
	}

	return a.String()
}

// DateTime combines a date and an optional time into the date-time string
// required by CFDI.
func DateTime(date cal.Date, time *cal.Time) string {
	if time == nil {
		time = new(cal.Time) // zero
	}
	return date.WithTime(*time).String()
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
