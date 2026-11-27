package cfdi

import (
	"bytes"
	"encoding/xml"

	"github.com/invopop/gobl"
	"github.com/invopop/gobl.mx.cfdi/addon"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/convert"
	"github.com/invopop/gobl/i18n"
	"github.com/invopop/gobl/l10n"
	"github.com/invopop/gobl/rules"
	"github.com/invopop/gobl/schema"
)

// KeyCFDI identifies the CFDI 4.0 format in the GOBL convert register.
const KeyCFDI cbc.Key = "mx-cfdi-v4"

var (
	invoiceSchema = schema.Lookup(bill.Invoice{})
	paymentSchema = schema.Lookup(bill.Payment{})
)

// acceptRules lists the conditions a document must meet to be exported.
var acceptRules = convert.Rules(KeyCFDI,
	rules.For(new(bill.Invoice), convert.RequireAddons("01", addon.V4)),
	rules.For(new(bill.Payment), convert.RequireAddons("01", addon.V4)),
)

func init() {
	convert.Register(converter{})
}

// converter registers the CFDI format with the GOBL convert register.
// Exported documents are neither sealed nor stamped.
type converter struct{}

func (converter) Formats() []*convert.Format {
	return []*convert.Format{{
		Key:       KeyCFDI,
		Name:      i18n.NewString("CFDI 4.0"),
		MIME:      "application/xml",
		Syntax:    "cfdi",
		Countries: []l10n.Code{l10n.MX},
		Import:    []schema.ID{invoiceSchema},
		Export:    []schema.ID{invoiceSchema, paymentSchema},
	}}
}

type headerKey struct{}

// header holds the root element details used to detect CFDI documents.
type header struct {
	namespace string
	docType   string
}

// Detect claims CFDI 4.0 invoices and credit notes by their root namespace
// and type. Other types, such as payments, are left out, as Parse only
// imports invoices.
func (converter) Detect(in *convert.Input) cbc.Key {
	v, ok := in.Get(headerKey{})
	if !ok {
		v = readHeader(in.Data)
		in.Set(headerKey{}, v)
	}
	h := v.(header)
	if h.namespace != CFDINamespace {
		return cbc.KeyEmpty
	}
	switch cbc.Code(h.docType) {
	case addon.DocTypeInvoice, addon.DocTypeCreditNote:
		return KeyCFDI
	}
	return cbc.KeyEmpty
}

// readHeader reads only the root element's namespace and type.
func readHeader(data []byte) header {
	d := xml.NewDecoder(bytes.NewReader(data))
	for {
		tok, err := d.Token()
		if err != nil {
			return header{}
		}
		if se, ok := tok.(xml.StartElement); ok {
			h := header{namespace: se.Name.Space}
			for _, a := range se.Attr {
				if a.Name.Local == "TipoDeComprobante" {
					h.docType = a.Value
				}
			}
			return h
		}
	}
}

func (converter) Import(_ cbc.Key, data []byte, _ *convert.ImportOptions) (*convert.ImportResult, error) {
	env, err := Parse(data)
	if err != nil {
		return nil, err
	}
	return &convert.ImportResult{Envelope: env}, nil
}

// Accepts checks the document includes the CFDI addon.
func (converter) Accepts(_ cbc.Key, env *gobl.Envelope) error {
	if fs := acceptRules.Validate(env.Extract()); fs != nil {
		return fs
	}
	return nil
}

func (converter) Export(_ cbc.Key, env *gobl.Envelope, _ *convert.ExportOptions) ([]byte, error) {
	doc, err := Convert(env)
	if err != nil {
		return nil, err
	}
	return doc.Bytes()
}
