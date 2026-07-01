package addon_test

import (
	"testing"

	"github.com/invopop/gobl.mx.cfdi/addon"
	"github.com/invopop/gobl/norm"
	"github.com/invopop/gobl/org"
	"github.com/invopop/gobl/tax"
	"github.com/stretchr/testify/assert"
)

func TestMigratePartyIdentities(t *testing.T) {
	customer := &org.Party{
		Name: "Test Customer",
		Identities: []*org.Identity{
			{
				Key:  addon.ExtKeyFiscalRegime,
				Code: "608",
			},
			{
				Key:  addon.ExtKeyUse,
				Code: "G01",
			},
			{
				Key:  "random",
				Code: "12345678",
			},
		},
	}

	norm.Normalize(customer, tax.AddonContext(addon.V4))

	assert.Len(t, customer.Identities, 1)
	assert.Equal(t, 2, customer.Ext.Len())
	assert.Equal(t, "608", customer.Ext.Get(addon.ExtKeyFiscalRegime).String())
	assert.Equal(t, "G01", customer.Ext.Get(addon.ExtKeyUse).String())
	assert.Equal(t, "12345678", customer.Identities[0].Code.String())
}

func TestNormalizePartyWithNilIdentities(t *testing.T) {
	t.Run("party with nil identity in array", func(t *testing.T) {
		customer := &org.Party{
			Name: "Test Customer",
			Identities: []*org.Identity{
				nil,
				{
					Key:  addon.ExtKeyFiscalRegime,
					Code: "608",
				},
				nil,
			},
		}

		norm.Normalize(customer, tax.AddonContext(addon.V4))

		// Should not panic with nil identities
		assert.Len(t, customer.Identities, 0)
		assert.Equal(t, 1, customer.Ext.Len())
		assert.Equal(t, "608", customer.Ext.Get(addon.ExtKeyFiscalRegime).String())
	})

	t.Run("party with only nil identities", func(t *testing.T) {
		customer := &org.Party{
			Name:       "Test Customer",
			Identities: []*org.Identity{nil, nil},
		}

		norm.Normalize(customer, tax.AddonContext(addon.V4))

		// Should not panic with only nil identities
		assert.Len(t, customer.Identities, 0)
		assert.True(t, customer.Ext.IsZero())
	})
}
