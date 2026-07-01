package cfdi_test

import (
	"testing"

	"github.com/invopop/gobl.mx.cfdi/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCfdiRelacionados(t *testing.T) {
	t.Run("should return a Document with the CfdiRelacionados data", func(t *testing.T) {
		doc, err := test.NewDocumentFrom("credit-note.json")
		require.NoError(t, err)

		assert.Equal(t, "01", doc.CFDIRelacionados.TipoRelacion)

		rel := doc.CFDIRelacionados.CfdiRelacionado[0]

		assert.Equal(t, "1fac4464-1111-0000-1111-cd37179db12e", rel.UUID)
	})
}
