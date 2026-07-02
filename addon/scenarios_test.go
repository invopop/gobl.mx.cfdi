package addon_test

import (
	"testing"

	"github.com/invopop/gobl.mx.cfdi/addon"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/num"
	"github.com/invopop/gobl/pay"
	"github.com/invopop/gobl/rules"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInvoiceScenarios(t *testing.T) {
	t.Run("regular", func(t *testing.T) {
		inv := validInvoice()
		require.NoError(t, inv.Calculate())
		assert.Equal(t, "PPD", inv.Tax.Ext.Get(addon.ExtKeyPaymentMethod).String())
		assert.NoError(t, rules.Validate(inv))
	})
	t.Run("prepaid", func(t *testing.T) {
		inv := validInvoice()
		inv.Payment = &bill.PaymentDetails{
			Advances: []*pay.Record{
				{
					Key:         "card",
					Description: "Pago anticipado",
					Percent:     num.NewPercentage(100, 2),
				},
			},
		}
		require.NoError(t, inv.Calculate())
		assert.NoError(t, rules.Validate(inv))
		assert.Equal(t, "PUE", inv.Tax.Ext.Get(addon.ExtKeyPaymentMethod).String())
	})
}
