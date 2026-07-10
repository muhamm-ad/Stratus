package gcp

import (
	"encoding/json"
	"os"

	"github.com/muhamm-ad/stratus/internal/core"
)

func init() {
	core.RegisterProvider(ProviderID, func(raw json.RawMessage) (core.CloudProvider, error) {
		cfg, err := ParseConfig(raw, os.Getenv)
		if err != nil {
			return nil, err
		}
		return New(cfg), nil
	})
}
