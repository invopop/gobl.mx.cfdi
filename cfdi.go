// Package cfdi implements the conversion from GOBL to CFDI XML
package cfdi

import (
	"encoding/xml"
	"errors"
	"fmt"

	"github.com/invopop/gobl"
	"github.com/invopop/gobl.mx.cfdi/addendas"
	"github.com/invopop/gobl.mx.cfdi/addon"
	"github.com/invopop/gobl.mx.cfdi/internal"
	"github.com/invopop/gobl.mx.cfdi/internal/format"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cal"
	"github.com/invopop/gobl/currency"
	"github.com/invopop/gobl/num"
	"github.com/invopop/gobl/pay"
	"github.com/invopop/gobl/schema"
	"github.com/invopop/gobl/tax"
	"github.com/invopop/validation"
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
	FormaPagoPorDefinir = "99"
	ImpuestoIVA         = "002"
)

// Generic supplier constants
const (
	NombreReceptorGenerico       = "PÚBLICO EN GENERAL"
	RegimenFiscalSinObligaciones = "616" // no tax obligations
	UsoCFDISinEfectos            = "S01" // no tax effects
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
	ComplementoTimbreFiscalDigital     *TimbreFiscalDigital       `xml:"cfdi:Complemento>tfd:TimbreFiscalDigital,omitempty"`

	// Supported addendas
	AddendaMabe *addendas.MabeFactura `xml:"cfdi:Addenda>mabe:Factura,omitempty"`
}

// GlobalInformation is used for invoices that contain a summary of B2C documents.
type GlobalInformation struct {
	Period string `xml:"Periodicidad,attr"`
	Month  string `xml:"Meses,attr"`
	Year   string `xml:"Año,attr"`
}

// Convert converts a GOBL envelope into a CFDI document
func Convert(env *gobl.Envelope) (*Document, error) {
	inv, ok := env.Extract().(*bill.Invoice)
	if !ok {
		return nil, fmt.Errorf("invalid type %T", env.Document)
	}

	if err := validateSupport(inv); err != nil {
		return nil, err
	}
	if err := inv.RemoveIncludedTaxes(); err != nil {
		return nil, fmt.Errorf("removing included taxes: %w", err)
	}

	issuePlace := issuePlace(inv)

	doc := &Document{
		CFDINamespace:  CFDINamespace,
		XSINamespace:   XSINamespace,
		SchemaLocation: format.SchemaLocation(CFDINamespace, CFDISchemaLocation),
		Version:        CFDIVersion,

		TipoDeComprobante: lookupTipoDeComprobante(inv),
		Serie:             inv.Series.String(),
		Folio:             inv.Code.String(),
		Fecha:             formatIssueDateTime(inv),
		LugarExpedicion:   issuePlace,
		Descuento:         internal.TotalInvoiceDiscount(inv),
		Moneda:            string(inv.Currency),
		TipoCambio:        tipoCambio(inv),
		Exportacion:       ExportacionNoAplica,
		MetodoPago:        metodoPago(inv),
		FormaPago:         formaPago(inv),
		CondicionesDePago: paymentTermsNotes(inv),

		NoCertificado: FakeNoCertificado,

		Global:           newGlobalInformation(inv),
		CFDIRelacionados: newCfdiRelacionados(inv),
		Emisor:           newEmisor(inv.Supplier),
		Receptor:         newReceptor(inv.Customer, issuePlace),
		Conceptos:        newConceptos(inv.Lines, inv.Ordering), // nolint:misspell
		Impuestos:        newImpuestos(inv.Totals, inv.Lines, inv.Currency),
	}

	// Determine the subtotal directly from the concepts, as there may be some
	// additional taxes included in the line charges that needed to be taken into
	// account for the totals.
	zero := inv.Currency.Def().Zero()
	doc.SubTotal = zero
	for _, c := range doc.Conceptos.Concepto {
		doc.SubTotal = doc.SubTotal.MatchPrecision(c.Importe)
		doc.SubTotal = doc.SubTotal.Add(c.Importe)
	}

	// Recalculate the total so that we can avoid any rounding issues
	doc.Total = doc.SubTotal
	if doc.Descuento != nil {
		doc.Total = doc.Total.MatchPrecision(*doc.Descuento)
		doc.Total = doc.Total.Subtract(*doc.Descuento)
	}
	taxes := zero
	if doc.Impuestos != nil {
		if tit := doc.Impuestos.TotalImpuestosTrasladados; tit != nil {
			taxes = taxes.MatchPrecision(*tit)
			taxes = taxes.Add(*tit)
		}
		if tir := doc.Impuestos.TotalImpuestosRetenidos; tir != nil {
			taxes = taxes.MatchPrecision(*tir)
			taxes = taxes.Subtract(*tir)
		}
	}
	doc.Total = doc.Total.Add(taxes)

	if err := addComplementos(doc, inv.Complements); err != nil {
		return nil, err
	}

	if err := addAddendas(doc, inv); err != nil {
		return nil, err
	}

	// Perform rounding on the totals at the last possible moment
	doc.SubTotal = doc.SubTotal.Rescale(zero.Exp())
	doc.Total = doc.Total.Rescale(zero.Exp())
	if doc.Descuento != nil {
		adjustDiscount(doc, taxes, zero)
	}

	return doc, nil
}

func newGlobalInformation(inv *bill.Invoice) *GlobalInformation {
	if inv.Tax == nil || !inv.Tax.Ext.Has(addon.ExtKeyGlobalPeriod) {
		return nil
	}
	return &GlobalInformation{
		Period: inv.Tax.Ext.Get(addon.ExtKeyGlobalPeriod).String(),
		Month:  inv.Tax.Ext.Get(addon.ExtKeyGlobalMonth).String(),
		Year:   inv.Tax.Ext.Get(addon.ExtKeyGlobalYear).String(),
	}
}

func validateSupport(inv *bill.Invoice) error {
	errs := validation.Errors{}

	if len(inv.Charges) > 0 {
		errs["charges"] = ErrNotSupported
	}

	// Deprecation pending...
	if inv.HasTags(tax.TagSelfBilled) {
		errs["self-billed"] = ErrNotSupported
	}
	if inv.HasTags(tax.TagCustomerRates) {
		errs["customer-rates"] = ErrNotSupported
	}

	if len(errs) > 0 {
		return errs
	}

	return nil
}

func issuePlace(inv *bill.Invoice) string {
	if inv.Tax != nil && inv.Tax.Ext.Has(addon.ExtKeyIssuePlace) {
		return inv.Tax.Ext.Get(addon.ExtKeyIssuePlace).String()
	}
	// Fallback
	return inv.Supplier.Ext.Get(addon.ExtKeyIssuePlace).String()
}

// Bytes returns the XML representation of the document in bytes
func (d *Document) Bytes() ([]byte, error) {
	bytes, err := xml.MarshalIndent(d, "", "  ")
	if err != nil {
		return nil, err
	}

	return append([]byte(xml.Header), bytes...), nil
}

func addComplementos(doc *Document, complements []*schema.Object) error {
	for _, c := range complements {
		switch o := c.Instance().(type) {
		case *addon.FuelAccountBalance:
			addEstadoCuentaCombustible(doc, o)
		case *addon.FoodVouchers:
			addValesDeDespensa(doc, o)
		default:
			return fmt.Errorf("unsupported complement %T", o)
		}
	}

	return nil
}

func addAddendas(doc *Document, inv *bill.Invoice) error {
	ads, err := addendas.For(inv)
	if err != nil {
		return err
	}

	for _, ad := range ads {
		switch ad := ad.(type) {
		case *addendas.MabeFactura:
			doc.AddendaMabe = ad
		default:
			return fmt.Errorf("unsupported addenda %T", ad)
		}
	}

	return nil
}

func formatIssueDateTime(inv *bill.Invoice) string {
	tn := inv.IssueTime
	if tn == nil {
		tn = new(cal.Time) // zero
	}
	return inv.IssueDate.WithTime(*tn).String()
}

func lookupTipoDeComprobante(inv *bill.Invoice) string {
	if inv.Tax == nil {
		return ""
	}

	return inv.Tax.Ext.Get(addon.ExtKeyDocType).String()
}

func tipoCambio(inv *bill.Invoice) *num.Amount {
	r := currency.MatchExchangeRate(inv.ExchangeRates, inv.Currency, currency.MXN)
	if r == nil {
		return nil
	}
	a := r.Amount
	return &a
}

func metodoPago(inv *bill.Invoice) string {
	if inv.Tax != nil && inv.Tax.Ext.Has(addon.ExtKeyPaymentMethod) {
		return inv.Tax.Ext.Get(addon.ExtKeyPaymentMethod).String()
	}
	// Fallback to the payment method based on the detected payment advances
	if isPrepaid(inv) {
		return addon.ExtCodePaymentMethodPUE.String()
	}
	return addon.ExtCodePaymentMethodPPD.String()
}

func formaPago(inv *bill.Invoice) string {
	adv := largestAdvance(inv)
	if !isPrepaid(inv) || adv == nil {
		return FormaPagoPorDefinir
	}
	return adv.Ext.Get(addon.ExtKeyPaymentMeans).String()
}

func isPrepaid(inv *bill.Invoice) bool {
	return inv.Totals.Due != nil && inv.Totals.Due.IsZero()
}

func largestAdvance(inv *bill.Invoice) *pay.Record {
	if inv.Payment == nil || len(inv.Payment.Advances) == 0 {
		return nil
	}

	la := inv.Payment.Advances[0]
	for _, a := range inv.Payment.Advances {
		if a.Amount.Compare(la.Amount) == 1 {
			la = a
		}
	}
	return la
}

func paymentTermsNotes(inv *bill.Invoice) string {
	if inv.Payment == nil || inv.Payment.Terms == nil {
		return ""
	}

	return inv.Payment.Terms.Notes
}

// adjustDiscount adjusts the document's discount to ensure it's consistent with the
// totals after rounding. It also adjusts one concept's discount to ensure it's consistent
// with the adjusted total discount. This can cause the data in the CFDI to be
// different from the data in the GOBL envelope, but we couldn't find another way to
// comply with the SAT requirements.
func adjustDiscount(doc *Document, taxes num.Amount, zero num.Amount) {
	// Recalculate the discount from the other totals
	desc := doc.SubTotal.Add(taxes).Subtract(doc.Total)
	diff := desc.MatchPrecision(*doc.Descuento).Subtract(*doc.Descuento)

	// Set the document's discount to the adjusted value
	doc.Descuento = &desc

	// Determine the minimum increment necessary to match the adjusted total discount
	inc := diff.Subtract(num.MakeAmount(5, zero.Exp()+1))
	if !inc.IsPositive() {
		// No adjustment is needed
		return
	}

	// Apply the increment to the first concept with a discount
	for _, c := range doc.Conceptos.Concepto {
		if c.Descuento != nil {
			disc := c.Descuento.MatchPrecision(inc).Add(inc)
			c.Descuento = &disc
			c.Importe = c.Importe.MatchPrecision(disc) // Importe and Descuento must match precision
			break
		}
	}
}
