package cfdi_test

import (
	"testing"

	"github.com/invopop/gobl.mx.cfdi/test"
	"github.com/invopop/gobl/tax"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestImpuestos(t *testing.T) {
	t.Run("should return a Document with the Impuestos data", func(t *testing.T) {
		doc, err := test.NewDocumentFrom("invoice-b2b-full.json")
		require.NoError(t, err)

		assert.Equal(t, "32.03", doc.Impuestos.TotalImpuestosTrasladados.String())

		tr := doc.Impuestos.Traslados.Traslado[0]

		assert.Equal(t, "200.20", tr.Base.String())   // SAT expects 2 decimals only
		assert.Equal(t, "32.03", tr.Importe.String()) // SAT expects 2 decimals only
		assert.Equal(t, "002", tr.Impuesto)
		assert.Equal(t, "Tasa", tr.TipoFactor)
		assert.Equal(t, "0.160000", tr.TasaOCuota.String())
	})

	t.Run("should return a Document with the Impuestos data when all taxes are exempt", func(t *testing.T) {
		inv, err := test.LoadTestInvoice("invoice-b2b-bare.json")
		require.NoError(t, err)

		inv.Lines[0].Taxes[0].Key = tax.KeyExempt

		doc, err := test.GenerateCFDIFrom(inv)
		require.NoError(t, err)

		assert.Nil(t, doc.Impuestos.TotalImpuestosTrasladados)
	})

	t.Run("should return a Document with the Impuestos data of each Concepto", func(t *testing.T) {
		doc, err := test.NewDocumentFrom("invoice-b2b-full.json")
		require.NoError(t, err)

		tr := doc.Conceptos.Concepto[0].Impuestos.Traslados.Traslado[0]

		assert.Equal(t, "200.2020", tr.Base.String())
		assert.Equal(t, "32.0323", tr.Importe.String())
		assert.Equal(t, "002", tr.Impuesto)
		assert.Equal(t, "Tasa", tr.TipoFactor)
		assert.Equal(t, "0.160000", tr.TasaOCuota.String())

		tr = doc.Conceptos.Concepto[1].Impuestos.Traslados.Traslado[0]
		assert.Equal(t, "10.5000", tr.Base.String()) // more precision
		assert.Nil(t, tr.Importe)
		assert.Equal(t, "002", tr.Impuesto)
		assert.Equal(t, "Exento", tr.TipoFactor)
		assert.Nil(t, tr.TasaOCuota)

		assert.Nil(t, doc.Conceptos.Concepto[2].Impuestos)
	})
}
