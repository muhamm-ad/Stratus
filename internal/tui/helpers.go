package tui

import (
	"strconv"
	"strings"
)

func cycle[T comparable](current T, vals ...T) T {
	if len(vals) == 0 {
		return current
	}
	for i, v := range vals {
		if v == current {
			return vals[(i+1)%len(vals)]
		}
	}
	return vals[0]
}

func anyMarked(m map[string]bool) bool {
	for _, v := range m {
		if v {
			return true
		}
	}
	return false
}

func itoa(n int) string { return strconv.Itoa(n) }

func formatTags(tags map[string]string) string {
	if len(tags) == 0 {
		return "—"
	}
	var parts []string
	for k, v := range tags {
		parts = append(parts, k+":"+v)
	}
	return strings.Join(parts, " · ")
}
