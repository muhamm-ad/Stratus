package tui

import (
	"fmt"
	"strings"
)

const maxLogLines = 6

type logPane struct {
	lines []string
}

func newLogPane() logPane { return logPane{} }

func (l *logPane) add(level, msg string) {
	line := fmt.Sprintf("%s %s", level, msg)
	l.lines = append(l.lines, line)
	if len(l.lines) > maxLogLines {
		l.lines = l.lines[len(l.lines)-maxLogLines:]
	}
}

func (l logPane) View(s Styles, w int) string {
	head := s.SectionHead.Render("LOG")
	if len(l.lines) == 0 {
		return s.Dim.Width(w).Render(head + "\n(no events)")
	}
	return s.Dim.Width(w).Render(head + "\n" + strings.Join(l.lines, "\n"))
}
