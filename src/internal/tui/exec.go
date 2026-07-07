// exec.go (same package) — builds the *exec.Cmd from a SessionSpec.
package tui

import "os/exec"

func buildExecCmd(spec SessionSpec) *exec.Cmd {
	// #nosec G204 -- argv comes from the service layer, not raw user input.
	return exec.Command(spec.Bin, spec.Args...)
}
