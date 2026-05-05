package cfdi

import (
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/num"
	"github.com/invopop/gobl/regimes/mx"
	"github.com/invopop/gobl/tax"
)

var goblTaxCategoryMap = map[string]cbc.Code{
	satTaxVAT:  tax.CategoryVAT,
	satTaxIEPS: mx.TaxCategoryIEPS,
}

var goblRetainedTaxCategoryMap = map[string]cbc.Code{
	satTaxISR:  mx.TaxCategoryISR,
	satTaxVAT:  mx.TaxCategoryRVAT,
	satTaxIEPS: mx.TaxCategoryRIEPS,
}

func goblLineTaxes(c *Concepto) tax.Set {
	if c == nil || c.Impuestos == nil {
		return nil
	}

	var taxes tax.Set

	if c.Impuestos.Traslados != nil {
		for _, t := range c.Impuestos.Traslados.Traslado {
			if combo := goblLineTax(t, false); combo != nil {
				taxes = append(taxes, combo)
			}
		}
	}
	if c.Impuestos.Retenciones != nil {
		for _, t := range c.Impuestos.Retenciones.Retencion {
			if combo := goblLineTax(t, true); combo != nil {
				taxes = append(taxes, combo)
			}
		}
	}

	return taxes
}

func goblLineCharges(c *Concepto) []*bill.LineCharge {
	if c == nil || c.Impuestos == nil || c.Impuestos.Traslados == nil {
		return nil
	}

	var charges []*bill.LineCharge

	for _, t := range c.Impuestos.Traslados.Traslado {
		if charge := goblLineCharge(t); charge != nil {
			charges = append(charges, charge)
		}
	}

	return charges
}

func goblLineTax(t *Impuesto, retained bool) *tax.Combo {
	if t == nil {
		return nil
	}

	if t.Impuesto == taxCategoryMap[mx.TaxCategoryIEPS] {
		// IEPS maps to a line charge, not a tax combo
		return nil
	}

	category := goblTaxCategory(t, retained)
	if category == "" {
		return nil
	}

	return &tax.Combo{
		Category: category,
		Key:      goblTaxKey(t),
		Percent:  goblTaxPercent(t),
	}
}

func goblLineCharge(t *Impuesto) *bill.LineCharge {
	if t == nil {
		return nil
	}

	if t.Impuesto != taxCategoryMap[mx.TaxCategoryIEPS] {
		// Taxes other than IEPS map to a tax combo, not a line charge
		return nil
	}

	charge := &bill.LineCharge{
		Code: mx.TaxCategoryIEPS,
	}

	if t.Importe != nil {
		charge.Amount = *t.Importe
	}

	switch t.TipoFactor {
	case TipoFactorTasa:
		charge.Percent = goblTaxPercent(t)
	case TipoFactorCuota:
		charge.Rate = t.TasaOCuota
		charge.Quantity = t.Base
	default:
		// Other IEPS factors are ignored.
		return nil
	}

	return charge
}

func goblTaxPercent(t *Impuesto) *num.Percentage {
	if t == nil || t.TipoFactor != TipoFactorTasa {
		return nil
	}

	val := t.TasaOCuota.Value()
	exp := t.TasaOCuota.Exp()

	// CFDI tasas are typically expressed with 6 decimal places, most of them zeros.
	// Here we try to compact them.
	for val%100 == 0 && exp > 3 {
		val /= 10
		exp--
	}

	return num.NewPercentage(val, exp)
}

func goblTaxKey(t *Impuesto) cbc.Key {
	if t.TipoFactor == TipoFactorExento {
		return tax.KeyExempt
	}
	return ""
}

func goblTaxCategory(t *Impuesto, retained bool) cbc.Code {
	if retained {
		return goblRetainedTaxCategoryMap[t.Impuesto]
	}
	return goblTaxCategoryMap[t.Impuesto]
}

func goblNewTaxTotal(doc *Document) *tax.Total {
	if doc.Impuestos == nil {
		return nil
	}

	tt := &tax.Total{
		Sum:      zero,
		Retained: doc.Impuestos.TotalImpuestosRetenidos,
	}
	if doc.Impuestos.TotalImpuestosTrasladados != nil {
		tt.Sum = *doc.Impuestos.TotalImpuestosTrasladados
	}

	if doc.Impuestos.Traslados != nil {
		for _, t := range doc.Impuestos.Traslados.Traslado {
			if t.Impuesto == taxCategoryMap[mx.TaxCategoryIEPS] {
				// IEPS is handled as a line charge, subtract it from the total
				if t.Importe != nil {
					tt.Sum = tt.Sum.MatchPrecision(*t.Importe).Subtract(*t.Importe)
				}
				continue
			}
			cat := goblTaxCategory(t, false)
			ct := tt.Category(cat)
			if ct == nil {
				ct = &tax.CategoryTotal{
					Code: cat,
				}
				tt.Categories = append(tt.Categories, ct)
			}
			r := new(tax.RateTotal)
			if t.Importe != nil {
				r.Amount = *t.Importe
				ct.Amount = ct.Amount.MatchPrecision(*t.Importe).Add(*t.Importe)
			}
			if t.Base != nil {
				r.Base = *t.Base
			}
			switch t.TipoFactor {
			case TipoFactorExento:
				r.Key = tax.KeyExempt
				r.Amount = zero
			case TipoFactorTasa:
				r.Percent = goblTaxPercent(t)
			}
			ct.Rates = append(ct.Rates, r)
		}
	}

	if doc.Impuestos.Retenciones != nil {
		for _, t := range doc.Impuestos.Retenciones.Retencion {
			cat := goblTaxCategory(t, true)
			ct := tt.Category(cat)
			if ct == nil {
				ct = &tax.CategoryTotal{
					Code:     cat,
					Retained: true,
				}
				tt.Categories = append(tt.Categories, ct)
			}
			r := new(tax.RateTotal)
			if t.Importe != nil {
				r.Amount = *t.Importe
				ct.Amount = ct.Amount.MatchPrecision(*t.Importe).Add(*t.Importe)
			}

			// Base is mandatory in GOBL but not available in the CFDI totals, so we need to calculate it from the concepts
			for _, c := range doc.Conceptos.Concepto {
				if c.Impuestos == nil || c.Impuestos.Retenciones == nil {
					continue
				}
				for _, cr := range c.Impuestos.Retenciones.Retencion {
					if cr.Impuesto == t.Impuesto {
						r.Base = r.Base.MatchPrecision(*cr.Base).Add(*cr.Base)
					}
				}
			}
			ct.Rates = append(ct.Rates, r)
		}
	}

	return tt
}
