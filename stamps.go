package cfdi

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/invopop/gobl"
	addon "github.com/invopop/gobl/addons/mx/cfdi"
	"github.com/invopop/gobl/head"
	"github.com/invopop/gobl/regimes/mx"
	"github.com/invopop/validation"
	"github.com/invopop/validation/is"
)

const (
	satVerifyBaseURL = "https://verificacfdi.facturaelectronica.sat.gob.mx/default.aspx"
	// ?&id=7789e672-4d89-4b5b-8ed5-c85eb1ddca27&re=RBL2206107N8&rr=INS210623QU7&tt=000000000000022576.570000&fe=A9fALQ==
)

// StampData defines all the fields that are expected from provider.
type StampData struct {
	UUID        string
	CFDI        *Signature
	SAT         *Signature
	ProviderRFC string
	Chain       string
	Timestamp   string
}

// Signature represents the data for a signature.
type Signature struct {
	Serial string
	Value  string
}

// NewSignature instantiates a new signature with the provided serial and value.
func NewSignature(serial, value string) *Signature {
	return &Signature{
		Serial: serial,
		Value:  value,
	}
}

// Stamp takes the provided structured stamp data and applies it to the envelope.
// This should be done after the CFDI document has been processed and signed
// by a PAC (Proveedor Autorizado de Certificación) in Mexico.
func Stamp(env *gobl.Envelope, sd *StampData, doc *Document) error {
	if err := sd.Validate(); err != nil {
		return err
	}

	// Add all the stamps
	env.Head.AddStamp(&head.Stamp{
		Provider: mx.StampSATUUID,
		Value:    sd.UUID,
	})
	env.Head.AddStamp(&head.Stamp{
		Provider: addon.StampSignature,
		Value:    sd.CFDI.Value,
	})
	env.Head.AddStamp(&head.Stamp{
		Provider: addon.StampSerial,
		Value:    sd.CFDI.Serial,
	})
	env.Head.AddStamp(&head.Stamp{
		Provider: mx.StampSATSignature,
		Value:    sd.SAT.Value,
	})
	env.Head.AddStamp(&head.Stamp{
		Provider: mx.StampSATSerial,
		Value:    sd.SAT.Serial,
	})
	env.Head.AddStamp(&head.Stamp{
		Provider: mx.StampSATTimestamp,
		Value:    sd.Timestamp,
	})
	env.Head.AddStamp(&head.Stamp{
		Provider: mx.StampSATChain,
		Value:    sd.Chain,
	})
	// Provider RFC not required, but if its there, add it.
	if sd.ProviderRFC != "" {
		env.Head.AddStamp(&head.Stamp{
			Provider: mx.StampSATProviderRFC,
			Value:    sd.ProviderRFC,
		})
	}

	// Generate and add the QR code.
	base, err := url.Parse(satVerifyBaseURL)
	if err != nil {
		return fmt.Errorf("parsing base URL: %w", err)
	}
	// Manually build up the query so that we don't escape the `=` included
	// in the signature. When encoded, the SAT website does not read them
	// correctly :facepalm:.
	q := []string{
		fmt.Sprintf("id=%s", sd.UUID),
		fmt.Sprintf("tt=%s", doc.Total),
		fmt.Sprintf("re=%s", doc.Emisor.Rfc),
		fmt.Sprintf("rr=%s", doc.Receptor.Rfc),
		fmt.Sprintf("fe=%s", sd.CFDI.Value[len(sd.CFDI.Value)-8:]),
	}
	base.RawQuery = strings.Join(q, "&")
	env.Head.AddStamp(&head.Stamp{
		Provider: mx.StampSATURL,
		Value:    base.String(),
	})

	return nil
}

// Validate ensures the incoming Stamp data looks correct.
func (sd *StampData) Validate() error {
	return validation.ValidateStruct(sd,
		validation.Field(&sd.UUID, validation.Required, is.UUID),
		validation.Field(&sd.CFDI, validation.Required),
		validation.Field(&sd.SAT, validation.Required),
		validation.Field(&sd.ProviderRFC),
		validation.Field(&sd.Chain, validation.Required),
		validation.Field(&sd.Timestamp),
	)
}

// Validate ensures the signature looks good
func (sig *Signature) Validate() error {
	return validation.ValidateStruct(sig,
		validation.Field(&sig.Serial, validation.Required),
		validation.Field(&sig.Value, validation.Required),
	)
}
