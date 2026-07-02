package cfdi_test

import (
	"testing"

	"github.com/invopop/gobl.mx.cfdi/addon"
	"github.com/invopop/gobl.mx.cfdi/test"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/l10n"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseEmisor(t *testing.T) {
	t.Run("should return an Invoice with the Supplier data", func(t *testing.T) {
		inv, err := test.LoadParsedInvoice("invoice-b2b-bare.xml")
		require.NoError(t, err)

		s := inv.Supplier
		require.NotNil(t, s)

		assert.Equal(t, "ESCUELA KEMPER URGATE", s.Name)
		require.NotNil(t, s.TaxID)
		assert.Equal(t, l10n.MX.Tax(), s.TaxID.Country)
		assert.Equal(t, cbc.Code("EKU9003173C9"), s.TaxID.Code)
		assert.Equal(t, cbc.Code("601"), s.Ext.Get(addon.ExtKeyFiscalRegime))
	})
}

func TestParseReceptor(t *testing.T) {
	t.Run("should return an Invoice with the Customer data", func(t *testing.T) {
		inv, err := test.LoadParsedInvoice("invoice-b2b-bare.xml")
		require.NoError(t, err)

		c := inv.Customer
		require.NotNil(t, c)

		assert.Equal(t, "UNIVERSIDAD ROBOTICA ESPAÑOLA", c.Name)
		require.NotNil(t, c.TaxID)
		assert.Equal(t, l10n.MX.Tax(), c.TaxID.Country)
		assert.Equal(t, cbc.Code("URE180429TM6"), c.TaxID.Code)
		assert.Equal(t, cbc.Code("601"), c.Ext.Get(addon.ExtKeyFiscalRegime))
		assert.Equal(t, cbc.Code("G01"), c.Ext.Get(addon.ExtKeyUse))

		require.Len(t, c.Addresses, 1)
		assert.Equal(t, cbc.Code("65000"), c.Addresses[0].Code)
	})

	t.Run("should not return Customer for simplified invoices", func(t *testing.T) {
		inv, err := test.LoadParsedInvoice("invoice-b2c.xml")
		require.NoError(t, err)

		assert.Nil(t, inv.Customer)
	})

	t.Run("should return an Invoice with foreign Customer data on export invoices", func(t *testing.T) {
		inv, err := test.LoadParsedInvoice("invoice-export.xml")
		require.NoError(t, err)

		c := inv.Customer
		require.NotNil(t, c)

		assert.Equal(t, "EXAMPLE CUSTOMER S.A.S.", c.Name)
		require.NotNil(t, c.TaxID)
		assert.Equal(t, l10n.TaxCountryCode("CO"), c.TaxID.Country)
		assert.Equal(t, cbc.Code("9014514805"), c.TaxID.Code)
	})

	t.Run("should parse third party in line seller", func(t *testing.T) {
		inv, err := test.LoadParsedInvoice("invoice-b2c-third-party.xml")
		require.NoError(t, err)

		require.GreaterOrEqual(t, len(inv.Lines), 1)

		// Find a line with a seller
		var lineWithSeller *bill.Line
		for _, l := range inv.Lines {
			if l.Seller != nil {
				lineWithSeller = l
				break
			}
		}

		if lineWithSeller != nil {
			require.NotNil(t, lineWithSeller.Seller)
			assert.NotEmpty(t, lineWithSeller.Seller.Name)
			require.NotNil(t, lineWithSeller.Seller.TaxID)
			assert.Equal(t, l10n.MX.Tax(), lineWithSeller.Seller.TaxID.Country)
			assert.NotEmpty(t, lineWithSeller.Seller.TaxID.Code)
		}
	})
}
