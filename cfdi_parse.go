// Package cfdi implements the conversion from CFDI XML to GOBL.
package cfdi

import (
	"fmt"

	"cloud.google.com/go/civil"
	"github.com/invopop/gobl"
	addon "github.com/invopop/gobl/addons/mx/cfdi"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cal"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/currency"
	"github.com/invopop/gobl/num"
	"github.com/invopop/gobl/pay"
	"github.com/invopop/gobl/regimes/mx"
	"github.com/invopop/gobl/tax"
	"github.com/nbio/xml"
)

var (
	// ErrUnknownDocumentType is returned when the document namespace is not the expected one.
	ErrUnknownDocumentType = fmt.Errorf("unknown document namespace")
)

// Parse parses a raw CFDI XML document and converts it into a GOBL envelope. The
// TimbreFiscalDigital, if present, is parsed as GOBL stamps. The resulting envelope must
// be signed to make it valid.
func Parse(data []byte) (*gobl.Envelope, error) {
	// Unmarshal the CFDI document.
	cfdiDoc := new(Document)
	if err := xml.Unmarshal(data, cfdiDoc); err != nil {
		return nil, err
	}

	// Convert the CFDI document to a GOBL invoice.
	goblDoc, err := goblInvoice(cfdiDoc)
	if err != nil {
		return nil, err
	}

	// Create the GOBL envelope.
	env, err := gobl.Envelop(goblDoc)
	if err != nil {
		return nil, err
	}

	// Extract stamps from TimbreFiscalDigital if present
	if err := Stamp(env, cfdiDoc); err != nil {
		return nil, err
	}

	return env, nil
}

func goblInvoice(doc *Document) (*bill.Invoice, error) {
	out := &bill.Invoice{
		Addons:        tax.WithAddons(addon.V4),
		Tags:          tax.WithTags(tax.TagBypass),
		Code:          cbc.Code(doc.Folio),
		Series:        cbc.Code(doc.Serie),
		Type:          goblInvoiceType(doc.TipoDeComprobante),
		Currency:      currency.Code(doc.Moneda),
		Supplier:      goblNewSupplier(doc.Emisor),
		Customer:      goblNewCustomer(doc.Receptor),
		ExchangeRates: goblNewExchangeRates(doc),
		Payment:       goblNewPaymentDetails(doc),
		Tax: &bill.Tax{
			Ext: tax.Extensions{
				addon.ExtKeyDocType:       cbc.Code(doc.TipoDeComprobante),
				addon.ExtKeyIssuePlace:    cbc.Code(doc.LugarExpedicion),
				addon.ExtKeyPaymentMethod: cbc.Code(doc.MetodoPago),
			},
		},
		Totals: goblNewBillTotals(doc),
	}

	if doc.Fecha != "" {
		dt, err := civil.ParseDateTime(doc.Fecha)
		if err != nil {
			return nil, err
		}
		out.IssueDate = cal.MakeDate(dt.Date.Year, dt.Date.Month, dt.Date.Day)
		out.IssueTime = cal.NewTime(dt.Time.Hour, dt.Time.Minute, dt.Time.Second)
	}

	if doc.Global != nil {
		out.SetTags(addon.TagGlobal)
		out.Tax.Ext = out.Tax.Ext.Merge(tax.Extensions{
			addon.ExtKeyGlobalPeriod: cbc.Code(doc.Global.Period),
			addon.ExtKeyGlobalMonth:  cbc.Code(doc.Global.Month),
			addon.ExtKeyGlobalYear:   cbc.Code(doc.Global.Year),
		})
	}

	if err := goblAddLines(doc, out); err != nil {
		return nil, err
	}

	goblAddPreceding(doc, out)

	return out, nil
}

func goblNewExchangeRates(doc *Document) []*currency.ExchangeRate {
	if doc.TipoCambio == nil {
		return nil
	}

	return []*currency.ExchangeRate{
		{
			From:   currency.Code(doc.Moneda),
			To:     currency.MXN,
			Amount: *doc.TipoCambio,
		},
	}
}
func goblNewPaymentDetails(doc *Document) *bill.PaymentDetails {
	if doc == nil {
		return nil
	}

	payment := new(bill.PaymentDetails)

	if doc.CondicionesDePago != "" {
		payment.Terms = &pay.Terms{
			Notes: doc.CondicionesDePago,
		}
	}

	if cbc.Code(doc.MetodoPago) == addon.ExtCodePaymentMethodPUE {
		payment.Advances = []*pay.Advance{{
			Description: "Pago en una sola exhibición",
			Percent:     num.NewPercentage(100, 2),
			Amount:      doc.Total,
			Ext: tax.Extensions{
				addon.ExtKeyPaymentMeans: cbc.Code(doc.FormaPago),
			},
		}}
	}

	if payment.Terms == nil && payment.Advances == nil {
		return nil
	}
	return payment
}

func goblNewBillTotals(doc *Document) *bill.Totals {
	bt := &bill.Totals{
		Sum:          doc.SubTotal,
		Total:        doc.SubTotal,
		TotalWithTax: doc.Total,
		Payable:      doc.Total,
		Taxes:        goblNewTaxTotal(doc),
	}

	if doc.Impuestos != nil {
		// Copy the tax and retained tax totals from the taxes total
		bt.Tax = bt.Taxes.Sum
		bt.RetainedTax = bt.Taxes.Retained

		// Adjust the total with tax to include the retained tax
		if bt.RetainedTax != nil {
			bt.TotalWithTax = bt.TotalWithTax.Add(*bt.RetainedTax)
		}

		// Ajust sum and total to include IEPS
		if doc.Impuestos.Traslados != nil {
			for _, c := range doc.Impuestos.Traslados.Traslado {
				if c.Impuesto == taxCategoryMap[mx.TaxCategoryIEPS] && c.Importe != nil {
					bt.Sum = bt.Sum.MatchPrecision(*c.Importe).Add(*c.Importe)
					bt.Total = bt.Total.MatchPrecision(*c.Importe).Add(*c.Importe)
				}
			}
		}
	}

	// Adjust sum and total to include discount
	if doc.Descuento != nil {
		bt.Sum = bt.Sum.Subtract(*doc.Descuento)
		bt.Total = bt.Total.Subtract(*doc.Descuento)
	}

	// Set due to zero if payment method is PUE consistently with the advance
	if cbc.Code(doc.MetodoPago) == addon.ExtCodePaymentMethodPUE {
		bt.Due = &zero
	}

	return bt
}

func goblInvoiceType(code string) cbc.Key {
	switch code {
	case "I":
		return bill.InvoiceTypeStandard
	case "E":
		return bill.InvoiceTypeCreditNote
	default:
		return bill.InvoiceTypeOther
	}
}
