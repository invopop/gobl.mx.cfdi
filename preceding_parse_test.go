package cfdi_test

import (
	"testing"

	"github.com/invopop/gobl.mx.cfdi/addon"
	"github.com/invopop/gobl.mx.cfdi/test"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/regimes/mx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseCfdiRelacionados(t *testing.T) {
	t.Run("should return an Invoice with the Preceding documents data", func(t *testing.T) {
		inv, err := test.LoadParsedInvoice("credit-note.xml")
		require.NoError(t, err)

		assert.Equal(t, cbc.Code("01"), inv.Tax.Ext.Get(addon.ExtKeyRelType))

		require.Len(t, inv.Preceding, 1)
		prec := inv.Preceding[0]

		assert.Equal(t, cbc.Code("1fac4464-1111-0000-1111-cd37179db12e"), prec.Code)

		require.Len(t, prec.Stamps, 1)
		assert.Equal(t, mx.StampSATUUID, prec.Stamps[0].Provider)
		assert.Equal(t, "1fac4464-1111-0000-1111-cd37179db12e", prec.Stamps[0].Value)
	})
}
