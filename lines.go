package cfdi

import (
	"github.com/invopop/gobl.mx.cfdi/internal"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/num"
)

// Conceptos list invoice lines
// nolint:misspell
type Conceptos struct {
	Concepto []*Concepto `xml:"cfdi:Concepto"`
}

// Concepto stores an invoice line data
type Concepto struct {
	ClaveProdServ string      `xml:",attr"`
	Ref           string      `xml:"NoIdentificacion,attr,omitempty"`
	Cantidad      string      `xml:",attr"`
	ClaveUnidad   string      `xml:",attr"`
	Desc          string      `xml:"Descripcion,attr"` // nolint:misspell
	ValorUnitario num.Amount  `xml:",attr"`
	Importe       num.Amount  `xml:",attr"`
	Descuento     *num.Amount `xml:",attr,omitempty"`
	ObjetoImp     string      `xml:",attr"`

	Impuestos  *ConceptoImpuestos `xml:"cfdi:Impuestos,omitempty"`
	ThirdParty *ThirdParty        `xml:"cfdi:ACuentaTerceros,omitempty"`
}

// nolint:misspell
func newConceptos(lines []*bill.Line, ordering *bill.Ordering) *Conceptos {
	var conceptos []*Concepto

	for _, line := range lines {
		if line.Sum == nil {
			continue
		}
		conceptos = append(conceptos, newConcepto(line, ordering))
	}

	return &Conceptos{conceptos}
}

func newConcepto(line *bill.Line, ordering *bill.Ordering) *Concepto {
	concepto := &Concepto{
		ClaveProdServ: internal.ClaveProdServ(line).String(),
		Ref:           line.Item.Ref.String(),
		Cantidad:      line.Quantity.String(),
		ClaveUnidad:   internal.ClaveUnidad(line).String(),
		Desc:          line.Item.Name,
		ValorUnitario: *line.Item.Price,
		Importe:       *line.Sum,
		Descuento:     internal.TotalLineDiscount(line),
		ObjetoImp:     lineSubjectToTax(line),
		Impuestos:     newConceptoImpuestos(line),
	}
	if line.Seller != nil {
		concepto.ThirdParty = newThirdParty(line.Seller)
	} else if ordering != nil && ordering.Seller != nil {
		concepto.ThirdParty = newThirdParty(ordering.Seller)
	}
	return concepto
}

func lineSubjectToTax(line *bill.Line) string {
	if len(line.Taxes) == 0 {
		return ObjetoImpNo
	}
	return ObjetoImpSi
}
