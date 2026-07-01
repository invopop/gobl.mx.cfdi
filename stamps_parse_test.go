package cfdi_test

import (
	"os"
	"path/filepath"
	"testing"

	cfdi "github.com/invopop/gobl.mx.cfdi"
	"github.com/invopop/gobl.mx.cfdi/addon"
	"github.com/invopop/gobl.mx.cfdi/test"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/regimes/mx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseStamps(t *testing.T) {
	t.Run("should extract stamps from TimbreFiscalDigital", func(t *testing.T) {
		path := filepath.Join(test.GetDataPath(), "parse", "in", "invoice-timbre.xml")
		xmlData, err := os.ReadFile(path)
		require.NoError(t, err)

		env, err := cfdi.Parse(xmlData)
		require.NoError(t, err)

		tests := []struct {
			name  string
			stamp cbc.Key
			value string
		}{
			{
				name:  "UUID",
				stamp: mx.StampSATUUID,
				value: "fbff00b3-6577-437b-b15e-cbf25be09663",
			},
			{
				name:  "Provider RFC",
				stamp: mx.StampSATProviderRFC,
				value: testProvCertifRFC,
			},
			{
				name:  "SAT Serial",
				stamp: mx.StampSATSerial,
				value: "30001000000500003456",
			},
			{
				name:  "SAT Signature",
				stamp: mx.StampSATSignature,
				value: "V+qcMJ2KcJI2Of6+AW1dgNDJCrfkqHiCwFmMN0vTKr+N58vj5+NXwbvL2jm6/UJWErjiObHWyK5EhG92mfNu4zJmP1v0CMb6VgYnYHk/JvdL0DCsqOhf89Ybt6vKUHZB9GsFHNUaQX6mtVt0N0yhOjaoJGAXLR+9xvE2JuWBmgPpiAddGkE9D8lYWeB47TOlYU3tBJlaki1UCLM0lc5xiffxx6ktN0q8syQnfs5iIrGblt2CJDRMDHIvUwsV776i2ARxE1YAsoJGt1RAh/AkjYbNJSZTYWngAOatcosEoaCtchUFo+pWy6MYe+KKyO8KrNRWwvZr3ZqtaxCsMTi/LQ==",
			},
			{
				name:  "CFDI Serial",
				stamp: addon.StampSerial,
				value: "30001000000500003416",
			},
			{
				name:  "CFDI Signature",
				stamp: addon.StampSignature,
				value: "rnjrKDV2nqcILrqkDA+l+f1PhEZisAMEiYwQogTsijNF1nap8AkLYGsuV5KfEy/SUR+M1V7I4d/JH/2m6yBKphM/C6W/uNXJqevTQQ6zO+I2tlFb9yx4/Zz6pqkCjn3MQAt4fFDY1XpKn1HwD+gHiFGayjvsISrNr34rFDOCj9L/p9JJ5H+z3dwI728DRVmqogts7fNPhZY6ou0sZwkGIFaOUjg64b+OfVj2GnP1s0UJFMjsftEmYpdkF3JNtqunAF+vXL3/wE93wVleccvVBjs69AVGBSNedqjeEhgtjQrGeT56cPp/J0xt6Na3J1mAdmIyo49Of+8uLk0GW2BGVg==",
			},
			{
				name:  "Timestamp",
				stamp: mx.StampSATTimestamp,
				value: "2026-02-06T00:08:17",
			},
			{
				name:  "Chain",
				stamp: mx.StampSATChain,
				value: "||1.1|fbff00b3-6577-437b-b15e-cbf25be09663|2026-02-06T00:08:17|" + testProvCertifRFC + "|rnjrKDV2nqcILrqkDA+l+f1PhEZisAMEiYwQogTsijNF1nap8AkLYGsuV5KfEy/SUR+M1V7I4d/JH/2m6yBKphM/C6W/uNXJqevTQQ6zO+I2tlFb9yx4/Zz6pqkCjn3MQAt4fFDY1XpKn1HwD+gHiFGayjvsISrNr34rFDOCj9L/p9JJ5H+z3dwI728DRVmqogts7fNPhZY6ou0sZwkGIFaOUjg64b+OfVj2GnP1s0UJFMjsftEmYpdkF3JNtqunAF+vXL3/wE93wVleccvVBjs69AVGBSNedqjeEhgtjQrGeT56cPp/J0xt6Na3J1mAdmIyo49Of+8uLk0GW2BGVg==|30001000000500003456||",
			},
			{
				name:  "URL",
				stamp: mx.StampSATURL,
				value: "https://verificacfdi.facturaelectronica.sat.gob.mx/default.aspx?id=fbff00b3-6577-437b-b15e-cbf25be09663&tt=211.36&re=EKU9003173C9&rr=URE180429TM6&fe=W2BGVg==",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				st := env.Head.GetStamp(tt.stamp)
				require.NotNil(t, st, "stamp %s should exist", tt.stamp)
				assert.Equal(t, tt.value, st.Value)
			})
		}
	})

	t.Run("should not fail when TimbreFiscalDigital is not present", func(t *testing.T) {
		path := filepath.Join(test.GetDataPath(), "parse", "in", "invoice-b2b-bare.xml")
		xmlData, err := os.ReadFile(path)
		require.NoError(t, err)

		env, err := cfdi.Parse(xmlData)
		require.NoError(t, err)

		// Should have no stamps in head
		assert.Nil(t, env.Head.Stamps)
	})
}
