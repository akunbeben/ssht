package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/akunbeben/ssht/internal/config"
)

func (m *model) handleDeleteKey(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "y", "Y":
		if m.deleteTarget == nil {
			m.mode = modeList
			return m, nil
		}
		next := make([]config.Server, 0, len(m.profile.Servers))
		for _, s := range m.profile.Servers {
			if s.ID == m.deleteTarget.ID {
				continue
			}
			next = append(next, s)
		}
		deletedName := m.deleteTarget.Name

		port := m.deleteTarget.Port
		if port == 0 {
			port = 22
		}
		exKey := fmt.Sprintf("%s@%s:%d", m.deleteTarget.User, m.deleteTarget.Host, port)
		if !containsString(m.profile.ImportExceptions, exKey) {
			m.profile.ImportExceptions = append(m.profile.ImportExceptions, exKey)
		}

		m.profile.Servers = next
		m.deleteTarget = nil
		m.mode = modeList
		m.syncServers()
		cmd := m.setStatus(fmt.Sprintf("✓ deleted %q", deletedName), successStyle)
		return m, cmd
	case "n", "N", "esc", "q":
		m.deleteTarget = nil
		m.mode = modeList
	}
	return m, nil
}

func (m *model) confirmDeleteView() string {
	if m.deleteTarget == nil {
		return ""
	}
	var b strings.Builder
	b.WriteString(titleStyle.Render("Delete Server") + "\n\n")
	b.WriteString(warnStyle.Render("  ⚠ Delete") + " ")
	b.WriteString(selectedStyle.Render(m.deleteTarget.Name))
	b.WriteString(warnStyle.Render("?") + "\n")
	host := m.deleteTarget.Host
	user := m.deleteTarget.User
	if m.masked {
		host = strings.Repeat("*", min(len(host), 12))
		user = strings.Repeat("*", min(len(user), 8))
	}
	b.WriteString(dimStyle.Render(fmt.Sprintf("  %s@%s:%d", user, host, m.deleteTarget.Port)) + "\n\n")
	b.WriteString(helpStyle.Render("  y: confirm · n/Esc: cancel"))
	return m.renderFullScreen(b.String())
}
