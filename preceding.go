package cfdi

import (
	addon "github.com/invopop/gobl/addons/mx/cfdi"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/org"
	"github.com/invopop/gobl/regimes/mx"
)

// CFDIRelacionados list the preceding CFDI documents (e.g., the preceding
// invoices of a credit note)
type CFDIRelacionados struct { // nolint:revive
	TipoRelacion    string            `xml:",attr"`
	CfdiRelacionado []CFDIRelacionado `xml:"cfdi:CfdiRelacionado"`
}

// CFDIRelacionado stores the data of a preceding CFDI document
type CFDIRelacionado struct { // nolint:revive
	UUID string `xml:",attr"`
}

func newCfdiRelacionados(inv *bill.Invoice) *CFDIRelacionados {
	if len(inv.Preceding) == 0 {
		return nil
	}

	crs := &CFDIRelacionados{
		TipoRelacion: lookupTipoRelacion(inv),
	}

	for _, p := range inv.Preceding {
		uuid := lookupUUID(p)
		if uuid != "" {
			cr := CFDIRelacionado{uuid}
			crs.CfdiRelacionado = append(crs.CfdiRelacionado, cr)
		}
	}

	return crs
}

func lookupUUID(p *org.DocumentRef) string {
	for _, s := range p.Stamps {
		if s.Provider == mx.StampSATUUID {
			return s.Value
		}
	}

	return ""
}

func lookupTipoRelacion(inv *bill.Invoice) string {
	if inv.Tax == nil {
		return ""
	}

	return inv.Tax.Ext[addon.ExtKeyRelType].String()
}
