package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/akunbeben/ssht/internal/config"
)

func newVPNInput() textinput.Model {
	ti := textinput.New()
	ti.Placeholder = "/path/to/your/wg0.conf"
	ti.CharLimit = 256
	ti.Focus()
	ti.PromptStyle = focusedInputStyle
	ti.TextStyle = focusedInputStyle
	ti.Cursor.Style = cursorStyle
	return ti
}

func (m *model) handleVPNConfigKey(key string, msg tea.Msg) (tea.Model, tea.Cmd) {
	switch key {
	case "esc":
		m.mode = modeList
		return m, nil
	case "enter":
		path := strings.TrimSpace(m.vpnInput.Value())
		if path == "" {
			m.err = fmt.Errorf("config path is required")
			return m, nil
		}

		m.profile.VPN = &config.VPNConf{
			Type:     "wireguard",
			ConfPath: path,
			AutoUp:   true,
		}
		m.syncServers()
		m.mode = modeList
		cmd := m.setStatus("✓ VPN configured", successStyle)
		return m, cmd
	}

	var cmd tea.Cmd
	m.vpnInput, cmd = m.vpnInput.Update(msg)
	return m, cmd
}

func (m *model) vpnConfigView(width, height int, helperWrapped bool) string {
	var body strings.Builder
	body.WriteString(titleStyle.Render("VPN Configuration") + "\n\n")
	body.WriteString(dimStyle.Copy().Width(width).Render("Enter WireGuard configuration path:") + "\n\n")
	body.WriteString("  " + m.vpnInput.View() + "\n\n")
	if m.err != nil {
		errStyleWrap := errorStyle.Copy().Width(width)
		errMsg := m.err.Error()
		if !helperWrapped {
			errMsg = truncate(errMsg, width)
			errStyleWrap = errStyleWrap.MaxHeight(1)
		}
		body.WriteString(errStyleWrap.Render("✗ "+errMsg) + "\n\n")
	}

	help := "Enter: save · Esc: cancel"
	helpStyleWrap := helpStyle.Copy().Width(width)
	if !helperWrapped {
		help = truncate(help, width)
		helpStyleWrap = helpStyleWrap.MaxHeight(1)
	}
	renderedHelp := helpStyleWrap.Render(help)

	bodyContent := body.String()
	gap := height - lipgloss.Height(bodyContent) - lipgloss.Height(renderedHelp)
	if gap > 0 {
		return bodyContent + strings.Repeat("\n", gap) + renderedHelp
	}
	return bodyContent + "\n\n" + renderedHelp
}
