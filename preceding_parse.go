package cfdi

import (
	addon "github.com/invopop/gobl/addons/mx/cfdi"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/head"
	"github.com/invopop/gobl/org"
	"github.com/invopop/gobl/regimes/mx"
	"github.com/invopop/gobl/tax"
)

func goblAddPreceding(doc *Document, out *bill.Invoice) {
	if doc == nil || doc.CFDIRelacionados == nil {
		return
	}

	if out.Tax == nil {
		out.Tax = new(bill.Tax)
	}
	if doc.CFDIRelacionados.TipoRelacion != "" {
		out.Tax.Ext = out.Tax.Ext.Merge(tax.Extensions{
			addon.ExtKeyRelType: cbc.Code(doc.CFDIRelacionados.TipoRelacion),
		})
	}

	for _, rel := range doc.CFDIRelacionados.CfdiRelacionado {
		if rel.UUID == "" {
			continue
		}
		ref := &org.DocumentRef{
			Code: cbc.Code(rel.UUID),
			Stamps: []*head.Stamp{
				{
					Provider: mx.StampSATUUID,
					Value:    rel.UUID,
				},
			},
		}
		out.Preceding = append(out.Preceding, ref)
	}
}
