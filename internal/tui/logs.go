package tui

import "time"

// maxLogEntries caps retained history well beyond a single toast.
const maxLogEntries = 200

type logEntry struct{ ts, level, msg string }

type logPane struct {
	entries []logEntry
	toasted map[string]struct{}
}

func newLogPane() logPane { return logPane{toasted: make(map[string]struct{})} }

func (l *logPane) add(level, msg string) {
	l.entries = append(l.entries, logEntry{ts: time.Now().Format("15:04:05"), level: level, msg: msg})
	if len(l.entries) > maxLogEntries {
		l.entries = l.entries[len(l.entries)-maxLogEntries:]
	}
}

func (l *logPane) shouldToast(msg string) bool {
	if l.toasted == nil {
		l.toasted = make(map[string]struct{})
	}
	if _, ok := l.toasted[msg]; ok {
		return false
	}
	l.toasted[msg] = struct{}{}
	return true
}
