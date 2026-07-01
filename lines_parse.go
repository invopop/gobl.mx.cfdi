package cfdi

import (
	"github.com/invopop/gobl.cfdi/addon"
	"github.com/invopop/gobl.cfdi/internal"
	"github.com/invopop/gobl/bill"
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
	if c == nil || c.ClaveProdServ == "" || c.ClaveProdServ == internal.DefaultClaveProdServ {
		return tax.Extensions{}
	}
	return tax.ExtensionsOf(cbc.CodeMap{
		addon.ExtKeyProdServ: cbc.Code(c.ClaveProdServ),
	})
}

func goblItemUnit(cu string) org.Unit {
	for _, def := range org.UnitDefinitions {
		if def.UNECE == cbc.Code(cu) {
			return def.Unit
		}
	}
	// No unit found, use empty unit
	return org.UnitEmpty
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
