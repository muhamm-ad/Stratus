package tui

import (
	"context"
	"fmt"
	"sync"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/muhamm-ad/stratus/service"
)

type session struct {
	ID, Target, Provider, Method string
	Opened                       time.Time
}

// SessionSpec is what the TUI turns into an *exec.Cmd. The service decides the
// method+args; the TUI just runs it via tea.ExecProcess. This keeps the CLI
// invocation logic in the (provider-aware) service layer, not in the UI.
type SessionSpec struct {
	SessionID string
	VMName    string
	Provider  string
	Bin       string   // "aws" | "az" | "gcloud"
	Args      []string // full argv
}

type sessionsModel struct {
	svc      *service.Service
	sessions []session
	cursor   int
	seq      int

	mu sync.Mutex
}

func newSessionsModel(svc *service.Service) sessionsModel {
	return sessionsModel{svc: svc}
}

func (m *sessionsModel) Update(msg tea.KeyPressMsg) tea.Cmd {
	switch msg.String() {
	case "j", "down":
		if m.cursor < len(m.getSessions())-1 {
			m.cursor++
		}
	case "k", "up":
		if m.cursor > 0 {
			m.cursor--
		}
	}
	return nil
}

func (m *sessionsModel) View(s Styles, w, h int) string {
	head := s.SectionHead.Render("ACTIVE SESSIONS · coming soon")
	if len(m.getSessions()) == 0 {
		return lipgloss.Place(w, h, lipgloss.Left, lipgloss.Top,
			head+"\n\n"+s.Dim.Render("no active sessions — connect from inventory (c)"))
	}
	var rows []string
	for i, sess := range m.getSessions() {
		cur := "  "
		if i == m.cursor {
			cur = s.Accent.Render("▸ ")
		}
		line := fmt.Sprintf("%s%s · %s · %s · opened %s",
			cur, sess.Target, sess.Provider, sess.Method,
			sess.Opened.Format(time.Kitchen))
		rows = append(rows, line)
	}
	return lipgloss.Place(w, h, lipgloss.Left, lipgloss.Top,
		head+"\n\n"+lipgloss.JoinVertical(lipgloss.Left, rows...))
}

// BuildSessionSpec constructs the native CLI argv for connecting to a VM.
// Provider-specific knowledge lives here so the TUI only runs tea.ExecProcess.
func BuildSessionSpec(vm VM, sessionID string) SessionSpec { // FIXME: Implement this
	spec := SessionSpec{
		SessionID: sessionID,
		VMName:    vm.Name,
		Provider:  vm.Provider,
	}
	// switch vm.Provider {
	// case "aws":
	// 	spec.Bin, spec.Args = "aws", []string{
	// 		"ssm", "start-session", "--target", vm.ID, "--region", vm.Region,
	// 	}
	// case "azure":
	// 	spec.Bin, spec.Args = "az", []string{
	// 		"network", "bastion", "ssh",
	// 		"--name", "stratus-bastion", "--resource-group", "stratus-prod",
	// 		"--target-resource-id", vm.ID, "--auth-type", "AAD",
	// 	}
	// case "gcp":
	// 	spec.Bin, spec.Args = "gcloud", []string{
	// 		"compute", "ssh", vm.Name,
	// 		"--tunnel-through-iap", "--zone=" + vm.Region + "-a", "--project=stratus-dev",
	// 	}
	// }
	// if spec.Bin != "" {
	// 	if _, err := exec.LookPath(spec.Bin); err != nil {
	// 		spec.Bin, spec.Args = "sh", []string{"-c",
	// 			fmt.Sprintf("echo 'stratus: connected to %s via %s. type exit to return.'; exec ${SHELL:-sh}",
	// 				vm.Name, vm.Method)}
	// 	}
	// }
	return spec
}

func (m *sessionsModel) openSession(ctx context.Context, vmID string) (SessionSpec, error) { // FIXME: Implement this
	// vms, err := m.svc.ListInstances(ctx, cloudprovider.ID, account)
	// if err != nil {
	// 	return SessionSpec{}, err
	// }
	// var vm VM
	// for _, v := range vms {
	// 	if v.ID == vmID || v.Name == vmID {
	// 		vm = v
	// 		break
	// 	}
	// }
	// if vm.ID == "" {
	// 	return SessionSpec{}, fmt.Errorf("gateway: vm %q not found", vmID)
	// }
	// m.mu.Lock()
	// m.seq++
	// id := fmt.Sprintf("sess-%d", m.seq)
	// m.mu.Unlock()
	// spec := BuildSessionSpec(vm, id)
	// m.mu.Lock()
	// m.sessions = append(m.sessions, session{
	// 	ID: id, Target: vm.Name, Provider: vm.Provider, Method: vm.Method, Opened: time.Now(),
	// })
	// m.mu.Unlock()
	// return spec, nil

	return SessionSpec{}, nil
}

func (m *sessionsModel) CloseSession(ctx context.Context, sessionID string) error { // REVIEW: Implement this
	m.mu.Lock()
	defer m.mu.Unlock()
	for i, s := range m.sessions {
		if s.ID == sessionID {
			m.sessions = append(m.sessions[:i], m.sessions[i+1:]...)
			break
		}
	}
	return nil
}

func (m *sessionsModel) getSessions() []session { // REVIEW: Implement this
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]session, len(m.sessions))
	copy(out, m.sessions)
	return out
}
