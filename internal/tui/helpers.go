package tui

import (
	"strconv"
	"strings"
)

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
