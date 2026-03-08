package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

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

func (m *model) vpnConfigView() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("VPN Configuration") + "\n\n")
	b.WriteString(dimStyle.Render("  Enter WireGuard configuration path:") + "\n\n")
	b.WriteString("  " + m.vpnInput.View() + "\n\n")
	if m.err != nil {
		b.WriteString("  " + errorStyle.Render("✗ "+m.err.Error()) + "\n\n")
	}
	b.WriteString(helpStyle.Render("  Enter: save · Esc: cancel"))
	return m.renderFullScreen(b.String())
}
