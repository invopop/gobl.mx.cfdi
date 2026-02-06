package cfdi

import (
	"encoding/xml"

	"github.com/invopop/gobl.cfdi/internal/format"
	addon "github.com/invopop/gobl/addons/mx/cfdi"
	"github.com/invopop/gobl/regimes/mx"
	"github.com/invopop/gobl/tax"
)

// ECC Schema constants
const (
	ECCVersion        = "1.2"
	ECCTipoOperacion  = "Tarjeta"
	ECCNamespace      = "http://www.sat.gob.mx/EstadoDeCuentaCombustible12"
	ECCSchemaLocation = "http://www.sat.gob.mx/sitio_internet/cfd/EstadoDeCuentaCombustible/ecc12.xsd"
)

// EstadoDeCuentaCombustible stores the fuel account balance data
type EstadoDeCuentaCombustible struct {
	XMLName       xml.Name `xml:"ecc12:EstadoDeCuentaCombustible"`
	Version       string   `xml:",attr"`
	TipoOperacion string   `xml:",attr"`

	NumeroDeCuenta string `xml:",attr"`
	SubTotal       string `xml:",attr"`
	Total          string `xml:",attr"`

	Conceptos []*ECCConcepto `xml:"ecc12:Conceptos>ecc12:ConceptoEstadoDeCuentaCombustible"` // nolint:misspell
}

// ECCConcepto stores the data of a fuel purchase
type ECCConcepto struct {
	Identificador     string `xml:",attr"`
	Fecha             string `xml:",attr"`
	Rfc               string `xml:",attr"`
	ClaveEstacion     string `xml:",attr"`
	Cantidad          string `xml:",attr"`
	TipoCombustible   string `xml:",attr"`
	Unidad            string `xml:",attr,omitempty"`
	NombreCombustible string `xml:",attr"`
	FolioOperacion    string `xml:",attr"`
	ValorUnitario     string `xml:",attr"`
	Importe           string `xml:",attr"`

	Traslados []*ECCTraslado `xml:"ecc12:Traslados>ecc12:Traslado"`
}

// ECCTraslado stores the tax data of a fuel purchase
type ECCTraslado struct {
	Impuesto   string `xml:",attr"`
	TasaOCuota string `xml:",attr"`
	Importe    string `xml:",attr"`
}

func addEstadoCuentaCombustible(doc *Document, fc *addon.FuelAccountBalance) {
	ecc := &EstadoDeCuentaCombustible{
		Version:       ECCVersion,
		TipoOperacion: ECCTipoOperacion,

		NumeroDeCuenta: fc.AccountNumber,
		SubTotal:       fc.Subtotal.RescaleRange(1, 2).String(),
		Total:          fc.Total.RescaleRange(1, 2).String(),

		Conceptos: newECCConceptos(fc.Lines), // nolint:misspell
	}

	doc.ECCNamespace = ECCNamespace
	doc.SchemaLocation = doc.SchemaLocation + " " + format.SchemaLocation(ECCNamespace, ECCSchemaLocation)
	doc.AppendComplemento(ecc)
}

// nolint:misspell
func newECCConceptos(lines []*addon.FuelAccountLine) []*ECCConcepto {
	cs := make([]*ECCConcepto, len(lines))

	for i, l := range lines {
		cs[i] = &ECCConcepto{
			Identificador:     l.EWalletID.String(),
			Fecha:             l.PurchaseDateTime.String(),
			Rfc:               l.VendorTaxCode.String(),
			ClaveEstacion:     l.ServiceStationCode.String(),
			Cantidad:          l.Quantity.RescaleRange(1, 3).String(),
			TipoCombustible:   l.Item.Type.String(),
			Unidad:            l.Item.Unit.UNECE().String(),
			NombreCombustible: l.Item.Name,
			FolioOperacion:    l.PurchaseCode.String(),
			ValorUnitario:     l.Item.Price.RescaleRange(1, 3).String(),
			Importe:           l.Total.RescaleRange(1, 2).String(),
			Traslados:         newECCTraslados(l.Taxes),
		}
	}

	return cs
}

func newECCTraslados(taxes []*addon.FuelAccountTax) []*ECCTraslado {
	ts := make([]*ECCTraslado, len(taxes))

	for i, t := range taxes {
		tasa := ""
		if t.Percent != nil {
			tasa = t.Percent.Base().RescaleDown(6).String()
		} else if t.Rate != nil {
			tasa = t.Rate.RescaleDown(6).String()
		}
		impuesto := ""
		switch t.Category {
		case tax.CategoryVAT:
			impuesto = "IVA"
		case mx.TaxCategoryIEPS:
			impuesto = "IEPS"
		}
		ts[i] = &ECCTraslado{
			Impuesto:   impuesto,
			TasaOCuota: tasa,
			Importe:    t.Amount.RescaleRange(1, 2).String(),
		}
	}

	return ts
}
