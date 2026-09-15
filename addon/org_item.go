package addon

import (
	"regexp"

	"github.com/invopop/gobl/catalogues/untdid"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/org"
	"github.com/invopop/gobl/tax"
)

// SAT item identity codes (ClaveProdServ) regular expression.
var (
	itemExtensionValidCodeRegexp        = regexp.MustCompile(`^\d{8}$`)
	itemExtensionNormalizableCodeRegexp = regexp.MustCompile(`^\d{6}$`)
)

// extension keys that have been migrated from identities to
// extensions.
var migratedExtensionKeys = []cbc.Key{
	ExtKeyProdServ,
	ExtKeyFiscalRegime,
	ExtKeyUse,
}

func normalizeItem(item *org.Item) {
	if item == nil {
		return
	}
	normalizeItemUnit(item)
	// 2023-08-25: Migrate identities to extensions
	// Pending removal after migrations completed.
	idents := make([]*org.Identity, 0)
	for _, v := range item.Identities {
		if v == nil {
			continue
		}
		if v.Key.In(migratedExtensionKeys...) {
			if item.Ext.IsZero() {
				item.Ext = tax.MakeExtensions()
			}
			item.Ext = item.Ext.Set(v.Key, v.Code)
		} else {
			idents = append(idents, v)
		}
	}
	item.Identities = idents
	// end.
	for k, v := range item.Ext.All() {
		if k == ExtKeyProdServ {
			if itemExtensionNormalizableCodeRegexp.MatchString(v.String()) {
				item.Ext = item.Ext.Set(k, cbc.Code(v.String()+"00"))
			}
		}
	}
}

// normalizeItemUnit keeps the item's unit and its UN/ECE code in step. The
// code is what the CFDI carries as the line's "ClaveUnidad", and SAT's
// c_ClaveUnidad catalogue is built on the same UN/ECE recommendations as the
// UNTDID extension, so a code with no GOBL key of its own is preserved there.
func normalizeItemUnit(item *org.Item) {
	code := item.Ext.Get(untdid.ExtKeyUnit)
	if unit := untdid.UnitKey(code); unit != cbc.KeyEmpty {
		item.Unit = unit
	}
	if code == cbc.CodeEmpty {
		if code = untdid.UnitCode(item.Unit); code != cbc.CodeEmpty {
			item.Ext = item.Ext.Set(untdid.ExtKeyUnit, code)
		}
	}
}
