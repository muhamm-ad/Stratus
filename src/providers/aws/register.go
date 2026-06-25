package aws

import (
	"encoding/json"
	"fmt"

	"github.com/muhamm-ad/stratus/core"
)

// Register builds the AWS connector from its configuration section and adds it
// to the registry. Called once at startup from app.go:
//
//	secs, _ := config.Load()
//	err := aws.Register(app.registry, secs.AWS)
//
// Passing the section (rather than loading config here) keeps this package
// decoupled from the config package and lets app.go load config.json once.
func Register(reg *core.Registry, raw json.RawMessage, opts ...Option) error {
	cfg, err := ParseConfig(raw)
	if err != nil {
		return fmt.Errorf("aws: %w", err)
	}
	provider, err := NewProvider(cfg, opts...)
	if err != nil {
		return fmt.Errorf("aws: %w", err)
	}
	reg.Register(provider)
	return nil
}