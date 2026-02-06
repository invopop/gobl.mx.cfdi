package cfdi

import (
	"encoding/xml"

	"github.com/invopop/gobl.cfdi/internal/format"
	addon "github.com/invopop/gobl/addons/mx/cfdi"
)

// VD Schema constants
const (
	VDVersion        = "1.0"
	VDTipoOperacion  = "monedero electrónico"
	VDNamespace      = "http://www.sat.gob.mx/valesdedespensa"
	VDSchemaLocation = "http://www.sat.gob.mx/sitio_internet/cfd/valesdedespensa/valesdedespensa.xsd"
)

// ValesDeDespensa stores the food vouchers data
type ValesDeDespensa struct {
	XMLName       xml.Name `xml:"valesdedespensa:ValesDeDespensa"`
	Version       string   `xml:"version,attr"`
	TipoOperacion string   `xml:"tipoOperacion,attr"`

	RegistroPatronal string `xml:"registroPatronal,attr,omitempty"`
	NumeroDeCuenta   string `xml:"numeroDeCuenta,attr"`
	Total            string `xml:"total,attr"`

	Conceptos []*VDConcepto `xml:"valesdedespensa:Conceptos>valesdedespensa:Concepto"` // nolint:misspell
}

// VDConcepto stores the data of a single food voucher
type VDConcepto struct {
	Identificador      string `xml:"identificador,attr"`
	Fecha              string `xml:"fecha,attr"`
	Rfc                string `xml:"rfc,attr"`
	Curp               string `xml:"curp,attr"`
	Nombre             string `xml:"nombre,attr"`
	NumSeguridadSocial string `xml:"numSeguridadSocial,attr,omitempty"`
	Importe            string `xml:"importe,attr"`
}

func addValesDeDespensa(doc *Document, fvc *addon.FoodVouchers) {
	vd := &ValesDeDespensa{
		Version:       VDVersion,
		TipoOperacion: VDTipoOperacion,

		RegistroPatronal: fvc.EmployerRegistration,
		NumeroDeCuenta:   fvc.AccountNumber,
		Total:            fvc.Total.RescaleDown(2).String(),

		Conceptos: newVDConceptos(fvc.Lines), // nolint:misspell
	}

	doc.VDNamespace = VDNamespace
	doc.SchemaLocation = doc.SchemaLocation + " " + format.SchemaLocation(VDNamespace, VDSchemaLocation)
	doc.AppendComplemento(vd)
}

func newVDConceptos(lines []*addon.FoodVouchersLine) []*VDConcepto {
	cs := make([]*VDConcepto, len(lines))

	for i, l := range lines {
		cs[i] = &VDConcepto{
			Identificador:      l.EWalletID.String(),
			Fecha:              l.IssueDateTime.String(),
			Rfc:                l.Employee.TaxCode.String(),
			Curp:               l.Employee.CURP.String(),
			Nombre:             l.Employee.Name,
			NumSeguridadSocial: l.Employee.SocialSecurity.String(),
			Importe:            l.Amount.RescaleDown(2).String(),
		}
	}

	return cs
}
