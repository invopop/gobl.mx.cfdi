// Package cfdi implements the conversion from GOBL to CFDI XML
package cfdi

import (
	"encoding/xml"
	"errors"
	"fmt"

	"github.com/invopop/gobl"
	"github.com/invopop/gobl.mx.cfdi/addendas"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/num"
	"github.com/invopop/gobl/tax"
)

// CFDI schema constants
const (
	CFDINamespace      = "http://www.sat.gob.mx/cfd/4"
	CFDISchemaLocation = "http://www.sat.gob.mx/sitio_internet/cfd/4/cfdv40.xsd"
	XSINamespace       = "http://www.w3.org/2001/XMLSchema-instance"
	CFDIVersion        = "4.0"
)

// Hard-coded values for (yet) unsupported mappings
const (
	FakeNoCertificado   = "00000000000000000000"
	ExportacionNoAplica = "01"
	ImpuestoIVA         = "002"
)

// Generic supplier constants
const (
	NombreReceptorGenerico       = "PÚBLICO EN GENERAL"
	RegimenFiscalSinObligaciones = "616" // no tax obligations
)

// TipoFactor definitions.
const (
	TipoFactorTasa   = "Tasa"
	TipoFactorCuota  = "Cuota" // Not supported
	TipoFactorExento = "Exento"
)

// Subject to tax constants
const (
	ObjetoImpNo = "01" // not subject to tax
	ObjetoImpSi = "02" // subject to tax
)

// ErrNotSupported is returned when the conversion of the invoice is not supported
var ErrNotSupported = errors.New("not supported")

// regime global
var regime = tax.RegimeDefFor("MX")

// zero global
var zero = regime.Currency.Def().Zero()

// Document is a pseudo-model for containing the XML document being created
type Document struct {
	XMLName        xml.Name `xml:"cfdi:Comprobante"`
	CFDINamespace  string   `xml:"xmlns:cfdi,attr"`
	XSINamespace   string   `xml:"xmlns:xsi,attr"`
	ECCNamespace   string   `xml:"xmlns:ecc12,attr,omitempty"`
	VDNamespace    string   `xml:"xmlns:valesdedespensa,attr,omitempty"`
	PagosNamespace string   `xml:"xmlns:pago20,attr,omitempty"`
	SchemaLocation string   `xml:"xsi:schemaLocation,attr"`
	Version        string   `xml:"Version,attr"`

	TipoDeComprobante string      `xml:",attr"`
	Serie             string      `xml:",attr,omitempty"`
	Folio             string      `xml:",attr,omitempty"`
	Fecha             string      `xml:",attr"`
	LugarExpedicion   string      `xml:",attr"`
	SubTotal          num.Amount  `xml:",attr"`
	Descuento         *num.Amount `xml:",attr,omitempty"`
	Total             num.Amount  `xml:",attr"`
	Moneda            string      `xml:",attr"`
	TipoCambio        *num.Amount `xml:",attr,omitempty"`
	Exportacion       string      `xml:",attr"`
	MetodoPago        string      `xml:",attr,omitempty"`
	FormaPago         string      `xml:",attr,omitempty"`
	CondicionesDePago string      `xml:",attr,omitempty"`
	Sello             string      `xml:",attr"`
	NoCertificado     string      `xml:",attr"`
	Certificado       string      `xml:",attr"`

	Global           *GlobalInformation `xml:"cfdi:InformacionGlobal,omitempty"`
	CFDIRelacionados *CFDIRelacionados  `xml:"cfdi:CfdiRelacionados,omitempty"`
	Emisor           *Emisor            `xml:"cfdi:Emisor"`
	Receptor         *Receptor          `xml:"cfdi:Receptor"`
	Conceptos        *Conceptos         `xml:"cfdi:Conceptos"` //nolint:misspell
	Impuestos        *Impuestos         `xml:"cfdi:Impuestos,omitempty"`

	// Supported complements
	ComplementoValesDeDespensa         *ValesDeDespensa           `xml:"cfdi:Complemento>valesdedespensa:ValesDeDespensa,omitempty"`
	ComplementoEstadoCuentaCombustible *EstadoDeCuentaCombustible `xml:"cfdi:Complemento>ecc12:EstadoDeCuentaCombustible,omitempty"`
	ComplementoPagos                   *Pagos                     `xml:"cfdi:Complemento>pago20:Pagos,omitempty"`
	ComplementoTimbreFiscalDigital     *TimbreFiscalDigital       `xml:"cfdi:Complemento>tfd:TimbreFiscalDigital,omitempty"`

	// Supported addendas
	AddendaMabe *addendas.MabeFactura `xml:"cfdi:Addenda>mabe:Factura,omitempty"`
}

// Convert converts a GOBL envelope into a CFDI document
func Convert(env *gobl.Envelope) (*Document, error) {
	switch doc := env.Extract().(type) {
	case *bill.Invoice:
		return convertInvoice(doc)
	case *bill.Payment:
		return convertPayment(doc), nil
	default:
		return nil, fmt.Errorf("invalid type %T", doc)
	}
}

// Bytes returns the XML representation of the document in bytes
func (d *Document) Bytes() ([]byte, error) {
	bytes, err := xml.MarshalIndent(d, "", "  ")
	if err != nil {
		return nil, err
	}

	return append([]byte(xml.Header), bytes...), nil
}
