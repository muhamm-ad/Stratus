package tui

import (
	"fmt"
	"sort"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/bubbles/v2/table"
	"charm.land/lipgloss/v2"
	"github.com/muhamm-ad/stratus/internal/service"
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
	svc      *service.Service
	all      []VM
	view     []VM // after filters+sort
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

func newInventoryModel(svc *service.Service) inventoryModel {
	t := table.New(table.WithColumns([]table.Column{
		{Title: "", Width: 2},
		{Title: "", Width: 2},
		{Title: "NAME", Width: 26},
		{Title: "PROVIDER", Width: 9},
		{Title: "REGION", Width: 15},
		{Title: "TYPE", Width: 18},
		{Title: "STATE", Width: 12},
	}))
	return inventoryModel{svc: svc, marked: map[string]bool{}, tbl: t, sortAsc: true}
}

func (m *inventoryModel) mergeProvider(provider string, vms []VM) {
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

func stateMatches(filter string, s VMState) bool {
	switch filter {
	case "", "all": return true
	case "running": return s == StateRunning
	case "stopped": return s == StateStopped
	case "transition": return s == StateStarting || s == StateStopping
	case "unknown": return s == StateUnknown
	}
	return true
}

// stateGlyph renders the exact glyphs from the mockup with theme colors.
func stateGlyph(s Styles, st VMState) string {
	switch st {
	case StateRunning: return s.OK.Render("● running")
	case StateStopped: return s.Err.Render("○ stopped")
	case StateStarting: return s.Warn.Render("◐ starting")
	case StateStopping: return s.Warn.Render("◑ stopping")
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
func (m inventoryModel) connectTargets() []VM {
	var out []VM
	if anyMarked(m.marked) {
		for _, v := range m.view { if m.marked[v.ID] { out = append(out, v) } }
	} else if len(m.view) > 0 {
		out = append(out, m.view[m.cursor])
	}
	return out
}

func (m *inventoryModel) syncRows() {
	if m.cursor >= len(m.view) && len(m.view) > 0 {
		m.cursor = len(m.view) - 1
	}
	if len(m.view) == 0 {
		m.cursor = 0
	}
}

func (m *inventoryModel) toggleMarkAll() {
	if !anyMarked(m.marked) {
		for _, v := range m.view {
			m.marked[v.ID] = true
		}
		return
	}
	for id := range m.marked {
		delete(m.marked, id)
	}
}

func (m *inventoryModel) cycleRegion() {
	regions := m.distinctRegions()
	if len(regions) == 0 {
		m.fRegion = ""
		return
	}
	if m.fRegion == "" {
		m.fRegion = regions[0]
		return
	}
	for i, r := range regions {
		if r == m.fRegion {
			if i+1 < len(regions) {
				m.fRegion = regions[i+1]
			} else {
				m.fRegion = ""
			}
			return
		}
	}
	m.fRegion = regions[0]
}

func (m *inventoryModel) distinctRegions() []string {
	seen := map[string]bool{}
	var out []string
	for _, v := range m.all {
		if v.Region != "" && !seen[v.Region] {
			seen[v.Region] = true
			out = append(out, v.Region)
		}
	}
	sort.Strings(out)
	return out
}

func (m *inventoryModel) clearFilters() {
	m.fProvider = ""
	m.fState = ""
	m.fRegion = ""
	m.query = ""
}

func (m inventoryModel) View(s Styles, w, h int) string {
	if len(m.view) == 0 {
		msg := s.Dim.Render("no vms match filters — press u to refresh")
		if len(m.all) == 0 {
			msg = s.Dim.Render("no vms loaded — sign in and press u or R to sync providers")
		}
		return lipgloss.Place(w, h, lipgloss.Left, lipgloss.Top, msg)
	}

	head := s.SectionHead.Render(fmt.Sprintf(
		"  NAME%20sPROVIDER  REGION          TYPE            STATE", "",
	))

	var rows []string
	rows = append(rows, head)
	visible := h - 1
	if m.showDetail {
		visible -= 6
	}
	if visible < 1 {
		visible = 1
	}
	start := 0
	if m.cursor >= visible {
		start = m.cursor - visible + 1
	}
	end := start + visible
	if end > len(m.view) {
		end = len(m.view)
	}

	for i := start; i < end; i++ {
		v := m.view[i]
		rows = append(rows, m.renderRow(s, v, i == m.cursor))
	}

	body := strings.Join(rows, "\n")
	if m.showDetail && m.cursor < len(m.view) {
		body += "\n\n" + m.detailView(s, m.view[m.cursor])
	}
	return lipgloss.NewStyle().Width(w).Height(h).Render(body)
}

func (m inventoryModel) renderRow(s Styles, v VM, selected bool) string {
	mark := "  "
	if m.marked[v.ID] {
		mark = s.Warn.Render("* ")
	}
	cur := "  "
	if selected {
		cur = s.Accent.Render("▸ ")
	}
	dot := providerDot(s, v.Provider)
	name := padRight(v.Name, 26)
	prov := padRight(v.Provider, 9)
	region := padRight(v.Region, 15)
	typ := padRight(v.Type, 18)
	state := stateGlyph(s, v.State)
	line := cur + mark + dot + " " + name + prov + region + typ + state
	if selected {
		return s.Cursor.Render(line)
	}
	if m.marked[v.ID] {
		return s.Marked.Render(line)
	}
	return line
}

func (m inventoryModel) detailView(s Styles, v VM) string {
	lines := []string{
		s.SectionHead.Render("INSTANCE DETAIL"),
		s.Text.Render("name:    ") + s.Accent.Render(v.Name),
		s.Text.Render("id:      ") + s.Dim.Render(v.ID),
		s.Text.Render("ip:      ") + s.Cyan.Render(v.PrivateIP),
		s.Text.Render("method:  ") + s.Text.Render(vmMethodLabel(v)),
		s.Text.Render("tags:    ") + s.Dim.Render(formatTags(v.Tags)),
	}
	if len(v.Recent) > 0 {
		lines = append(lines, s.Text.Render("recent:  ") + s.Dim.Render(v.Recent[0].Text))
	}
	return strings.Join(lines, "\n")
}

func padRight(s string, n int) string {
	if len(s) >= n {
		return s[:n]
	}
	return s + strings.Repeat(" ", n-len(s))
}
