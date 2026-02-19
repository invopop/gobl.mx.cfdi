package cfdi_test

import (
	"testing"

	"github.com/invopop/gobl.cfdi/test"
	addon "github.com/invopop/gobl/addons/mx/cfdi"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/currency"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseComprobanteIngreso(t *testing.T) {
	t.Run("should return an Invoice with the Comprobante data", func(t *testing.T) {
		inv, err := test.LoadParsedInvoice("invoice-b2b-full.xml")
		require.NoError(t, err)

		assert.Equal(t, bill.InvoiceTypeStandard, inv.Type)
		assert.Equal(t, cbc.Code("LMC"), inv.Series)
		assert.Equal(t, cbc.Code("0010"), inv.Code)
		assert.Equal(t, "2023-05-29", inv.IssueDate.String())
		assert.Equal(t, "12:00:00", inv.IssueTime.String())
		assert.Equal(t, currency.MXN, inv.Currency)

		assert.Equal(t, cbc.Code("I"), inv.Tax.Ext[addon.ExtKeyDocType])
		assert.Equal(t, cbc.Code("26015"), inv.Tax.Ext[addon.ExtKeyIssuePlace])
		assert.Equal(t, cbc.Code("PUE"), inv.Tax.Ext[addon.ExtKeyPaymentMethod])

		// After discount: 200.2020 + 10.50 + 10.00 = 220.70
		assert.Equal(t, "220.70", inv.Totals.Sum.String())
		assert.Equal(t, "220.70", inv.Totals.Total.String())
		assert.Equal(t, "252.73", inv.Totals.TotalWithTax.String())
		assert.Equal(t, "211.36", inv.Totals.Payable.String())
	})

	t.Run("should parse payment details properly", func(t *testing.T) {
		inv, err := test.LoadParsedInvoice("invoice-b2b-full.xml")
		require.NoError(t, err)

		require.NotNil(t, inv.Payment)
		require.NotNil(t, inv.Payment.Terms)
		assert.Equal(t, "Pago a 30 días.", inv.Payment.Terms.Notes)

		require.Len(t, inv.Payment.Advances, 1)
		adv := inv.Payment.Advances[0]
		assert.Equal(t, "Pago en una sola exhibición", adv.Description)
		assert.Equal(t, "100%", adv.Percent.String())
		assert.Equal(t, "211.36", adv.Amount.String())
		assert.Equal(t, cbc.Code("03"), adv.Ext[addon.ExtKeyPaymentMeans])
	})

	t.Run("should parse exchange rate when currency is not MXN", func(t *testing.T) {
		inv, err := test.LoadParsedInvoice("invoice-multi-currency.xml")
		require.NoError(t, err)

		assert.Equal(t, currency.Code("USD"), inv.Currency)
		require.Len(t, inv.ExchangeRates, 1)
		assert.Equal(t, currency.Code("USD"), inv.ExchangeRates[0].From)
		assert.Equal(t, currency.MXN, inv.ExchangeRates[0].To)
		assert.Equal(t, "17.46", inv.ExchangeRates[0].Amount.String())
	})

	t.Run("should parse IEPS taxes in line charges", func(t *testing.T) {
		inv, err := test.LoadParsedInvoice("invoice-ieps.xml")
		require.NoError(t, err)

		assert.Equal(t, bill.InvoiceTypeStandard, inv.Type)
		assert.Equal(t, cbc.Code("TEST"), inv.Series)
		assert.Equal(t, cbc.Code("00001"), inv.Code)
		assert.Equal(t, "2023-07-10", inv.IssueDate.String())
		assert.Equal(t, currency.MXN, inv.Currency)
		assert.Equal(t, cbc.Code("PPD"), inv.Tax.Ext[addon.ExtKeyPaymentMethod])

		require.NotNil(t, inv.Payment)
		require.NotNil(t, inv.Payment.Terms)
		assert.Equal(t, "Condiciones de pago", inv.Payment.Terms.Notes)
	})

	t.Run("should parse global invoices", func(t *testing.T) {
		inv, err := test.LoadParsedInvoice("invoice-global.xml")
		require.NoError(t, err)

		assert.True(t, inv.HasTags(addon.TagGlobal), "global tag should be present")
		assert.Equal(t, cbc.Code("04"), inv.Tax.Ext[addon.ExtKeyGlobalPeriod])
		assert.Equal(t, cbc.Code("03"), inv.Tax.Ext[addon.ExtKeyGlobalMonth])
		assert.Equal(t, cbc.Code("2025"), inv.Tax.Ext[addon.ExtKeyGlobalYear])
	})

	t.Run("should not parse payment details when MetodoPago is PPD", func(t *testing.T) {
		inv, err := test.LoadParsedInvoice("invoice-b2b-bare.xml")
		require.NoError(t, err)

		// PPD invoices should not have advances
		if inv.Payment != nil {
			assert.Nil(t, inv.Payment.Advances)
		}
	})
}

func TestParseComprobanteEgreso(t *testing.T) {
	t.Run("should return an Invoice with credit note data", func(t *testing.T) {
		inv, err := test.LoadParsedInvoice("credit-note.xml")
		require.NoError(t, err)

		assert.Equal(t, bill.InvoiceTypeCreditNote, inv.Type)
		assert.Equal(t, cbc.Code("CN"), inv.Series)
		assert.Equal(t, cbc.Code("0003"), inv.Code)
		assert.Equal(t, cbc.Code("E"), inv.Tax.Ext[addon.ExtKeyDocType])
	})
}
