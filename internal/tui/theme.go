package tui

import "github.com/charmbracelet/lipgloss"

var (
	colorAccent = lipgloss.AdaptiveColor{Light: "#5A56E0", Dark: "#8B87FF"}
	colorText   = lipgloss.AdaptiveColor{Light: "#1F1F24", Dark: "#E4E4E7"}
	colorMuted  = lipgloss.AdaptiveColor{Light: "#62626D", Dark: "#9A9AA6"}
	colorFaint  = lipgloss.AdaptiveColor{Light: "#A1A1AA", Dark: "#5C5C66"}
	colorBorder = lipgloss.AdaptiveColor{Light: "#D4D4D8", Dark: "#3F3F46"}
	colorSelBg  = lipgloss.AdaptiveColor{Light: "#EAE8FF", Dark: "#2B2945"}
	colorGood   = lipgloss.AdaptiveColor{Light: "#15803D", Dark: "#4ADE80"}
	colorWarn   = lipgloss.AdaptiveColor{Light: "#B45309", Dark: "#FBBF24"}
	colorBad    = lipgloss.AdaptiveColor{Light: "#B91C1C", Dark: "#F87171"}
)

var (
	textStyle   = lipgloss.NewStyle().Foreground(colorText)
	boldStyle   = textStyle.Bold(true)
	mutedStyle  = lipgloss.NewStyle().Foreground(colorMuted)
	faintStyle  = lipgloss.NewStyle().Foreground(colorFaint)
	accentStyle = lipgloss.NewStyle().Foreground(colorAccent).Bold(true)
	goodStyle   = lipgloss.NewStyle().Foreground(colorGood)
	warnStyle   = lipgloss.NewStyle().Foreground(colorWarn)
	badStyle    = lipgloss.NewStyle().Foreground(colorBad)

	columnStyle   = faintStyle.Bold(true)
	selectedStyle = lipgloss.NewStyle().Background(colorSelBg).Foreground(colorText).Bold(true)
	labelStyle    = mutedStyle
	detailsStyle  = lipgloss.NewStyle().Border(lipgloss.NormalBorder(), false, false, false, true).BorderForeground(colorBorder).PaddingLeft(2)
	dialogStyle   = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(colorAccent).Padding(1, 2)
)
