package internal

import (
	"github.com/invopop/gobl.mx.cfdi/addon"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/catalogues/untdid"
	"github.com/invopop/gobl/cbc"
)

// Default keys
const (
	DefaultClaveUnidad   = "ZZ" // Mutuamente definida
	DefaultClaveProdServ = "01010101"
)

// ClaveUnidad determines the line item's "ClaveUnidad" value from the item's
// unit, falling back to the mutually defined code when there is nothing to
// map. Normalization keeps the UN/ECE code in the extension, including codes
// that have no GOBL unit key of their own.
func ClaveUnidad(line *bill.Line) cbc.Code {
	if code := line.Item.Ext.Get(untdid.ExtKeyUnit); code != cbc.CodeEmpty {
		return code
	}
	if code := untdid.UnitCode(line.Item.Unit); code != cbc.CodeEmpty {
		return code
	}
	return DefaultClaveUnidad
}

// ClaveProdServ determines the line's Product-Service code
func ClaveProdServ(line *bill.Line) cbc.Code {
	if line.Item == nil {
		return ""
	}
	val := line.Item.Ext.Get(addon.ExtKeyProdServ)
	if val == "" {
		val = DefaultClaveProdServ
	}
	return val
}
