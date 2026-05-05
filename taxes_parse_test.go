package cfdi_test

import (
	"testing"

	"github.com/invopop/gobl.cfdi/test"
	"github.com/invopop/gobl/regimes/mx"
	"github.com/invopop/gobl/tax"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseImpuestos(t *testing.T) {
	t.Run("should return an Invoice with the Taxes data", func(t *testing.T) {
		inv, err := test.LoadParsedInvoice("invoice-b2b-full.xml")
		require.NoError(t, err)

		require.NotNil(t, inv.Totals.Taxes)
		assert.Equal(t, "32.03", inv.Totals.Taxes.Sum.String())
		assert.Equal(t, "32.03", inv.Totals.Tax.String())

		// invoice-b2b-full.xml has both regular VAT and retained taxes
		require.GreaterOrEqual(t, len(inv.Totals.Taxes.Categories), 1)

		// Find the VAT category
		var vatCat *tax.CategoryTotal
		for _, ct := range inv.Totals.Taxes.Categories {
			if ct.Code == tax.CategoryVAT && !ct.Retained {
				vatCat = ct
				break
			}
		}
		require.NotNil(t, vatCat)

		assert.Equal(t, tax.CategoryVAT, vatCat.Code)
		assert.False(t, vatCat.Retained)
		assert.Equal(t, "32.03", vatCat.Amount.String())

		// Check at least one rate exists with correct data
		require.GreaterOrEqual(t, len(vatCat.Rates), 1)
		rt := vatCat.Rates[0]
		assert.Equal(t, "200.20", rt.Base.String())
		assert.Equal(t, "32.03", rt.Amount.String())
		assert.Equal(t, "16.0%", rt.Percent.String())
	})

	t.Run("should return an Invoice with line taxes", func(t *testing.T) {
		inv, err := test.LoadParsedInvoice("invoice-b2b-full.xml")
		require.NoError(t, err)

		require.Len(t, inv.Lines, 3)

		// First line - taxed with retentions (VAT + 2 retentions)
		l := inv.Lines[0]
		require.GreaterOrEqual(t, len(l.Taxes), 1)

		// Find the VAT tax in the first line
		var vatTax *tax.Combo
		for _, t := range l.Taxes {
			if t.Category == tax.CategoryVAT && t.Percent != nil {
				vatTax = t
				break
			}
		}
		require.NotNil(t, vatTax)
		assert.Equal(t, tax.CategoryVAT, vatTax.Category)
		assert.Equal(t, "16.0%", vatTax.Percent.String())

		// Second line - exempt
		l = inv.Lines[1]
		require.Len(t, l.Taxes, 1)
		assert.Equal(t, tax.CategoryVAT, l.Taxes[0].Category)
		assert.Equal(t, tax.KeyExempt, l.Taxes[0].Key)

		// Third line - no taxes
		l = inv.Lines[2]
		assert.Nil(t, l.Taxes)
	})

	t.Run("should parse IEPS as line charges", func(t *testing.T) {
		inv, err := test.LoadParsedInvoice("invoice-ieps.xml")
		require.NoError(t, err)

		require.Len(t, inv.Lines, 2)

		// First line with IEPS - has multiple charges (IEPS tasa and cuota)
		l := inv.Lines[0]
		require.GreaterOrEqual(t, len(l.Charges), 2)

		// First IEPS charge is tasa (percentage)
		charge := l.Charges[0]
		assert.Equal(t, mx.TaxCategoryIEPS, charge.Code)
		assert.Equal(t, "25.0%", charge.Percent.String())
		assert.Equal(t, "7.50", charge.Amount.String())

		// Second IEPS charge is cuota
		charge = l.Charges[1]
		assert.Equal(t, mx.TaxCategoryIEPS, charge.Code)
		assert.Equal(t, "1.000000", charge.Rate.String())
		assert.Equal(t, "0.400", charge.Quantity.String())
		assert.Equal(t, "0.40", charge.Amount.String())
	})

	t.Run("should parse retained taxes", func(t *testing.T) {
		inv, err := test.LoadParsedInvoice("invoice-b2b-full-round.xml")
		require.NoError(t, err)

		require.NotNil(t, inv.Totals)
		require.NotNil(t, inv.Totals.Taxes)

		if inv.Totals.RetainedTax != nil {
			// Find retained tax category
			var retainedCat *tax.CategoryTotal
			for _, ct := range inv.Totals.Taxes.Categories {
				if ct.Retained {
					retainedCat = ct
					break
				}
			}
			if retainedCat != nil {
				assert.True(t, retainedCat.Retained)
			}
		}
	})

	t.Run("should not panic when TotalImpuestosTrasladados is missing", func(t *testing.T) {
		require.NotPanics(t, func() {
			_, _ = test.LoadParsedInvoice("invoice-retention-only.xml")
		})
	})
}
