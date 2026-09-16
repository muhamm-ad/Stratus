package tui

import (
	"fmt"
	"sort"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/table"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/muhamm-ad/stratus/internal/core"
	"github.com/muhamm-ad/stratus/internal/query"
	"github.com/muhamm-ad/stratus/internal/service"
)

// detailSidebarWidth is a terminal-column width for the right VM-details
// drawer. Separate from sidebarWidth (notifications / logs / settings).
const detailSidebarWidth = 48

const detailLabelWidth = 11

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

	detailOn     bool
	detailScroll int
	detailVMID   string
}

func newInventoryModel(svc *service.Service, s Styles) inventoryModel {
	t := table.New(
		table.WithColumns(fullTableColumns(80, "○")),
		table.WithKeyMap(vimTableKeyMap()),
		table.WithFocused(true),
		table.WithStyles(setTableStyles(s)),
	)
	return inventoryModel{
		svc:    svc,
		styles: s,
		marked: map[string]bool{},
		tbl:    t,
		input:  newQueryInput(s),
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
// terminal; PROVIDER/REGION/TYPE/STATE stay fixed on the right. The details
// drawer overlays this table — it does not shrink the column set.
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
	if _, ok := m.selectedVM(); !ok {
		m.detailOn = false
	}
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
	m.syncDetailVM()
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
	return table.Row{check, name, cloudProvider, string(vm.Region), string(vm.Type), state}
}

func (m *inventoryModel) Update(msg tea.KeyPressMsg, k KeyMap, s Styles) (inventoryModel, tea.Cmd, appIntent) {
	m.SetStyles(s)
	if m.input.Focused() {
		return m.updateQuery(msg, k)
	}
	if m.detailOn {
		switch k.Nav.Match(msg) {
		case ActionPageUp:
			m.detailScrollBy(-m.detailPageSize())
			return *m, nil, appIntent{}
		case ActionPageDown:
			m.detailScrollBy(m.detailPageSize())
			return *m, nil, appIntent{}
		}
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
	// Movement stays on the table keymap so j/k match Nav without a second
	// dispatch table. Mouse later calls Handle directly.
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
			if m.detailOn {
				m.detailScroll = 0
			}
		}
	case ActionBack:
		if m.detailOn {
			m.detailOn = false
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
		m.detailOn = false
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

// SetSize stores the available area and resizes the table.
func (m *inventoryModel) SetSize(w, h int) {
	m.width, m.height = w, h
	m.applyTableLayout()
	m.detailScrollBy(0)
}

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
	w := max(1, m.width)
	m.tbl.SetColumns(fullTableColumns(w, m.checkHeaderTitle()))
	m.tbl.SetWidth(w)
	// Reserve one row for the inset header rule drawn in tableView.
	m.tbl.SetHeight(max(1, h-headerRuleHeight))
	m.syncTableRows()
}

func (m *inventoryModel) View() string {
	main := m.mainView()
	if vm, ok := m.selectedVM(); m.detailOn && ok {
		return m.composeDetailSidebar(main, vm)
	}
	return main
}

func (m inventoryModel) mainView() string {
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
		body = m.tableView(m.width, restH)
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

func (m *inventoryModel) syncDetailVM() {
	id := ""
	if vm, ok := m.selectedVM(); ok {
		id = vm.ID
	}
	if id != m.detailVMID {
		m.detailVMID = id
		m.detailScroll = 0
	}
}

// composeDetailSidebar draws the VM-details drawer over the right edge of
// inventory. It is a different overlay from the right-hand notifications /
// logs / settings sidebar: this one is inventory-only and has no tabs.
func (m inventoryModel) composeDetailSidebar(background string, vm core.VM) string {
	w := max(1, m.width)
	h := max(1, m.height)
	blurred := blurContent(padBlock(background, w, h), m.styles.Dim)
	panel := opaqueOverlay(m.detailSidebarView(vm, h), m.styles.th.Bg)
	px, py := sidebarOrigin(panel, w)
	comp := lipgloss.NewCompositor(
		lipgloss.NewLayer(blurred),
		lipgloss.NewLayer(panel).X(px).Y(py).Z(1),
	)
	return lipgloss.NewCanvas(w, h).Compose(comp).Render()
}

func (m inventoryModel) detailSidebarView(vm core.VM, height int) string {
	s := m.styles
	innerW := detailSidebarInnerWidth(s)
	innerH := max(1, height-s.DetailSidebar.GetVerticalFrameSize())

	header := m.detailHeader(vm, innerW)
	footer := m.detailFooter(innerW)
	listH := max(1, innerH-blockHeight(header)-blockHeight(footer))

	body := m.detailBody(vm, innerW)
	bodyLines := strings.Split(body, "\n")
	if body == "" {
		bodyLines = nil
	}
	n := len(bodyLines)
	scroll := max(0, min(m.detailScroll, max(0, n-listH)))
	if n > listH {
		end := min(scroll+listH, n)
		body = joinScrollbar(strings.Join(bodyLines[scroll:end], "\n"), innerW, listH, n, scroll, s)
	}

	inner := stackTopMidBottom(header, body, footer, innerW, innerH)
	return boxNoWrap(s.DetailSidebar.PaddingBottom(0).MarginBottom(0), inner, detailSidebarWidth, height)
}

func (m inventoryModel) detailHeader(vm core.VM, innerW int) string {
	s := m.styles
	title := s.Text.Bold(true).Render("details")
	name := s.Text.Bold(true).Render(dash(vm.Name)) + "  " + stateGlyph(s, vm.State, false)
	id := s.Dim.Render(dash(vm.ID))
	return lipgloss.JoinVertical(lipgloss.Left,
		title,
		"",
		clipLine(name, innerW),
		clipLine(id, innerW),
	)
}

func (m inventoryModel) detailFooter(innerW int) string {
	hint := "esc/⏎ close"
	if m.detailBodyOverflows() {
		hint = "pgup/pgdn scroll · " + hint
	}
	return m.styles.Dim.Render(clipLine(hint, innerW))
}

func (m inventoryModel) detailBody(vm core.VM, innerW int) string {
	s := m.styles
	section := func(title string) string { return s.SectionHead.Render(title) }
	kv := func(label, value string, vs lipgloss.Style) string {
		return detailField(s, label, value, vs, innerW)
	}
	launch := "—"
	if !vm.LaunchTime.IsZero() {
		launch = vm.LaunchTime.Local().Format("2006-01-02 15:04")
	}
	provider := dash(string(vm.Provider))
	providerStyle := lipgloss.NewStyle().Foreground(ProviderColor(vm.Provider))

	rows := []string{
		section("INSTANCE"),
		kv("name", dash(vm.Name), s.Text),
		kv("id", dash(vm.ID), s.Dim),
		kv("provider", provider, providerStyle),
		kv("region", dash(string(vm.Region)), s.Text),
		kv("type", dash(string(vm.Type)), s.Text),
		kv("platform", dash(string(vm.Platform)), s.Text),
		detailFieldRaw(s, "state", stateGlyph(s, vm.State, false), innerW),
		kv("launched", launch, s.Dim),
		"",
		section("NETWORK"),
		kv("private ip", dash(string(vm.PrivateIP)), s.Cyan),
		kv("public ip", dash(string(vm.PublicIP)), s.Cyan),
		"",
		section("ACCESS"),
		kv("user", dash(string(vm.OSUser)), s.Text),
		kv("method", "not implemented", s.Dim),
		"",
		section("TAGS"),
		kv("tags", formatTags(vm.Tags), s.Dim),
	}
	return strings.Join(rows, "\n")
}

func detailField(s Styles, label, value string, vs lipgloss.Style, innerW int) string {
	l := s.Dim.Width(detailLabelWidth).Render(label)
	vw := max(4, innerW-detailLabelWidth-1)
	wrapped := lipgloss.Wrap(value, vw, "")
	lines := strings.Split(wrapped, "\n")
	out := make([]string, len(lines))
	indent := strings.Repeat(" ", detailLabelWidth+1)
	for i, line := range lines {
		styled := vs.Render(line)
		if i == 0 {
			out[i] = l + " " + styled
			continue
		}
		out[i] = indent + styled
	}
	return strings.Join(out, "\n")
}

func detailFieldRaw(s Styles, label, renderedValue string, innerW int) string {
	l := s.Dim.Width(detailLabelWidth).Render(label)
	vw := max(4, innerW-detailLabelWidth-1)
	wrapped := lipgloss.Wrap(renderedValue, vw, "")
	lines := strings.Split(wrapped, "\n")
	out := make([]string, len(lines))
	indent := strings.Repeat(" ", detailLabelWidth+1)
	for i, line := range lines {
		if i == 0 {
			out[i] = l + " " + line
			continue
		}
		out[i] = indent + line
	}
	return strings.Join(out, "\n")
}

func detailSidebarInnerWidth(s Styles) int {
	return max(8, detailSidebarWidth-s.DetailSidebar.GetHorizontalFrameSize())
}

func (m inventoryModel) detailListHeight() int {
	innerH := max(1, max(1, m.height)-m.styles.DetailSidebar.GetVerticalFrameSize())
	headerH := blockHeight(m.detailHeader(core.VM{}, detailSidebarInnerWidth(m.styles)))
	footerH := 1
	return max(1, innerH-headerH-footerH)
}

func (m inventoryModel) detailBodyOverflows() bool {
	vm, ok := m.selectedVM()
	if !ok {
		return false
	}
	n := lipgloss.Height(m.detailBody(vm, detailSidebarInnerWidth(m.styles)))
	return n > m.detailListHeight()
}

func (m inventoryModel) detailMaxScroll() int {
	vm, ok := m.selectedVM()
	if !ok {
		return 0
	}
	n := lipgloss.Height(m.detailBody(vm, detailSidebarInnerWidth(m.styles)))
	return max(0, n-m.detailListHeight())
}

func (m inventoryModel) detailPageSize() int {
	return m.detailListHeight()
}

func (m *inventoryModel) detailScrollBy(delta int) {
	m.detailScroll += delta
	if m.detailScroll < 0 {
		m.detailScroll = 0
	}
	maxScroll := m.detailMaxScroll()
	if m.detailScroll > maxScroll {
		m.detailScroll = maxScroll
	}
}
