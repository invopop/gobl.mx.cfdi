package main

import (
	"encoding/json"
	"fmt"
	"io"

	cfdi "github.com/invopop/gobl.cfdi"
	"github.com/spf13/cobra"
)

type parseOpts struct {
	*rootOpts
}

func parse(o *rootOpts) *parseOpts {
	return &parseOpts{rootOpts: o}
}

func (p *parseOpts) cmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "parse <infile> [outfile]",
		Short: "Parse a CFDI XML document and convert it to a GOBL envelope",
		RunE:  p.runE,
	}

	return cmd
}

func (p *parseOpts) runE(cmd *cobra.Command, args []string) error {
	// ctx := commandContext(cmd)

	if len(args) == 0 || len(args) > 2 {
		return fmt.Errorf("expected one or two arguments, the command usage is `gobl.cfdi parse <infile> [outfile]`")
	}

	input, err := openInput(cmd, args)
	if err != nil {
		return err
	}
	defer input.Close() // nolint:errcheck

	out, err := p.openOutput(cmd, args)
	if err != nil {
		return err
	}
	defer out.Close() // nolint:errcheck

	inData, err := io.ReadAll(input)
	if err != nil {
		return fmt.Errorf("reading input: %w", err)
	}

	env, err := cfdi.Parse(inData)
	if err != nil {
		return fmt.Errorf("parsing CFDI document: %w", err)
	}

	outData, err := json.MarshalIndent(env, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling GOBL envelope: %w", err)
	}

	if _, err = out.Write(outData); err != nil {
		return fmt.Errorf("writing json output: %w", err)
	}

	// Add a newline at the end for better formatting
	if _, err = out.Write([]byte("\n")); err != nil {
		return fmt.Errorf("writing json output: %w", err)
	}

	return nil
}
