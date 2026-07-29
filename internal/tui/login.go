package tui

import (
	"fmt"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/muhamm-ad/stratus/internal/core"
	"github.com/muhamm-ad/stratus/internal/service"
)

type loginStep int

const (
	stepSelect loginStep = iota
	stepWaiting
)

type loginModel struct {
	svc        *service.Service
	styles     Styles
	cursor     int
	step       loginStep
	selected   string
	spinner    spinner.Model
	deviceCode core.DeviceCode
	useDevice  bool // from selected IdP; false → browser opens automatically
	err        error
}

func newLoginModel(svc *service.Service, s Styles) loginModel {
	sp := spinner.New(spinner.WithSpinner(spinner.Spinner{Frames: SpinnerFrames, FPS: 12}))
	return loginModel{svc: svc, styles: s, spinner: sp}
}

// Not the Update function from the Model interface.
// Update is called manually when a key is pressed.
func (m *loginModel) Update(msg tea.KeyPressMsg, send func(tea.Msg), s Styles) (loginModel, tea.Cmd) {
	m.styles = s
	switch m.step {
	case stepSelect:
		idpIDs := m.svc.IdentityProvidersIDs()
		switch msg.String() {
		case "j", "down":
			if m.cursor < len(idpIDs)-1 {
				m.cursor++
			}
		case "k", "up":
			if m.cursor > 0 {
				m.cursor--
			}
		case "enter":
			idpID := idpIDs[m.cursor]
			m.selected = string(idpID)
			m.useDevice = m.svc.IdentityUsesDeviceFlow(idpID)
			m.deviceCode = core.DeviceCode{}
			m.err = nil
			m.step = stepWaiting
			return *m, tea.Batch(m.spinner.Tick, loginCmd(m.svc, idpID, send))
		}
	case stepWaiting:
		if msg.String() == "esc" {
			m.step = stepSelect
		}
	}
	return *m, nil
}

func (m *loginModel) View(w, h int) string {
	title := m.styles.Title.Render("S T R A T U S")
	sub := m.styles.Dim.Render("multi-cloud vm gateway")
	rule := m.styles.Dim.Render(lipgloss.NewStyle().Width(40).Render("────────────────────────────────────────"))

	var inner string
	if m.step == stepSelect {
		head := m.styles.SectionHead.Render("SELECT IDENTITY PROVIDER")
		var rows []string
		idpIDs := m.svc.IdentityProvidersIDs()
		for i, idpID := range idpIDs {
			cur := "  "
			if i == m.cursor {
				cur = m.styles.Cursor.Render("▸ ")
			}
			name := m.styles.Text.Bold(true).Render(string(idpID))
			// desc := s.Dim.Render(" — " + idp.Description)
			// line := cur + name + desc
			line := cur + name
			rows = append(rows, line)
		}
		hint := m.styles.Dim.Render("j/k move · ⏎ select")
		if m.err != nil {
			hint = m.styles.Err.Render(m.err.Error()) + "\n" + hint
		}
		inner = head + "\n\n" + lipgloss.JoinVertical(lipgloss.Left, rows...) + "\n\n" + hint
	} else {
		label := m.styles.Accent.Render(m.selected)
		wait := m.styles.Warn.Render(m.spinner.View() + " waiting for authentication…")
		if m.err != nil {
			wait = m.styles.Err.Render(m.err.Error()) + "\n" + wait
		}
		if m.useDevice {
			code := m.styles.CodeBox.Render(m.deviceCode.UserCode)
			url := m.styles.Cyan.Render(m.deviceCode.VerificationURI)
			inner = fmt.Sprintf("%s\n\nFirst, copy your one-time code:\n\n%s\n\nThen enter it at %s\n\n%s\n\n%s",
				label, code, url, wait, m.styles.Dim.Render("esc cancel"))
		} else {
			inner = fmt.Sprintf("%s\n\n%s\n\n%s\n\n%s",
				label,
				m.styles.Text.Render("Your browser will open automatically."),
				m.styles.Dim.Render("Complete sign-in there, then return here."),
				wait+"\n\n"+m.styles.Dim.Render("esc cancel"))
		}
	}

	// box := s.Box.Render(title + "  " + s.Dim.Render("v0.4.0") + "\n" + sub + "\n" + rule + "\n\n" + inner)
	box := m.styles.Box.Render(title + "\n" + sub + "\n" + rule + "\n\n" + inner)
	caption := m.styles.Dim.Render("sign in once · connect aws, azure & gcp")
	block := lipgloss.JoinVertical(lipgloss.Center, box, "", caption)
	return lipgloss.Place(w, h, lipgloss.Center, lipgloss.Center, block)
}
