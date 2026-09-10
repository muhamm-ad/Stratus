package tui

import (
	"context"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/muhamm-ad/stratus/internal/core"
	"github.com/muhamm-ad/stratus/internal/service"
)

// ---- messages -------------------------------------------------------------

type deviceCodeMsg core.DeviceCode
type loginResultMsg struct {
	identityProvider core.IdentityProvider
	cpErrors         map[core.CloudProviderID]error
	err              error
}
type vmsLoadedMsg struct {
	provider core.CloudProviderID
	vms      []core.VM
}
type vmsLoadErrMsg struct {
	provider core.CloudProviderID
	err      error
}

type sessionOpenedMsg struct{ spec SessionSpec }
type sessionClosedMsg struct {
	id  string
	err error
}
type tokenExpiredMsg struct{ provider core.CloudProviderID }
type autoRefreshMsg time.Time
type ggResetMsg struct{}

type providerReconnectOKMsg struct{ provider core.CloudProviderID }
type providerReconnectErrMsg struct {
	provider core.CloudProviderID
	err      error
}

// ---- commands -------------------------------------------------------------

// loginCmd wraps the blocking OIDC login in a tea.Cmd. The device code is not
// returned here; it's pushed asynchronously via program.Send inside onCode.
// Cloud providers are auto-connected inside LoginWith; failures land in
// loginResultMsg.cpErrors without failing the overall login.
func loginCmd(svc *service.Service, idpID core.IdentityProviderID, send func(tea.Msg)) tea.Cmd {
	return func() tea.Msg {
		idp, err, cpErrors := svc.LoginWith(context.Background(), idpID, func(dc core.DeviceCode) {
			send(deviceCodeMsg(dc)) // inject the code into the program from the callback
		})
		return loginResultMsg{identityProvider: idp, cpErrors: cpErrors, err: err}
	}
}

// syncProviderCmd lists one provider's VMs after an optional delay.
func syncProviderCmd(svc *service.Service, cp core.CloudProviderID, delay time.Duration) tea.Cmd {
	return tea.Tick(delay, func(time.Time) tea.Msg {
		vms, err := svc.ListVMs(context.Background(), cp)
		if err != nil {
			return vmsLoadErrMsg{provider: cp, err: err}
		}
		return vmsLoadedMsg{provider: cp, vms: vms}
	})
}

// syncProviderCmds loads all providers' VMs after a staggered delay
func syncProviderCmds(svc *service.Service, delay bool) []tea.Cmd {
	cloudProviders := svc.GetCloudProvidersIDs()
	cmds := make([]tea.Cmd, len(cloudProviders))

	for i, cp := range cloudProviders {
		d := time.Duration(0)
		if delay {
			d = time.Duration(i+1) * 200 * time.Millisecond
		}
		cmds[i] = syncProviderCmd(svc, cp, d)
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

func autoRefreshCmd() tea.Cmd {
	return tea.Tick(60*time.Second, func(t time.Time) tea.Msg { return autoRefreshMsg(t) })
}

func ggResetCmd() tea.Cmd {
	return tea.Tick(450*time.Millisecond, func(time.Time) tea.Msg { return ggResetMsg{} })
}
