package cfdi

import (
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/currency"
	"github.com/invopop/gobl/num"
	"github.com/invopop/gobl/regimes/mx"
	"github.com/invopop/gobl/tax"
)

// Impuestos store the invoice tax totals
type Impuestos struct {
	TotalImpuestosTrasladados *num.Amount  `xml:",attr,omitempty"`
	TotalImpuestosRetenidos   *num.Amount  `xml:",attr,omitempty"`
	Retenciones               *Retenciones `xml:"cfdi:Retenciones,omitempty"`
	Traslados                 *Traslados   `xml:"cfdi:Traslados,omitempty"`
}

// ConceptoImpuestos store the line tax totals
type ConceptoImpuestos struct {
	Traslados   *Traslados   `xml:"cfdi:Traslados,omitempty"`
	Retenciones *Retenciones `xml:"cfdi:Retenciones,omitempty"`
}

// Traslados lists the non-retained taxes of a line or the invoice
type Traslados struct {
	Traslado []*Impuesto `xml:"cfdi:Traslado"`
}

// Retenciones lists the retained taxes of a line or the invoice
type Retenciones struct {
	Retencion []*Impuesto `xml:"cfdi:Retencion"`
}

// Impuesto stores the tax data of the invoice or a line
type Impuesto struct {
	Base       *num.Amount `xml:",attr,omitempty"`
	Importe    *num.Amount `xml:",attr,omitempty"`
	Impuesto   string      `xml:",attr"`
	TasaOCuota *num.Amount `xml:",attr,omitempty"`
	TipoFactor string      `xml:",attr,omitempty"`
}

// SAT tax type codes used in the Impuesto field
const (
	satTaxISR  = "001"
	satTaxVAT  = "002"
	satTaxIEPS = "003"
)

// Map of tax categories to SAT tax types
var taxCategoryMap = map[cbc.Code]string{
	mx.TaxCategoryISR:   satTaxISR,
	tax.CategoryVAT:     satTaxVAT,
	mx.TaxCategoryRVAT:  satTaxVAT,
	mx.TaxCategoryIEPS:  satTaxIEPS,
	mx.TaxCategoryRIEPS: satTaxIEPS,
}

func newImpuestos(totals *bill.Totals, lines []*bill.Line, currency currency.Code) *Impuestos {
	if totals.Taxes == nil {
		return nil
	}
	var traslados, retenciones []*Impuesto

	for _, cat := range totals.Taxes.Categories {
		for _, rate := range cat.Rates {
			imp := newImpuesto(cat.Code, rate, currency)
			if cat.Retained {
				// Clear out fields not supported by retained totals
				imp.Base = nil
				imp.TasaOCuota = nil
				imp.TipoFactor = ""

				retenciones = append(retenciones, imp)
			} else {
				traslados = append(traslados, imp)
			}
		}
	}

	traslados = taxesAddLineCharges(traslados, lines, currency)

	impuestos := &Impuestos{}
	empty := true
	if len(traslados) > 0 {
		impuestos.Traslados = &Traslados{traslados}

		// Set tax total only for non-exempt taxes
		for _, t := range traslados {
			if t.TipoFactor != TipoFactorExento {
				impuestos.TotalImpuestosTrasladados = amountAdd(impuestos.TotalImpuestosTrasladados, t.Importe)
			}
		}

		empty = false
	}
	if len(retenciones) > 0 {
		impuestos.Retenciones = &Retenciones{retenciones}
		for _, r := range retenciones {
			impuestos.TotalImpuestosRetenidos = amountAdd(impuestos.TotalImpuestosRetenidos, r.Importe)
		}
		empty = false
	}
	if empty {
		return nil
	}

	return impuestos
}

func taxesAddLineCharges(traslados []*Impuesto, lines []*bill.Line, cur currency.Code) []*Impuesto {
	taxes := []*Impuesto{}
	// generate the lines that need to be there
	for _, line := range lines {
		for _, charge := range line.Charges {
			tl := newImpuestoFromLineCharge(line, charge)
			if tl != nil {
				// ensure bases are rounded as they may have come from quantities
				b := tl.Base.Rescale(cur.Def().Subunits)
				tl.Base = &b
				taxes = append(taxes, tl)
			}
		}
	}
	if len(taxes) == 0 {
		return traslados
	}

	// see if any of the new lines are already present to add to the existing ones
	for _, tl := range taxes {
		found := false
		for _, tlt := range traslados {
			if tlt.Impuesto == tl.Impuesto &&
				tlt.TipoFactor == tl.TipoFactor &&
				tlt.TasaOCuota.Equals(*tl.TasaOCuota) {

				tlt.Base = amountAdd(tlt.Base, tl.Base)
				tlt.Importe = amountAdd(tlt.Importe, tl.Importe)
				found = true
			}
		}
		if !found {
			traslados = append(traslados, tl)
		}
	}

	return traslados
}

func amountAdd(a, b *num.Amount) *num.Amount {
	if a == nil {
		return b
	}
	if b == nil {
		return a
	}
	r := a.Add(*b)
	return &r
}

func amountPtr(a num.Amount) *num.Amount {
	return &a
}

func newImpuesto(catCode cbc.Code, rate *tax.RateTotal, currency currency.Code) *Impuesto {
	cu := currency.Def().Subunits // SAT expects tax total amounts with no more decimals than supported by the currency

	if rate.Percent == nil {
		return &Impuesto{
			Base:       amountPtr(rate.Base.Rescale(cu)),
			Impuesto:   taxCategoryMap[catCode],
			TipoFactor: TipoFactorExento,
		}
	}

	return &Impuesto{
		Base:       amountPtr(rate.Base.Rescale(cu)),
		Importe:    amountPtr(rate.Amount.Rescale(cu)),
		Impuesto:   taxCategoryMap[catCode],
		TasaOCuota: amountPtr(rate.Percent.Base().Rescale(6)),
		TipoFactor: TipoFactorTasa,
	}
}

func newConceptoImpuestos(line *bill.Line) *ConceptoImpuestos {
	if line.Total == nil {
		// nothing to do when no line total
		return nil
	}
	var traslados, retenciones []*Impuesto

	for _, tax := range line.Taxes {
		catDef := regime.CategoryDef(tax.Category)
		imp := newImpuestoFromCombo(line, tax)
		if catDef.Retained {
			retenciones = append(retenciones, imp)
		} else {
			traslados = append(traslados, imp)
		}
	}
	for _, charge := range line.Charges {
		imp := newImpuestoFromLineCharge(line, charge)
		if imp != nil {
			traslados = append(traslados, imp)
		}
	}

	impuestos := &ConceptoImpuestos{}
	if len(traslados) > 0 {
		impuestos.Traslados = &Traslados{traslados}
	}
	if len(retenciones) > 0 {
		impuestos.Retenciones = &Retenciones{retenciones}
	}
	if impuestos.Traslados == nil && impuestos.Retenciones == nil {
		return nil
	}

	return impuestos
}

func newImpuestoFromCombo(line *bill.Line, tax *tax.Combo) *Impuesto {
	if line.Total == nil {
		return nil
	}

	// Start with 4 decimal places of precision to try and avoid rounding issues.
	total := line.Total.RescaleUp(4)

	if tax.Percent == nil {
		return &Impuesto{
			Base:       &total,
			Impuesto:   taxCategoryMap[tax.Category],
			TipoFactor: TipoFactorExento,
		}
	}

	// GOBL doesn't provide an amount at line level, so we calculate it
	taxAmount := tax.Percent.Of(total)
	return &Impuesto{
		Base:       &total,
		Importe:    amountPtr(taxAmount),
		Impuesto:   taxCategoryMap[tax.Category],
		TasaOCuota: amountPtr(tax.Percent.Base().Rescale(6)),
		TipoFactor: TipoFactorTasa,
	}
}

func newImpuestoFromLineCharge(line *bill.Line, charge *bill.LineCharge) *Impuesto {
	if charge.Code != mx.TaxCategoryIEPS {
		// only handle IEPS at the moment
		return nil
	}

	i := &Impuesto{
		Impuesto: taxCategoryMap[charge.Code],
		Importe:  amountPtr(charge.Amount),
	}
	if charge.Percent != nil {
		i.Base = line.Sum
		i.TasaOCuota = amountPtr(charge.Percent.Base().Rescale(6))
		i.TipoFactor = TipoFactorTasa
	} else if charge.Rate != nil {
		q := line.Quantity
		if charge.Quantity != nil {
			q = *charge.Quantity
		}
		i.Base = &q
		i.TasaOCuota = amountPtr(charge.Rate.Rescale(6))
		i.TipoFactor = TipoFactorCuota
	} else {
		// Not enough details to process, ignore.
		return nil
	}

	return i
}
