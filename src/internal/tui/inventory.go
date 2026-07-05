package tui

import (
	"sort"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/bubbles/v2/table"
	// "charm.land/lipgloss/v2"
	"github.com/muhamm-ad/stratus/service"
)

type sortKey int

const (
	sortNone sortKey = iota
	sortName
	sortProvider
	sortRegion
	sortType
	sortState
)

type inventoryModel struct {
	gw       service.Gateway
	all      []service.VM
	view     []service.VM // after filters+sort
	cursor   int
	marked   map[string]bool
	tbl      table.Model
	showDetail bool

	fProvider string // "" | aws | azure | gcp
	fState    string // "" | running | stopped | transition | unknown
	fRegion   string
	query     string
	sortK     sortKey
	sortAsc   bool
}

func newInventoryModel(gw service.Gateway) inventoryModel {
	t := table.New(table.WithColumns([]table.Column{
		{Title: "", Width: 2},
		{Title: "", Width: 2},
		{Title: "NAME", Width: 26},
		{Title: "PROVIDER", Width: 9},
		{Title: "REGION", Width: 15},
		{Title: "TYPE", Width: 18},
		{Title: "STATE", Width: 12},
	}))
	return inventoryModel{gw: gw, marked: map[string]bool{}, tbl: t, sortAsc: true}
}

func (m *inventoryModel) mergeProvider(provider string, vms []service.VM) {
	// Drop existing VMs for that provider, append fresh ones.
	kept := m.all[:0]
	for _, v := range m.all {
		if v.Provider != provider {
			kept = append(kept, v)
		}
	}
	m.all = append(kept, vms...)
	m.recompute()
}

func (m *inventoryModel) recompute() {
	m.view = m.view[:0]
	for _, v := range m.all {
		if m.fProvider != "" && v.Provider != m.fProvider { continue }
		if !stateMatches(m.fState, v.State) { continue }
		if m.fRegion != "" && v.Region != m.fRegion { continue }
		if m.query != "" && !strings.Contains(strings.ToLower(v.Name), strings.ToLower(m.query)) { continue }
		m.view = append(m.view, v)
	}
	m.applySort()
	m.syncRows()
}

func (m *inventoryModel) applySort() {
	if m.sortK == sortNone { return }
	less := func(i, j int) bool {
		a, b := m.view[i], m.view[j]
		var r bool
		switch m.sortK {
		case sortName: r = a.Name < b.Name
		case sortProvider: r = a.Provider < b.Provider
		case sortRegion: r = a.Region < b.Region
		case sortType: r = a.Type < b.Type
		case sortState: r = a.State < b.State
		}
		if !m.sortAsc { return !r }
		return r
	}
	sort.SliceStable(m.view, less)
}

func stateMatches(filter string, s service.VMState) bool {
	switch filter {
	case "", "all": return true
	case "running": return s == service.StateRunning
	case "stopped": return s == service.StateStopped
	case "transition": return s == service.StateStarting || s == service.StateStopping
	case "unknown": return s == service.StateUnknown
	}
	return true
}

// stateGlyph renders the exact glyphs from the mockup with theme colors.
func stateGlyph(s Styles, st service.VMState) string {
	switch st {
	case service.StateRunning: return s.OK.Render("● running")
	case service.StateStopped: return s.Err.Render("○ stopped")
	case service.StateStarting: return s.Warn.Render("◐ starting")
	case service.StateStopping: return s.Warn.Render("◑ stopping")
	default: return s.Dim.Render("◌ unknown")
	}
}

// Update handles inventory keys; connect/stop return commands to App.
func (m inventoryModel) Update(msg tea.KeyPressMsg, s Styles) (inventoryModel, tea.Cmd, appIntent) {
	switch msg.String() {
	case "j", "down":
		if m.cursor < len(m.view)-1 { m.cursor++ }
	case "k", "up":
		if m.cursor > 0 { m.cursor-- }
	case "G":
		m.cursor = len(m.view) - 1
	case " ":
		if len(m.view) > 0 { id := m.view[m.cursor].ID; m.marked[id] = !m.marked[id] }
	case "a":
		m.toggleMarkAll()
	case "enter":
		m.showDetail = !m.showDetail
	case "c":
		return m, nil, appIntent{kind: intentConnect, targets: m.connectTargets()}
	case "S":
		return m, nil, appIntent{kind: intentStop, targets: m.connectTargets()}
	case "p": m.fProvider = cycle(m.fProvider, "", "aws", "azure", "gcp"); m.recompute()
	case "f": m.fState = cycle(m.fState, "all", "running", "stopped", "transition", "unknown"); m.recompute()
	case "r": m.cycleRegion(); m.recompute()
	case "x": m.clearFilters(); m.recompute()
	case "o": m.sortK = (m.sortK + 1) % 6; m.recompute()
	case "O": m.sortAsc = !m.sortAsc; m.recompute()
	case "u":
		return m, nil, appIntent{kind: intentRefresh}
	}
	m.syncRows()
	return m, nil, appIntent{}
}

// connectTargets returns marked VMs (or the cursor VM), splitting out the ones
// with no permission so App can flash "N opened · M skipped (no permission)".
func (m inventoryModel) connectTargets() []service.VM {
	var out []service.VM
	if anyMarked(m.marked) {
		for _, v := range m.view { if m.marked[v.ID] { out = append(out, v) } }
	} else if len(m.view) > 0 {
		out = append(out, m.view[m.cursor])
	}
	return out
}
