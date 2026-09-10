package tui

import "time"

// maxLogEntries caps retained history well beyond a single toast.
const maxLogEntries = 200

type logEntry struct{ ts, level, msg string }

type logPane struct {
	entries []logEntry
}

func newLogPane() logPane { return logPane{} }

func (l *logPane) add(level, msg string) {
	l.entries = append(l.entries, logEntry{ts: time.Now().Format("15:04:05"), level: level, msg: msg})
	if len(l.entries) > maxLogEntries {
		l.entries = l.entries[len(l.entries)-maxLogEntries:]
	}
}
