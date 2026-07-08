package tui

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type auditModel struct {
	gw     *Gateway
	list   []AuditEntry
	cursor int
}

func newAuditModel(gw *Gateway) auditModel {
	return auditModel{gw: gw, list: gw.AuditLog()}
}

func (m auditModel) Update(msg tea.KeyPressMsg) (auditModel, tea.Cmd) {
	m.list = m.gw.AuditLog()
	switch msg.String() {
	case "j", "down":
		if m.cursor < len(m.list)-1 {
			m.cursor++
		}
	case "k", "up":
		if m.cursor > 0 {
			m.cursor--
		}
	}
	return m, nil
}

func (m auditModel) View(s Styles, w, h int) string {
	head := s.SectionHead.Render("AUDIT LOG · coming soon")
	if len(m.list) == 0 {
		return lipgloss.Place(w, h, lipgloss.Left, lipgloss.Top,
			head+"\n\n"+s.Dim.Render("no audit entries yet"))
	}
	var rows []string
	for i, e := range m.list {
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
