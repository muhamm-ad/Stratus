package main

// Package tui is the full-screen, k9s-style terminal interface for Stratus,
// built with Bubble Tea. It is a thin PRESENTATION layer: every operation is
// delegated to *service.Service — the exact same logic the Wails UI uses. The
// TUI never talks to providers or Entra directly.

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/muhamm-ad/stratus/core"
	"github.com/muhamm-ad/stratus/service"
)

type view int

const (
	viewProviders view = iota
	viewInstances
)

type connState int

const (
	stIdle connState = iota
	stConnecting
	stConnected
	stFailed
)

type provState struct {
	st  connState
	err string
}

// Model is the Bubble Tea model.
type Model struct {
	svc  *service.Service
	view view

	spin      spinner.Model
	provTable table.Model
	instTable table.Model

	prov     map[core.ProviderID]provState
	selected core.ProviderID
	authed   bool

	busy     string // non-empty → a background op is running (label shown)
	msg      string // transient status line
	errMsg   string
	warnings []string

	w, h int
}

// New builds the TUI model from the shared service and any bootstrap warnings.
func New(svc *service.Service, warnings []error) Model {
	sp := spinner.New()
	sp.Spinner = spinner.Dot

	prov := make(map[core.ProviderID]provState)
	for _, id := range svc.Providers() {
		prov[id] = provState{st: stIdle}
	}

	pt := table.New(
		table.WithColumns([]table.Column{{Title: "PROVIDER", Width: 14}, {Title: "STATUS", Width: 26}}),
		table.WithFocused(true),
		table.WithHeight(10),
	)
	pt.SetStyles(tableStyles())

	it := table.New(
		table.WithColumns([]table.Column{
			{Title: "NAME", Width: 22}, {Title: "ID", Width: 20}, {Title: "STATE", Width: 10},
			{Title: "PLATFORM", Width: 10}, {Title: "REGION", Width: 14}, {Title: "PRIVATE IP", Width: 16},
		}),
		table.WithHeight(10),
	)
	it.SetStyles(tableStyles())

	m := Model{
		svc: svc, view: viewProviders, spin: sp,
		provTable: pt, instTable: it, prov: prov,
		authed:   svc.IsAuthenticated(),
		warnings: errsToStrings(warnings),
	}
	m.provTable.SetRows(m.providerRows())
	return m
}

func (m Model) Init() tea.Cmd { return m.spin.Tick }

// ── messages produced by background commands ────────────────────────────────

type loginDoneMsg struct{ err error }
type connectDoneMsg struct {
	id  core.ProviderID
	err error
}
type instancesMsg struct {
	id    core.ProviderID
	items []core.Instance
	err   error
}

// ── background commands (run the shared service, off the UI loop) ────────────

func loginCmd(svc *service.Service) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		return loginDoneMsg{err: svc.Login(ctx)}
	}
}

func connectCmd(svc *service.Service, id core.ProviderID) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		return connectDoneMsg{id: id, err: svc.Connect(ctx, id)}
	}
}

func listInstancesCmd(svc *service.Service, id core.ProviderID) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()
		items, err := svc.ListInstances(ctx, id, "")
		return instancesMsg{id: id, items: items, err: err}
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.w, m.h = msg.Width, msg.Height
		bodyH := m.h - 7
		if bodyH < 3 {
			bodyH = 3
		}
		m.provTable.SetHeight(bodyH)
		m.instTable.SetHeight(bodyH)
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit

		case "l":
			if m.busy == "" {
				m.busy = "Signing in to Microsoft Entra ID (browser)…"
				m.errMsg, m.msg = "", ""
				return m, tea.Batch(m.spin.Tick, loginCmd(m.svc))
			}

		case "r":
			m.authed = m.svc.IsAuthenticated()
			m.provTable.SetRows(m.providerRows())
			return m, nil

		case "esc":
			if m.view == viewInstances {
				m.view = viewProviders
				return m, nil
			}

		case "enter":
			if m.busy != "" || m.view != viewProviders {
				return m, nil
			}
			id := m.selectedProvider()
			if id == "" {
				return m, nil
			}
			if !m.authed {
				m.msg = "Sign in first — press l"
				return m, nil
			}
			if m.prov[id].st == stConnected {
				m.view, m.selected = viewInstances, id
				m.busy, m.msg, m.errMsg = "Listing instances…", "", ""
				m.instTable.SetRows(nil)
				return m, tea.Batch(m.spin.Tick, listInstancesCmd(m.svc, id))
			}
			ps := m.prov[id]
			ps.st = stConnecting
			m.prov[id] = ps
			m.provTable.SetRows(m.providerRows())
			m.busy, m.msg, m.errMsg = "Exchanging credentials for "+string(id)+"…", "", ""
			return m, tea.Batch(m.spin.Tick, connectCmd(m.svc, id))
		}

		// Forward navigation keys to the active table.
		var cmd tea.Cmd
		if m.view == viewProviders {
			m.provTable, cmd = m.provTable.Update(msg)
		} else {
			m.instTable, cmd = m.instTable.Update(msg)
		}
		return m, cmd

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spin, cmd = m.spin.Update(msg)
		return m, cmd

	case loginDoneMsg:
		m.busy = ""
		if msg.err != nil {
			m.errMsg = "Login failed: " + msg.err.Error()
		} else {
			m.authed = m.svc.IsAuthenticated()
			m.msg = "Signed in. Select a provider and press enter to connect."
		}
		return m, nil

	case connectDoneMsg:
		m.busy = ""
		ps := m.prov[msg.id]
		if msg.err != nil {
			ps.st, ps.err = stFailed, msg.err.Error()
			m.errMsg = string(msg.id) + ": " + msg.err.Error()
		} else {
			ps.st, ps.err = stConnected, ""
			m.msg = string(msg.id) + " connected. Press enter to list its instances."
		}
		m.prov[msg.id] = ps
		m.provTable.SetRows(m.providerRows())
		return m, nil

	case instancesMsg:
		m.busy = ""
		switch {
		case msg.err != nil && errors.Is(msg.err, core.ErrNotImplemented):
			m.msg = "Instance listing isn't implemented yet (Phase 3)."
			m.instTable.SetRows(nil)
		case msg.err != nil:
			m.errMsg = "List failed: " + msg.err.Error()
			m.instTable.SetRows(nil)
		default:
			m.instTable.SetRows(instanceRows(msg.items))
			m.msg = fmt.Sprintf("%d instance(s) on %s.", len(msg.items), msg.id)
		}
		return m, nil
	}
	return m, nil
}

// ── helpers ─────────────────────────────────────────────────────────────────

func (m Model) selectedProvider() core.ProviderID {
	row := m.provTable.SelectedRow()
	if len(row) == 0 {
		return ""
	}
	return core.ProviderID(row[0])
}

func (m Model) providerRows() []table.Row {
	ids := m.svc.Providers()
	rows := make([]table.Row, 0, len(ids))
	for _, id := range ids {
		rows = append(rows, table.Row{string(id), statusLabel(m.prov[id])})
	}
	return rows
}

func instanceRows(items []core.Instance) []table.Row {
	rows := make([]table.Row, 0, len(items))
	for _, in := range items {
		rows = append(rows, table.Row{in.Name, in.ID, in.State, in.Platform, in.Region, in.PrivateIP})
	}
	return rows
}

func statusLabel(ps provState) string {
	switch ps.st {
	case stConnecting:
		return "connecting…"
	case stConnected:
		return "✓ connected"
	case stFailed:
		return "✗ failed"
	default:
		return "—"
	}
}

func errsToStrings(errs []error) []string {
	out := make([]string, 0, len(errs))
	for _, e := range errs {
		out = append(out, e.Error())
	}
	return out
}