package cfdi_test

import (
	"testing"

	"github.com/invopop/gobl.mx.cfdi/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEmisor(t *testing.T) {
	t.Run("should return a Document with the Emisor data", func(t *testing.T) {
		doc, err := test.NewDocumentFrom("invoice-b2b-bare.json")
		require.NoError(t, err)

		e := doc.Emisor

		assert.Equal(t, "EKU9003173C9", e.Rfc)
		assert.Equal(t, "ESCUELA KEMPER URGATE", e.Nombre)
		assert.Equal(t, "601", e.RegimenFiscal)
	})
}

func TestReceptor(t *testing.T) {
	t.Run("should return a Document with the Receptor data", func(t *testing.T) {
		doc, err := test.NewDocumentFrom("invoice-b2b-bare.json")
		require.NoError(t, err)

		r := doc.Receptor

		assert.Equal(t, "URE180429TM6", r.Rfc)
		assert.Equal(t, "UNIVERSIDAD ROBOTICA ESPAÑOLA", r.Nombre)
		assert.Equal(t, "65000", r.DomicilioFiscalReceptor)
		assert.Equal(t, "601", r.RegimenFiscalReceptor)
		assert.Equal(t, "G01", r.UsoCFDI)
	})

	t.Run("should return a Document with the Receptor data on simplified invoices", func(t *testing.T) {
		doc, err := test.NewDocumentFrom("invoice-b2c.json")
		require.NoError(t, err)

		r := doc.Receptor

		assert.Equal(t, "XAXX010101000", r.Rfc)
		assert.Equal(t, "PÚBLICO EN GENERAL", r.Nombre)
		assert.Equal(t, "26015", r.DomicilioFiscalReceptor)
		assert.Equal(t, "616", r.RegimenFiscalReceptor)
		assert.Equal(t, "S01", r.UsoCFDI)
	})

	t.Run("should return a Document with the Receptor data on export invoices", func(t *testing.T) {
		doc, err := test.NewDocumentFrom("invoice-export.json")
		require.NoError(t, err)

		r := doc.Receptor

		assert.Equal(t, "XEXX010101000", r.Rfc)
		assert.Equal(t, "EXAMPLE CUSTOMER S.A.S.", r.Nombre)
		assert.Equal(t, "26015", r.DomicilioFiscalReceptor)
		assert.Equal(t, "616", r.RegimenFiscalReceptor)
		assert.Equal(t, "S01", r.UsoCFDI)
		assert.Equal(t, "9014514805", r.NumRegIdTrib)
		assert.Equal(t, "COL", r.ResidenciaFiscal)
	})
}
