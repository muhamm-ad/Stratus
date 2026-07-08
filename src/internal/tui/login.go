package tui

import (
	"fmt"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/muhamm-ad/stratus/core"
	"github.com/muhamm-ad/stratus/service"
)

type loginStep int

const (
	stepSelect loginStep = iota
	stepWaiting
)

type loginModel struct {
	svc        *service.Service
	cursor     int
	step       loginStep
	selected   string
	spinner    spinner.Model
	deviceCode core.DeviceCode
	useDevice  bool // from selected IdP; false → browser opens automatically
	err        error
}

func newLoginModel(svc *service.Service) loginModel {
	sp := spinner.New(spinner.WithSpinner(spinner.Spinner{Frames: SpinnerFrames, FPS: 12}))
	return loginModel{svc: svc, spinner: sp}
}

// Not the Update function from the Model interface.
// Update is called manually when a key is pressed.
func (m loginModel) Update(msg tea.KeyPressMsg, send func(tea.Msg)) (loginModel, tea.Cmd) {
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
			return m, tea.Batch(m.spinner.Tick, loginCmd(m.svc, idpID, send))
		}
	case stepWaiting:
		if msg.String() == "esc" {
			m.step = stepSelect
		}
	}
	return m, nil
}

func (m loginModel) View(s Styles, w, h int) string {
	title := s.Title.Render("S T R A T U S")
	sub := s.Dim.Render("multi-cloud vm gateway")
	rule := s.Dim.Render(lipgloss.NewStyle().Width(40).Render("────────────────────────────────────────"))

	var inner string
	if m.step == stepSelect {
		head := s.SectionHead.Render("SELECT IDENTITY PROVIDER")
		var rows []string
		idpIDs := m.svc.IdentityProvidersIDs()
		for i, idpID := range idpIDs {
			cur := "  "
			if i == m.cursor {
				cur = s.Accent.Render("▸ ")
			}
			name := s.Text.Bold(true).Render(string(idpID))
			// desc := s.Dim.Render(" — " + idp.Description)
			// line := cur + name + desc
			line := cur + name
			rows = append(rows, line)
		}
		hint := s.Dim.Render("j/k move · ⏎ select")
		if m.err != nil {
			hint = s.Err.Render(m.err.Error()) + "\n" + hint
		}
		inner = head + "\n\n" + lipgloss.JoinVertical(lipgloss.Left, rows...) + "\n\n" + hint
	} else {
		label := s.Accent.Render(m.selected)
		wait := s.Warn.Render(m.spinner.View() + " waiting for authentication…")
		if m.err != nil {
			wait = s.Err.Render(m.err.Error()) + "\n" + wait
		}
		if m.useDevice {
			code := s.CodeBox.Render(m.deviceCode.UserCode)
			url := s.Cyan.Render(m.deviceCode.VerificationURI)
			inner = fmt.Sprintf("%s\n\nFirst, copy your one-time code:\n\n%s\n\nThen enter it at %s\n\n%s\n\n%s",
				label, code, url, wait, s.Dim.Render("esc cancel"))
		} else {
			inner = fmt.Sprintf("%s\n\n%s\n\n%s\n\n%s",
				label,
				s.Text.Render("Your browser will open automatically."),
				s.Dim.Render("Complete sign-in there, then return here."),
				wait+"\n\n"+s.Dim.Render("esc cancel"))
		}
	}

	// box := s.Box.Render(title + "  " + s.Dim.Render("v0.4.0") + "\n" + sub + "\n" + rule + "\n\n" + inner)
	box := s.Box.Render(title + "\n" + sub + "\n" + rule + "\n\n" + inner)
	caption := s.Dim.Render("sign in once · connect aws, azure & gcp")
	block := lipgloss.JoinVertical(lipgloss.Center, box, "", caption)
	return lipgloss.Place(w, h, lipgloss.Center, lipgloss.Center, block)
}
