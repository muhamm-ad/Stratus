package tui

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"github.com/muhamm-ad/stratus/internal/service"
)

type AuditEntry struct {
	When                       time.Time
	User, VM, Provider, Method string
	Success                    bool
}

type auditItem AuditEntry

func (i auditItem) FilterValue() string { return i.User + " " + i.VM + " " + i.Provider + " " + i.Method }

func auditItems(entries []AuditEntry) []list.Item {
	items := make([]list.Item, len(entries))
	for i, e := range entries {
		items[i] = auditItem(e)
	}
	return items
}

type auditModel struct {
	svc    *service.Service
	styles Styles
	list   list.Model

	mu     sync.Mutex
	audits []AuditEntry
}

func newAuditModel(svc *service.Service, s Styles) auditModel {
	// DefaultDelegate is only used for height/pagination math — rows render in View.
	d := list.NewDefaultDelegate()
	d.ShowDescription = false
	d.SetSpacing(0)

	l := list.New(nil, d, 0, 0)
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetShowHelp(false)
	l.SetShowPagination(true)
	l.SetFilteringEnabled(true)
	// list.Model's default Quit ("q"/"esc") would otherwise fire once a
	// keypress falls through to this tab, since nothing else intercepts esc here.
	l.DisableQuitKeybindings()
	return auditModel{svc: svc, styles: s, list: l}
}

func (m *auditModel) applyStyles(s Styles) { m.styles = s }

func (m *auditModel) Update(msg tea.KeyPressMsg) tea.Cmd {
	var cmds []tea.Cmd
	// Only resync items (and thus re-run the filter) when the underlying data
	// actually changed — SetItems every keystroke would clobber in-progress filtering.
	if fresh := m.getAudits(); len(fresh) != len(m.audits) {
		m.audits = fresh
		if cmd := m.list.SetItems(auditItems(m.audits)); cmd != nil {
			cmds = append(cmds, cmd)
		}
	}
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}
	return tea.Batch(cmds...)
}

// SetSize reserves 2 lines for the hand-rolled header (+ blank line) which
// sits outside list.Model's own layout accounting.
func (m *auditModel) SetSize(w, h int) {
	m.list.SetSize(w, max(1, h-2))
}

func (m *auditModel) View() string {
	s := m.styles
	head := s.SectionHead.Render("AUDIT LOG · coming soon")
	if len(m.list.Items()) == 0 {
		return head + "\n\n" + s.Dim.Render("no audit entries yet")
	}

	var rows []string
	if m.list.SettingFilter() {
		rows = append(rows, m.list.FilterInput.View(), "")
	}

	items := m.list.VisibleItems()
	start, end := m.list.Paginator.GetSliceBounds(len(items))
	for i, item := range items[start:end] {
		e := item.(auditItem)
		cur := "  "
		if start+i == m.list.Index() {
			cur = s.Cursor.Render("▸ ")
		}
		okGlyph := s.OK.Render("✓")
		if !e.Success {
			okGlyph = s.Err.Render("✗")
		}
		rows = append(rows, fmt.Sprintf("%s%s %s · %s · %s · %s · %s",
			cur, okGlyph, e.When.Format("Jan 2 15:04"), e.User, e.VM, e.Provider, e.Method))
	}

	if m.list.Paginator.TotalPages > 1 {
		rows = append(rows, "", m.list.Paginator.View())
	}
	return head + "\n\n" + strings.Join(rows, "\n")
}

func (m *auditModel) getAudits() []AuditEntry {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]AuditEntry, len(m.audits))
	copy(out, m.audits)
	return out
}
