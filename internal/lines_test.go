package internal_test

import (
	"testing"

	"github.com/invopop/gobl.mx.cfdi/internal"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/catalogues/untdid"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/org"
	"github.com/invopop/gobl/tax"
	"github.com/stretchr/testify/assert"
)

func TestClaveUnidad(t *testing.T) {
	tests := []struct {
		name string
		item *org.Item
		code cbc.Code
	}{
		{
			name: "from the unit",
			item: &org.Item{Unit: org.UnitLitre},
			code: "LTR",
		},
		{
			name: "from the extension",
			item: &org.Item{
				Unit: org.UnitLitre,
				Ext:  tax.ExtensionsOf(cbc.CodeMap{untdid.ExtKeyUnit: "A59"}),
			},
			code: "A59",
		},
		{
			name: "code without a GOBL unit",
			item: &org.Item{
				Ext: tax.ExtensionsOf(cbc.CodeMap{untdid.ExtKeyUnit: "A59"}),
			},
			code: "A59",
		},
		{
			name: "no unit at all",
			item: &org.Item{},
			code: internal.DefaultClaveUnidad,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			line := &bill.Line{Item: tt.item}
			assert.Equal(t, tt.code, internal.ClaveUnidad(line))
		})
	}
}
