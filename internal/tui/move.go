package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/akunbeben/ssht/internal/config"
)

func (m *model) handleMoveKey(key string) (tea.Model, tea.Cmd) {
	if m.moveProfiles == nil || m.moveTarget == nil {
		m.mode = modeList
		return m, nil
	}

	otherCount := 0
	for _, n := range m.moveProfiles.names {
		if n != m.profileName {
			otherCount++
		}
	}

	switch key {
	case "esc", "q":
		m.moveProfiles = nil
		m.moveTarget = nil
		m.mode = modeList
	case "j", "down":
		m.moveProfiles.moveWithMax(1, otherCount)
	case "k", "up":
		m.moveProfiles.moveWithMax(-1, otherCount)
	case "enter":
		return m.handleMoveSelection()
	}
	return m, nil
}

func (m *model) handleMoveSelection() (tea.Model, tea.Cmd) {
	selectedIdx := m.moveProfiles.index
	realIdx := 0
	selectedName := ""
	for _, name := range m.moveProfiles.names {
		if name == m.profileName {
			continue
		}
		if realIdx == selectedIdx {
			selectedName = name
			break
		}
		realIdx++
	}

	if selectedName == "" {
		m.moveNewName = newMoveProfileInput()
		m.mode = modeMoveNewProfile
		return m, nil
	}

	if selectedName == m.profileName {
		cmd := m.setStatus("cannot move to current profile", dimStyle)
		m.moveProfiles = nil
		m.moveTarget = nil
		m.mode = modeList
		return m, cmd
	}

	targetProfile, ok := m.cfg.Profiles[selectedName]
	if ok {
		for _, s := range targetProfile.Servers {
			if s.Name == m.moveTarget.Name {
				m.moveTargetProfile = selectedName
				m.mode = modeMoveConfirmOverwrite
				return m, nil
			}
		}
	}

	return m.moveServerToProfile(selectedName)
}

func (m *model) moveServerToProfile(targetName string) (tea.Model, tea.Cmd) {
	targetProfile, ok := m.cfg.Profiles[targetName]
	if !ok {
		m.err = fmt.Errorf("profile %q not found", targetName)
		m.moveProfiles = nil
		m.moveTarget = nil
		m.mode = modeList
		return m, nil
	}

	movedServer := *m.moveTarget
	serverName := movedServer.Name

	port := movedServer.Port
	if port == 0 {
		port = 22
	}
	exKey := fmt.Sprintf("%s@%s:%d", movedServer.User, movedServer.Host, port)
	if !containsString(m.profile.ImportExceptions, exKey) {
		m.profile.ImportExceptions = append(m.profile.ImportExceptions, exKey)
	}

	next := make([]config.Server, 0, len(m.profile.Servers))
	for _, s := range m.profile.Servers {
		if s.ID == movedServer.ID {
			continue
		}
		next = append(next, s)
	}
	m.profile.Servers = next

	targetProfile.ImportExceptions = removeString(targetProfile.ImportExceptions, exKey)

	targetProfile.Servers = append(targetProfile.Servers, movedServer)
	m.cfg.Profiles[targetName] = targetProfile

	m.moveProfiles = nil
	m.moveTarget = nil
	m.moveTargetProfile = ""
	m.mode = modeList
	m.syncServers()
	cmd := m.setStatus(fmt.Sprintf("✓ moved %q → %s", serverName, targetName), successStyle)
	return m, cmd
}

func (m *model) handleMoveConfirmOverwriteKey(key string) (tea.Model, tea.Cmd) {
	switch strings.ToLower(key) {
	case "y":
		targetName := m.moveTargetProfile
		targetProfile := m.cfg.Profiles[targetName]
		nextT := make([]config.Server, 0, len(targetProfile.Servers))
		for _, s := range targetProfile.Servers {
			if s.Name == m.moveTarget.Name {
				continue
			}
			nextT = append(nextT, s)
		}
		targetProfile.Servers = nextT
		m.cfg.Profiles[targetName] = targetProfile

		return m.moveServerToProfile(targetName)

	case "n", "esc", "q":
		m.moveTargetProfile = ""
		m.mode = modeMoveServer

	}
	return m, nil
}

func (m *model) moveConfirmOverwriteView(width, height int, helperWrapped bool) string {
	if m.moveTarget == nil {
		return ""
	}
	var body strings.Builder
	body.WriteString(titleStyle.Render("Move Collision") + "\n\n")
	msg := fmt.Sprintf("Server %s already exists in profile %s.", m.moveTarget.Name, m.moveTargetProfile)
	warnStyleWrap := warnStyle.Copy().Width(width)
	if !helperWrapped {
		msg = truncate(msg, width)
		warnStyleWrap = warnStyleWrap.MaxHeight(1)
	}
	body.WriteString(warnStyleWrap.Render(msg))
	body.WriteString("\n\n")

	help := "y overwrite and move  n cancel overwrite  Esc back"
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

func (m *model) moveServerView(width, height int, helperWrapped bool) string {
	if m.moveProfiles == nil || m.moveTarget == nil {
		return ""
	}
	var body strings.Builder
	body.WriteString(titleStyle.Render("Move Server") + "\n")
	body.WriteString(dimStyle.Render("Move removes the server from this profile and adds it to the target profile.") + "\n\n")
	body.WriteString("  " + selectedStyle.Render(m.moveTarget.Name) + "  ")
	body.WriteString(dimStyle.Render(serverAddress(*m.moveTarget, m.masked, true)))
	body.WriteString("\n\n")
	body.WriteString(dimStyle.Render("  Destination workspace:") + "\n\n")

	realIdx := 0
	for _, name := range m.moveProfiles.names {
		if name == m.profileName {
			continue
		}
		cursor := inactiveCursor
		labelText := renderMoveProfileRow(name, m.cfg, width)
		label := dimStyle.Render(labelText)
		if realIdx == m.moveProfiles.index {
			cursor = focusedInputStyle.Render(activeCursor)
			label = selectedStyle.Render(labelText)
		}
		body.WriteString(cursor + label + "\n")
		realIdx++
	}

	newCursor := inactiveCursor
	newLabel := dimStyle.Render("+ New profile")
	if m.moveProfiles.index == realIdx {
		newCursor = focusedInputStyle.Render(activeCursor)
		newLabel = selectedStyle.Render("+ New profile")
	}
	body.WriteString(newCursor + newLabel + "\n")

	help := "j/k: navigate  Enter: move  Esc: cancel"
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

func newMoveProfileInput() textinput.Model {
	ti := textinput.New()
	ti.Placeholder = "profile name"
	ti.CharLimit = 64
	ti.Focus()
	ti.PromptStyle = focusedInputStyle
	ti.TextStyle = focusedInputStyle
	ti.Cursor.Style = cursorStyle
	return ti
}

func (m *model) handleMoveNewProfileKey(key string, msg tea.Msg) (tea.Model, tea.Cmd) {
	switch key {
	case "esc":
		m.moveTarget = nil
		m.moveProfiles = nil
		m.mode = modeList
		return m, nil
	case "enter":
		name := strings.TrimSpace(m.moveNewName.Value())
		if err := config.ValidateProfileName(name); err != nil {
			m.err = err
			return m, nil
		}
		if _, exists := m.cfg.Profiles[name]; exists {
			m.err = fmt.Errorf("profile %q already exists", name)
			return m, nil
		}
		m.cfg.Profiles[name] = config.Profile{Name: name, Servers: []config.Server{}}
		m.err = nil

		return m.moveServerToProfile(name)
	}

	var cmd tea.Cmd
	m.moveNewName, cmd = m.moveNewName.Update(msg)
	return m, cmd
}

func (m *model) moveNewProfileView(width, height int, helperWrapped bool) string {
	if m.moveTarget == nil {
		return ""
	}
	var body strings.Builder
	body.WriteString(titleStyle.Render("Move Server") + "\n\n")
	body.WriteString("  " + selectedStyle.Render(m.moveTarget.Name) + "  ")
	body.WriteString(dimStyle.Render(serverAddress(*m.moveTarget, m.masked, true)))
	body.WriteString("\n\n")
	body.WriteString(dimStyle.Render("  New profile name:") + "\n\n")
	body.WriteString("  " + m.moveNewName.View() + "\n\n")
	if m.err != nil {
		errStyleWrap := errorStyle.Copy().Width(width)
		errMsg := m.err.Error()
		if !helperWrapped {
			errMsg = truncate(errMsg, width)
			errStyleWrap = errStyleWrap.MaxHeight(1)
		}
		body.WriteString(errStyleWrap.Render("✗ "+errMsg) + "\n\n")
	}

	help := "Enter: create profile and move  Esc: cancel"
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

func renderMoveProfileRow(name string, cfg *config.Config, width int) string {
	profile := cfg.Profiles[name]
	vpn := "vpn none"
	if profile.VPN != nil {
		vpn = "vpn " + vpnType(profile.VPN)
	}
	row := fmt.Sprintf("%-18s %3d servers  %s", name, len(profile.Servers), vpn)
	return truncate(row, max(width-4, 1))
}
