package addon_test

import (
	"testing"

	"github.com/invopop/gobl"
	"github.com/invopop/gobl.mx.cfdi/addon"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/org"
	"github.com/invopop/gobl/tax"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func cancelStatus(reason cbc.Code) *bill.Status {
	return &bill.Status{
		Addons: tax.WithAddons(addon.V4),
		Type:   bill.StatusTypeUpdate,
		Code:   "CANCEL-0001",
		Supplier: &org.Party{
			Name:  "ESCUELA KEMPER URGATE",
			TaxID: &tax.Identity{Country: "MX", Code: "EKU9003173C9"},
		},
		Lines: []*bill.StatusLine{{
			Key: bill.StatusLineRejected,
			Doc: &org.DocumentRef{Code: "0001"},
			Ext: tax.ExtensionsOf(cbc.CodeMap{addon.ExtKeyCancelReason: reason}),
		}},
	}
}

func TestBillStatusRules(t *testing.T) {
	t.Run("accepts a valid cancel reason", func(t *testing.T) {
		env := gobl.NewEnvelope()
		require.NoError(t, env.Insert(cancelStatus(addon.CancelReasonErrorsWithoutRelation)))
		assert.NoError(t, env.Validate())
	})

	t.Run("rejects an unknown cancel reason", func(t *testing.T) {
		env := gobl.NewEnvelope()
		require.NoError(t, env.Insert(cancelStatus("05")))
		assert.ErrorContains(t, env.Validate(), "mx-cfdi-cancel-reason")
	})
}
