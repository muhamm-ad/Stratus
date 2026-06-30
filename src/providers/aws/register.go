package aws

import (
	"encoding/json"
	"os"

	"github.com/muhamm-ad/stratus/core"
)

// init registers the AWS connector factory in the global catalog. The app
// enables it via a blank import of providers/all.
func init() {
	core.RegisterProvider(core.ProviderAWS, func(raw json.RawMessage) (core.ProviderConnector, error) {
		cfg, err := ParseConfig(raw, os.Getenv)
		if err != nil {
			return nil, err
		}
		return New(cfg), nil
	})
}
