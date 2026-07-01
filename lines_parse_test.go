package cfdi_test

import (
	"testing"

	"github.com/invopop/gobl.mx.cfdi/addon"
	"github.com/invopop/gobl.mx.cfdi/test"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/org"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseConceptos(t *testing.T) {
	// nolint:misspell
	t.Run("should return an Invoice with the Lines data", func(t *testing.T) {
		inv, err := test.LoadParsedInvoice("invoice-b2b-full.xml")
		require.NoError(t, err)

		require.Len(t, inv.Lines, 3)

		// First line
		l := inv.Lines[0]
		assert.Equal(t, 1, l.Index)
		assert.Equal(t, "2", l.Quantity.String())
		assert.Equal(t, "Cigarros", l.Item.Name)
		assert.Equal(t, "200.2020", l.Item.Price.String())
		assert.Equal(t, org.Unit("piece"), l.Item.Unit)
		assert.Equal(t, cbc.Code("50211502"), l.Item.Ext.Get(addon.ExtKeyProdServ))
		assert.Equal(t, "400.4040", l.Sum.String())

		require.Len(t, l.Discounts, 1)
		assert.Equal(t, "200.2020", l.Discounts[0].Amount.String())
		assert.Equal(t, "200.2020", l.Total.String())

		// Second line
		l = inv.Lines[1]
		assert.Equal(t, 2, l.Index)
		assert.Equal(t, "1", l.Quantity.String())
		assert.Equal(t, "Cerveza", l.Item.Name)
		assert.Equal(t, "10.50", l.Item.Price.String())

		// Third line
		l = inv.Lines[2]
		assert.Equal(t, 3, l.Index)
	})

	t.Run("should return the default Unit when no ClaveUnidad is given", func(t *testing.T) {
		inv, err := test.LoadParsedInvoice("invoice-b2b-bare.xml")
		require.NoError(t, err)

		require.Len(t, inv.Lines, 1)
		l := inv.Lines[0]

		assert.Equal(t, org.UnitEmpty, l.Item.Unit)
	})

	t.Run("should parse item references", func(t *testing.T) {
		inv, err := test.LoadParsedInvoice("invoice-global.xml")
		require.NoError(t, err)

		require.Len(t, inv.Lines, 2)
		l := inv.Lines[0]

		assert.Equal(t, cbc.Code("SALE1"), l.Item.Ref)
	})
}
