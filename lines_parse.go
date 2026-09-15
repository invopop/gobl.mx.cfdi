package cfdi

import (
	"github.com/invopop/gobl.mx.cfdi/addon"
	"github.com/invopop/gobl.mx.cfdi/internal"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/catalogues/untdid"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/num"
	"github.com/invopop/gobl/org"
	"github.com/invopop/gobl/tax"
)

func goblAddLines(doc *Document, out *bill.Invoice) error {
	if doc == nil || doc.Conceptos == nil {
		return nil
	}

	for _, c := range doc.Conceptos.Concepto {
		line, err := goblNewLine(c, len(out.Lines)+1)
		if err != nil {
			return err
		}
		if line != nil {
			out.Lines = append(out.Lines, line)
		}
	}
	return nil
}

func goblNewLine(c *Concepto, index int) (*bill.Line, error) {
	if c == nil {
		return nil, nil
	}

	qty, err := num.AmountFromString(c.Cantidad)
	if err != nil {
		return nil, err
	}

	total := c.Importe
	if c.Descuento != nil {
		total = total.Subtract(*c.Descuento)
	}

	return &bill.Line{
		Index:    index,
		Quantity: qty,
		Item: &org.Item{
			Name:  c.Desc,
			Price: &c.ValorUnitario,
			Unit:  goblItemUnit(c.ClaveUnidad),
			Ref:   cbc.Code(c.Ref),
			Ext:   goblItemExt(c),
		},
		Sum:       &c.Importe,
		Total:     &total,
		Discounts: goblLineDiscounts(c.Descuento),
		Seller:    goblNewThirdParty(c.ThirdParty),
		Taxes:     goblLineTaxes(c),
		Charges:   goblLineCharges(c),
	}, nil
}

func goblItemExt(c *Concepto) tax.Extensions {
	ext := tax.Extensions{}
	if c == nil {
		return ext
	}
	if c.ClaveProdServ != "" && c.ClaveProdServ != internal.DefaultClaveProdServ {
		ext = ext.Set(addon.ExtKeyProdServ, cbc.Code(c.ClaveProdServ))
	}
	// Keep the unit code as given, including the many SAT codes with no GOBL
	// unit, but not the mutually defined default, which says nothing.
	if c.ClaveUnidad != "" && c.ClaveUnidad != internal.DefaultClaveUnidad {
		ext = ext.Set(untdid.ExtKeyUnit, cbc.Code(c.ClaveUnidad))
	}
	return ext
}

// goblItemUnit maps a "ClaveUnidad" onto the GOBL unit it corresponds to,
// which is empty for the many SAT codes GOBL has no unit for. The code itself
// is kept in the item's extensions either way, so it survives a round trip.
func goblItemUnit(cu string) cbc.Key {
	return untdid.UnitKey(cbc.Code(cu))
}

func goblLineDiscounts(d *num.Amount) []*bill.LineDiscount {
	if d == nil {
		return nil
	}
	return []*bill.LineDiscount{
		{
			Amount: *d,
		},
	}
}
