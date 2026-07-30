package tui

import (
	"strings"
	"sync"
	"time"

	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/muhamm-ad/stratus/internal/core"
	"github.com/muhamm-ad/stratus/internal/service"
)

type AuditEntry struct {
	When                       time.Time
	User, VM, Provider, Method string
	Success                    bool
}

type auditModel struct {
	svc    *service.Service
	styles Styles
	tbl    table.Model

	width, height int

	mu              sync.Mutex
	audits          []AuditEntry // last-synced snapshot, change-detected against getAudits()
	filteredEntries []AuditEntry
	query           string
}

func newAuditModel(svc *service.Service, s Styles) auditModel {
	t := table.New(
		table.WithColumns(auditColumns(80)),
		table.WithKeyMap(vimTableKeyMap()),
		table.WithFocused(true),
		table.WithStyles(setTableStyles(s)),
	)
	return auditModel{svc: svc, styles: s, tbl: t}
}

// auditColumns sizes the VM column responsively, mirroring the mockup's
// grid-template-columns: fixed timestamp/user/provider/method/result
// columns, VM takes whatever width remains.
func auditColumns(w int) []table.Column {
	const timestampW, userW, providerW, methodW, resultW = 17, 10, 9, 9, 10
	vmW := w - (timestampW + userW + providerW + methodW + resultW)
	if vmW < 12 {
		vmW = 12
	}
	return []table.Column{
		{Title: "TIMESTAMP", Width: timestampW},
		{Title: "USER", Width: userW},
		{Title: "VM", Width: vmW},
		{Title: "PROVIDER", Width: providerW},
		{Title: "METHOD", Width: methodW},
		{Title: "RESULT", Width: resultW},
	}
}

func (m *auditModel) applyStyles(s Styles) {
	m.styles = s
	m.tbl.SetStyles(setTableStyles(s))
	m.syncTableRows()
}

func (m *auditModel) Update(msg tea.KeyPressMsg) tea.Cmd {
	if fresh := m.getAudits(); len(fresh) != len(m.audits) {
		m.audits = fresh
		m.recompute()
	}
	var cmd tea.Cmd
	m.tbl, cmd = m.tbl.Update(msg)
	m.syncTableRows()
	return cmd
}

// recompute re-derives filteredEntries from audits via m.query. Nothing
// wires a key to set m.query yet (see app_keys.go's Search case, which only
// arms searchMode for the inventory tab today) — this machinery exists so
// audit filtering doesn't structurally regress versus the list.Model it
// replaces once that gap is closed.
func (m *auditModel) recompute() {
	m.filteredEntries = m.filteredEntries[:0]
	q := strings.ToLower(m.query)
	for _, e := range m.audits {
		if q != "" {
			hay := strings.ToLower(e.User + " " + e.VM + " " + e.Provider + " " + e.Method)
			if !strings.Contains(hay, q) {
				continue
			}
		}
		m.filteredEntries = append(m.filteredEntries, e)
	}
	m.syncTableRows()
}

func (m *auditModel) syncTableRows() {
	rows := make([]table.Row, len(m.filteredEntries))
	for i, e := range m.filteredEntries {
		rows[i] = auditToRow(m.styles, e, i == m.tbl.Cursor())
	}
	m.tbl.SetRows(rows)
}

// auditToRow mirrors inventory.go's vmToRow: cells lose their explicit
// color when the row is selected, deferring to table.Styles.Selected's
// highlight instead of clashing with it.
func auditToRow(s Styles, e AuditEntry, selected bool) table.Row {
	ts := e.When.Format("Jan 2 15:04")
	prov := e.Provider
	method := e.Method
	result := "✓ success"
	resultStyle := s.OK
	if !e.Success {
		result, resultStyle = "✗ failure", s.Err
	}
	if !selected {
		ts = s.Dim.Render(ts)
		prov = lipgloss.NewStyle().Foreground(ProviderColor(core.CloudProviderID(e.Provider))).Render(prov)
		method = s.Dim.Render(method)
		result = resultStyle.Render(result)
	}
	return table.Row{ts, e.User, e.VM, prov, method, result}
}

// SetSize resizes the table and re-derives the VM column's responsive width.
func (m *auditModel) SetSize(w, h int) {
	m.width, m.height = w, h
	m.tbl.SetColumns(auditColumns(w))
	m.tbl.SetWidth(w)
	if h < 1 {
		h = 1
	}
	m.tbl.SetHeight(h)
}

func (m *auditModel) View() string {
	if len(m.audits) == 0 {
		return m.styles.Dim.Render("no audit entries yet")
	}
	if len(m.filteredEntries) == 0 {
		return m.styles.Dim.Render("no entries match your search")
	}
	return m.tbl.View()
}

func (m *auditModel) getAudits() []AuditEntry {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]AuditEntry, len(m.audits))
	copy(out, m.audits)
	return out
}
