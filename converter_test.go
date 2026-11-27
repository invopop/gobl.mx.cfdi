package cfdi_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/invopop/gobl"
	cfdi "github.com/invopop/gobl.mx.cfdi"
	"github.com/invopop/gobl.mx.cfdi/addon"
	"github.com/invopop/gobl.mx.cfdi/test"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/convert"
	"github.com/invopop/gobl/l10n"
	"github.com/invopop/gobl/rules"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConverterFormat(t *testing.T) {
	f := convert.FormatFor(cfdi.KeyCFDI)
	require.NotNil(t, f)
	assert.Equal(t, cbc.Key("cfdi"), f.Syntax)
	assert.Equal(t, []l10n.Code{l10n.MX}, f.Countries)
	assert.Len(t, f.Import, 1)
	assert.Len(t, f.Export, 2)
	assert.False(t, f.Attachments)
}

func TestConverterExport(t *testing.T) {
	t.Run("attachments", func(t *testing.T) {
		env, err := test.LoadTestEnvelope("invoice-b2b-full.json")
		require.NoError(t, err)
		pdf := &convert.Attachment{Name: "invoice.pdf", MIME: "application/pdf", Data: []byte("%PDF")}
		_, err = convert.Export(env, cfdi.KeyCFDI, convert.WithAttachments(pdf))
		assert.ErrorIs(t, err, convert.ErrNotSupported)
	})

	for _, name := range []string{"invoice-b2b-full.json", "payment-basic.json"} {
		t.Run(name, func(t *testing.T) {
			env, err := test.LoadTestEnvelope(name)
			require.NoError(t, err)
			out, err := convert.Export(env, cfdi.KeyCFDI)
			require.NoError(t, err)
			assert.Equal(t, cfdi.KeyCFDI, out.Format.Key)

			doc, err := cfdi.Convert(env)
			require.NoError(t, err)
			data, err := doc.Bytes()
			require.NoError(t, err)
			assert.Equal(t, string(data), string(out.Data))
		})
	}

	faultCode := func(t *testing.T, err error) rules.Code {
		t.Helper()
		assert.ErrorIs(t, err, convert.ErrNotSupported)
		var ge *gobl.Error
		require.ErrorAs(t, err, &ge)
		require.NotNil(t, ge.Faults())
		return ge.Faults().First().Code()
	}

	t.Run("invoice missing addon", func(t *testing.T) {
		env, err := test.LoadTestEnvelope("invoice-b2b-full.json")
		require.NoError(t, err)
		env.Extract().(*bill.Invoice).SetAddons()
		_, err = convert.Export(env, cfdi.KeyCFDI)
		assert.Equal(t, rules.Code("MX-CFDI-V4-BILL-INVOICE-01"), faultCode(t, err))
	})

	t.Run("payment missing addon", func(t *testing.T) {
		env, err := test.LoadTestEnvelope("payment-basic.json")
		require.NoError(t, err)
		env.Extract().(*bill.Payment).SetAddons()
		_, err = convert.Export(env, cfdi.KeyCFDI)
		assert.Equal(t, rules.Code("MX-CFDI-V4-BILL-PAYMENT-01"), faultCode(t, err))
	})
}

func TestConverterImport(t *testing.T) {
	t.Run("detected and imported", func(t *testing.T) {
		env, err := test.LoadTestEnvelope("invoice-b2b-full.json")
		require.NoError(t, err)
		out, err := convert.Export(env, cfdi.KeyCFDI)
		require.NoError(t, err)

		f, err := convert.Detect(out.Data)
		require.NoError(t, err)
		assert.Equal(t, cfdi.KeyCFDI, f.Key)

		imported, err := convert.Import(out.Data)
		require.NoError(t, err)
		inv := imported.Envelope.Extract().(*bill.Invoice)
		assert.Empty(t, imported.Attachments)
		assert.Equal(t, env.Extract().(*bill.Invoice).Totals.Payable.String(), inv.Totals.Payable.String())
		assert.True(t, addon.V4.In(inv.GetAddons()...))
	})

	t.Run("credit note", func(t *testing.T) {
		data, err := os.ReadFile(filepath.Join(test.GetDataPath(), "parse", "in", "credit-note.xml"))
		require.NoError(t, err)
		imported, err := convert.Import(data)
		require.NoError(t, err)
		assert.Equal(t, cfdi.KeyCFDI, imported.Format.Key)
		assert.Equal(t, bill.InvoiceTypeCreditNote, imported.Envelope.Extract().(*bill.Invoice).Type)
	})

	t.Run("other types not detected", func(t *testing.T) {
		data, err := os.ReadFile(filepath.Join(test.GetDataPath(), "parse", "in", "invoice-b2b-full.xml"))
		require.NoError(t, err)
		for _, dt := range []string{"P", "T", "N"} {
			doc := strings.Replace(string(data), `TipoDeComprobante="I"`, `TipoDeComprobante="`+dt+`"`, 1)
			require.NotEqual(t, string(data), doc)
			_, err := convert.Detect([]byte(doc))
			assert.ErrorIs(t, err, convert.ErrUnknownFormat, dt)
		}
	})

	t.Run("other documents not detected", func(t *testing.T) {
		for _, data := range []string{`<Invoice xmlns="urn:example"/>`, `{}`, ``} {
			_, err := convert.Detect([]byte(data))
			assert.ErrorIs(t, err, convert.ErrUnknownFormat)
		}
	})

	t.Run("invalid document", func(t *testing.T) {
		_, err := convert.Import([]byte(`<cfdi:Comprobante xmlns:cfdi="` + cfdi.CFDINamespace + `" TipoDeComprobante="I"><`))
		assert.Error(t, err)
	})
}
