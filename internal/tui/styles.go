package tui

import "github.com/charmbracelet/lipgloss"

var (
	boxStyle      = lipgloss.NewStyle().Padding(0, 1)
	titleStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39"))
	selectedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true)
	dimStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	errorStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	helpStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	warnStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("11")).Bold(true)
	successStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))

	formLabelStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Bold(true).Width(10)
	focusedInputStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	blurredInputStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	cursorStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("39"))

	profileActiveStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true)
	statusBarStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Italic(true)
)
