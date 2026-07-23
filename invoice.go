package cfdi

import (
	"fmt"

	"github.com/invopop/gobl.mx.cfdi/addendas"
	"github.com/invopop/gobl.mx.cfdi/addon"
	"github.com/invopop/gobl.mx.cfdi/internal"
	"github.com/invopop/gobl.mx.cfdi/internal/format"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/currency"
	"github.com/invopop/gobl/num"
	"github.com/invopop/gobl/pay"
	"github.com/invopop/gobl/schema"
	"github.com/invopop/gobl/tax"
	"github.com/invopop/validation"
)

// GlobalInformation is used for invoices that contain a summary of B2C documents.
type GlobalInformation struct {
	Period string `xml:"Periodicidad,attr"`
	Month  string `xml:"Meses,attr"`
	Year   string `xml:"Año,attr"`
}

func convertInvoice(inv *bill.Invoice) (*Document, error) {
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
		Fecha:             format.DateTime(inv.IssueDate, inv.IssueTime),
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
		return addon.PaymentMethodPUE.String()
	}
	return addon.PaymentMethodPPD.String()
}

func formaPago(inv *bill.Invoice) string {
	adv := largestAdvance(inv)
	if !isPrepaid(inv) || adv == nil {
		return addon.PaymentMeansToDefine.String()
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
