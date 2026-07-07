package tui

import "github.com/muhamm-ad/stratus/service"

type intentKind int

const (
	intentNone intentKind = iota
	intentConnect
	intentStop
	intentRefresh
	intentReconnect
)

type appIntent struct {
	kind    intentKind
	targets []service.VM
	provider string
}
