package tui

import (
	"strconv"
	"strings"

	"charm.land/lipgloss/v2"
)

func cycle(current string, vals ...string) string {
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

func providerDot(s Styles, provider string) string {
	return lipgloss.NewStyle().Foreground(ProviderColor(provider)).Render("●")
}

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

func vmMethodLabel(vm VM) string {
	if vm.Method != "" {
		return vm.Method
	}
	switch vm.Provider {
	case "aws":
		return "SSM"
	case "azure":
		return "Bastion"
	case "gcp":
		return "IAP"
	}
	return "—"
}
