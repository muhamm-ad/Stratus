package azure

import (
	"encoding/json"
	"os"

	"github.com/muhamm-ad/stratus/core"
)

func init() {
	core.RegisterProvider(core.ProviderAzure, func(raw json.RawMessage) (core.ProviderConnector, error) {
		cfg, err := ParseConfig(raw, os.Getenv)
		if err != nil {
			return nil, err
		}
		return New(cfg), nil
	})
}
