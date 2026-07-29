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
			{Title: "", Width: 1}, // ✓
			{Title: "NAME", Width: 30},
			{Title: "PROVIDER", Width: 9},
			{Title: "REGION", Width: 15},
			{Title: "TYPE", Width: 18},
			{Title: "STATE", Width: 12},
		}),
		table.WithKeyMap(inventoryTableKeyMap()),
		table.WithFocused(true),
		table.WithStyles(setTableStyles(s)),
	)
	return inventoryModel{svc: svc, styles: s, marked: map[string]bool{}, tbl: t, sortAsc: true}
}

// inventoryTableKeyMap mirrors the app's own navigation keys (k/j/g/G) but
// strips PageDown/HalfPageDown's default "f"/"space" bindings — those are
// already claimed by FilterState and other app keys.
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

func setTableStyles(s Styles) table.Styles {
	style := table.DefaultStyles()
	style.Header = s.SectionHead.BorderBottom(true).Padding(0, 1)
	style.Cell = lipgloss.NewStyle().Padding(0, 1)
	// Full-row selection style from bubbles table, themed with Accent.
	style.Selected = lipgloss.NewStyle().
		Background(s.th.Surface2).
		Foreground(s.th.Text).
		Bold(true)
	return style
}

// SetStyles re-themes the table: both its chrome (SetStyles) and its row
// content, since cell values carry theme-colored glyphs baked in as ANSI text.
func (m *inventoryModel) SetStyles(s Styles) {
	m.styles = s
	m.tbl.SetStyles(setTableStyles(s))
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
		rows[i] = m.vmToRow(vm, i == m.tbl.Cursor())
	}
	m.tbl.SetRows(rows)
}

func (m inventoryModel) vmToRow(vm core.VM, selected bool) table.Row {
	check := " "
	if m.marked[vm.ID] {
		if selected {
			check = "✓"
		} else {
			check = m.styles.OK.Render("✓")
		}
	}
	dot := "●"
	if !selected {
		dot = lipgloss.NewStyle().Foreground(ProviderColor(vm.Provider)).Render("●")
	}
	return table.Row{
		check,
		dot + " " + vm.Name,
		string(vm.Provider),
		string(vm.Region),
		string(vm.Type),
		stateGlyph(m.styles, vm.State, selected),
	}
}

func (m *inventoryModel) Update(msg tea.KeyPressMsg, k KeyMap, s Styles) (inventoryModel, tea.Cmd, appIntent) {
	m.SetStyles(s)
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
		if _, ok := m.selectedVM(); ok {
			return *m, nil, appIntent{kind: intentShowDetail}
		}
	case key.Matches(msg, k.Connect):
		return *m, nil, appIntent{kind: intentConnect, targets: m.selectedVMs()}
	case key.Matches(msg, k.Start):
		return *m, nil, appIntent{kind: intentStart, targets: m.selectedVMs()}
	case key.Matches(msg, k.Stop):
		return *m, nil, appIntent{kind: intentStop, targets: m.selectedVMs()}
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
		m.syncTableRows()
	}
	return *m, nil, appIntent{}
}

// stateGlyph renders the exact glyphs from the mockup with theme colors.
func stateGlyph(s Styles, st core.VMState, selected bool) string {
	var label string
	switch st {
	case core.StateRunning:
		label = "● running"
	case core.StateStopped:
		label = "○ stopped"
	case core.StateStarting:
		label = "◐ starting"
	case core.StateStopping:
		label = "◑ stopping"
	default:
		label = "◌ unknown"
	}

	if !selected {
		switch st {
		case core.StateRunning:
			return s.OK.Render(label)
		case core.StateStopped:
			return s.Err.Render(label)
		case core.StateStarting:
			return s.Warn.Render(label)
		case core.StateStopping:
			return s.Warn.Render(label)
		default:
			return s.Dim.Render(label)
		}
	}
	return label
}

// selectedVMs returns every marked VM, or — when nothing is marked — the VM
// under the cursor. App.connectCmd opens sessions for these (not implemented yet).
func (m *inventoryModel) selectedVMs() []core.VM {
	var out []core.VM
	if anyMarked(m.marked) {
		for _, v := range m.filteredVM {
			if m.marked[v.ID] {
				out = append(out, v)
			}
		}
		return out
	}
	if len(m.filteredVM) > 0 {
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

// SetSize stores the available area and resizes the table.
func (m *inventoryModel) SetSize(w, h int) {
	m.width, m.height = w, h
	m.applyTableHeight()
}

func (m *inventoryModel) applyTableHeight() {
	h := m.height
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
	return lipgloss.NewStyle().Width(m.width).Height(m.height).Render(m.tbl.View())
}

// selectedVM returns the VM under the table cursor, if any.
func (m *inventoryModel) selectedVM() (core.VM, bool) {
	c := m.tbl.Cursor()
	if c < 0 || c >= len(m.filteredVM) {
		return core.VM{}, false
	}
	return m.filteredVM[c], true
}

// detailView renders the VM detail as a floating modal (App.composeOverlay
// centers it over the inventory table via the Lip Gloss v2 compositor).
func (m *inventoryModel) detailView(vm core.VM) string {
	lines := []string{
		m.styles.Accent.Bold(true).Render("instance detail"),
		"",
		m.styles.Text.Render("name:    ") + m.styles.Accent.Render(vm.Name),
		m.styles.Text.Render("id:      ") + m.styles.Dim.Render(vm.ID),
		m.styles.Text.Render("provider:") + " " + m.styles.Text.Render(string(vm.Provider)),
		m.styles.Text.Render("region:  ") + m.styles.Text.Render(string(vm.Region)),
		m.styles.Text.Render("type:    ") + m.styles.Text.Render(string(vm.Type)),
		m.styles.Text.Render("state:   ") + stateGlyph(m.styles, vm.State, false),
		m.styles.Text.Render("ip:      ") + m.styles.Cyan.Render(string(vm.PrivateIP)),
		m.styles.Text.Render("method:  ") + m.styles.Text.Render("unknown (not implemented yet)"),
		m.styles.Text.Render("tags:    ") + m.styles.Dim.Render(formatTags(vm.Tags)),
		"",
		m.styles.Dim.Render("esc/⏎ close"),
	}
	return m.styles.OverlayBox.Width(56).Render(strings.Join(lines, "\n"))
}
