package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type paletteAction struct {
	Name        string
	Description string
	Key         string
	Enabled     bool
}

type actionPaletteState struct {
	query string
	index int
}

func (m *model) openActionPalette() {
	m.palette = actionPaletteState{}
	m.mode = modeActionPalette
}

func (m *model) handleActionPaletteKey(key string, msg tea.Msg) (tea.Model, tea.Cmd) {
	switch key {
	case "esc", "ctrl+c":
		m.mode = modeList
		return m, nil
	case "enter":
		actions := m.filteredPaletteActions()
		if len(actions) == 0 {
			return m, nil
		}
		return m.executePaletteAction(actions[m.palette.index])
	case "j", "down":
		m.movePalette(1)
	case "k", "up":
		m.movePalette(-1)
	case "home":
		m.palette.index = 0
	case "end", "G":
		actions := m.filteredPaletteActions()
		if len(actions) > 0 {
			m.palette.index = len(actions) - 1
		}
	case "backspace":
		if len(m.palette.query) > 0 {
			m.palette.query = m.palette.query[:len(m.palette.query)-1]
			m.palette.index = 0
		}
	default:
		if len(key) == 1 {
			m.palette.query += key
			m.palette.index = 0
		}
	}
	_ = msg
	return m, nil
}

func (m *model) executePaletteAction(action paletteAction) (tea.Model, tea.Cmd) {
	if !action.Enabled {
		return m, m.setStatus("action unavailable", errorStyle)
	}

	m.mode = modeList
	return m.handleListKey(action.Key)
}

func (m *model) movePalette(delta int) {
	actions := m.filteredPaletteActions()
	if len(actions) == 0 {
		m.palette.index = 0
		return
	}
	next := m.palette.index + delta
	if next < 0 {
		next = 0
	}
	if next >= len(actions) {
		next = len(actions) - 1
	}
	m.palette.index = next
}

func (m *model) filteredPaletteActions() []paletteAction {
	actions := m.paletteActions()
	query := strings.ToLower(strings.TrimSpace(m.palette.query))
	if query == "" {
		return actions
	}

	filtered := make([]paletteAction, 0, len(actions))
	for _, action := range actions {
		candidate := strings.ToLower(action.Name + " " + action.Description + " " + action.Key)
		if strings.Contains(candidate, query) {
			filtered = append(filtered, action)
		}
	}
	return filtered
}

func (m *model) paletteActions() []paletteAction {
	hasSelection := len(m.filtered) > 0
	return []paletteAction{
		{Name: "Connect", Description: selectedActionDescription(m, "Open SSH session"), Key: "enter", Enabled: hasSelection},
		{Name: "Search Servers", Description: "Filter by name, address, user, tag, or note", Key: "/", Enabled: true},
		{Name: "Add Server", Description: "Create a new SSH target", Key: "a", Enabled: true},
		{Name: "Edit Server", Description: selectedActionDescription(m, "Edit selected SSH target"), Key: "e", Enabled: hasSelection},
		{Name: "Copy Server", Description: selectedActionDescription(m, "Duplicate selected SSH target"), Key: "c", Enabled: hasSelection},
		{Name: "Delete Server", Description: selectedActionDescription(m, "Remove selected SSH target"), Key: "d", Enabled: hasSelection},
		{Name: "Move Server", Description: selectedActionDescription(m, "Move selected server to another profile"), Key: "m", Enabled: hasSelection},
		{Name: "Switch Profile", Description: "Change workspace/profile", Key: "p", Enabled: len(m.cfg.Profiles) > 1},
		{Name: "Configure Or Toggle VPN", Description: "Set up or toggle profile VPN", Key: "v", Enabled: true},
		{Name: "Public Keys", Description: "Copy or generate SSH public keys", Key: "K", Enabled: true},
		{Name: "Toggle Privacy", Description: "Mask sensitive server details", Key: "*", Enabled: true},
		{Name: "Toggle Helper Wrapping", Description: "Wrap or truncate footer/help text", Key: "H", Enabled: true},
		{Name: "Open Help", Description: "Show all shortcuts", Key: "?", Enabled: true},
	}
}

func selectedActionDescription(m *model, fallback string) string {
	if len(m.filtered) == 0 {
		return fallback
	}
	return fmt.Sprintf("%s: %s", fallback, m.filtered[m.index].Name)
}

func (m *model) actionPaletteView() string {
	innerW, innerH := m.innerSize()
	actions := m.filteredPaletteActions()
	if m.palette.index >= len(actions) {
		m.palette.index = max(len(actions)-1, 0)
	}

	query := m.palette.query
	if query == "" {
		query = "type to filter actions"
	}

	header := titleStyle.Render("Action Palette")
	search := focusedInputStyle.Render(activeCursor) + query
	footer := helpStyle.Render("Enter run  Esc close  j/k move")
	bodyHeight := max(innerH-renderedHeight(header)-renderedHeight(search)-renderedHeight(footer)-3, 1)
	body := m.renderPaletteRows(actions, bodyHeight, innerW)

	top := strings.Join([]string{header, search, body}, "\n")
	content := pinFooter(top, footer, innerH)
	return m.renderFullScreen(content)
}

func (m *model) renderPaletteRows(actions []paletteAction, height, width int) string {
	if len(actions) == 0 {
		return dimStyle.Render("  no matching actions")
	}

	start := max(m.palette.index-height+1, 0)
	end := min(start+height, len(actions))
	rows := make([]string, 0, height)
	for i := start; i < end; i++ {
		action := actions[i]
		cursor := inactiveCursor
		style := lipgloss.NewStyle()
		if !action.Enabled {
			style = dimStyle
		}
		if i == m.palette.index {
			cursor = activeCursor
			style = selectedStyle
		}
		key := dimStyle.Render(action.Key)
		name := fmt.Sprintf("%-24s", action.Name)
		row := cursor + name + " " + key + "  " + action.Description
		rows = append(rows, style.Render(truncate(row, width)))
	}
	return strings.Join(padRows(rows, height), "\n")
}
