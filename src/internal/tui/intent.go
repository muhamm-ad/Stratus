package tui

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
	targets []VM
	provider string
}
