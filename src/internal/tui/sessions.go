package tui

import (
	"fmt"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type sessionsModel struct {
	gw     *Gateway
	list   []Session
	cursor int
}

func newSessionsModel(gw *Gateway) sessionsModel {
	return sessionsModel{gw: gw, list: gw.Sessions()}
}

func (m sessionsModel) Update(msg tea.KeyPressMsg) (sessionsModel, tea.Cmd) {
	m.list = m.gw.Sessions()
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

func (m sessionsModel) View(s Styles, w, h int) string {
	head := s.SectionHead.Render("ACTIVE SESSIONS · coming soon")
	if len(m.list) == 0 {
		return lipgloss.Place(w, h, lipgloss.Left, lipgloss.Top,
			head+"\n\n"+s.Dim.Render("no active sessions — connect from inventory (c)"))
	}
	var rows []string
	for i, sess := range m.list {
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
