// Package test provides tools for testing the library
package test

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/invopop/gobl"
	cfdi "github.com/invopop/gobl.cfdi"
	"github.com/invopop/gobl/bill"
)

// NewDocumentFrom creates a CFDI Document from a GOBL file in the `test/data` folder
func NewDocumentFrom(name string) (*cfdi.Document, error) {
	env, err := LoadTestEnvelope(name)
	if err != nil {
		return nil, err
	}

	return cfdi.NewDocument(env)
}

// LoadTestInvoice returns a GOBL Invoice from a file in the `test/data` folder
func LoadTestInvoice(name string) (*bill.Invoice, error) {
	env, err := LoadTestEnvelope(name)
	if err != nil {
		return nil, err
	}

	return env.Extract().(*bill.Invoice), nil
}

// LoadTestEnvelope returns a GOBL Envelope from a file in the `test/data` folder
func LoadTestEnvelope(name string) (*gobl.Envelope, error) {
	path := filepath.Join(GetDataPath(), "convert", "in", name)
	src, err := os.Open(path)
	if err != nil {
		// Backwards compatible path.
		src, _ = os.Open(filepath.Join(GetDataPath(), name))
	}

	buf := new(bytes.Buffer)
	if _, err := buf.ReadFrom(src); err != nil {
		return nil, err
	}

	out, err := gobl.Parse(buf.Bytes())
	if err != nil {
		return nil, fmt.Errorf("parsing file: %w", err)
	}

	var env *gobl.Envelope
	switch obj := out.(type) {
	case *gobl.Envelope:
		env = obj
	default:
		env = gobl.NewEnvelope()
		if err := env.Insert(obj); err != nil {
			return nil, fmt.Errorf("inserting object: %w", err)
		}
	}

	if err := env.Calculate(); err != nil {
		return nil, err
	}

	if err := env.Validate(); err != nil {
		return nil, err
	}

	return env, nil
}

// GenerateCFDIFrom returns a CFDI Document from a GOBL Invoice
func GenerateCFDIFrom(inv *bill.Invoice) (*cfdi.Document, error) {
	if err := inv.Calculate(); err != nil {
		return nil, err
	}

	env, err := gobl.Envelop(inv)
	if err != nil {
		return nil, err
	}

	return cfdi.NewDocument(env)
}

// GetDataPath returns the path to the `test/data` folder
func GetDataPath() string {
	return filepath.Join(GetTestPath(), "data")
}

// GetTestPath returns the path to the `test` folder
func GetTestPath() string {
	return filepath.Join(getRootFolder(), "test")
}

func getRootFolder() string {
	cwd, _ := os.Getwd()

	for !isRootFolder(cwd) {
		cwd = removeLastEntry(cwd)
	}

	return cwd
}

func isRootFolder(dir string) bool {
	files, _ := os.ReadDir(dir)

	for _, file := range files {
		if file.Name() == "go.mod" {
			return true
		}
	}

	return false
}

func removeLastEntry(dir string) string {
	lastEntry := "/" + filepath.Base(dir)
	i := strings.LastIndex(dir, lastEntry)
	return dir[:i]
}
