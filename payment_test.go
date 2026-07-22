package cfdi_test

import (
	"testing"

	"github.com/invopop/gobl.mx.cfdi/addon"
	"github.com/invopop/gobl.mx.cfdi/test"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cal"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/currency"
	"github.com/invopop/gobl/head"
	"github.com/invopop/gobl/num"
	"github.com/invopop/gobl/org"
	"github.com/invopop/gobl/pay"
	"github.com/invopop/gobl/tax"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestComprobantePago(t *testing.T) {
	t.Run("should return a Document with the Comprobante data", func(t *testing.T) {
		doc, err := test.GenerateCFDIFrom(validPayment())
		require.NoError(t, err)

		assert.Equal(t, "http://www.sat.gob.mx/cfd/4", doc.CFDINamespace)
		assert.Equal(t, "http://www.w3.org/2001/XMLSchema-instance", doc.XSINamespace)
		assert.Equal(t, "http://www.sat.gob.mx/Pagos20", doc.PagosNamespace)
		assert.Equal(t, "http://www.sat.gob.mx/cfd/4 http://www.sat.gob.mx/sitio_internet/cfd/4/cfdv40.xsd http://www.sat.gob.mx/Pagos20 http://www.sat.gob.mx/sitio_internet/cfd/Pagos/Pagos20.xsd", doc.SchemaLocation)
		assert.Equal(t, "4.0", doc.Version)

		assert.Equal(t, "P", doc.TipoDeComprobante)
		assert.Equal(t, "P", doc.Serie)
		assert.Equal(t, "0123", doc.Folio)
		assert.Equal(t, "2025-03-01T10:30:00", doc.Fecha)
		assert.Equal(t, "21000", doc.LugarExpedicion)
		assert.Equal(t, "0", doc.SubTotal.String())
		assert.Equal(t, "0", doc.Total.String())
		assert.Equal(t, "XXX", doc.Moneda)
		assert.Equal(t, "01", doc.Exportacion)

		assert.Nil(t, doc.Descuento)
		assert.Nil(t, doc.TipoCambio)
		assert.Empty(t, doc.MetodoPago)
		assert.Empty(t, doc.FormaPago)
		assert.Empty(t, doc.CondicionesDePago)

		assert.Equal(t, "00000000000000000000", doc.NoCertificado)

		assert.Nil(t, doc.Global)
		assert.Nil(t, doc.Impuestos)
		assert.NotNil(t, doc.ComplementoPagos)
	})

	t.Run("should return a Document with the Emisor and Receptor data", func(t *testing.T) {
		doc, err := test.GenerateCFDIFrom(validPayment())
		require.NoError(t, err)

		assert.Equal(t, "AAA010101AAA", doc.Emisor.Rfc)
		assert.Equal(t, "Test Supplier", doc.Emisor.Nombre)
		assert.Equal(t, "601", doc.Emisor.RegimenFiscal)

		assert.Equal(t, "URE180429TM6", doc.Receptor.Rfc)
		assert.Equal(t, "Test Customer", doc.Receptor.Nombre)
		assert.Equal(t, "65000", doc.Receptor.DomicilioFiscalReceptor)
		assert.Equal(t, "608", doc.Receptor.RegimenFiscalReceptor)
		assert.Equal(t, "CP01", doc.Receptor.UsoCFDI)
	})

	t.Run("should force the CFDI use of a generic Receptor", func(t *testing.T) {
		pmt := validPayment()
		pmt.Customer = nil

		doc, err := test.GenerateCFDIFrom(pmt)
		require.NoError(t, err)

		assert.Equal(t, "XAXX010101000", doc.Receptor.Rfc)
		assert.Equal(t, "PÚBLICO EN GENERAL", doc.Receptor.Nombre)
		assert.Equal(t, "21000", doc.Receptor.DomicilioFiscalReceptor)
		assert.Equal(t, "616", doc.Receptor.RegimenFiscalReceptor)
		assert.Equal(t, "CP01", doc.Receptor.UsoCFDI)
	})

	t.Run("should return a Document with the fixed Concepto", func(t *testing.T) {
		doc, err := test.GenerateCFDIFrom(validPayment())
		require.NoError(t, err)

		require.Len(t, doc.Conceptos.Concepto, 1)

		c := doc.Conceptos.Concepto[0]

		assert.Equal(t, "84111506", c.ClaveProdServ)
		assert.Empty(t, c.Ref)
		assert.Equal(t, "1", c.Cantidad)
		assert.Equal(t, "ACT", c.ClaveUnidad)
		assert.Equal(t, "Pago", c.Desc)
		assert.Equal(t, "0", c.ValorUnitario.String())
		assert.Equal(t, "0", c.Importe.String())
		assert.Nil(t, c.Descuento)
		assert.Equal(t, "01", c.ObjetoImp)
		assert.Nil(t, c.Impuestos)
	})

	t.Run("should return a Document with the CfdiRelacionados data", func(t *testing.T) {
		pmt := validPayment()
		pmt.Preceding = []*org.DocumentRef{
			{
				Code: "0122",
				Stamps: []*head.Stamp{
					{
						Provider: "sat-uuid",
						Value:    "12345678-aaaa-bbbb-cccc-000000000000",
					},
				},
			},
		}

		doc, err := test.GenerateCFDIFrom(pmt)
		require.NoError(t, err)

		require.NotNil(t, doc.CFDIRelacionados)
		assert.Equal(t, "04", doc.CFDIRelacionados.TipoRelacion)
		require.Len(t, doc.CFDIRelacionados.CfdiRelacionado, 1)
		assert.Equal(t, "12345678-aaaa-bbbb-cccc-000000000000", doc.CFDIRelacionados.CfdiRelacionado[0].UUID)
	})

	t.Run("should generate a valid CFDI XML document", func(t *testing.T) {
		schema, err := loadSchema()
		require.NoError(t, err)

		withheldAndExempt := validPayment()
		withheldAndExempt.Lines[0].Tax = &tax.Total{
			Categories: []*tax.CategoryTotal{
				{
					Code: "VAT",
					Rates: []*tax.RateTotal{
						{
							Base: num.MakeAmount(200000, 2), // 2000.00 exempt
						},
					},
				},
				{
					Code:     "ISR",
					Retained: true,
					Rates: []*tax.RateTotal{
						{
							Base:    num.MakeAmount(200000, 2), // 2000.00
							Percent: num.NewPercentage(100, 3), // 10.0%
						},
					},
				},
			},
		}

		foreignCurrency := validPayment()
		foreignCurrency.Currency = "USD"
		foreignCurrency.ExchangeRates = []*currency.ExchangeRate{
			{
				From:   "USD",
				To:     "MXN",
				Amount: num.MakeAmount(1750, 2), // 17.50
			},
		}

		payments := map[string]*bill.Payment{
			"basic":               validPayment(),
			"withheld and exempt": withheldAndExempt,
			"foreign currency":    foreignCurrency,
		}

		for name, pmt := range payments {
			t.Run(name, func(t *testing.T) {
				doc, err := test.GenerateCFDIFrom(pmt)
				require.NoError(t, err)

				data, err := doc.Bytes()
				require.NoError(t, err)

				errs := validateDoc(schema, data)
				for _, e := range errs {
					assert.NoError(t, e)
				}
			})
		}
	})

}

func TestPagos(t *testing.T) {
	t.Run("should return a Document with the Pagos complement", func(t *testing.T) {
		doc, err := test.GenerateCFDIFrom(validPayment())
		require.NoError(t, err)

		p := doc.ComplementoPagos

		require.NotNil(t, p)
		assert.Equal(t, "2.0", p.Version)
		assert.NotNil(t, p.Totales)
		assert.Len(t, p.Pago, 1)
	})
}

func TestPago(t *testing.T) {
	t.Run("should return the Pago data", func(t *testing.T) {
		doc, err := test.GenerateCFDIFrom(validPayment())
		require.NoError(t, err)

		p := doc.ComplementoPagos.Pago[0]

		assert.Equal(t, "2025-02-25T12:00:00", p.FechaPago)
		assert.Equal(t, "03", p.FormaDePagoP)
		assert.Equal(t, "MXN", p.MonedaP)
		assert.Equal(t, "1", p.TipoCambioP)
		assert.Equal(t, "2320.00", p.Monto)
		assert.Equal(t, "0123456789", p.NumOperacion)
	})

	t.Run("should fall back to the value date and the issue date", func(t *testing.T) {
		pmt := validPayment()
		pmt.Methods[0].Date = nil
		pmt.ValueDate = cal.NewDate(2025, 2, 27)

		doc, err := test.GenerateCFDIFrom(pmt)
		require.NoError(t, err)
		assert.Equal(t, "2025-02-27T12:00:00", doc.ComplementoPagos.Pago[0].FechaPago)

		pmt = validPayment()
		pmt.Methods[0].Date = nil

		doc, err = test.GenerateCFDIFrom(pmt)
		require.NoError(t, err)
		assert.Equal(t, "2025-03-01T12:00:00", doc.ComplementoPagos.Pago[0].FechaPago)
	})

	t.Run("should omit the NumOperacion when the method has no reference", func(t *testing.T) {
		pmt := validPayment()
		pmt.Methods[0].Ref = ""

		doc, err := test.GenerateCFDIFrom(pmt)
		require.NoError(t, err)

		assert.Empty(t, doc.ComplementoPagos.Pago[0].NumOperacion)
	})

	t.Run("should return the TipoCambioP when the currency is not MXN", func(t *testing.T) {
		pmt := validPayment()
		pmt.Currency = "USD"
		pmt.ExchangeRates = []*currency.ExchangeRate{
			{
				From:   "USD",
				To:     "MXN",
				Amount: num.MakeAmount(1750, 2), // 17.50
			},
		}

		doc, err := test.GenerateCFDIFrom(pmt)
		require.NoError(t, err)

		p := doc.ComplementoPagos.Pago[0]

		assert.Equal(t, "USD", p.MonedaP)
		assert.Equal(t, "17.50", p.TipoCambioP)
		assert.Equal(t, "2320.00", p.Monto)
	})
}

func TestDoctoRelacionado(t *testing.T) {
	t.Run("should return the DoctoRelacionado data", func(t *testing.T) {
		doc, err := test.GenerateCFDIFrom(validPayment())
		require.NoError(t, err)

		require.Len(t, doc.ComplementoPagos.Pago[0].DoctoRelacionado, 1)

		dr := doc.ComplementoPagos.Pago[0].DoctoRelacionado[0]

		assert.Equal(t, "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee", dr.IDDocumento)
		assert.Equal(t, "F", dr.Serie)
		assert.Equal(t, "00123", dr.Folio)
		assert.Equal(t, "MXN", dr.MonedaDR)
		assert.Equal(t, "1", dr.EquivalenciaDR)
		assert.Equal(t, "3", dr.NumParcialidad)
		assert.Equal(t, "7000.00", dr.ImpSaldoAnt)
		assert.Equal(t, "2320.00", dr.ImpPagado)
		assert.Equal(t, "4680.00", dr.ImpSaldoInsoluto)
		assert.Equal(t, "02", dr.ObjetoImpDR)
	})

	t.Run("should return the balances of a full settlement", func(t *testing.T) {
		pmt := validPayment()
		pmt.Lines[0].Installment = 1
		pmt.Lines[0].Payable = num.NewAmount(232000, 2) // 2320.00
		pmt.Lines[0].Advances = nil

		doc, err := test.GenerateCFDIFrom(pmt)
		require.NoError(t, err)

		dr := doc.ComplementoPagos.Pago[0].DoctoRelacionado[0]

		assert.Equal(t, "1", dr.NumParcialidad)
		assert.Equal(t, "2320.00", dr.ImpSaldoAnt)
		assert.Equal(t, "2320.00", dr.ImpPagado)
		assert.Equal(t, "0.00", dr.ImpSaldoInsoluto)
	})

	t.Run("should return the balances of a first partial payment", func(t *testing.T) {
		pmt := validPayment()
		pmt.Lines[0].Installment = 1
		pmt.Lines[0].Advances = nil

		doc, err := test.GenerateCFDIFrom(pmt)
		require.NoError(t, err)

		dr := doc.ComplementoPagos.Pago[0].DoctoRelacionado[0]

		assert.Equal(t, "10000.00", dr.ImpSaldoAnt)
		assert.Equal(t, "2320.00", dr.ImpPagado)
		assert.Equal(t, "7680.00", dr.ImpSaldoInsoluto)
	})

	t.Run("should omit the Serie when the document has no series", func(t *testing.T) {
		pmt := validPayment()
		pmt.Lines[0].Document.Series = ""

		doc, err := test.GenerateCFDIFrom(pmt)
		require.NoError(t, err)

		dr := doc.ComplementoPagos.Pago[0].DoctoRelacionado[0]

		assert.Empty(t, dr.Serie)
		assert.Equal(t, "00123", dr.Folio)
	})

	t.Run("should flag documents not subject to tax", func(t *testing.T) {
		pmt := validPayment()
		pmt.Lines[0].Tax = nil

		doc, err := test.GenerateCFDIFrom(pmt)
		require.NoError(t, err)

		dr := doc.ComplementoPagos.Pago[0].DoctoRelacionado[0]

		assert.Equal(t, "01", dr.ObjetoImpDR)
		assert.Nil(t, dr.ImpuestosDR)
	})
}

func TestImpuestosDR(t *testing.T) {
	t.Run("should return the transferred taxes of the document", func(t *testing.T) {
		doc, err := test.GenerateCFDIFrom(validPayment())
		require.NoError(t, err)

		impuestos := doc.ComplementoPagos.Pago[0].DoctoRelacionado[0].ImpuestosDR

		require.NotNil(t, impuestos)
		assert.Nil(t, impuestos.RetencionesDR)
		require.NotNil(t, impuestos.TrasladosDR)
		require.Len(t, impuestos.TrasladosDR.TrasladoDR, 1)

		tr := impuestos.TrasladosDR.TrasladoDR[0]

		assert.Equal(t, "2000.00", tr.BaseDR)
		assert.Equal(t, "002", tr.ImpuestoDR)
		assert.Equal(t, "Tasa", tr.TipoFactorDR)
		assert.Equal(t, "0.160000", tr.TasaOCuotaDR)
		assert.Equal(t, "320.00", tr.ImporteDR)
	})

	t.Run("should return the withheld taxes of the document", func(t *testing.T) {
		pmt := validPayment()
		pmt.Lines[0].Tax.Categories = append(pmt.Lines[0].Tax.Categories, &tax.CategoryTotal{
			Code:     "ISR",
			Retained: true,
			Rates: []*tax.RateTotal{
				{
					Base:    num.MakeAmount(200000, 2), // 2000.00
					Percent: num.NewPercentage(100, 3), // 10.0%
				},
			},
		})

		doc, err := test.GenerateCFDIFrom(pmt)
		require.NoError(t, err)

		impuestos := doc.ComplementoPagos.Pago[0].DoctoRelacionado[0].ImpuestosDR

		require.NotNil(t, impuestos)
		require.NotNil(t, impuestos.RetencionesDR)
		require.Len(t, impuestos.RetencionesDR.RetencionDR, 1)

		ret := impuestos.RetencionesDR.RetencionDR[0]

		assert.Equal(t, "2000.00", ret.BaseDR)
		assert.Equal(t, "001", ret.ImpuestoDR)
		assert.Equal(t, "Tasa", ret.TipoFactorDR)
		assert.Equal(t, "0.100000", ret.TasaOCuotaDR)
		assert.Equal(t, "200.00", ret.ImporteDR)
	})

	t.Run("should return the exempt taxes of the document", func(t *testing.T) {
		pmt := validPayment()
		pmt.Lines[0].Tax.Categories[0].Rates = []*tax.RateTotal{
			{
				Base: num.MakeAmount(200000, 2), // 2000.00
			},
		}

		doc, err := test.GenerateCFDIFrom(pmt)
		require.NoError(t, err)

		impuestos := doc.ComplementoPagos.Pago[0].DoctoRelacionado[0].ImpuestosDR

		require.NotNil(t, impuestos)
		require.NotNil(t, impuestos.TrasladosDR)
		require.Len(t, impuestos.TrasladosDR.TrasladoDR, 1)

		tr := impuestos.TrasladosDR.TrasladoDR[0]

		assert.Equal(t, "2000.00", tr.BaseDR)
		assert.Equal(t, "002", tr.ImpuestoDR)
		assert.Equal(t, "Exento", tr.TipoFactorDR)
		assert.Empty(t, tr.TasaOCuotaDR)
		assert.Empty(t, tr.ImporteDR)
	})
}

func TestImpuestosP(t *testing.T) {
	t.Run("should group the taxes of multiple documents", func(t *testing.T) {
		pmt := paymentWithTwoLines()

		doc, err := test.GenerateCFDIFrom(pmt)
		require.NoError(t, err)

		impuestos := doc.ComplementoPagos.Pago[0].ImpuestosP

		require.NotNil(t, impuestos)

		require.NotNil(t, impuestos.RetencionesP)
		require.Len(t, impuestos.RetencionesP.RetencionP, 1)
		assert.Equal(t, "001", impuestos.RetencionesP.RetencionP[0].ImpuestoP)
		assert.Equal(t, "300.00", impuestos.RetencionesP.RetencionP[0].ImporteP)

		require.NotNil(t, impuestos.TrasladosP)
		require.Len(t, impuestos.TrasladosP.TrasladoP, 1)
		tr := impuestos.TrasladosP.TrasladoP[0]
		assert.Equal(t, "3000.00", tr.BaseP)
		assert.Equal(t, "002", tr.ImpuestoP)
		assert.Equal(t, "Tasa", tr.TipoFactorP)
		assert.Equal(t, "0.160000", tr.TasaOCuotaP)
		assert.Equal(t, "480.00", tr.ImporteP)
	})

	t.Run("should return exempt taxes with base only", func(t *testing.T) {
		pmt := validPayment()
		pmt.Lines[0].Tax.Categories[0].Rates = []*tax.RateTotal{
			{
				Base: num.MakeAmount(200000, 2), // 2000.00
			},
		}

		doc, err := test.GenerateCFDIFrom(pmt)
		require.NoError(t, err)

		impuestos := doc.ComplementoPagos.Pago[0].ImpuestosP

		require.NotNil(t, impuestos)
		assert.Nil(t, impuestos.RetencionesP)
		require.NotNil(t, impuestos.TrasladosP)
		require.Len(t, impuestos.TrasladosP.TrasladoP, 1)

		tr := impuestos.TrasladosP.TrasladoP[0]

		assert.Equal(t, "2000.00", tr.BaseP)
		assert.Equal(t, "002", tr.ImpuestoP)
		assert.Equal(t, "Exento", tr.TipoFactorP)
		assert.Empty(t, tr.TasaOCuotaP)
		assert.Empty(t, tr.ImporteP)
	})

	t.Run("should omit the tax summary when no document has taxes", func(t *testing.T) {
		pmt := validPayment()
		pmt.Lines[0].Tax = nil

		doc, err := test.GenerateCFDIFrom(pmt)
		require.NoError(t, err)

		assert.Nil(t, doc.ComplementoPagos.Pago[0].ImpuestosP)
	})
}

func TestPagosTotales(t *testing.T) {
	t.Run("should return the Totales data", func(t *testing.T) {
		doc, err := test.GenerateCFDIFrom(validPayment())
		require.NoError(t, err)

		totales := doc.ComplementoPagos.Totales

		assert.Equal(t, "2000.00", totales.TotalTrasladosBaseIVA16)
		assert.Equal(t, "320.00", totales.TotalTrasladosImpuestoIVA16)
		assert.Equal(t, "2320.00", totales.MontoTotalPagos)

		assert.Empty(t, totales.TotalRetencionesIVA)
		assert.Empty(t, totales.TotalRetencionesISR)
		assert.Empty(t, totales.TotalRetencionesIEPS)
		assert.Empty(t, totales.TotalTrasladosBaseIVA8)
		assert.Empty(t, totales.TotalTrasladosImpuestoIVA8)
		assert.Empty(t, totales.TotalTrasladosBaseIVA0)
		assert.Empty(t, totales.TotalTrasladosImpuestoIVA0)
		assert.Empty(t, totales.TotalTrasladosBaseIVAExento)
	})

	t.Run("should return the totals of every tax group", func(t *testing.T) {
		pmt := validPayment()
		pmt.Lines[0].Payable = nil
		pmt.Lines[0].Advances = nil
		pmt.Lines[0].Tax = &tax.Total{
			Categories: []*tax.CategoryTotal{
				{
					Code: "VAT",
					Rates: []*tax.RateTotal{
						{
							Base:    num.MakeAmount(100000, 2), // 1000.00
							Percent: num.NewPercentage(160, 3), // 16.0%
						},
						{
							Base:    num.MakeAmount(50000, 2), // 500.00
							Percent: num.NewPercentage(80, 3), // 8.0%
						},
						{
							Base:    num.MakeAmount(30000, 2), // 300.00
							Percent: num.NewPercentage(0, 3),  // 0.0%
						},
						{
							Base: num.MakeAmount(20000, 2), // 2000.00 exempt
						},
					},
				},
				{
					Code:     "RVAT",
					Retained: true,
					Rates: []*tax.RateTotal{
						{
							Base:    num.MakeAmount(100000, 2), // 1000.00
							Percent: num.NewPercentage(40, 3),  // 4.0%
						},
					},
				},
				{
					Code:     "ISR",
					Retained: true,
					Rates: []*tax.RateTotal{
						{
							Base:    num.MakeAmount(100000, 2), // 1000.00
							Percent: num.NewPercentage(100, 3), // 10.0%
						},
					},
				},
				{
					Code:     "RIEPS",
					Retained: true,
					Rates: []*tax.RateTotal{
						{
							Base:    num.MakeAmount(10000, 2), // 100.00
							Percent: num.NewPercentage(30, 3), // 3.0%
						},
					},
				},
			},
		}

		doc, err := test.GenerateCFDIFrom(pmt)
		require.NoError(t, err)

		totales := doc.ComplementoPagos.Totales

		assert.Equal(t, "40.00", totales.TotalRetencionesIVA)
		assert.Equal(t, "100.00", totales.TotalRetencionesISR)
		assert.Equal(t, "3.00", totales.TotalRetencionesIEPS)
		assert.Equal(t, "1000.00", totales.TotalTrasladosBaseIVA16)
		assert.Equal(t, "160.00", totales.TotalTrasladosImpuestoIVA16)
		assert.Equal(t, "500.00", totales.TotalTrasladosBaseIVA8)
		assert.Equal(t, "40.00", totales.TotalTrasladosImpuestoIVA8)
		assert.Equal(t, "300.00", totales.TotalTrasladosBaseIVA0)
		assert.Equal(t, "0.00", totales.TotalTrasladosImpuestoIVA0)
		assert.Equal(t, "200.00", totales.TotalTrasladosBaseIVAExento)
		assert.Equal(t, "2320.00", totales.MontoTotalPagos)
	})

	t.Run("should exclude nonstandard IVA rates from the traslados totals", func(t *testing.T) {
		pmt := validPayment()
		pmt.Lines[0].Tax.Categories[0].Rates = []*tax.RateTotal{
			{
				Base:    num.MakeAmount(200000, 2), // 2000.00
				Percent: num.NewPercentage(40, 3),  // 4.0%
			},
		}

		doc, err := test.GenerateCFDIFrom(pmt)
		require.NoError(t, err)

		// The traslado is grouped in the payment's tax summary...
		impuestos := doc.ComplementoPagos.Pago[0].ImpuestosP

		require.NotNil(t, impuestos)
		require.NotNil(t, impuestos.TrasladosP)
		require.Len(t, impuestos.TrasladosP.TrasladoP, 1)

		tr := impuestos.TrasladosP.TrasladoP[0]

		assert.Equal(t, "2000.00", tr.BaseP)
		assert.Equal(t, "002", tr.ImpuestoP)
		assert.Equal(t, "Tasa", tr.TipoFactorP)
		assert.Equal(t, "0.040000", tr.TasaOCuotaP)
		assert.Equal(t, "80.00", tr.ImporteP)

		// ...but it doesn't contribute to the totals of any of the rates the
		// SAT distinguishes.
		totales := doc.ComplementoPagos.Totales

		assert.Empty(t, totales.TotalTrasladosBaseIVA16)
		assert.Empty(t, totales.TotalTrasladosImpuestoIVA16)
		assert.Empty(t, totales.TotalTrasladosBaseIVA8)
		assert.Empty(t, totales.TotalTrasladosImpuestoIVA8)
		assert.Empty(t, totales.TotalTrasladosBaseIVA0)
		assert.Empty(t, totales.TotalTrasladosImpuestoIVA0)
		assert.Empty(t, totales.TotalTrasladosBaseIVAExento)
		assert.Equal(t, "2320.00", totales.MontoTotalPagos)
	})

	t.Run("should convert the totals to MXN when the currency is not MXN", func(t *testing.T) {
		pmt := validPayment()
		pmt.Currency = "USD"
		pmt.ExchangeRates = []*currency.ExchangeRate{
			{
				From:   "USD",
				To:     "MXN",
				Amount: num.MakeAmount(1750, 2), // 17.50
			},
		}

		doc, err := test.GenerateCFDIFrom(pmt)
		require.NoError(t, err)

		totales := doc.ComplementoPagos.Totales

		assert.Equal(t, "35000.00", totales.TotalTrasladosBaseIVA16)
		assert.Equal(t, "5600.00", totales.TotalTrasladosImpuestoIVA16)
		assert.Equal(t, "40600.00", totales.MontoTotalPagos)
	})
}

func validPayment() *bill.Payment {
	return &bill.Payment{
		Addons:    tax.WithAddons(addon.V4),
		Series:    "P",
		Code:      "0123",
		Currency:  "MXN",
		IssueDate: cal.MakeDate(2025, 3, 1),
		IssueTime: cal.NewTime(10, 30, 0),
		Ext: tax.ExtensionsOf(cbc.CodeMap{
			addon.ExtKeyIssuePlace: "21000",
		}),
		Supplier: &org.Party{
			Name: "Test Supplier",
			Ext: tax.ExtensionsOf(cbc.CodeMap{
				addon.ExtKeyFiscalRegime: "601",
			}),
			TaxID: &tax.Identity{
				Country: "MX",
				Code:    "AAA010101AAA",
			},
		},
		Customer: &org.Party{
			Name: "Test Customer",
			Ext: tax.ExtensionsOf(cbc.CodeMap{
				addon.ExtKeyFiscalRegime: "608",
			}),
			TaxID: &tax.Identity{
				Country: "MX",
				Code:    "URE180429TM6",
			},
			Addresses: []*org.Address{
				{
					Locality: "Mexico",
					Code:     "65000",
				},
			},
		},
		Lines: []*bill.PaymentLine{
			{
				Installment: 3,
				Document: &org.DocumentRef{
					Series:    "F",
					Code:      "00123",
					IssueDate: cal.NewDate(2025, 2, 1),
					Stamps: []*head.Stamp{
						{
							Provider: "sat-uuid",
							Value:    "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
						},
					},
				},
				Payable:  num.NewAmount(1000000, 2), // 10000.00
				Advances: num.NewAmount(300000, 2),  // 3000.00
				Amount:   num.MakeAmount(232000, 2), // 2320.00
				Tax: &tax.Total{
					Categories: []*tax.CategoryTotal{
						{
							Code: "VAT",
							Rates: []*tax.RateTotal{
								{
									Base:    num.MakeAmount(200000, 2), // 2000.00
									Percent: num.NewPercentage(160, 3), // 16.0%
								},
							},
						},
					},
				},
			},
		},
		Methods: []*pay.Record{
			{
				Key:  pay.MeansKeyCreditTransfer,
				Date: cal.NewDate(2025, 2, 25),
				Ref:  "0123456789",
			},
		},
	}
}

func paymentWithTwoLines() *bill.Payment {
	pmt := validPayment()
	pmt.Lines[0].Tax.Categories = append(pmt.Lines[0].Tax.Categories, &tax.CategoryTotal{
		Code:     "ISR",
		Retained: true,
		Rates: []*tax.RateTotal{
			{
				Base:    num.MakeAmount(200000, 2), // 2000.00
				Percent: num.NewPercentage(100, 3), // 10.0%
			},
		},
	})
	pmt.Lines = append(pmt.Lines, &bill.PaymentLine{
		Installment: 1,
		Document: &org.DocumentRef{
			Series:    "F",
			Code:      "00124",
			IssueDate: cal.NewDate(2025, 2, 10),
			Stamps: []*head.Stamp{
				{
					Provider: "sat-uuid",
					Value:    "bbbbbbbb-cccc-dddd-eeee-ffffffffffff",
				},
			},
		},
		Amount: num.MakeAmount(106000, 2), // 1060.00
		Tax: &tax.Total{
			Categories: []*tax.CategoryTotal{
				{
					Code: "VAT",
					Rates: []*tax.RateTotal{
						{
							Base:    num.MakeAmount(100000, 2), // 1000.00
							Percent: num.NewPercentage(160, 3), // 16.0%
						},
					},
				},
				{
					Code:     "ISR",
					Retained: true,
					Rates: []*tax.RateTotal{
						{
							Base:    num.MakeAmount(100000, 2), // 1000.00
							Percent: num.NewPercentage(100, 3), // 10.0%
						},
					},
				},
			},
		},
	})
	return pmt
}
