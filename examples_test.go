package cfdi_test

import (
	"bytes"
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/invopop/gobl"
	cfdi "github.com/invopop/gobl.cfdi"
	"github.com/invopop/gobl.cfdi/test"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/dsig"
	"github.com/invopop/gobl/uuid"
	"github.com/lestrrat-go/libxml2"
	"github.com/lestrrat-go/libxml2/xsd"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	pathConvert = "convert"
	pathParse   = "parse"
	pathIn      = "in"
	pathOut     = "out"
)

var (
	updateOut = flag.Bool("update", false, "Update the JSON and XML files in test/data/convert and test/data/parse")

	staticUUID uuid.UUID = "0195ce71-dc9c-72c8-bf2c-9890a4a9f0a2"
)

func TestConvertExamples(t *testing.T) {
	schema, err := loadSchema()
	require.NoError(t, err)

	examples := findSourceFiles(t, pathConvert, "*.json")
	for _, example := range examples {
		inName := filepath.Base(example)
		outName := strings.Replace(inName, ".json", ".xml", 1)

		t.Run(inName, func(t *testing.T) {
			env := loadEnvelope(t, inputFilepath(pathConvert, inName))
			doc, err := cfdi.Convert(env)
			require.NoError(t, err)

			data, err := doc.Bytes()
			require.NoError(t, err)

			outPath := outputFilepath(pathConvert, outName)
			if *updateOut {
				errs := validateDoc(schema, data)
				for _, e := range errs {
					assert.NoError(t, e)
				}
				if len(errs) > 0 {
					assert.Fail(t, "Invalid XML:\n"+string(data))
					return
				}
				require.NoError(t, os.MkdirAll(filepath.Dir(outPath), 0755))
				require.NoError(t, os.WriteFile(outPath, data, 0644))
				return
			}

			output := loadOutputFile(t, pathConvert, outName)
			assert.Equal(t, strings.TrimSpace(string(output)), strings.TrimSpace(string(data)), "Output should match the expected XML. Update with --update flag.")
		})
	}
}

func TestParseExamples(t *testing.T) {
	examples := findSourceFiles(t, pathParse, "*.xml")
	for _, example := range examples {
		inName := filepath.Base(example)
		outName := strings.Replace(inName, ".xml", ".json", 1)

		t.Run(inName, func(t *testing.T) {
			xmlData, err := os.ReadFile(example)
			require.NoError(t, err)

			env, err := cfdi.Parse(xmlData)
			require.NoError(t, err)

			// Set the static UUID to avoid different UUIDs between executions.
			env.Head.UUID = staticUUID
			if inv, ok := env.Extract().(*bill.Invoice); ok {
				inv.UUID = staticUUID
			}
			require.NoError(t, env.Calculate())

			// Add a mock signature to make the envelope with the stamps valid
			env.Signatures = []*dsig.Signature{new(dsig.Signature)}

			writeEnvelope(t, dataPath(pathParse, pathOut, outName), env)

			data, err := json.MarshalIndent(env, "", "\t")
			require.NoError(t, err)

			output := loadOutputFile(t, pathParse, outName)
			var expectedEnv gobl.Envelope
			require.NoError(t, json.Unmarshal(output, &expectedEnv))

			expectedData, err := json.MarshalIndent(expectedEnv, "", "\t")
			require.NoError(t, err)

			assert.JSONEq(t, string(expectedData), string(data), "Invoice should match the expected JSON. Update with --update flag.")
		})
	}
}

func loadSchema() (*xsd.Schema, error) {
	schemaPath := filepath.Join(test.GetTestPath(), "schema", "schema.xsd")
	schema, err := xsd.ParseFromFile(schemaPath)
	if err != nil {
		return nil, err
	}

	return schema, nil
}

func loadEnvelope(t *testing.T, path string) *gobl.Envelope {
	t.Helper()

	src, err := os.ReadFile(path)
	require.NoError(t, err)

	env := new(gobl.Envelope)
	require.NoError(t, json.Unmarshal(src, env))
	require.NoError(t, env.Calculate())
	require.NoError(t, env.Validate())

	writeEnvelope(t, path, env)

	return env
}

func writeEnvelope(t *testing.T, path string, env *gobl.Envelope) {
	t.Helper()
	if !*updateOut {
		return
	}
	require.NoError(t, env.Validate())
	data, err := json.MarshalIndent(env, "", "\t")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, append(data, '\n'), 0644))
}

func outputFilepath(path, name string) string {
	return filepath.Join(dataPath(path, pathOut, name))
}

func inputFilepath(path, name string) string {
	return filepath.Join(dataPath(path, pathIn, name))
}

func loadOutputFile(t *testing.T, path, name string) []byte {
	t.Helper()
	src, err := os.Open(outputFilepath(path, name))
	require.NoError(t, err)
	buf := new(bytes.Buffer)
	if _, err := buf.ReadFrom(src); err != nil {
		require.NoError(t, err)
	}
	return buf.Bytes()
}

func findSourceFiles(t *testing.T, path, pattern string) []string {
	path = inputFilepath(path, pattern)
	files, err := filepath.Glob(path)
	require.NoError(t, err)
	return files
}

func dataPath(files ...string) string {
	files = append([]string{test.GetDataPath()}, files...)
	return filepath.Join(files...)
}

func validateDoc(schema *xsd.Schema, doc []byte) []error {
	xmlDoc, err := libxml2.ParseString(string(doc))
	if err != nil {
		return []error{err}
	}

	err = schema.Validate(xmlDoc)
	if err != nil {
		return err.(xsd.SchemaValidationError).Errors()
	}

	return nil
}
