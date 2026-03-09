package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

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

func (m *model) moveConfirmOverwriteView() string {
	if m.moveTarget == nil {
		return ""
	}
	var b strings.Builder
	b.WriteString(titleStyle.Render("Move Collision") + "\n\n")
	b.WriteString(warnStyle.Render("  ⚠ Server ") + selectedStyle.Render(m.moveTarget.Name))
	b.WriteString(warnStyle.Render(" already exists in profile ") + selectedStyle.Render(m.moveTargetProfile))
	b.WriteString(warnStyle.Render("."))
	b.WriteString("\n\n")
	b.WriteString(helpStyle.Render("  Overwrite in target? (y/n)"))
	return m.renderFullScreen(b.String())
}

func (m *model) moveServerView() string {
	if m.moveProfiles == nil || m.moveTarget == nil {
		return ""
	}
	var b strings.Builder
	b.WriteString(titleStyle.Render("Move Server") + "\n\n")
	b.WriteString("  " + selectedStyle.Render(m.moveTarget.Name))
	host := m.moveTarget.Host
	user := m.moveTarget.User
	if m.masked {
		host = strings.Repeat("*", min(len(host), 12))
		user = strings.Repeat("*", min(len(user), 8))
	}
	b.WriteString(dimStyle.Render(fmt.Sprintf("  %s@%s", user, host)))
	b.WriteString("\n\n")
	b.WriteString(dimStyle.Render("  Move to:") + "\n\n")

	realIdx := 0
	for _, name := range m.moveProfiles.names {
		if name == m.profileName {
			continue
		}
		cursor := "  "
		label := dimStyle.Render(name)
		if realIdx == m.moveProfiles.index {
			cursor = focusedInputStyle.Render("▸ ")
			label = selectedStyle.Render(name)
		}
		b.WriteString(cursor + label + "\n")
		realIdx++
	}

	newCursor := "  "
	newLabel := dimStyle.Render("+ New profile")
	if m.moveProfiles.index == realIdx {
		newCursor = focusedInputStyle.Render("▸ ")
		newLabel = selectedStyle.Render("+ New profile")
	}
	b.WriteString(newCursor + newLabel + "\n")

	b.WriteString("\n")
	b.WriteString(helpStyle.Render("j/k: navigate · Enter: move · Esc: cancel"))

	return m.renderFullScreen(b.String())
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
		if name == "" {
			m.err = fmt.Errorf("profile name is required")
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

func (m *model) moveNewProfileView() string {
	if m.moveTarget == nil {
		return ""
	}
	var b strings.Builder
	b.WriteString(titleStyle.Render("Move Server") + "\n\n")
	b.WriteString("  " + selectedStyle.Render(m.moveTarget.Name))
	host := m.moveTarget.Host
	user := m.moveTarget.User
	if m.masked {
		host = strings.Repeat("*", min(len(host), 12))
		user = strings.Repeat("*", min(len(user), 8))
	}
	b.WriteString(dimStyle.Render(fmt.Sprintf("  %s@%s", user, host)))
	b.WriteString("\n\n")
	b.WriteString(dimStyle.Render("  New profile name:") + "\n\n")
	b.WriteString("  " + m.moveNewName.View() + "\n\n")
	if m.err != nil {
		b.WriteString("  " + errorStyle.Render("✗ "+m.err.Error()) + "\n\n")
	}
	b.WriteString(helpStyle.Render("  Enter: create & move · Esc: cancel"))

	return m.renderFullScreen(b.String())
}
