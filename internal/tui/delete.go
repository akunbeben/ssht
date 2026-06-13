package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

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
	innerW, innerH := m.innerSize()
	var b strings.Builder
	b.WriteString(titleStyle.Render("Delete Server") + "\n")
	b.WriteString(dimStyle.Render("This removes the server from the current profile and prevents auto-import from adding it back.") + "\n\n")
	b.WriteString(warnStyle.Render("Delete") + " ")
	b.WriteString(selectedStyle.Render(m.deleteTarget.Name))
	b.WriteString(warnStyle.Render("?") + "\n")
	b.WriteString(dimStyle.Render(serverAddress(*m.deleteTarget, m.masked, true)) + "\n")
	if len(m.deleteTarget.Tags) > 0 {
		b.WriteString(dimStyle.Render("Tags: "+strings.Join(m.deleteTarget.Tags, ", ")) + "\n")
	}
	if strings.TrimSpace(m.deleteTarget.Note) != "" && !m.masked {
		b.WriteString(dimStyle.Copy().Width(innerW).Render("Note: "+m.deleteTarget.Note) + "\n")
	}
	b.WriteString("\n")

	help := "y confirm delete  n/Esc cancel"
	b.WriteString(helpStyle.Render(truncate(help, innerW)))

	content := b.String()
	if lipgloss.Height(content) < innerH {
		content += strings.Repeat("\n", innerH-lipgloss.Height(content))
	}
	return m.renderFullScreen(content)
}
