package addendas

import (
	"encoding/xml"
	"fmt"
	"strings"

	"github.com/invopop/gobl.mx.cfdi/internal"
	"github.com/invopop/gobl.mx.cfdi/internal/format"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/i18n"
	"github.com/invopop/gobl/l10n"
	"github.com/invopop/gobl/num"
	"github.com/invopop/gobl/org"
	"github.com/invopop/gobl/tax"
	"github.com/invopop/validation"
)

// Mabe schema constants
const (
	MabeVersion        = "1.0"
	MabeNamespace      = "https://recepcionfe.mabempresa.com/cfd/addenda/v1"
	MabeSchemaLocation = "https://recepcionfe.mabempresa.com/cfd/addenda/v1/mabev1.xsd"
	MabeNotApplicable  = "NA"
)

// TipoDocumento valid values
const (
	MabeTipoDocumentoFactura     = "FACTURA"
	MabeTipoDocumentoNotaCredito = "NOTA CREDITO"
	MabeTipoDocumentoNotaCargo   = "NOTA CARGO"
)

// MabeTipoDocumentoMap maps GOBL invoice types to Mabe's TipoDocumento
var MabeTipoDocumentoMap = map[cbc.Key]string{
	bill.InvoiceTypeStandard:   MabeTipoDocumentoFactura,
	bill.InvoiceTypeCreditNote: MabeTipoDocumentoNotaCredito,
	bill.InvoiceTypeDebitNote:  MabeTipoDocumentoNotaCargo,
}

// Mabe specific identity codes.
const (
	MabeKeyIdentityPurchaseOrder = "mx-mabe-purchase-order"
	MabeKeyIdentityProviderCode  = "mx-mabe-provider-code"
	MabeKeyIdentityRef1          = "mx-mabe-reference1"
	MabeKeyIdentityRef2          = "mx-mabe-reference2"
	MabeKeyIdentityDeliveryPlant = "mx-mabe-delivery-plant"
	MabeKeyIdentityArticleCode   = "mx-mabe-article-code"
	MabeKeyIdentityUnit          = "mx-mabe-unit"
)

// MabeFactura is the root element of the Mabe addendum
type MabeFactura struct {
	XMLName        xml.Name `xml:"mabe:Factura"`
	Namespace      string   `xml:"xmlns:mabe,attr"`
	SchemaLocation string   `xml:"xsi:schemaLocation,attr"`

	Version       string `xml:"version,attr"`
	TipoDocumento string `xml:"tipoDocumento,attr"`
	Folio         string `xml:"folio,attr"`
	Fecha         string `xml:"fecha,attr"`
	OrdenCompra   string `xml:"ordenCompra,attr"`
	Referencia1   string `xml:"referencia1,attr"`
	Referencia2   string `xml:"referencia2,attr,omitempty"`

	Moneda    *MabeMoneda    `xml:"mabe:Moneda"`
	Proveedor *MabeProveedor `xml:"mabe:Proveedor"`
	Entrega   *MabeEntrega   `xml:"mabe:Entrega"`
	Detalles  *MabeDetalles  `xml:"mabe:Detalles"`

	Descuentos  *MabeDescuentos  `xml:"mabe:Descuentos,omitempty"`
	Subtotal    *MabeImporte     `xml:"mabe:Subtotal"`
	Traslados   *MabeTraslados   `xml:"mabe:Traslados"`
	Retenciones *MabeRetenciones `xml:"mabe:Retenciones"`
	Total       *MabeImporte     `xml:"mabe:Total"`
}

// MabeMoneda carries the data about the invoice's currency
type MabeMoneda struct {
	TipoMoneda      string `xml:"tipoMoneda,attr"`
	TipoCambio      string `xml:"tipoCambio,attr,omitempty"`      // Not implemented yet
	ImporteConLetra string `xml:"importeConLetra,attr,omitempty"` // Not implemented yet
}

// MabeProveedor carries the data about the invoice's supplier
type MabeProveedor struct {
	Codigo string `xml:"codigo,attr"`
}

// MabeEntrega carries the data about the invoice's delivery
type MabeEntrega struct {
	PlantaEntrega string `xml:"plantaEntrega,attr"`
	Calle         string `xml:"calle,attr,omitempty"`
	NoExterior    string `xml:"noExterior,attr,omitempty"`
	NoInterior    string `xml:"noInterior,attr,omitempty"`
	CodigoPostal  string `xml:"codigoPostal,attr,omitempty"`
}

// MabeDetalles carries the data about an invoice's lines
type MabeDetalles struct {
	Detalle []*MabeDetalle `xml:"mabe:Detalle"`
}

// MabeDetalle carries the data about one invoice's line
type MabeDetalle struct {
	NoLineaArticulo int    `xml:"noLineaArticulo,attr"`
	CodigoArticulo  string `xml:"codigoArticulo,attr"`
	Descripcion     string `xml:"descripcion,attr"` //nolint:misspell
	Unidad          string `xml:"unidad,attr"`
	Cantidad        string `xml:"cantidad,attr"`
	PrecioSinIva    string `xml:"precioSinIva,attr"`
	ImporteSinIva   string `xml:"importeSinIva,attr"`
	PrecioConIva    string `xml:"precioConIva,attr,omitempty"`  // Not implemented yet
	ImporteConIva   string `xml:"importeConIva,attr,omitempty"` // Not implemented yet
}

// MabeImporte carries the data about an invoice's total
type MabeImporte struct {
	Importe string `xml:"importe,attr"`
}

// MabeTraslados carries the data about an invoice's taxes (expect retained ones)
type MabeTraslados struct {
	Traslado []*MabeImpuesto `xml:"mabe:Traslado"`
}

// MabeRetenciones carries the data about an invoice's retained taxes
type MabeRetenciones struct {
	Retencion []*MabeImpuesto `xml:"mabe:Retencion"`
}

// MabeImpuesto carries the data about an invoice's tax
type MabeImpuesto struct {
	Tipo    string `xml:"tipo,attr"`
	Tasa    string `xml:"tasa,attr"`
	Importe string `xml:"importe,attr"`
}

// MabeDescuentos carries the data about an invoice's discount
type MabeDescuentos struct {
	Tipo        string `xml:"tipo,attr"`
	Descripcion string `xml:"descripcion,attr"` //nolint:misspell
	Importe     string `xml:"importe,attr"`
}

func isMabe(inv *bill.Invoice) bool {
	if inv.Supplier == nil {
		return false
	}
	id := extractIdentity(inv.Supplier.Identities, MabeKeyIdentityProviderCode)
	return id != cbc.CodeEmpty
}

// newMabe provides a new Mabe addenda.
func newMabe(inv *bill.Invoice) (*MabeFactura, error) {
	if err := validateInvoiceForMabe(inv); err != nil {
		return nil, err
	}

	// Ref2 is not currently used by Mabe, so we set the default
	// value to "NA".
	ref2 := extractIdentity(inv.Ordering.Identities, MabeKeyIdentityRef2)
	if ref2 == "" {
		ref2 = "NA"
	}

	f := &MabeFactura{
		Namespace:      MabeNamespace,
		SchemaLocation: format.SchemaLocation(MabeNamespace, MabeSchemaLocation),

		Version:       MabeVersion,
		TipoDocumento: MabeTipoDocumentoMap[inv.Type],
		Folio:         formatMabeFolio(inv),
		Fecha:         inv.IssueDate.String(),
		OrdenCompra:   extractIdentity(inv.Ordering.Identities, MabeKeyIdentityPurchaseOrder).String(),
		Referencia1:   extractIdentity(inv.Ordering.Identities, MabeKeyIdentityRef1).String(),
		Referencia2:   ref2.String(),

		Moneda:     newMabeMoneda(inv),
		Proveedor:  newMabeProveedor(inv),
		Entrega:    newMabeEntrega(inv),
		Descuentos: newMabeDescuentos(inv),
		Detalles:   newMabeDetalles(inv),

		Subtotal: newMabeImporte(inv.Totals.Sum),
		Total:    newMabeImporte(inv.Totals.Payable),
	}

	setMabeTaxes(inv, f)

	return f, nil
}

func validateInvoiceForMabe(inv *bill.Invoice) error {
	return validation.ValidateStruct(inv,
		validation.Field(&inv.Type,
			validation.In(validMabeInvoiceTypes()...),
			validation.Skip,
		),
		validation.Field(&inv.Supplier,
			validation.By(validateSupplierForMabe),
			validation.Skip,
		),
		validation.Field(&inv.Lines,
			validation.Each(validation.By(validateLineForMabe), validation.Skip),
			validation.Skip, // prevent GOBL validations from running
		),
		validation.Field(&inv.Delivery,
			validation.Required,
			validation.By(validateDeliveryForMabe),
			validation.Skip,
		),
		validation.Field(&inv.Ordering,
			validation.Required,
			validation.By(validateOrderingForMabe),
			validation.Skip,
		),
	)
}

func validateSupplierForMabe(value any) error {
	sup, _ := value.(*org.Party)
	if sup == nil {
		return nil
	}
	return validation.ValidateStruct(sup,
		validation.Field(&sup.Identities,
			requireIdentityKey(MabeKeyIdentityProviderCode),
			validation.Skip,
		),
	)
}

func validateLineForMabe(value any) error {
	line, _ := value.(*bill.Line)
	if line == nil {
		return nil
	}
	return validation.ValidateStruct(line,
		validation.Field(&line.Item,
			validation.By(validateItemForMabe),
			validation.Skip,
		),
	)
}

func validateItemForMabe(value any) error {
	item, _ := value.(*org.Item)
	if item == nil {
		return nil
	}
	return validation.ValidateStruct(item,
		validation.Field(&item.Identities,
			requireIdentityKey(MabeKeyIdentityArticleCode),
			validation.Skip,
		),
	)
}

func validateDeliveryForMabe(value any) error {
	del, _ := value.(*bill.DeliveryDetails)
	if del == nil {
		return nil
	}
	return validation.ValidateStruct(del,
		validation.Field(&del.Receiver,
			validation.Required,
			validation.By(validateReceiverForMabe),
			validation.Skip,
		),
	)
}

func validateReceiverForMabe(value any) error {
	rec, _ := value.(*org.Party)
	if rec == nil {
		return nil
	}
	return validation.ValidateStruct(rec,
		validation.Field(&rec.Identities,
			requireIdentityKey(MabeKeyIdentityDeliveryPlant),
			validation.Skip,
		),
	)
}

func validateOrderingForMabe(value any) error {
	ord, _ := value.(*bill.Ordering)
	if ord == nil {
		return nil
	}
	return validation.ValidateStruct(ord,
		validation.Field(&ord.Identities,
			requireIdentityKey(MabeKeyIdentityPurchaseOrder),
			requireIdentityKey(MabeKeyIdentityRef1),
			validation.Skip,
		),
	)
}

func validMabeInvoiceTypes() []any {
	var types []interface{}
	for t := range MabeTipoDocumentoMap {
		types = append(types, t)
	}
	return types
}

func newMabeMoneda(inv *bill.Invoice) *MabeMoneda {
	return &MabeMoneda{TipoMoneda: string(inv.Currency)}
}

func newMabeProveedor(inv *bill.Invoice) *MabeProveedor {
	if inv.Supplier == nil {
		return nil
	}
	id := extractIdentity(inv.Supplier.Identities, MabeKeyIdentityProviderCode)
	return &MabeProveedor{
		Codigo: id.String(),
	}
}

func newMabeEntrega(inv *bill.Invoice) *MabeEntrega {
	rec := inv.Delivery.Receiver
	id := extractIdentity(rec.Identities, MabeKeyIdentityDeliveryPlant)
	e := &MabeEntrega{
		PlantaEntrega: id.String(),
	}

	if len(rec.Addresses) > 0 {
		addr := rec.Addresses[0]

		e.Calle = addr.Street
		e.NoExterior = addr.Number
		e.NoInterior = MabeNotApplicable
		e.CodigoPostal = addr.Code.String()
	}

	return e
}

func newMabeDescuentos(inv *bill.Invoice) *MabeDescuentos {
	d := internal.TotalInvoiceDiscount(inv)
	if d == nil {
		return nil
	}

	return &MabeDescuentos{
		Tipo:        MabeNotApplicable,
		Descripcion: MabeNotApplicable, //nolint:misspell
		Importe:     d.String(),
	}
}

func newMabeDetalles(inv *bill.Invoice) *MabeDetalles {
	var detalles []*MabeDetalle

	for _, line := range inv.Lines {
		id := extractIdentity(line.Item.Identities, MabeKeyIdentityArticleCode)
		unit := extractIdentity(line.Item.Identities, MabeKeyIdentityUnit)
		if unit == cbc.CodeEmpty {
			unit = internal.ClaveUnidad(line)
		}
		d := &MabeDetalle{
			NoLineaArticulo: line.Index,
			CodigoArticulo:  id.String(),
			Descripcion:     line.Item.Name, //nolint:misspell
			Unidad:          unit.String(),
			Cantidad:        line.Quantity.String(),
			PrecioSinIva:    line.Item.Price.String(),
			ImporteSinIva:   line.Sum.String(),
		}

		detalles = append(detalles, d)
	}

	if len(detalles) == 0 {
		return nil
	}

	return &MabeDetalles{detalles}
}

func newMabeImporte(amount num.Amount) *MabeImporte {
	return &MabeImporte{
		Importe: amount.String(),
	}
}

func setMabeTaxes(inv *bill.Invoice, mabe *MabeFactura) {
	var traslados, retenciones []*MabeImpuesto

	regime := tax.RegimeDefFor(l10n.MX)

	for _, cat := range inv.Totals.Taxes.Categories {
		catDef := regime.CategoryDef(cat.Code)

		for _, rate := range cat.Rates {
			t := &MabeImpuesto{
				Tipo:    catDef.Name.In(i18n.ES), // this is risky
				Tasa:    format.TaxPercent(rate.Percent),
				Importe: rate.Amount.String(),
			}

			if cat.Retained {
				retenciones = append(retenciones, t)
			} else {
				traslados = append(traslados, t)
			}
		}
	}

	if len(traslados) > 0 {
		mabe.Traslados = &MabeTraslados{traslados}
	}

	if len(retenciones) > 0 {
		mabe.Retenciones = &MabeRetenciones{retenciones}
	}
}

func formatMabeFolio(inv *bill.Invoice) string {
	return fmt.Sprintf("%s%s", inv.Series, inv.Code)
}

// requireIdentityKey replaces the org.RequireIdentityKey helper that GOBL core
// dropped during its migration to the rules package. It validates that the
// identity set contains at least one identity matching any of the given keys.
func requireIdentityKey(key ...cbc.Key) validation.Rule {
	return validation.By(func(value any) error {
		ids, ok := value.([]*org.Identity)
		if !ok || org.IdentityForKey(ids, key...) != nil {
			return nil
		}
		strs := make([]string, len(key))
		for i, k := range key {
			strs[i] = k.String()
		}
		return fmt.Errorf("missing key '%s'", strings.Join(strs, "', '"))
	})
}

func extractIdentity(ids []*org.Identity, key cbc.Key) cbc.Code {
	if ids == nil {
		return ""
	}
	for _, id := range ids {
		if id.Key == key {
			return id.Code
		}
	}
	return cbc.CodeEmpty
}
