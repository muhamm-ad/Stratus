package tui

import (
	"fmt"
	"sort"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/table"
	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/muhamm-ad/stratus/internal/core"
	"github.com/muhamm-ad/stratus/internal/query"
	"github.com/muhamm-ad/stratus/internal/service"
)

// detailPanelWidth is a terminal-column width, not the mockup's 380 CSS
// pixels — chosen to read well at typical 80-160 col terminal widths.
const detailPanelWidth = 42

// The query bar language is defined in docs/inventory-query.md and parsed by
// internal/query — shortcuts only rewrite the string, they do not filter.
const queryPlaceholder = "search, or provider=aws · name+ to sort, ? for filters help"

type inventoryModel struct {
	svc           *service.Service
	styles        Styles
	allVM         []core.VM
	filteredVM    []core.VM
	marked        map[string]bool
	tbl           table.Model
	input         textinput.Model
	width, height int

	detailOn bool
	detailVP viewport.Model
}

func newInventoryModel(svc *service.Service, s Styles) inventoryModel {
	t := table.New(
		table.WithColumns(fullTableColumns(80, "○")),
		table.WithKeyMap(vimTableKeyMap()),
		table.WithFocused(true),
		table.WithStyles(setTableStyles(s)),
	)
	return inventoryModel{
		svc:      svc,
		styles:   s,
		marked:   map[string]bool{},
		tbl:      t,
		input:    newQueryInput(s),
		detailVP: viewport.New(),
	}
}

func newQueryInput(s Styles) textinput.Model {
	ti := textinput.New()
	ti.Prompt = "/ "
	ti.Placeholder = queryPlaceholder
	ti.SetStyles(queryInputStyles(s))
	ti.SetVirtualCursor(true)
	return ti
}

func queryInputStyles(s Styles) textinput.Styles {
	st := textinput.DefaultDarkStyles()
	st.Focused.Prompt = s.Accent
	st.Focused.Text = s.Text
	st.Focused.Placeholder = s.Dim
	st.Blurred.Prompt = s.Dim
	st.Blurred.Text = s.Text
	st.Blurred.Placeholder = s.Dim
	st.Cursor.Color = s.th.Accent
	st.Cursor.Shape = tea.CursorBar
	st.Cursor.Blink = true
	return st
}

// tableCellPad is the horizontal padding from setTableStyles (Padding(0, 1)
// on Header and Cell). Bubbles sizes columns by content width only, so this
// extra must be subtracted or the table overflows the terminal.
const tableCellPad = 2

// headerRuleHeight is the inset ─ drawn under the header (not a cell border).
const headerRuleHeight = 1
const headerRulePad = 1

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
func fullTableColumns(w int, check string) []table.Column {
	fixed := invColCheck + invColProvider + invColRegion + invColType + invColState
	return []table.Column{
		{Title: check, Width: invColCheck}, // select-all ○ / ●
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
func narrowTableColumns(w int, check string) []table.Column {
	fixed := invColCheck + invColProvider + invColState
	return []table.Column{
		{Title: check, Width: invColCheck}, // select-all ○ / ●
		{Title: "NAME", Width: flexNameWidth(w, 4, fixed)},
		{Title: "PROVIDER", Width: invColProvider},
		{Title: "STATE", Width: invColState},
	}
}

// vimTableKeyMap mirrors the app's own navigation keys (k/j/g/G) but
// strips PageDown/HalfPageDown's default "f"/"space" bindings — those are
// already claimed by FilterState and other app keys.
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
	style.Header = s.SectionHead.Padding(0, 1)
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
	m.input.SetStyles(queryInputStyles(s))
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
	m.filteredVM = query.Apply(m.allVM, query.Parse(m.input.Value()))
	m.syncTableRows()
}

// syncTableRows rebuilds the table's rows from filteredVM. table.Model.SetRows
// clamps the cursor to the new row count itself, replacing the old manual
// syncRows() cursor-clamp.
func (m *inventoryModel) syncTableRows() {
	m.syncHeaderCheck()
	rows := make([]table.Row, len(m.filteredVM))
	for i, vm := range m.filteredVM {
		rows[i] = m.vmToRow(vm, i == m.tbl.Cursor())
	}
	m.tbl.SetRows(rows)
}

// checkHeaderTitle is ○ unless every visible VM is marked, then ●.
func (m *inventoryModel) checkHeaderTitle() string {
	if len(m.filteredVM) == 0 {
		return "○"
	}
	for _, vm := range m.filteredVM {
		if !m.marked[vm.ID] {
			return "○"
		}
	}
	return "●"
}

func (m *inventoryModel) syncHeaderCheck() {
	cols := m.tbl.Columns()
	if len(cols) == 0 {
		return
	}
	title := m.checkHeaderTitle()
	if cols[0].Title == title {
		return
	}
	cols[0].Title = title
	m.tbl.SetColumns(cols)
}

func (m inventoryModel) vmToRow(vm core.VM, selected bool) table.Row {
	check := "○"
	if m.marked[vm.ID] {
		check = "●"
	}
	if !selected {
		if m.marked[vm.ID] {
			check = m.styles.Accent.Render(check)
		} else {
			check = m.styles.Dim.Render(check)
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
	if m.input.Focused() {
		return m.updateQuery(msg, k)
	}
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

func (m *inventoryModel) updateQuery(msg tea.KeyPressMsg, k KeyMap) (inventoryModel, tea.Cmd, appIntent) {
	if k.Inventory.Match(msg) == ActionQueryHelp {
		return *m, nil, appIntent{kind: intentQueryHelp}
	}
	switch k.Nav.Match(msg) {
	case ActionBack, ActionSelect:
		m.input.Blur()
		return *m, nil, appIntent{}
	case ActionMoveUp, ActionMoveDown, ActionPageUp, ActionPageDown:
		m.tbl, _ = m.tbl.Update(msg)
		m.syncTableRows()
		return *m, nil, appIntent{}
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	m.recompute()
	return *m, cmd, appIntent{}
}

// QueryFocused reports whether keystrokes belong to the query bar.
func (m inventoryModel) QueryFocused() bool { return m.input.Focused() }

// QueryString is the raw query bar value.
func (m inventoryModel) QueryString() string { return m.input.Value() }

// HandleMsg forwards non-key messages (cursor blink) to the query bar.
func (m *inventoryModel) HandleMsg(msg tea.Msg) tea.Cmd {
	if !m.input.Focused() {
		return nil
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return cmd
}

func (m *inventoryModel) parsed() query.Query {
	return query.Parse(m.input.Value())
}

func (m *inventoryModel) setQuery(q query.Query) {
	m.input.SetValue(q.String())
	m.input.CursorEnd()
	m.recompute()
}

func (m *inventoryModel) applyProvider(id core.CloudProviderID) {
	if id == "" {
		m.setQuery(m.parsed().ClearFilter(query.FieldProvider))
		return
	}
	m.setQuery(m.parsed().SetFilter(query.FieldProvider, string(id)))
}

func (m *inventoryModel) applyState(st core.VMState) {
	if st == "" {
		m.setQuery(m.parsed().ClearFilter(query.FieldState))
		return
	}
	m.setQuery(m.parsed().SetFilter(query.FieldState, string(st)))
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
		ids := m.svc.GetCloudProvidersIDs()
		vals := make([]string, 0, len(ids)+1)
		for _, id := range ids {
			vals = append(vals, string(id))
		}
		vals = append(vals, "")
		m.setQuery(m.parsed().CycleFilter(query.FieldProvider, vals))
	case ActionFilterState:
		m.setQuery(m.parsed().CycleFilter(query.FieldState, []string{
			string(core.StateRunning),
			string(core.StateStopped),
			string(core.StateStarting),
			string(core.StateStopping),
			string(core.StateUnknown),
			"",
		}))
	case ActionFilterRegion:
		m.cycleRegion()
	case ActionClearFilters:
		m.clearFilters()
	case ActionSortKey:
		m.setQuery(m.parsed().CycleSort())
	case ActionSortDir:
		m.setQuery(m.parsed().ToggleSortDir())
	case ActionSearch:
		return *m, m.input.Focus(), appIntent{}
	case ActionQueryHelp:
		return *m, nil, appIntent{kind: intentQueryHelp}
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
		label = "running"
	case core.StateStopped:
		label = "stopped"
	case core.StateStarting:
		label = "starting"
	case core.StateStopping:
		label = "stopping"
	default:
		label = "unknown"
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
	vals := make([]string, 0, len(regions)+1)
	for _, r := range regions {
		vals = append(vals, string(r))
	}
	vals = append(vals, "")
	m.setQuery(m.parsed().CycleFilter(query.FieldRegion, vals))
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
	m.input.Reset()
	m.recompute()
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
func (m *inventoryModel) queryBarHeight() int {
	return 1 + m.styles.FilterLine.GetVerticalFrameSize()
}

func (m *inventoryModel) bodyHeight() int {
	h := m.height - m.queryBarHeight()
	if h < 1 {
		return 1
	}
	return h
}

func (m *inventoryModel) applyTableLayout() {
	h := m.bodyHeight()
	w := m.tableWidth()
	check := m.checkHeaderTitle()
	if m.detailOn {
		m.tbl.SetColumns(narrowTableColumns(w, check))
	} else {
		m.tbl.SetColumns(fullTableColumns(w, check))
	}
	m.tbl.SetWidth(w)
	// Reserve one row for the inset header rule drawn in tableView.
	m.tbl.SetHeight(max(1, h-headerRuleHeight))
	m.syncTableRows()
}

func (m *inventoryModel) View() string {
	bar := m.queryBarView()
	barH := m.queryBarHeight()
	if m.height < barH+1 {
		return bar
	}
	restH := m.bodyHeight()
	var body string
	if len(m.filteredVM) == 0 {
		msg := m.styles.Dim.Render("no vms match filters — press u to refresh")
		if len(m.allVM) == 0 {
			msg = m.styles.Dim.Render("no vms loaded — sign in and press u or R to sync providers")
		}
		body = lipgloss.Place(m.width, restH, lipgloss.Left, lipgloss.Top, msg)
	} else {
		tableView := m.tableView(m.tableWidth(), restH)
		if vm, ok := m.selectedVM(); m.detailOn && ok {
			body = lipgloss.JoinHorizontal(lipgloss.Top, tableView, m.detailPanelView(vm, restH))
		} else {
			body = tableView
		}
	}
	return lipgloss.JoinVertical(lipgloss.Top, bar, body)
}

func (m *inventoryModel) queryBarView() string {
	st := m.styles.FilterLine.UnsetForeground()
	innerW := max(1, m.width-st.GetHorizontalFrameSize())
	count := m.styles.Dim.Render(fmt.Sprintf("%d/%d vms", len(m.filteredVM), len(m.allVM)))
	right := count
	if m.input.Focused() {
		right = m.styles.Dim.Render("esc to table") + " · " + count
	}
	promptW := lipgloss.Width(m.input.Prompt)
	m.input.SetWidth(max(1, innerW-lipgloss.Width(right)-promptW))
	return boxNoWrap(st, joinClipRow(m.input.View(), right, innerW), m.width, m.queryBarHeight())
}

func (m *inventoryModel) tableView(w, h int) string {
	raw := m.tbl.View()
	header, body, ok := strings.Cut(raw, "\n")
	if !ok {
		return boxNoWrap(lipgloss.NewStyle(), raw, w, h)
	}
	return boxNoWrap(lipgloss.NewStyle(), header+"\n"+m.headerRule(w)+"\n"+body, w, h)
}

func (m *inventoryModel) headerRule(w int) string {
	inner := w - 2*headerRulePad
	if inner < 1 {
		return m.styles.Dim.Render(strings.Repeat("─", max(0, w)))
	}
	return strings.Repeat(" ", headerRulePad) + m.styles.Dim.Render(strings.Repeat("─", inner))
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
func (m *inventoryModel) detailPanelView(vm core.VM, height int) string {
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
	bodyH := height - lipgloss.Height(header) - lipgloss.Height(footer) - 2
	if bodyH < 1 {
		bodyH = 1
	}
	m.detailVP.SetWidth(innerW)
	m.detailVP.SetHeight(bodyH)
	m.detailVP.SetContent(strings.Join(fields, "\n"))

	body := lipgloss.JoinVertical(lipgloss.Left, header, "", m.detailVP.View(), footer)
	return boxNoWrap(m.styles.SidePanel, body, detailPanelWidth, height)
}
