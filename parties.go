package cfdi

import (
	"github.com/invopop/gobl.cfdi/addon"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/l10n"
	"github.com/invopop/gobl/org"
	"github.com/invopop/gobl/regimes/mx"
)

// Emisor stores the invoice supplier data
type Emisor struct {
	Rfc           string `xml:",attr"`
	Nombre        string `xml:",attr"`
	RegimenFiscal string `xml:",attr"`
}

// Receptor stores the invoice customer data
type Receptor struct {
	Rfc                     string `xml:",attr"`
	Nombre                  string `xml:",attr"`
	DomicilioFiscalReceptor string `xml:",attr"`
	RegimenFiscalReceptor   string `xml:",attr"`
	UsoCFDI                 string `xml:",attr"`
	NumRegIdTrib            string `xml:",attr,omitempty"` //nolint:staticcheck,revive
	ResidenciaFiscal        string `xml:",attr,omitempty"`
}

// ThirdParty or "ACuentaTerceros" defines the details of a third part for whom this
// transaction has been performed.
type ThirdParty struct {
	RFC          string   `xml:"RfcACuentaTerceros,attr"`
	Name         string   `xml:"NombreACuentaTerceros,attr"`
	FiscalRegime cbc.Code `xml:"RegimenFiscalACuentaTerceros,attr"`
	PostCode     cbc.Code `xml:"DomicilioFiscalACuentaTerceros,attr"`
}

func newEmisor(supplier *org.Party) *Emisor {
	emisor := &Emisor{
		Rfc:           supplier.TaxID.Code.String(),
		Nombre:        supplier.Name,
		RegimenFiscal: supplier.Ext.Get(addon.ExtKeyFiscalRegime).String(),
	}
	return emisor
}

func newThirdParty(p *org.Party) *ThirdParty {
	if p == nil || p.TaxID == nil || len(p.Addresses) == 0 {
		return nil
	}
	out := &ThirdParty{
		RFC:          p.TaxID.Code.String(),
		Name:         p.Name,
		FiscalRegime: p.Ext.Get(addon.ExtKeyFiscalRegime),
		PostCode:     p.Addresses[0].Code,
	}
	return out
}

func newReceptor(customer *org.Party, issuePlace string) *Receptor {
	if customer == nil {
		return &Receptor{
			Nombre:                  NombreReceptorGenerico,
			Rfc:                     mx.TaxIdentityCodeGeneric.String(),
			RegimenFiscalReceptor:   RegimenFiscalSinObligaciones,
			UsoCFDI:                 UsoCFDISinEfectos,
			DomicilioFiscalReceptor: issuePlace,
		}
	}

	if customer.TaxID.Country.Code() != l10n.MX {
		cd := l10n.Countries().Code(customer.TaxID.Country.Code())
		return &Receptor{
			Nombre:                  customer.Name,
			Rfc:                     mx.TaxIdentityCodeForeign.String(),
			RegimenFiscalReceptor:   RegimenFiscalSinObligaciones,
			UsoCFDI:                 UsoCFDISinEfectos,
			DomicilioFiscalReceptor: issuePlace,
			NumRegIdTrib:            customer.TaxID.Code.String(),
			ResidenciaFiscal:        cd.Alpha3,
		}
	}

	var postcode string
	if len(customer.Addresses) > 0 {
		postcode = customer.Addresses[0].Code.String()
	}

	return &Receptor{
		Nombre:                  customer.Name,
		Rfc:                     customer.TaxID.Code.String(),
		RegimenFiscalReceptor:   customer.Ext.Get(addon.ExtKeyFiscalRegime).String(),
		UsoCFDI:                 customer.Ext.Get(addon.ExtKeyUse).String(),
		DomicilioFiscalReceptor: postcode,
	}
}
