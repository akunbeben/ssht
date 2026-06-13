package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/akunbeben/ssht/internal/config"
)

func newVPNInputs() (textinput.Model, textinput.Model) {
	typeInput := textinput.New()
	typeInput.Placeholder = "wireguard, shadowsocks, trojan, openvpn"
	typeInput.CharLimit = 64
	typeInput.Focus()
	typeInput.PromptStyle = focusedInputStyle
	typeInput.TextStyle = focusedInputStyle
	typeInput.Cursor.Style = cursorStyle

	confInput := textinput.New()
	confInput.Placeholder = "~/wg0.conf, ss://..., trojan://..., or .ovpn"
	confInput.CharLimit = 512
	confInput.PromptStyle = blurredInputStyle
	confInput.TextStyle = blurredInputStyle
	confInput.Cursor.Style = cursorStyle
	return typeInput, confInput
}

func (m *model) handleVPNConfigKey(key string, msg tea.Msg) (tea.Model, tea.Cmd) {
	switch key {
	case "esc":
		m.mode = modeList
		return m, nil
	case "tab", "down":
		m.focusVPNField((m.vpnFocus + 1) % 2)
		return m, nil
	case "shift+tab", "up":
		m.focusVPNField((m.vpnFocus + 1) % 2)
		return m, nil
	case "enter":
		if m.vpnFocus == 0 {
			m.focusVPNField(1)
			return m, nil
		}
		vpnType := strings.TrimSpace(m.vpnTypeInput.Value())
		path := strings.TrimSpace(m.vpnInput.Value())
		if path == "" {
			m.err = fmt.Errorf("config path is required")
			return m, nil
		}
		if vpnType == "" {
			vpnType = "wireguard"
		}

		m.profile.VPN = &config.VPNConf{
			Type:     vpnType,
			ConfPath: path,
			AutoUp:   true,
		}
		m.syncServers()
		m.mode = modeList
		cmd := m.setStatus("✓ VPN configured", successStyle)
		return m, cmd
	}

	var cmd tea.Cmd
	if m.vpnFocus == 0 {
		m.vpnTypeInput, cmd = m.vpnTypeInput.Update(msg)
	} else {
		m.vpnInput, cmd = m.vpnInput.Update(msg)
	}
	return m, cmd
}

func (m *model) focusVPNField(idx int) {
	m.vpnFocus = idx
	if m.vpnFocus == 0 {
		m.vpnTypeInput.Focus()
		m.vpnTypeInput.PromptStyle = focusedInputStyle
		m.vpnTypeInput.TextStyle = focusedInputStyle
		m.vpnInput.Blur()
		m.vpnInput.PromptStyle = blurredInputStyle
		m.vpnInput.TextStyle = blurredInputStyle
		return
	}
	m.vpnInput.Focus()
	m.vpnInput.PromptStyle = focusedInputStyle
	m.vpnInput.TextStyle = focusedInputStyle
	m.vpnTypeInput.Blur()
	m.vpnTypeInput.PromptStyle = blurredInputStyle
	m.vpnTypeInput.TextStyle = blurredInputStyle
}

func (m *model) vpnConfigView(width, height int, helperWrapped bool) string {
	var body strings.Builder
	body.WriteString(titleStyle.Render("Profile VPN") + "\n")
	body.WriteString(dimStyle.Copy().Width(width).Render("This VPN applies to servers in this profile unless a server has its own override. SSH traffic is tunneled per connection; system traffic is unaffected.") + "\n\n")

	typeCursor := inactiveCursor
	confCursor := inactiveCursor
	if m.vpnFocus == 0 {
		typeCursor = focusedInputStyle.Render(activeCursor)
	} else {
		confCursor = focusedInputStyle.Render(activeCursor)
	}
	body.WriteString(typeCursor + formLabelStyle.Render("Type") + m.vpnTypeInput.View() + "\n")
	body.WriteString(confCursor + formLabelStyle.Render("Config") + m.vpnInput.View() + "\n")
	body.WriteString(dimStyle.Copy().Width(width).Render("Leave type blank for wireguard. Supported types depend on the configured dialer.") + "\n\n")
	if m.err != nil {
		errStyleWrap := errorStyle.Copy().Width(width)
		errMsg := m.err.Error()
		if !helperWrapped {
			errMsg = truncate(errMsg, width)
			errStyleWrap = errStyleWrap.MaxHeight(1)
		}
		body.WriteString(errStyleWrap.Render("✗ "+errMsg) + "\n\n")
	}

	help := "Tab/Up/Down: navigate  Enter: next/save  Esc: cancel"
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
