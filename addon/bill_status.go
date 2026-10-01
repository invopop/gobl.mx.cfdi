package addon

import (
	"fmt"

	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/rules"
	"github.com/invopop/gobl/tax"
)

func billStatusRules() *rules.Set {
	return rules.For(new(bill.Status),
		rules.Field("lines",
			rules.Each(
				rules.Field("ext",
					rules.Assert("01",
						fmt.Sprintf("line '%s' extension is not valid", ExtKeyCancelReason),
						tax.ExtensionHasValidCode(ExtKeyCancelReason),
					),
				),
			),
		),
	)
}
