package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m *model) renderMainView() string {
	innerW, innerH := m.innerSize()
	header := m.renderMainHeader(innerW)
	search := m.renderSearchLine(innerW)
	footer := m.renderMainFooter(innerW)

	reserved := renderedHeight(header) + renderedHeight(search) + renderedHeight(footer) + 3
	bodyHeight := max(innerH-reserved, 1)
	body := m.renderMainBody(innerW, bodyHeight)

	top := strings.Join([]string{header, search, body}, "\n")
	content := pinFooter(top, footer, innerH)
	return m.renderFullScreen(content)
}

func (m *model) renderMainBody(width, height int) string {
	if len(m.filtered) == 0 {
		return fitBlock(m.renderDashboardEmpty(width), height)
	}

	if width >= 110 {
		return m.renderDashboard(width, height)
	}
	return m.renderStackedBody(width, height)
}

func (m *model) renderDesktopBody(width, height int) string {
	detailWidth := min(44, max(34, width/3))
	listWidth := max(width-detailWidth-2, 30)
	if listWidth < 56 {
		return m.renderStackedBody(width, height)
	}

	header := renderListHeader(listWidth)
	listHeight := height
	if header != "" {
		listHeight--
	}
	rows := strings.Join(m.visibleServerRowsFor(listHeight, listWidth), "\n")
	list := rows
	if header != "" {
		list = header + "\n" + rows
	}

	details := renderSelectedDetails(m.filtered[m.index], m.profile.VPN, m.profileName, m.masked, detailWidth)
	details = fitBlock(details, height)
	return joinPanels(list, details, listWidth, detailWidth)
}

func (m *model) renderStackedBody(width, height int) string {
	details := renderSelectedDetails(m.filtered[m.index], m.profile.VPN, m.profileName, m.masked, width)
	detailHeight := renderedHeight(details)
	if width < 55 {
		detailHeight = min(detailHeight, 7)
	}
	if detailHeight > height/2 {
		detailHeight = max(height/3, 4)
	}

	header := renderListHeader(width)
	listHeight := height - detailHeight - 1
	if header != "" {
		listHeight--
	}
	if listHeight < 1 {
		listHeight = 1
		detailHeight = max(height-listHeight-1, 0)
	}

	rows := strings.Join(m.visibleServerRowsFor(listHeight, width), "\n")
	parts := []string{}
	if header != "" {
		parts = append(parts, header)
	}
	parts = append(parts, rows)
	if detailHeight > 0 {
		parts = append(parts, fitBlock(details, detailHeight))
	}

	return lipgloss.NewStyle().Width(width).Render(strings.Join(parts, "\n"))
}
