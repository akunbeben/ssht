package tui

import "github.com/charmbracelet/lipgloss"

func (m *model) renderMainFooter(width int) string {
	status := m.status
	style := m.statusStyle
	if m.err != nil {
		status = "x " + m.err.Error()
		style = errorStyle
	}
	if status != "" {
		return m.renderStatus(status, style, width)
	}

	help := "Enter connect  ctrl+k actions  / search  a add  e edit  p profile  ? help"
	if len(m.filtered) == 0 {
		help = "ctrl+k actions  a add server  p profile  v vpn  K keys  ? help"
	}
	if width < 70 {
		help = "Enter connect  ctrl+k actions  / search  a add  ? help"
	}
	if width < 48 {
		help = "Enter connect  ^K actions  / search"
	}

	s := helpStyle.Copy().Width(width)
	if !m.helperWrapped {
		help = truncate(help, width)
		s = s.MaxHeight(1)
	}
	return s.Render(help)
}

func renderedHeight(s string) int {
	if s == "" {
		return 0
	}
	return lipgloss.Height(s)
}
