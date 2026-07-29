package tui

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"github.com/muhamm-ad/stratus/internal/core"
	"github.com/muhamm-ad/stratus/internal/service"
)

type session struct {
	ID       string
	Target   string
	Provider string
	Method   string
	Opened   time.Time
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

type sessionItem session

func (i sessionItem) FilterValue() string { return i.Target + " " + i.Provider + " " + i.Method }

func sessionItems(sessions []session) []list.Item {
	items := make([]list.Item, len(sessions))
	for i, s := range sessions {
		items[i] = sessionItem(s)
	}
	return items
}

type sessionsModel struct {
	svc    *service.Service
	styles Styles
	list   list.Model
	seq    int

	mu       sync.Mutex
	sessions []session
}

func newSessionsModel(svc *service.Service, s Styles) sessionsModel {
	// DefaultDelegate is only used for height/pagination math — rows render in View.
	d := list.NewDefaultDelegate()
	d.ShowDescription = false
	d.SetSpacing(0)

	l := list.New(nil, d, 0, 0)
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetShowHelp(false)
	l.SetShowPagination(true)
	l.SetFilteringEnabled(false)
	// list.Model's default Quit ("q"/"esc") would otherwise fire once a
	// keypress falls through to this tab.
	l.DisableQuitKeybindings()
	return sessionsModel{svc: svc, list: l, styles: s}
}

func (m *sessionsModel) Render(w io.Writer, l list.Model, index int, item list.Item) {
	sess, ok := item.(sessionItem)
	if !ok {
		return
	}
	cur := "  "
	if index == l.Index() {
		cur = m.styles.Accent.Render("▸ ")
	}
	fmt.Fprintf(w, "%s%s · %s · %s · opened %s",
		cur, sess.Target, sess.Provider, sess.Method, sess.Opened.Format(time.Kitchen))
}

// Update handles CloseSess itself, before ever forwarding to list.Update:
// list.Model's default NextPage binding includes "d", which would otherwise
// collide with CloseSess ("x"/"d").
func (m *sessionsModel) Update(msg tea.KeyPressMsg, k KeyMap) tea.Cmd {
	if key.Matches(msg, k.CloseSess) {
		if item, ok := m.list.SelectedItem().(sessionItem); ok {
			_ = m.CloseSession(context.Background(), item.ID)
		}
		m.syncItems()
		return nil
	}
	m.syncItems()
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return cmd
}

// syncItems only calls SetItems when the underlying data actually changed —
// calling it unconditionally on every keystroke would be wasteful and, once
// filtering is ever enabled here, would clobber in-progress filter state.
func (m *sessionsModel) syncItems() {
	if fresh := m.getSessions(); len(fresh) != len(m.sessions) {
		m.sessions = fresh
		m.list.SetItems(sessionItems(m.sessions))
	}
}

func (m *sessionsModel) applyStyles(s Styles) {
	m.styles = s
	// m.list.SetDelegate(*m)
}

// SetSize reserves 2 lines for the hand-rolled header (+ blank line) which
// sits outside list.Model's own layout accounting.
func (m *sessionsModel) SetSize(w, h int) {
	m.list.SetSize(w, max(1, h-2))
}

func (m *sessionsModel) View() string {
	head := m.styles.SectionHead.Render("ACTIVE SESSIONS · coming soon")
	if len(m.list.Items()) == 0 {
		return head + "\n\n" + m.styles.Dim.Render("no active sessions — connect from inventory (c)")
	}
	return head + "\n\n" + m.list.View()
}

// BuildSessionSpec constructs the native CLI argv for connecting to a VM.
// Provider-specific knowledge lives here so the TUI only runs tea.ExecProcess.
func BuildSessionSpec(vm core.VM, sessionID string) SessionSpec { // FIXME: Implement this
	spec := SessionSpec{
		SessionID: sessionID,
		VMName:    vm.Name,
		Provider:  string(vm.Provider),
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
