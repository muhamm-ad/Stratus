package tui

import "github.com/muhamm-ad/stratus/internal/core"

type intentKind int

const (
	intentNone intentKind = iota
	intentConnect
	intentStop
	intentStart
	intentRefresh
	intentReconnect
	intentShowDetail
)

type appIntent struct {
	kind     intentKind
	targets  []core.VM
	provider string
}
