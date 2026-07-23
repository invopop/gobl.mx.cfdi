package addon_test

import (
	"testing"
	"time"

	"github.com/invopop/gobl.mx.cfdi/addon"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cal"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/currency"
	"github.com/invopop/gobl/head"
	"github.com/invopop/gobl/norm"
	"github.com/invopop/gobl/num"
	"github.com/invopop/gobl/org"
	"github.com/invopop/gobl/pay"
	_ "github.com/invopop/gobl/regimes/mx"
	"github.com/invopop/gobl/rules"
	"github.com/invopop/gobl/tax"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidPayment(t *testing.T) {
	t.Run("basic", func(t *testing.T) {
		pmt := validPayment()
		require.NoError(t, pmt.Calculate())
		require.NoError(t, rules.Validate(pmt))
		assert.Equal(t, cbc.Code("P"), pmt.Ext.Get(addon.ExtKeyDocType))
		assert.Equal(t, cbc.Code("03"), pmt.Methods[0].Ext.Get(addon.ExtKeyPaymentMeans))
	})
	t.Run("foreign customer", func(t *testing.T) {
		pmt := validPayment()
		pmt.Customer.TaxID.Country = "US"
		pmt.Customer.Ext = tax.Extensions{}
		pmt.Customer.Addresses = nil
		require.NoError(t, pmt.Calculate())
		require.NoError(t, rules.Validate(pmt))
	})
}

func TestPaymentTypeValidation(t *testing.T) {
	pmt := validPayment()
	pmt.Type = bill.PaymentTypeRequest
	assertPaymentValidationError(t, pmt, "only payment receipts are supported")

	pmt.Type = bill.PaymentTypeAdvice
	assertPaymentValidationError(t, pmt, "only payment receipts are supported")

	pmt.Type = bill.PaymentTypeReceipt
	require.NoError(t, pmt.Calculate())
	require.NoError(t, rules.Validate(pmt))
}

func TestPaymentCurrencyValidation(t *testing.T) {
	t.Run("non-MXN currency without exchange rates", func(t *testing.T) {
		pmt := validPayment()
		pmt.Currency = "USD"
		require.NoError(t, pmt.Calculate())
		err := rules.Validate(pmt)
		assert.ErrorContains(t, err, "[GOBL-MX-CFDI-BILL-PAYMENT-02] payment must be in MXN or provide exchange rate for conversion")
	})

	t.Run("non-MXN currency with exchange rates", func(t *testing.T) {
		pmt := validPayment()
		pmt.Currency = "USD"
		pmt.ExchangeRates = []*currency.ExchangeRate{
			{
				From:   "USD",
				To:     "MXN",
				Amount: num.MakeAmount(1800, 2),
			},
		}
		require.NoError(t, pmt.Calculate())
		assert.NoError(t, rules.Validate(pmt))
	})
}

func TestPaymentExtensionsValidation(t *testing.T) {
	t.Run("missing issue place", func(t *testing.T) {
		pmt := validPayment()
		pmt.Ext = tax.Extensions{}
		assertPaymentValidationError(t, pmt, "payment requires 'mx-cfdi-doc-type' and 'mx-cfdi-issue-place' extensions")
	})

	t.Run("missing document type", func(t *testing.T) {
		// The normalizer always sets the document type, so it needs to be
		// removed after calculation to trigger the validation error.
		pmt := validPayment()
		require.NoError(t, pmt.Calculate())
		pmt.Ext = pmt.Ext.Delete(addon.ExtKeyDocType)
		err := rules.Validate(pmt)
		assert.ErrorContains(t, err, "payment requires 'mx-cfdi-doc-type' and 'mx-cfdi-issue-place' extensions")
	})
}

func TestPaymentSupplierValidation(t *testing.T) {
	pmt := validPayment()
	pmt.Supplier.TaxID = nil
	assertPaymentValidationError(t, pmt, "supplier tax ID is required")

	pmt = validPayment()
	pmt.Supplier.TaxID.Code = ""
	assertPaymentValidationError(t, pmt, "supplier tax ID code is required")

	pmt = validPayment()
	pmt.Supplier.Ext = tax.Extensions{}
	assertPaymentValidationError(t, pmt, "supplier requires 'mx-cfdi-fiscal-regime' extension")
}

func TestPaymentCustomerValidation(t *testing.T) {
	pmt := validPayment()
	pmt.Customer = nil
	assertPaymentValidationError(t, pmt, "customer is required")

	pmt = validPayment()
	pmt.Customer.TaxID = nil
	assertPaymentValidationError(t, pmt, "customer tax ID is required")

	pmt = validPayment()
	pmt.Customer.TaxID.Code = ""
	assertPaymentValidationError(t, pmt, "customer tax ID code is required")

	pmt = validPayment()
	pmt.Customer.Ext = tax.Extensions{}
	assertPaymentValidationError(t, pmt, "Mexican customer requires 'mx-cfdi-fiscal-regime' extension")

	pmt = validPayment()
	pmt.Customer.Addresses = nil
	assertPaymentValidationError(t, pmt, "Mexican customer must have at least one address")

	pmt = validPayment()
	pmt.Customer.Addresses[0].Code = ""
	assertPaymentValidationError(t, pmt, "customer address postal code is required")

	pmt = validPayment()
	pmt.Customer.Addresses[0].Code = "ABC"
	assertPaymentValidationError(t, pmt, "customer address postal code format is invalid")
}

func TestPaymentCustomerUseValidation(t *testing.T) {
	// The normalizer always sets the CFDI use, so it needs to be changed after
	// calculation to trigger the validation error.
	t.Run("missing", func(t *testing.T) {
		pmt := validPayment()
		require.NoError(t, pmt.Calculate())
		pmt.Customer.Ext = pmt.Customer.Ext.Delete(addon.ExtKeyUse)
		err := rules.Validate(pmt)
		assert.ErrorContains(t, err, "customer's 'mx-cfdi-use' must be 'CP01' for payments")
	})

	t.Run("wrong code", func(t *testing.T) {
		pmt := validPayment()
		require.NoError(t, pmt.Calculate())
		pmt.Customer.Ext = pmt.Customer.Ext.Set(addon.ExtKeyUse, "G01")
		err := rules.Validate(pmt)
		assert.ErrorContains(t, err, "customer's 'mx-cfdi-use' must be 'CP01' for payments")
	})
}

func TestPaymentPayeeValidation(t *testing.T) {
	pmt := validPayment()
	pmt.Payee = &org.Party{
		Name: "Third Party",
		TaxID: &tax.Identity{
			Country: "MX",
			Code:    "AAA010101AAA",
		},
	}
	assertPaymentValidationError(t, pmt, "payee cannot be represented in a CFDI payment")
}

func TestPaymentMethodsValidation(t *testing.T) {
	t.Run("multiple methods", func(t *testing.T) {
		pmt := validPayment()
		pmt.Methods = []*pay.Record{
			{
				Key:    pay.MeansKeyCreditTransfer,
				Amount: num.MakeAmount(5800, 2),
			},
			{
				Key:    pay.MeansKeyCash,
				Amount: num.MakeAmount(5800, 2),
			},
		}
		assertPaymentValidationError(t, pmt, "exactly one payment method is required")
	})

	t.Run("missing means extension", func(t *testing.T) {
		pmt := validPayment()
		pmt.Methods[0].Key = pay.MeansKeyOther
		assertPaymentValidationError(t, pmt, "payment method requires 'mx-cfdi-payment-means' extension")
	})

	t.Run("means to define", func(t *testing.T) {
		pmt := validPayment()
		pmt.Methods[0].Key = pay.MeansKeyOther
		pmt.Methods[0].Ext = tax.ExtensionsOf(cbc.CodeMap{
			addon.ExtKeyPaymentMeans: "99",
		})
		assertPaymentValidationError(t, pmt, "payment means '99' (to define) is not allowed")
	})
}

func TestPaymentLinesValidation(t *testing.T) {
	tests := []struct {
		name   string
		modify func(l *bill.PaymentLine)
		err    string
	}{
		{
			name:   "valid line",
			modify: func(_ *bill.PaymentLine) {},
		},
		{
			name: "refund line",
			modify: func(l *bill.PaymentLine) {
				l.Refund = true
			},
			err: "refund lines are not supported",
		},
		{
			name: "missing installment",
			modify: func(l *bill.PaymentLine) {
				l.Installment = 0
			},
			err: "installment number is required",
		},
		{
			name: "missing payable",
			modify: func(l *bill.PaymentLine) {
				l.Payable = nil
			},
			err: "payable amount is required",
		},
		{
			name: "missing document",
			modify: func(l *bill.PaymentLine) {
				l.Document = nil
			},
			err: "line document is required",
		},
		{
			name: "wrong document stamp",
			modify: func(l *bill.PaymentLine) {
				l.Document.Stamps = []*head.Stamp{
					{
						Provider: "unexpected",
						Value:    "1234",
					},
				}
			},
			err: "line document requires the 'sat-uuid' stamp",
		},
		{
			name: "no document stamps",
			modify: func(l *bill.PaymentLine) {
				l.Document.Stamps = nil
			},
			err: "line document requires the 'sat-uuid' stamp",
		},
		{
			name: "tax rate surcharge",
			modify: func(l *bill.PaymentLine) {
				l.Tax = &tax.Total{
					Categories: []*tax.CategoryTotal{
						{
							Code: "VAT",
							Rates: []*tax.RateTotal{
								{
									Base:    num.MakeAmount(10000, 2),
									Percent: num.NewPercentage(16, 2),
									Surcharge: &tax.RateTotalSurcharge{
										Percent: num.MakePercentage(8, 3),
										Amount:  num.MakeAmount(80, 2),
									},
								},
							},
						},
					},
				}
			},
			err: "tax rate surcharges are not supported",
		},
		{
			name: "retained rate without percent",
			modify: func(l *bill.PaymentLine) {
				l.Tax = &tax.Total{
					Categories: []*tax.CategoryTotal{
						{
							Code:     "VAT",
							Retained: true,
							Rates: []*tax.RateTotal{
								{
									Base: num.MakeAmount(10000, 2),
								},
							},
						},
					},
				}
			},
			err: "retained tax rates must have a percent",
		},
	}

	for _, ts := range tests {
		t.Run(ts.name, func(t *testing.T) {
			pmt := validPayment()
			ts.modify(pmt.Lines[0])
			require.NoError(t, pmt.Calculate())
			err := rules.Validate(pmt)
			if ts.err == "" {
				assert.NoError(t, err)
			} else {
				if assert.Error(t, err) {
					assert.Contains(t, err.Error(), ts.err)
				}
			}
		})
	}
}

func TestPaymentLineDueValidation(t *testing.T) {
	// The core payment calculation fills in the due amount from payable, so it
	// needs to be removed after calculation to trigger the validation error.
	pmt := validPayment()
	require.NoError(t, pmt.Calculate())
	pmt.Lines[0].Due = nil
	err := rules.Validate(pmt)
	assert.ErrorContains(t, err, "due amount is required")
}

func TestPaymentLineDocumentCurrencyValidation(t *testing.T) {
	pmt := validPayment()
	pmt.ExchangeRates = []*currency.ExchangeRate{
		{
			From:   "USD",
			To:     "MXN",
			Amount: num.MakeAmount(1800, 2),
		},
	}
	pmt.Lines[0].Document.Currency = "USD"
	assertPaymentValidationError(t, pmt, "line document currency must match the payment currency")

	pmt.Lines[0].Document.Currency = ""
	require.NoError(t, pmt.Calculate())
	require.NoError(t, rules.Validate(pmt))
}

func TestPaymentPrecedingValidation(t *testing.T) {
	pmt := validPayment()
	pmt.Preceding = []*org.DocumentRef{
		{
			Code: "0122",
			Stamps: []*head.Stamp{
				{
					Provider: "unexpected",
					Value:    "1234",
				},
			},
		},
	}
	assertPaymentValidationError(t, pmt, "preceding row is missing 'sat-uuid' stamp")

	// A preceding row with no stamps at all is rejected the same way.
	pmt.Preceding[0].Stamps = nil
	assertPaymentValidationError(t, pmt, "preceding row is missing 'sat-uuid' stamp")

	pmt.Preceding = []*org.DocumentRef{
		{
			Code: "0122",
			Stamps: []*head.Stamp{
				{
					Provider: "sat-uuid",
					Value:    "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
				},
			},
		},
	}
	require.NoError(t, pmt.Calculate())
	require.NoError(t, rules.Validate(pmt))

	// The normalizer always sets the relation type, so it needs to be
	// removed after calculation to trigger the validation error.
	pmt.Preceding[0].Ext = pmt.Preceding[0].Ext.Delete(addon.ExtKeyRelType)
	err := rules.Validate(pmt)
	assert.ErrorContains(t, err, "preceding row's 'mx-cfdi-rel-type' must be '04'")
}

func TestNormalizePayment(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		var pmt *bill.Payment
		assert.NotPanics(t, func() {
			norm.Normalize(pmt, tax.AddonContext(addon.V4))
		})
	})

	t.Run("sets document type", func(t *testing.T) {
		pmt := validPayment()
		norm.Normalize(pmt, tax.AddonContext(addon.V4))
		assert.Equal(t, cbc.Code("P"), pmt.Ext.Get(addon.ExtKeyDocType))
	})

	t.Run("sets customer CFDI use", func(t *testing.T) {
		pmt := validPayment()
		pmt.Customer.Ext = tax.Extensions{}
		norm.Normalize(pmt, tax.AddonContext(addon.V4))
		assert.Equal(t, cbc.Code("CP01"), pmt.Customer.Ext.Get(addon.ExtKeyUse))
	})

	t.Run("overwrites document type", func(t *testing.T) {
		pmt := validPayment()
		pmt.Ext = pmt.Ext.Set(addon.ExtKeyDocType, "I")
		norm.Normalize(pmt, tax.AddonContext(addon.V4))
		assert.Equal(t, cbc.Code("P"), pmt.Ext.Get(addon.ExtKeyDocType))
	})

	t.Run("defaults preceding relation type", func(t *testing.T) {
		pmt := validPayment()
		pmt.Preceding = []*org.DocumentRef{
			{
				Code: "0122",
			},
		}
		norm.Normalize(pmt, tax.AddonContext(addon.V4))
		assert.Equal(t, cbc.Code("04"), pmt.Preceding[0].Ext.Get(addon.ExtKeyRelType))
	})

	t.Run("keeps preceding relation type", func(t *testing.T) {
		pmt := validPayment()
		pmt.Preceding = []*org.DocumentRef{
			{
				Code: "0122",
				Ext: tax.ExtensionsOf(cbc.CodeMap{
					addon.ExtKeyRelType: "07",
				}),
			},
		}
		norm.Normalize(pmt, tax.AddonContext(addon.V4))
		assert.Equal(t, cbc.Code("07"), pmt.Preceding[0].Ext.Get(addon.ExtKeyRelType))
	})

	t.Run("should set time and date", func(t *testing.T) {
		// These tests can fail very rarely if run on the exact transition of the milliseconds
		tz, err := time.LoadLocation("America/Mexico_City")
		require.NoError(t, err)
		pmt := validPayment()
		pmt.IssueTime = nil
		tn := time.Now().In(tz)
		require.NoError(t, pmt.Calculate())
		assert.NotNil(t, pmt.IssueTime)
		assert.Equal(t, tn.Format("2006-01-02"), pmt.IssueDate.String())
		assert.Equal(t, tn.Format("15:04:05"), pmt.IssueTime.String())
	})
}

func assertPaymentValidationError(t *testing.T, pmt *bill.Payment, expected string) {
	t.Helper()
	require.NoError(t, pmt.Calculate())
	err := rules.Validate(pmt)
	require.Error(t, err)
	assert.Contains(t, err.Error(), expected)
}

func validPayment() *bill.Payment {
	return &bill.Payment{
		Addons:    tax.WithAddons(addon.V4),
		Series:    "P",
		Code:      "0123",
		Currency:  "MXN",
		IssueDate: cal.MakeDate(2025, 3, 1),
		IssueTime: cal.NewTime(10, 30, 0),
		Ext: tax.ExtensionsOf(cbc.CodeMap{
			addon.ExtKeyIssuePlace: "21000",
		}),
		Supplier: &org.Party{
			Name: "Test Supplier",
			Ext: tax.ExtensionsOf(cbc.CodeMap{
				addon.ExtKeyFiscalRegime: "601",
			}),
			TaxID: &tax.Identity{
				Country: "MX",
				Code:    "AAA010101AAA",
			},
		},
		Customer: &org.Party{
			Name: "Test Customer",
			Ext: tax.ExtensionsOf(cbc.CodeMap{
				addon.ExtKeyFiscalRegime: "608",
			}),
			TaxID: &tax.Identity{
				Country: "MX",
				Code:    "ZZZ010101ZZZ",
			},
			Addresses: []*org.Address{
				{
					Locality: "Mexico",
					Code:     "65000",
				},
			},
		},
		Lines: []*bill.PaymentLine{
			{
				Installment: 1,
				Document: &org.DocumentRef{
					Series:    "F",
					Code:      "00123",
					IssueDate: cal.NewDate(2025, 2, 1),
					Stamps: []*head.Stamp{
						{
							Provider: "sat-uuid",
							Value:    "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
						},
					},
				},
				Payable: num.NewAmount(11600, 2),
				Amount:  num.MakeAmount(11600, 2),
				Due:     num.NewAmount(0, 2),
			},
		},
		Methods: []*pay.Record{
			{
				Key: pay.MeansKeyCreditTransfer,
			},
		},
	}
}
