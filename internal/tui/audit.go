package tui

import (
	"fmt"
	"sync"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/muhamm-ad/stratus/internal/service"
)

type AuditEntry struct {
	When                       time.Time
	User, VM, Provider, Method string
	Success                    bool
}

type auditModel struct {
	svc    *service.Service
	audits []AuditEntry
	cursor int

	mu sync.Mutex
}

func newAuditModel(svc *service.Service) auditModel {
	return auditModel{svc: svc}
}

func (m *auditModel) Update(msg tea.KeyPressMsg) tea.Cmd {
	m.audits = m.getAudits()
	switch msg.String() {
	case "j", "down":
		if m.cursor < len(m.audits)-1 {
			m.cursor++
		}
	case "k", "up":
		if m.cursor > 0 {
			m.cursor--
		}
	}
	return nil
}

func (m *auditModel) View(s Styles, w, h int) string {
	head := s.SectionHead.Render("AUDIT LOG · coming soon")
	if len(m.audits) == 0 {
		return lipgloss.Place(w, h, lipgloss.Left, lipgloss.Top,
			head+"\n\n"+s.Dim.Render("no audit entries yet"))
	}
	var rows []string
	for i, e := range m.audits {
		cur := "  "
		if i == m.cursor {
			cur = s.Accent.Render("▸ ")
		}
		ok := s.OK.Render("✓")
		if !e.Success {
			ok = s.Err.Render("✗")
		}
		line := fmt.Sprintf("%s%s %s · %s · %s · %s · %s",
			cur, ok, e.When.Format("Jan 2 15:04"), e.User, e.VM, e.Provider, e.Method)
		rows = append(rows, line)
	}
	return lipgloss.Place(w, h, lipgloss.Left, lipgloss.Top,
		head+"\n\n"+lipgloss.JoinVertical(lipgloss.Left, rows...))
}


func (m *auditModel) getAudits() []AuditEntry {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]AuditEntry, len(m.audits))
	copy(out, m.audits)
	return out
}