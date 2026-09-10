package tui

import (
	"sort"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/table"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/muhamm-ad/stratus/internal/core"
	"github.com/muhamm-ad/stratus/internal/service"
)

// detailPanelWidth is a terminal-column width, not the mockup's 380 CSS
// pixels — chosen to read well at typical 80-160 col terminal widths.
const detailPanelWidth = 42

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

	detailOn bool
	detailVP viewport.Model

	fProvider core.CloudProviderID
	fState    core.VMState
	fRegion   core.VMRegion
	query     string
	sortK     sortKey
	sortAsc   bool
}

func newInventoryModel(svc *service.Service, s Styles) inventoryModel {
	t := table.New(
		table.WithColumns(fullTableColumns(80)),
		table.WithKeyMap(vimTableKeyMap()),
		table.WithFocused(true),
		table.WithStyles(setTableStyles(s)),
	)
	return inventoryModel{svc: svc, styles: s, marked: map[string]bool{}, tbl: t, detailVP: viewport.New(), sortAsc: true}
}

// tableCellPad is the horizontal padding from setTableStyles (Padding(0, 1)
// on Header and Cell). Bubbles sizes columns by content width only, so this
// extra must be subtracted or the table overflows the terminal.
const tableCellPad = 2

const (
	invColCheck    = 1
	invColProvider = 9
	invColRegion   = 15
	invColType     = 18
	invColState    = 12
	invColNameMin  = 12
)

func flexNameWidth(tableW, nCols, fixedContent int) int {
	nameW := tableW - fixedContent - nCols*tableCellPad
	if nameW < invColNameMin {
		return invColNameMin
	}
	return nameW
}

// fullTableColumns sizes NAME to the leftover width so the table fills the
// terminal; PROVIDER/REGION/TYPE/STATE stay fixed on the right.
func fullTableColumns(w int) []table.Column {
	fixed := invColCheck + invColProvider + invColRegion + invColType + invColState
	return []table.Column{
		{Title: "", Width: invColCheck}, // ✓
		{Title: "NAME", Width: flexNameWidth(w, 6, fixed)},
		{Title: "PROVIDER", Width: invColProvider},
		{Title: "REGION", Width: invColRegion},
		{Title: "TYPE", Width: invColType},
		{Title: "STATE", Width: invColState},
	}
}

// narrowTableColumns drops REGION/TYPE when the detail panel is docked,
// mirroring the mockup's column-hiding behavior at reduced width. NAME still
// absorbs leftover space.
func narrowTableColumns(w int) []table.Column {
	fixed := invColCheck + invColProvider + invColState
	return []table.Column{
		{Title: "", Width: invColCheck}, // ✓
		{Title: "NAME", Width: flexNameWidth(w, 4, fixed)},
		{Title: "PROVIDER", Width: invColProvider},
		{Title: "STATE", Width: invColState},
	}
}

// vimTableKeyMap mirrors the app's own navigation keys (k/j/g/G) but
// strips PageDown/HalfPageDown's default "f"/"space" bindings — those are
// already claimed by FilterState and other app keys. Shared by the
// inventory and audit tables.
func vimTableKeyMap() table.KeyMap {
	return table.KeyMap{
		LineUp:       key.NewBinding(key.WithKeys("k", "up"), key.WithHelp("k/↑", "up")),
		LineDown:     key.NewBinding(key.WithKeys("j", "down"), key.WithHelp("j/↓", "down")),
		PageUp:       key.NewBinding(key.WithKeys("pgup"), key.WithHelp("pgup", "page up")),
		PageDown:     key.NewBinding(key.WithKeys("pgdown"), key.WithHelp("pgdn", "page down")),
		HalfPageUp:   key.NewBinding(key.WithKeys("ctrl+u"), key.WithHelp("ctrl+u", "½ page up")),
		HalfPageDown: key.NewBinding(key.WithKeys("ctrl+d"), key.WithHelp("ctrl+d", "½ page down")),
		// "g"/"G"/"home"/"end" are handled on the app Nav layer (gg + jump)
		// before dispatch reaches the table. Bound for symmetry only.
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
	cloudProvider := string(vm.Provider)
	if !selected {
		cloudProvider = cloudProviderGlyph(vm.Provider, selected)
	}
	name := string(vm.Name)
	state := stateGlyph(m.styles, vm.State, selected)
	if m.detailOn {
		return table.Row{check, name, cloudProvider, state}
	}
	return table.Row{check, name, cloudProvider, string(vm.Region), string(vm.Type), state}
}

func (m *inventoryModel) Update(msg tea.KeyPressMsg, k KeyMap, s Styles) (inventoryModel, tea.Cmd, appIntent) {
	m.SetStyles(s)
	act := k.Inventory.Match(msg)
	if act == ActionNone {
		switch nav := k.Nav.Match(msg); nav {
		case ActionSelect, ActionBack, ActionJumpTop, ActionJumpBottom:
			act = nav
		}
	}
	if act != ActionNone {
		return m.Handle(act)
	}
	// Movement / paging stays on the table keymap so j/k/pgup match Nav
	// without a second dispatch table. Mouse later calls Handle directly.
	m.tbl, _ = m.tbl.Update(msg)
	m.syncTableRows()
	return *m, nil, appIntent{}
}

func (m *inventoryModel) Handle(act Action) (inventoryModel, tea.Cmd, appIntent) {
	switch act {
	case ActionMark:
		if len(m.filteredVM) > 0 {
			id := m.filteredVM[m.tbl.Cursor()].ID
			m.marked[id] = !m.marked[id]
		}
		m.syncTableRows()
	case ActionMarkAll:
		m.toggleMarkAll()
		m.syncTableRows()
	case ActionSelect:
		if _, ok := m.selectedVM(); ok {
			m.detailOn = !m.detailOn
			m.applyTableLayout()
		}
	case ActionBack:
		if m.detailOn {
			m.detailOn = false
			m.applyTableLayout()
		}
	case ActionJumpTop:
		m.tbl.GotoTop()
		m.syncTableRows()
	case ActionJumpBottom:
		m.tbl.GotoBottom()
		m.syncTableRows()
	case ActionConnect:
		return *m, nil, appIntent{kind: intentConnect, targets: m.selectedVMs()}
	case ActionStart:
		return *m, nil, appIntent{kind: intentStart, targets: m.selectedVMs()}
	case ActionStop:
		return *m, nil, appIntent{kind: intentStop, targets: m.selectedVMs()}
	case ActionFilterProvider:
		providersIds := append(m.svc.GetCloudProvidersIDs(), core.CloudProviderID(""))
		m.fProvider = cycle(m.fProvider, providersIds...)
		m.recompute()
	case ActionFilterState:
		states := []core.VMState{core.StateRunning, core.StateStopped, core.StateStarting, core.StateStopping, core.StateUnknown, ""}
		m.fState = cycle(m.fState, states...)
		m.recompute()
	case ActionFilterRegion:
		m.cycleRegion()
		m.recompute()
	case ActionClearFilters:
		m.clearFilters()
		m.recompute()
	case ActionSortKey:
		m.sortK = (m.sortK + 1) % 6
		m.recompute()
	case ActionSortDir:
		m.sortAsc = !m.sortAsc
		m.recompute()
	case ActionRefresh:
		return *m, nil, appIntent{kind: intentRefresh}
	}
	return *m, nil, appIntent{}
}

func cloudProviderGlyph(cp core.CloudProviderID, selected bool) string {
	if !selected {
		return lipgloss.NewStyle().Foreground(ProviderColor(cp)).Render(string(cp))
	}
	return string(cp)
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

// SetSize stores the available area and resizes the table (and the detail
// panel, if docked).
func (m *inventoryModel) SetSize(w, h int) {
	m.width, m.height = w, h
	m.applyTableLayout()
}

// tableWidth is the full width normally, or the width remaining once the
// docked detail panel's column is subtracted.
func (m *inventoryModel) tableWidth() int {
	if !m.detailOn {
		return m.width
	}
	w := m.width - detailPanelWidth
	if w < 20 {
		w = 20
	}
	return w
}

// applyTableLayout resizes the table for the current width/height and swaps
// its column set (full vs. narrowed) to match whether the detail panel is
// docked — called on both SetSize and detailOn toggles.
func (m *inventoryModel) applyTableLayout() {
	h := m.height
	if h < 1 {
		h = 1
	}
	w := m.tableWidth()
	if m.detailOn {
		m.tbl.SetColumns(narrowTableColumns(w))
	} else {
		m.tbl.SetColumns(fullTableColumns(w))
	}
	m.tbl.SetWidth(w)
	m.tbl.SetHeight(h)
	m.syncTableRows()
}

func (m *inventoryModel) View() string {
	if len(m.filteredVM) == 0 {
		msg := m.styles.Dim.Render("no vms match filters — press u to refresh")
		if len(m.allVM) == 0 {
			msg = m.styles.Dim.Render("no vms loaded — sign in and press u or R to sync providers")
		}
		return lipgloss.Place(m.width, m.height, lipgloss.Left, lipgloss.Top, msg)
	}
	tableView := lipgloss.NewStyle().Width(m.tableWidth()).Height(m.height).Render(m.tbl.View())
	if vm, ok := m.selectedVM(); m.detailOn && ok {
		return lipgloss.JoinHorizontal(lipgloss.Top, tableView, m.detailPanelView(vm))
	}
	return tableView
}

// selectedVM returns the VM under the table cursor, if any.
func (m *inventoryModel) selectedVM() (core.VM, bool) {
	c := m.tbl.Cursor()
	if c < 0 || c >= len(m.filteredVM) {
		return core.VM{}, false
	}
	return m.filteredVM[c], true
}

// detailPanelView renders the VM detail as a side panel docked to the right
// of the table (see App.appView / inventoryModel.View), scrolled via a
// viewport so long tag lists never clip. The mockup's permission
// (canConnect) and "recent activity" blocks are dropped: core.VM has no
// permission field and there's no real audit data to source activity from.
func (m *inventoryModel) detailPanelView(vm core.VM) string {
	header := lipgloss.JoinVertical(lipgloss.Left,
		m.styles.Text.Bold(true).Render(vm.Name)+"  "+stateGlyph(m.styles, vm.State, false),
		m.styles.Dim.Render(vm.ID),
	)

	fields := []string{
		m.styles.Text.Render("name:    ") + m.styles.Accent.Render(vm.Name),
		m.styles.Text.Render("id:      ") + m.styles.Dim.Render(vm.ID),
		m.styles.Text.Render("provider:") + " " + m.styles.Text.Render(string(vm.Provider)),
		m.styles.Text.Render("region:  ") + m.styles.Text.Render(string(vm.Region)),
		m.styles.Text.Render("type:    ") + m.styles.Text.Render(string(vm.Type)),
		m.styles.Text.Render("state:   ") + stateGlyph(m.styles, vm.State, false),
		m.styles.Text.Render("ip:      ") + m.styles.Cyan.Render(string(vm.PrivateIP)),
		m.styles.Text.Render("method:  ") + m.styles.Text.Render("unknown (not implemented yet)"),
		m.styles.Text.Render("tags:    ") + m.styles.Dim.Render(formatTags(vm.Tags)),
	}
	footer := m.styles.Dim.Render("esc/⏎ close")

	innerW := detailPanelWidth - 2 // SidePanel's horizontal padding
	bodyH := m.height - lipgloss.Height(header) - lipgloss.Height(footer) - 2
	if bodyH < 1 {
		bodyH = 1
	}
	m.detailVP.SetWidth(innerW)
	m.detailVP.SetHeight(bodyH)
	m.detailVP.SetContent(strings.Join(fields, "\n"))

	body := lipgloss.JoinVertical(lipgloss.Left, header, "", m.detailVP.View(), footer)
	return m.styles.SidePanel.Width(detailPanelWidth).Height(m.height).Render(body)
}
