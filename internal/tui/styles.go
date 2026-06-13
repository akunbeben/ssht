package tui

import "github.com/charmbracelet/lipgloss"

const (
	inactiveCursor = "  "
	activeCursor   = "> "
)

var (
	boxStyle      = lipgloss.NewStyle().Padding(0, 1)
	titleStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39"))
	selectedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true)
	dimStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	errorStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	helpStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	warnStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("11")).Bold(true)
	successStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))

	formLabelStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Bold(true).Width(15).Align(lipgloss.Right).PaddingRight(1)
	formLabelStackedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Bold(true).PaddingBottom(0)
	focusedInputStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	blurredInputStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	placeholderStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	cursorStyle           = lipgloss.NewStyle().Foreground(lipgloss.Color("39"))
	badgeStyle            = lipgloss.NewStyle().Foreground(lipgloss.Color("230")).Background(lipgloss.Color("238")).Padding(0, 1)
	goodBadgeStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("16")).Background(lipgloss.Color("10")).Padding(0, 1)
	warnBadgeStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("16")).Background(lipgloss.Color("11")).Padding(0, 1)
	detailBoxStyle        = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("238")).Padding(0, 1)
	panelStyle            = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("238")).Padding(0, 1)
	panelTitleStyle       = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39"))
	heroStyle             = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("10"))

	profileActiveStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true)
	statusBarStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Italic(true)
	dimmedItalicStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Italic(true)
)
