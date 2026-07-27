package tui

import (
	"sort"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/muhamm-ad/stratus/internal/core"
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
	svc           *service.Service
	styles        Styles
	allVM         []core.VM
	filteredVM    []core.VM
	marked        map[string]bool
	tbl           table.Model
	showDetail    bool
	width, height int

	fProvider core.CloudProviderID
	fState    core.VMState
	fRegion   core.VMRegion
	query     string
	sortK     sortKey
	sortAsc   bool
}

func newInventoryModel(svc *service.Service, s Styles) inventoryModel {
	t := table.New(
		table.WithColumns([]table.Column{
			{Title: "", Width: 2},
			{Title: "NAME", Width: 28},
			{Title: "PROVIDER", Width: 9},
			{Title: "REGION", Width: 15},
			{Title: "TYPE", Width: 18},
			{Title: "STATE", Width: 12},
		}),
		table.WithKeyMap(inventoryTableKeyMap()),
		table.WithFocused(true),
		table.WithStyles(tableStylesFor(s)),
	)
	return inventoryModel{svc: svc, styles: s, marked: map[string]bool{}, tbl: t, sortAsc: true}
}

// inventoryTableKeyMap mirrors the app's own navigation keys (k/j/g/G) but
// strips PageUp/PageDown/HalfPageUp's default "b"/"f"/"space"/"u" bindings —
// those letters are already claimed by FilterState, Mark, and Refresh.
func inventoryTableKeyMap() table.KeyMap {
	return table.KeyMap{
		LineUp:       key.NewBinding(key.WithKeys("k", "up"), key.WithHelp("k/↑", "up")),
		LineDown:     key.NewBinding(key.WithKeys("j", "down"), key.WithHelp("j/↓", "down")),
		PageUp:       key.NewBinding(key.WithKeys("pgup"), key.WithHelp("pgup", "page up")),
		PageDown:     key.NewBinding(key.WithKeys("pgdown"), key.WithHelp("pgdn", "page down")),
		HalfPageUp:   key.NewBinding(key.WithKeys("ctrl+u"), key.WithHelp("ctrl+u", "½ page up")),
		HalfPageDown: key.NewBinding(key.WithKeys("ctrl+d"), key.WithHelp("ctrl+d", "½ page down")),
		// "g" never actually reaches the table: app_keys.go intercepts it for the
		// "gg" double-tap gesture before dispatch reaches here. Bound for symmetry only.
		GotoTop:    key.NewBinding(key.WithKeys("g", "home"), key.WithHelp("g", "top")),
		GotoBottom: key.NewBinding(key.WithKeys("G", "end"), key.WithHelp("G", "bottom")),
	}
}

func tableStylesFor(s Styles) table.Styles {
	return table.Styles{
		Header:   s.SectionHead.Bold(true).Padding(0, 1),
		Cell:     lipgloss.NewStyle().Padding(0, 1),
		Selected: s.Cursor,
	}
}

// applyStyles re-themes the table: both its chrome (SetStyles) and its row
// content, since cell values carry theme-colored glyphs baked in as ANSI text.
func (m *inventoryModel) applyStyles(s Styles) {
	m.styles = s
	m.tbl.SetStyles(tableStylesFor(m.styles))
	m.syncTableRows()
}

func (m *inventoryModel) mergeProvider(provider core.CloudProviderID, vms []core.VM) {
	// Drop existing VMs for that provider, append fresh ones.
	kept := m.allVM[:0]
	for _, v := range m.allVM {
		if v.Provider != provider {
			kept = append(kept, v)
		}
	}
	m.allVM = append(kept, vms...)
	m.recompute()
}

func (m *inventoryModel) recompute() {
	m.filteredVM = m.filteredVM[:0]
	for _, v := range m.allVM {
		if m.fProvider != "" && v.Provider != m.fProvider {
			continue
		}
		if m.fState != "" && v.State != m.fState {
			continue
		}
		if m.fRegion != "" && v.Region != m.fRegion {
			continue
		}
		// TODO: search query in a combined string of name, provider, region, type, state
		if m.query != "" && !strings.Contains(strings.ToLower(v.Name), strings.ToLower(m.query)) {
			continue
		}
		m.filteredVM = append(m.filteredVM, v)
	}
	m.applySort()
	m.syncTableRows()
}

func (m *inventoryModel) applySort() {
	if m.sortK == sortNone {
		return
	}
	less := func(i, j int) bool {
		a, b := m.filteredVM[i], m.filteredVM[j]
		var r bool
		switch m.sortK {
		case sortName:
			r = a.Name < b.Name
		case sortProvider:
			r = a.Provider < b.Provider
		case sortRegion:
			r = a.Region < b.Region
		case sortType:
			r = a.Type < b.Type
		case sortState:
			r = a.State < b.State
		}
		if !m.sortAsc {
			return !r
		}
		return r
	}
	sort.SliceStable(m.filteredVM, less)
}

// syncTableRows rebuilds the table's rows from filteredVM. table.Model.SetRows
// clamps the cursor to the new row count itself, replacing the old manual
// syncRows() cursor-clamp.
func (m *inventoryModel) syncTableRows() {
	rows := make([]table.Row, len(m.filteredVM))
	for i, vm := range m.filteredVM {
		rows[i] = m.vmToRow(vm)
	}
	m.tbl.SetRows(rows)
}

func (m inventoryModel) vmToRow(vm core.VM) table.Row {
	mark := "  "
	if m.marked[vm.ID] {
		mark = m.styles.Warn.Render("*") + " "
	}
	dot := lipgloss.NewStyle().Foreground(ProviderColor(vm.Provider)).Render("●")
	return table.Row{mark, dot + " " + vm.Name, string(vm.Provider), string(vm.Region), string(vm.Type), stateGlyph(m.styles, vm.State)}
}

// Update handles inventory keys; connect/stop return commands to App.
func (m *inventoryModel) Update(msg tea.KeyPressMsg, k KeyMap, s Styles) (inventoryModel, tea.Cmd, appIntent) {
	m.applyStyles(s)
	switch {
	case key.Matches(msg, k.Mark):
		if len(m.filteredVM) > 0 {
			id := m.filteredVM[m.tbl.Cursor()].ID
			m.marked[id] = !m.marked[id]
		}
		m.syncTableRows()
	case key.Matches(msg, k.MarkAll):
		m.toggleMarkAll()
		m.syncTableRows()
	case key.Matches(msg, k.Enter):
		m.showDetail = !m.showDetail
		m.applyTableHeight()
	case key.Matches(msg, k.Connect):
		return *m, nil, appIntent{kind: intentConnect, targets: m.connectTargets()}
	case key.Matches(msg, k.Stop):
		return *m, nil, appIntent{kind: intentStop, targets: m.connectTargets()}
	case key.Matches(msg, k.FilterProv):
		providersIds := append(m.svc.CloudProvidersIDs(), core.CloudProviderID(""))
		m.fProvider = cycle(m.fProvider, providersIds...)
		m.recompute()
	case key.Matches(msg, k.FilterState):
		states := []core.VMState{core.StateRunning, core.StateStopped, core.StateStarting, core.StateStopping, core.StateUnknown, ""}
		m.fState = cycle(m.fState, states...)
		m.recompute()
	case key.Matches(msg, k.FilterRegion):
		m.cycleRegion()
		m.recompute()
	case key.Matches(msg, k.ClearFilters):
		m.clearFilters()
		m.recompute()
	case key.Matches(msg, k.SortKey):
		m.sortK = (m.sortK + 1) % 6
		m.recompute()
	case key.Matches(msg, k.SortDir):
		m.sortAsc = !m.sortAsc
		m.recompute()
	case key.Matches(msg, k.Refresh):
		return *m, nil, appIntent{kind: intentRefresh}
	default:
		// Every remaining key (movement, paging) is safe to forward unconditionally:
		// inventoryTableKeyMap above guarantees no app-reserved letter is bound here.
		m.tbl, _ = m.tbl.Update(msg)
	}
	return *m, nil, appIntent{}
}

// stateGlyph renders the exact glyphs from the mockup with theme colors.
func stateGlyph(s Styles, st core.VMState) string {
	switch st {
	case core.StateRunning:
		return s.OK.Render("● running")
	case core.StateStopped:
		return s.Err.Render("○ stopped")
	case core.StateStarting:
		return s.Warn.Render("◐ starting")
	case core.StateStopping:
		return s.Warn.Render("◑ stopping")
	default:
		return s.Dim.Render("◌ unknown")
	}
}

// connectTargets returns marked VMs (or the cursor VM), splitting out the ones
// with no permission so App can flash "N opened · M skipped (no permission)".
func (m *inventoryModel) connectTargets() []core.VM {
	var out []core.VM
	if anyMarked(m.marked) {
		for _, v := range m.filteredVM {
			if m.marked[v.ID] {
				out = append(out, v)
			}
		}
	} else if len(m.filteredVM) > 0 {
		out = append(out, m.filteredVM[m.tbl.Cursor()])
	}
	return out
}

func (m *inventoryModel) toggleMarkAll() {
	if !anyMarked(m.marked) {
		for _, v := range m.filteredVM {
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

func (m *inventoryModel) distinctRegions() []core.VMRegion {
	seen := map[core.VMRegion]bool{}
	var out []core.VMRegion
	for _, v := range m.allVM {
		if v.Region != core.VMRegion("") && !seen[v.Region] {
			seen[v.Region] = true
			out = append(out, v.Region)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		return out[i] < out[j]
	})
	return out
}

func (m *inventoryModel) clearFilters() {
	m.fProvider = ""
	m.fState = ""
	m.fRegion = ""
	m.query = ""
}

// SetSize stores the available area and resizes the table (reserving room for
// the detail panel when it's open).
func (m *inventoryModel) SetSize(w, h int) {
	m.width, m.height = w, h
	m.applyTableHeight()
}

func (m *inventoryModel) applyTableHeight() {
	h := m.height
	if m.showDetail {
		h -= 6
	}
	if h < 1 {
		h = 1
	}
	m.tbl.SetWidth(m.width)
	m.tbl.SetHeight(h)
}

func (m *inventoryModel) View() string {
	if len(m.filteredVM) == 0 {
		msg := m.styles.Dim.Render("no vms match filters — press u to refresh")
		if len(m.allVM) == 0 {
			msg = m.styles.Dim.Render("no vms loaded — sign in and press u or R to sync providers")
		}
		return lipgloss.Place(m.width, m.height, lipgloss.Left, lipgloss.Top, msg)
	}

	body := m.tbl.View()
	if m.showDetail {
		if c := m.tbl.Cursor(); c < len(m.filteredVM) {
			body += "\n\n" + m.detailView(m.filteredVM[c])
		}
	}
	return lipgloss.NewStyle().Width(m.width).Height(m.height).Render(body)
}

func (m *inventoryModel) detailView(vm core.VM) string {
	lines := []string{
		m.styles.SectionHead.Render("INSTANCE DETAIL"),
		m.styles.Text.Render("name:    ") + m.styles.Accent.Render(vm.Name),
		m.styles.Text.Render("id:      ") + m.styles.Dim.Render(vm.ID),
		m.styles.Text.Render("ip:      ") + m.styles.Cyan.Render(string(vm.PrivateIP)),
		m.styles.Text.Render("method:  ") + m.styles.Text.Render("unknown (not implemented yet)"),
		m.styles.Text.Render("tags:    ") + m.styles.Dim.Render(formatTags(vm.Tags)),
	}
	return strings.Join(lines, "\n")
}
