package cfdi

import (
	"strings"

	addon "github.com/invopop/gobl/addons/mx/cfdi"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/l10n"
	"github.com/invopop/gobl/org"
	"github.com/invopop/gobl/regimes/mx"
	"github.com/invopop/gobl/tax"
)

func goblNewSupplier(e *Emisor) *org.Party {
	if e == nil {
		return nil
	}

	return &org.Party{
		Name: e.Nombre,
		TaxID: &tax.Identity{
			Country: l10n.MX.Tax(),
			Code:    cbc.Code(e.Rfc),
		},
		Ext: tax.Extensions{
			addon.ExtKeyFiscalRegime: cbc.Code(e.RegimenFiscal),
		},
	}
}

func goblNewCustomer(r *Receptor) *org.Party {
	if r == nil || isGenericReceiver(r) {
		return nil
	}

	if r.Rfc == mx.TaxIdentityCodeForeign.String() {
		return goblNewForeignCustomer(r)
	}

	out := &org.Party{
		Name: r.Nombre,
		TaxID: &tax.Identity{
			Country: l10n.MX.Tax(),
			Code:    cbc.Code(r.Rfc),
		},
		Ext: tax.Extensions{
			addon.ExtKeyFiscalRegime: cbc.Code(r.RegimenFiscalReceptor),
			addon.ExtKeyUse:          cbc.Code(r.UsoCFDI),
		},
	}
	if r.DomicilioFiscalReceptor != "" {
		out.Addresses = []*org.Address{
			{
				Code: cbc.Code(r.DomicilioFiscalReceptor),
			},
		}
	}

	return out
}

func goblNewForeignCustomer(r *Receptor) *org.Party {
	out := &org.Party{
		Name: r.Nombre,
		TaxID: &tax.Identity{
			Country: countryFromAlpha3(r.ResidenciaFiscal),
			Code:    cbc.Code(r.NumRegIdTrib),
		},
	}
	return out
}

func goblNewThirdParty(tp *ThirdParty) *org.Party {
	if tp == nil {
		return nil
	}
	out := &org.Party{
		Name: tp.Name,
		TaxID: &tax.Identity{
			Country: l10n.TaxCountryCode(l10n.MX),
			Code:    cbc.Code(tp.RFC),
		},
	}
	if tp.FiscalRegime != "" {
		out.Ext = tax.Extensions{
			addon.ExtKeyFiscalRegime: tp.FiscalRegime,
		}
	}
	if tp.PostCode != "" {
		out.Addresses = []*org.Address{
			{
				Code: tp.PostCode,
			},
		}
	}
	return out
}

func isGenericReceiver(r *Receptor) bool {
	if r == nil {
		return false
	}
	return r.Rfc == mx.TaxIdentityCodeGeneric.String()
}

func countryFromAlpha3(code string) l10n.TaxCountryCode {
	if code == "" {
		return ""
	}
	for _, def := range l10n.Countries() {
		if strings.EqualFold(def.Alpha3, code) {
			return def.Code.Tax()
		}
	}
	return ""
}
