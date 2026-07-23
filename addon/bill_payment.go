package addon

import (
	"fmt"

	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/currency"
	"github.com/invopop/gobl/head"
	"github.com/invopop/gobl/regimes/mx"
	"github.com/invopop/gobl/rules"
	"github.com/invopop/gobl/rules/is"
	"github.com/invopop/gobl/tax"
)

func normalizePayment(pmt *bill.Payment) {
	if pmt == nil {
		return
	}
	normalizePaymentIssueDateAndTime(pmt)

	// Normalize required extensions with a single allowed value
	pmt.Ext = pmt.Ext.Set(ExtKeyDocType, DocTypePayment)
	if pmt.Customer != nil {
		pmt.Customer.Ext = pmt.Customer.Ext.SetIfEmpty(ExtKeyUse, UsePayments)
	}
	for _, p := range pmt.Preceding {
		if p != nil {
			p.Ext = p.Ext.SetIfEmpty(ExtKeyRelType, RelTypeSubstitution)
		}
	}
}

func normalizePaymentIssueDateAndTime(pmt *bill.Payment) {
	if pmt == nil {
		return
	}
	// Overwrite the issue date and time to align with CFDI requirements for the
	// emission date, unless the issue time is already set.
	if pmt.IssueTime != nil && !pmt.IssueTime.IsZero() {
		return
	}
	pmt.IssueDate, pmt.IssueTime = currentIssueDateTime()
}

func billPaymentRules() *rules.Set {
	return rules.For(new(bill.Payment),
		rules.Assert("01", "only payment receipts are supported", bill.PaymentTypeIn(bill.PaymentTypeReceipt)),
		rules.Assert("02", "payment must be in MXN or provide exchange rate for conversion",
			is.Func("can convert to MXN", paymentCanConvertToMXN), // currency.CanConvertTo cannot be used here as bill.Payment does not implement the required interface.
		),
		// Document extensions
		rules.Field("ext",
			rules.Assert("03",
				fmt.Sprintf("payment requires '%s' and '%s' extensions", ExtKeyDocType, ExtKeyIssuePlace),
				tax.ExtensionsRequire(ExtKeyDocType, ExtKeyIssuePlace),
			),
			rules.Assert("04",
				fmt.Sprintf("payment document type must be '%s'", DocTypePayment),
				tax.ExtensionsHasCodes(ExtKeyDocType, DocTypePayment),
			),
		),
		// Supplier validation
		rules.Field("supplier",
			rules.Field("tax_id",
				rules.Assert("05", "supplier tax ID is required", is.Present),
				rules.Field("code",
					rules.Assert("06", "supplier tax ID code is required", is.Present),
				),
			),
			rules.Field("ext",
				rules.Assert("07",
					fmt.Sprintf("supplier requires '%s' extension", ExtKeyFiscalRegime),
					tax.ExtensionsRequire(ExtKeyFiscalRegime),
				),
			),
		),
		// Customer validation
		rules.Field("customer",
			// A payment cannot be issued to a generic customer
			rules.Assert("08", "customer is required", is.Present),
			rules.Field("tax_id",
				rules.Assert("09", "customer tax ID is required", is.Present),
				rules.Field("code",
					rules.Assert("10", "customer tax ID code is required", is.Present),
				),
			),
			rules.Field("ext",
				rules.Assert("11",
					fmt.Sprintf("customer's '%s' must be '%s' for payments", ExtKeyUse, UsePayments),
					tax.ExtensionsRequire(ExtKeyUse),
					tax.ExtensionsHasCodes(ExtKeyUse, UsePayments),
				),
			),
			rules.When(is.Func("customer is Mexican", partyIsMexican),
				rules.Field("ext",
					rules.Assert("12",
						fmt.Sprintf("Mexican customer requires '%s' extension", ExtKeyFiscalRegime),
						tax.ExtensionsRequire(ExtKeyFiscalRegime),
					),
				),
				rules.Field("addresses",
					rules.Assert("13", "Mexican customer must have at least one address", is.Present),
					rules.Each(
						rules.Field("code",
							rules.Assert("14", "customer address postal code is required", is.Present),
							rules.Assert("15", "customer address postal code format is invalid",
								is.Matches(PostCodePattern),
							),
						),
					),
				),
			),
		),
		rules.Field("payee",
			rules.Assert("16", "payee cannot be represented in a CFDI payment", is.Empty),
		),
		// Payment methods
		rules.Field("methods",
			rules.Assert("17", "exactly one payment method is required", is.Length(1, 1)),
			rules.Each(
				rules.Field("ext",
					rules.Assert("18",
						fmt.Sprintf("payment method requires '%s' extension", ExtKeyPaymentMeans),
						tax.ExtensionsRequire(ExtKeyPaymentMeans),
					),
					rules.Assert("19", "payment means '99' (to define) is not allowed",
						tax.ExtensionsExcludeCodes(ExtKeyPaymentMeans, PaymentMeansToDefine),
					),
				),
			),
		),
		// Line validation
		rules.Field("lines",
			rules.Each(
				rules.Field("refund",
					rules.Assert("20", "refund lines are not supported", is.Empty),
				),
				rules.Field("installment",
					rules.Assert("21", "installment number is required", is.Present),
				),
				rules.Field("payable",
					rules.Assert("22", "payable amount is required", is.Present),
				),
				rules.Field("due",
					rules.Assert("23", "due amount is required", is.Present),
				),
				rules.Field("document",
					rules.Assert("24", "line document is required", is.Present),
					rules.Field("stamps",
						rules.Assert("25", fmt.Sprintf("line document requires the '%s' stamp", mx.StampSATUUID),
							head.StampsHas(mx.StampSATUUID),
						),
					),
				),
				rules.Field("tax",
					rules.Assert("26", "tax rate surcharges are not supported",
						is.Func("no rate surcharges", lineTaxNoSurcharges),
					),
					rules.Assert("27", "retained tax rates must have a percent",
						is.Func("retained rates with percent", lineTaxRetainedRatesWithPercent),
					),
				),
			),
		),
		rules.Assert("28", "line document currency must match the payment currency",
			is.Func("line document currencies match", paymentLineDocCurrenciesMatch),
		),
		// Preceding validation
		rules.Field("preceding",
			rules.Each(
				rules.Field("ext",
					rules.Assert("29",
						fmt.Sprintf("preceding row's '%s' must be '%s'", ExtKeyRelType, RelTypeSubstitution),
						tax.ExtensionsRequire(ExtKeyRelType),
						tax.ExtensionsHasCodes(ExtKeyRelType, RelTypeSubstitution),
					),
				),
				rules.Field("stamps",
					rules.Assert("30", fmt.Sprintf("preceding row is missing '%s' stamp", mx.StampSATUUID),
						head.StampsHas(mx.StampSATUUID),
					),
				),
			),
		),
	)
}

// paymentCanConvertToMXN checks that the payment is either issued in MXN or provides an
// exchange rate to convert its amounts into MXN.
func paymentCanConvertToMXN(val any) bool {
	pmt, ok := val.(*bill.Payment)
	if !ok || pmt == nil {
		return true
	}
	if pmt.Currency == currency.MXN {
		return true
	}
	return currency.MatchExchangeRate(pmt.ExchangeRates, pmt.Currency, currency.MXN) != nil
}

// paymentLineDocCurrenciesMatch checks that every line document's currency is either
// empty or the same as the payment's currency. Cross-currency related documents are not
// supported.
func paymentLineDocCurrenciesMatch(val any) bool {
	pmt, ok := val.(*bill.Payment)
	if !ok || pmt == nil {
		return true
	}
	for _, l := range pmt.Lines {
		if l == nil || l.Document == nil {
			continue
		}
		if l.Document.Currency != currency.CodeEmpty && l.Document.Currency != pmt.Currency {
			return false
		}
	}
	return true
}

// lineTaxNoSurcharges checks that no rate in a line's tax total carries a surcharge.
func lineTaxNoSurcharges(val any) bool {
	t, ok := val.(*tax.Total)
	if !ok || t == nil {
		return true
	}
	for _, ct := range t.Categories {
		for _, rt := range ct.Rates {
			if rt.Surcharge != nil {
				return false
			}
		}
	}
	return true
}

// lineTaxRetainedRatesWithPercent checks that every retained rate in a line's tax total
// has a percent.
func lineTaxRetainedRatesWithPercent(val any) bool {
	t, ok := val.(*tax.Total)
	if !ok || t == nil {
		return true
	}
	for _, ct := range t.Categories {
		if !ct.Retained {
			continue
		}
		for _, rt := range ct.Rates {
			if rt.Percent == nil {
				return false
			}
		}
	}
	return true
}
