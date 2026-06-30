// Package all enables the built-in cloud providers by importing them for their
// registration side effects (each provider's init() calls core.RegisterProvider).
//
// This is the SINGLE place to enable or disable a provider: add or remove a
// blank import below. The main package imports this package once; nothing else
// needs to know which providers exist.
package all

import (
	_ "github.com/muhamm-ad/stratus/providers/aws"
	_ "github.com/muhamm-ad/stratus/providers/azure"
	_ "github.com/muhamm-ad/stratus/providers/gcp"
)