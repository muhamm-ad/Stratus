package tui

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/muhamm-ad/stratus/internal/core"
	"github.com/muhamm-ad/stratus/internal/service"
)

const sessionListWidth = 30

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
	svc           *service.Service
	styles        Styles
	list          list.Model
	seq           int
	width, height int

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
		cur = m.styles.Cursor.Render("▸ ")
	}
	fmt.Fprintf(w, "%s%s · %s · %s · opened %s",
		cur, sess.Target, sess.Provider, sess.Method, sess.Opened.Format(time.Kitchen))
}

// Update handles Close itself, before ever forwarding to list.Update:
// list.Model's default NextPage binding includes "d", which would otherwise
// collide with Close ("x"/"d").
func (m *sessionsModel) Update(msg tea.KeyPressMsg, k KeyMap) tea.Cmd {
	if k.Sessions.Match(msg) == ActionCloseSession {
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

// SetSize splits the available width between the session list and the
// right-hand detail pane, mirroring the mockup's two-pane layout.
func (m *sessionsModel) SetSize(w, h int) {
	m.width, m.height = w, h
	m.list.SetSize(m.listWidth(), max(1, h))
}

func (m *sessionsModel) listWidth() int {
	w := sessionListWidth
	if m.width > 0 && w > m.width {
		w = m.width
	}
	return w
}

func (m *sessionsModel) rightPaneWidth() int {
	w := m.width - m.listWidth()
	if w < 1 {
		w = 1
	}
	return w
}

// selected returns the session under the list cursor, if any.
func (m *sessionsModel) selected() (session, bool) {
	item, ok := m.list.SelectedItem().(sessionItem)
	if !ok {
		return session{}, false
	}
	return session(item), true
}

func (m *sessionsModel) View() string {
	return lipgloss.JoinHorizontal(lipgloss.Top, m.leftPaneView(), m.rightPaneView())
}

func (m *sessionsModel) leftPaneView() string {
	if len(m.list.Items()) == 0 {
		msg := m.styles.Dim.Render("no active sessions\nconnect from\ninventory (c)")
		return lipgloss.Place(m.listWidth(), m.height, lipgloss.Left, lipgloss.Top, msg)
	}
	return boxNoWrap(lipgloss.NewStyle(), m.list.View(), m.listWidth(), m.height)
}

// rightPaneView renders the mockup's terminal-pane chrome for the selected
// session — VISUAL STUB ONLY. It is not wired to a real pty or
// tea.ExecProcess (real connect/attach is separate, later work — see the
// BuildSessionSpec/openSession FIXMEs below). A textinput.Model is
// deliberately not used here: a component that silently discards keystrokes
// would be misleading rather than honestly "not implemented yet".
func (m *sessionsModel) rightPaneView() string {
	sess, ok := m.selected()
	if !ok {
		msg := m.styles.Dim.Render("no session selected")
		if len(m.list.Items()) == 0 {
			msg = m.styles.Dim.Render("no active sessions — connect from inventory (c)")
		}
		return lipgloss.Place(m.rightPaneWidth(), m.height, lipgloss.Center, lipgloss.Center, msg)
	}

	dot := m.styles.OK.Render("●")
	provStyle := lipgloss.NewStyle().Foreground(ProviderColor(core.CloudProviderID(sess.Provider)))
	left := dot + " " + m.styles.Text.Bold(true).Render(sess.Target) + "  " +
		provStyle.Render(sess.Provider) + "  " + m.styles.Dim.Render(sess.Method)
	header := lipgloss.JoinHorizontal(lipgloss.Bottom, left, m.styles.Dim.Render("[x] close"))

	body := lipgloss.JoinVertical(lipgloss.Left,
		m.styles.Dim.Render("Connecting to "+sess.Target+" via "+sess.Method+"…"),
		m.styles.Dim.Render("opened "+sess.Opened.Format(time.Kitchen)),
		"",
		m.styles.Dim.Render("(visual preview only — not yet wired to a live session)"),
	)

	content := lipgloss.JoinVertical(lipgloss.Left, header, "", body)
	return boxNoWrap(m.styles.SidePanel, content, m.rightPaneWidth(), m.height)
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
