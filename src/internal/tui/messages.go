package tui

import (
	"context"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/muhamm-ad/stratus/core"
)

// ---- messages -------------------------------------------------------------

type deviceCodeMsg core.DeviceCode
type loginResultMsg struct {
	identityProvider core.IdentityProvider
	err              error
}
type vmsLoadedMsg struct {
	provider string
	vms      []VM
}
type vmsLoadErrMsg struct {
	provider string
	err      error
}
type providerSyncedMsg struct {
	provider string
	count    int
}

type sessionOpenedMsg struct{ spec SessionSpec }
type sessionClosedMsg struct {
	id  string
	err error
}
type flashClearMsg struct{}
type tokenExpiredMsg struct{ provider string }
type autoRefreshMsg time.Time
type ggResetMsg struct{}

// ---- commands -------------------------------------------------------------

// loginCmd wraps the blocking OIDC login in a tea.Cmd. The device code is not
// returned here; it's pushed asynchronously via program.Send inside onCode.
func loginCmd(gw *Gateway, idp core.IdentityProviderID, send func(tea.Msg)) tea.Cmd {
	return func() tea.Msg {
		idp, err := gw.svc.LoginWith(context.Background(), idp, func(dc core.DeviceCode) {
			send(deviceCodeMsg(dc)) // inject the code into the program from the callback
		})
		return loginResultMsg{identityProvider: idp, err: err}
	}
}

// syncProviderCmds loads all providers' VMs after a staggered delay
func syncProviderCmds(gw *Gateway, delay bool) []tea.Cmd {
	cloudProviders := gw.svc.CloudProviders()
	cmds := make([]tea.Cmd, len(cloudProviders))

	for i, cp := range cloudProviders {
		provider := string(cp)
		d := time.Duration(0)
		if delay {
			d = time.Duration(i+1) * 500 * time.Millisecond
		}
		cmds[i] = tea.Tick(d, func(time.Time) tea.Msg {
			vms, err := gw.ListVMs(context.Background(), provider)
			if err != nil {
				return vmsLoadErrMsg{provider: provider, err: err}
			}
			return vmsLoadedMsg{provider: provider, vms: vms}
		})
	}
	return cmds
}

// execSessionCmd hands the terminal to the native CLI, then resumes the TUI.
func execSessionCmd(spec SessionSpec) tea.Cmd {
	c := buildExecCmd(spec) // *exec.Cmd, inherits our stdin/stdout/stderr
	return tea.ExecProcess(c, func(err error) tea.Msg {
		return sessionClosedMsg{id: spec.SessionID, err: err}
	})
}

func flashClearCmd() tea.Cmd {
	return tea.Tick(3200*time.Millisecond, func(time.Time) tea.Msg { return flashClearMsg{} })
}

func autoRefreshCmd() tea.Cmd {
	return tea.Tick(60*time.Second, func(t time.Time) tea.Msg { return autoRefreshMsg(t) })
}

func ggResetCmd() tea.Cmd {
	return tea.Tick(450*time.Millisecond, func(time.Time) tea.Msg { return ggResetMsg{} })
}
