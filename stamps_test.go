package cfdi_test

import (
	"testing"

	cfdi "github.com/invopop/gobl.cfdi"
	"github.com/invopop/gobl.cfdi/test"
	addon "github.com/invopop/gobl/addons/mx/cfdi"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/regimes/mx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testProvCertifRFC = "SPR190613I52"
	testCFDISignature = "NZaBl4TmCMq9zkUbWnD8a3AOzw4oScGyruxcZXonV1jIOWzrNGqwJbSDBLiSDJlKSXAueDBF+CVGuIu1wKok+FDT0pbSdKwwR+5K3X4U0uUEMiayfWAHInr2HsUbfNaCUWrndhsQyMVdnYh4v/qWAFkJfPJC+uZQHfqJD46GfgDORVMF0ZT93pu7qYuZLj2LEntQvbbp7GFmHMP96H1ccnnXXaik7fNSRKPmovPZfbC2hX5P4bwMBdh/aFwNI7iR7fsjNfdsIoluT1JQIrZdM/FsTvnm53GOJisdhEi5gSttiPCYsJ1Gd6U8H235IBBXXwJ9I8rF2ifHPquQizCINQ=="
)

func TestStamp(t *testing.T) {
	env, err := test.LoadTestEnvelope("invoice-b2b-full.json")
	require.NoError(t, err)

	doc, err := cfdi.Convert(env)
	require.NoError(t, err)

	// Add a dummy timbre fiscal digital
	doc.ComplementoTimbreFiscalDigital = &cfdi.TimbreFiscalDigital{
		Version:          "1.1",
		UUID:             "fd53505e-d737-43ab-815c-8090edec3655",
		FechaTimbrado:    "2022-09-05T13:04:29",
		RfcProvCertif:    testProvCertifRFC,
		SelloCFD:         testCFDISignature,
		NoCertificadoSAT: "30001000000400002495",
		SelloSAT:         "EVlPWO7rWg9xwNzQX3Wpt6tEGyizljFdH/9jpp0IklJsrEnxROOCpIpyZuzxfoSoEf7rR7v5sc4GImCvpGpb/6aXsFhmPLDko40rj26kkhGz/zDA++JHfUie9U5EBWPLnnQFFcpNydHyuCwyDWab8B7xgOsOtKbYqNg3VssjALid9QF3XRSHcVvJ7FV+i6rqwPgZMZUdkiQreDL7UKCx5WS2Hnvs5Vw3FiCDMxx/30duOzcOTCOPAOql1sKLb/5ohQqNyWRGQBdXBWYyGOgH2Y2W4ljCEty1HoTLPSAy4+gCoilXAER0I7KFe7aiidfj1QHwRzpyMd7XnWSWbUthyQ==",
	}
	doc.Sello = testCFDISignature
	doc.NoCertificado = "30001000000400002434"

	err = cfdi.Stamp(env, doc)
	assert.NoError(t, err)

	tests := []struct {
		name  string
		stamp cbc.Key
		value string
	}{
		{
			name:  "UUID",
			stamp: mx.StampSATUUID,
			value: "fd53505e-d737-43ab-815c-8090edec3655",
		},
		{
			name:  "URL",
			stamp: mx.StampSATURL,
			value: "https://verificacfdi.facturaelectronica.sat.gob.mx/default.aspx?id=fd53505e-d737-43ab-815c-8090edec3655&tt=211.36&re=EKU9003173C9&rr=URE180429TM6&fe=izCINQ==",
		},
		{
			name:  "Provider RFC",
			stamp: mx.StampSATProviderRFC,
			value: testProvCertifRFC,
		},
		{
			name:  "Chain",
			stamp: mx.StampSATChain,
			value: "||1.1|fd53505e-d737-43ab-815c-8090edec3655|2022-09-05T13:04:29|" + testProvCertifRFC + "|" + testCFDISignature + "|30001000000400002495||",
		},
		{
			name:  "SAT Serial",
			stamp: mx.StampSATSerial,
			value: "30001000000400002495",
		},
		{
			name:  "SAT Signature",
			stamp: mx.StampSATSignature,
			value: "EVlPWO7rWg9xwNzQX3Wpt6tEGyizljFdH/9jpp0IklJsrEnxROOCpIpyZuzxfoSoEf7rR7v5sc4GImCvpGpb/6aXsFhmPLDko40rj26kkhGz/zDA++JHfUie9U5EBWPLnnQFFcpNydHyuCwyDWab8B7xgOsOtKbYqNg3VssjALid9QF3XRSHcVvJ7FV+i6rqwPgZMZUdkiQreDL7UKCx5WS2Hnvs5Vw3FiCDMxx/30duOzcOTCOPAOql1sKLb/5ohQqNyWRGQBdXBWYyGOgH2Y2W4ljCEty1HoTLPSAy4+gCoilXAER0I7KFe7aiidfj1QHwRzpyMd7XnWSWbUthyQ==",
		},
		{
			name:  "CFDI Serial",
			stamp: addon.StampSerial,
			value: "30001000000400002434",
		},
		{
			name:  "CFDI Signature",
			stamp: addon.StampSignature,
			value: testCFDISignature,
		},
		{
			name:  "Timestamp",
			stamp: mx.StampSATTimestamp,
			value: "2022-09-05T13:04:29",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			st := env.Head.GetStamp(tt.stamp)
			require.NotNil(t, st)
			assert.Equal(t, tt.value, st.Value)
		})
	}

}
