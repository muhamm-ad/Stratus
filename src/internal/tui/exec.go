// exec.go (same package) — builds the *exec.Cmd from a SessionSpec.
package tui

import "os/exec"
import "github.com/muhamm-ad/stratus/service"

func buildExecCmd(spec service.SessionSpec) *exec.Cmd {
	// #nosec G204 -- argv comes from the service layer, not raw user input.
	return exec.Command(spec.Bin, spec.Args...)
}
