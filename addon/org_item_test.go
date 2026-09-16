package addon_test

import (
	"testing"

	"github.com/invopop/gobl.mx.cfdi/addon"
	"github.com/invopop/gobl/catalogues/untdid"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/norm"
	"github.com/invopop/gobl/org"
	"github.com/invopop/gobl/tax"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestItemIdentityNormalization(t *testing.T) {
	tests := []struct {
		Code     cbc.Code
		Expected cbc.Code
	}{
		{
			Code:     "123456",
			Expected: "12345600",
		},
		{
			Code:     "12345678",
			Expected: "12345678",
		},
		{
			Code:     "1234567",
			Expected: "1234567",
		},
	}
	for _, ts := range tests {
		item := &org.Item{Ext: tax.ExtensionsOf(cbc.CodeMap{addon.ExtKeyProdServ: ts.Code})}
		norm.Normalize(item, tax.AddonContext(addon.V4))
		assert.Equal(t, ts.Expected, item.Ext.Get(addon.ExtKeyProdServ))
	}

	// In context of invoice
	inv := validInvoice()
	inv.Lines[0].Item.Ext = inv.Lines[0].Item.Ext.Set(addon.ExtKeyProdServ, "010101")
	err := inv.Calculate()
	require.NoError(t, err)
	assert.Equal(t, cbc.Code("01010100"), inv.Lines[0].Item.Ext.Get(addon.ExtKeyProdServ))
}

func TestItemIdentityMigration(t *testing.T) {
	inv := validInvoice()

	inv.Lines[0].Item.Ext = tax.Extensions{}
	inv.Lines[0].Item.Identities = []*org.Identity{
		{
			Key:  addon.ExtKeyProdServ,
			Code: "01010101",
		},
		{
			Key:  "other",
			Code: "1234",
		},
	}

	err := inv.Calculate()
	require.NoError(t, err)
	assert.Equal(t, cbc.Code("01010101"), inv.Lines[0].Item.Ext.Get(addon.ExtKeyProdServ))
	assert.Equal(t, "1234", inv.Lines[0].Item.Identities[0].Code.String())
}

func TestItemNilIdentityHandling(t *testing.T) {
	t.Run("item with nil identity in array", func(t *testing.T) {
		inv := validInvoice()
		inv.Lines[0].Item.Identities = []*org.Identity{nil}
		require.NoError(t, inv.Calculate())
		// Should not panic with nil identity
	})

	t.Run("item with mixed nil and valid identities", func(t *testing.T) {
		inv := validInvoice()
		inv.Lines[0].Item.Ext = tax.Extensions{}
		inv.Lines[0].Item.Identities = []*org.Identity{
			nil,
			{
				Key:  addon.ExtKeyProdServ,
				Code: "01010101",
			},
			nil,
			{
				Key:  "other",
				Code: "5678",
			},
		}
		require.NoError(t, inv.Calculate())
		// Should not panic and should migrate valid identities
		assert.Equal(t, cbc.Code("01010101"), inv.Lines[0].Item.Ext.Get(addon.ExtKeyProdServ))
		assert.Len(t, inv.Lines[0].Item.Identities, 1)
		assert.Equal(t, "5678", inv.Lines[0].Item.Identities[0].Code.String())
	})
}

func TestItemUnitNormalization(t *testing.T) {
	t.Run("keeps the unit on its own", func(t *testing.T) {
		inv := validInvoice()
		inv.Lines[0].Item.Unit = org.UnitLitre

		require.NoError(t, inv.Calculate())
		assert.Equal(t, org.UnitLitre, inv.Lines[0].Item.Unit)
		assert.Empty(t, inv.Lines[0].Item.Ext.Get(untdid.ExtKeyUnit))
	})

	t.Run("takes the unit from the code", func(t *testing.T) {
		inv := validInvoice()
		inv.Lines[0].Item.Unit = cbc.KeyEmpty
		inv.Lines[0].Item.Ext = inv.Lines[0].Item.Ext.Set(untdid.ExtKeyUnit, "LTR")

		require.NoError(t, inv.Calculate())
		assert.Equal(t, org.UnitLitre, inv.Lines[0].Item.Unit)
		assert.Equal(t, cbc.Code("LTR"), inv.Lines[0].Item.Ext.Get(untdid.ExtKeyUnit), "the document stated the code")
	})

	t.Run("keeps a code with no GOBL unit", func(t *testing.T) {
		inv := validInvoice()
		inv.Lines[0].Item.Unit = cbc.KeyEmpty
		inv.Lines[0].Item.Ext = inv.Lines[0].Item.Ext.Set(untdid.ExtKeyUnit, "A59")

		require.NoError(t, inv.Calculate())
		assert.Equal(t, cbc.KeyEmpty, inv.Lines[0].Item.Unit)
		assert.Equal(t, cbc.Code("A59"), inv.Lines[0].Item.Ext.Get(untdid.ExtKeyUnit))
	})

	t.Run("takes a legacy raw code from the unit", func(t *testing.T) {
		inv := validInvoice()
		inv.Lines[0].Item.Unit = "H87"

		require.NoError(t, inv.Calculate())
		assert.Equal(t, org.UnitPiece, inv.Lines[0].Item.Unit)
		assert.Equal(t, cbc.Code("H87"), inv.Lines[0].Item.Ext.Get(untdid.ExtKeyUnit), "the code the document carried")
	})

	t.Run("leaves an item without a unit alone", func(t *testing.T) {
		inv := validInvoice()
		inv.Lines[0].Item.Unit = cbc.KeyEmpty

		require.NoError(t, inv.Calculate())
		assert.Equal(t, cbc.KeyEmpty, inv.Lines[0].Item.Unit)
		assert.Empty(t, inv.Lines[0].Item.Ext.Get(untdid.ExtKeyUnit))
	})
}
