package tui

import (
	"sort"
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
	keys := make([]string, 0, len(tags))
	for k := range tags {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, k+":"+tags[k])
	}
	return strings.Join(parts, " · ")
}

func dash(s string) string {
	if strings.TrimSpace(s) == "" {
		return "—"
	}
	return s
}
