package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/akunbeben/ssht/internal/config"
)

const selectedHeroHeight = 6

func (m *model) renderDashboard(width, height int) string {
	if len(m.filtered) == 0 {
		return fitBlock(m.renderDashboardEmpty(width), height)
	}

	rightWidth := min(44, max(36, width/4))
	launcherWidth := width - rightWidth - 3
	if launcherWidth < 58 {
		return m.renderStackedBody(width, height)
	}

	hero := m.renderSelectedHero(m.filtered[m.index], width)
	bodyHeight := max(height-selectedHeroHeight-1, 1)

	launcher := m.renderDenseLauncher(launcherWidth, bodyHeight)
	cockpit := m.renderCockpit(m.filtered[m.index], rightWidth, bodyHeight)
	body := lipgloss.JoinHorizontal(lipgloss.Top,
		launcher,
		lipgloss.NewStyle().Width(3).Render(""),
		cockpit,
	)

	return hero + "\n" + body
}

func (m *model) renderSelectedHero(s config.Server, width int) string {
	innerWidth := max(width-panelStyle.GetHorizontalFrameSize(), 1)
	name := maskText(s.Name, m.masked, 14)
	address := serverAddress(s, m.masked, true)
	badges := strings.Join([]string{vpnBadge(s.VPN, m.profile.VPN), keyBadge(s)}, "  ")
	command := commandPreview(s, m.masked)

	statusWidth := min(30, max(innerWidth/4, 18))
	badgeRaw := fmt.Sprintf("%-*s", statusWidth, truncate(badges, statusWidth))
	badgeText := dimStyle.Render(badgeRaw)
	nameText := heroStyle.Render(truncate(name, max(innerWidth-statusWidth-2, 10)))
	line1 := nameText
	gap := innerWidth - lipgloss.Width(line1) - statusWidth
	if gap > 0 {
		line1 += strings.Repeat(" ", gap) + badgeText
	}

	meta := address
	if len(s.Tags) > 0 {
		meta += "  #" + strings.Join(s.Tags, " #")
	}

	lines := []string{
		line1,
		dimStyle.Render(truncate(meta, innerWidth)),
		helpStyle.Render(truncate(command, innerWidth)),
	}
	content := strings.Join(lines, "\n")

	return panelStyle.Width(innerWidth).Height(selectedHeroHeight - panelStyle.GetVerticalFrameSize()).Render(content)
}

func (m *model) renderDenseLauncher(width, height int) string {
	header := panelTitleStyle.Render("Servers") + "  " + dimStyle.Render(fmt.Sprintf("%d matches", len(m.filtered)))
	rowsHeight := max(height-renderedHeight(header)-1, 1)
	rows := strings.Join(m.visibleDenseRows(rowsHeight, width), "\n")
	content := header + "\n" + rows
	return lipgloss.NewStyle().Width(width).Height(height).Render(content)
}

func (m *model) visibleDenseRows(height, width int) []string {
	if len(m.filtered) == 0 {
		return []string{dimStyle.Render("No servers")}
	}

	start := max(m.index-height+1, 0)
	end := min(start+height, len(m.filtered))
	rows := make([]string, 0, height)
	for i := start; i < end; i++ {
		rows = append(rows, renderDenseServerRow(m.filtered[i], m.profile.VPN, i == m.index, m.masked, width))
	}
	return padRows(rows, height)
}

func renderDenseServerRow(s config.Server, profileVPN *config.VPNConf, selected bool, masked bool, width int) string {
	cursor := inactiveCursor
	if selected {
		cursor = activeCursor
	}

	statusWidth := 26
	nameWidth := max(width/4, 18)
	leftWidth := max(width-statusWidth, 20)
	addrWidth := max(leftWidth-nameWidth-lipgloss.Width(cursor)-1, 10)

	name := truncate(maskText(s.Name, masked, 14), nameWidth)
	addr := truncate(serverAddress(s, masked, false), addrWidth)
	status := truncate(vpnBadge(s.VPN, profileVPN)+"  "+keyBadge(s), statusWidth)

	left := fmt.Sprintf("%s%-*s %-*s", cursor, nameWidth, name, addrWidth, addr)
	left = truncate(left, leftWidth)
	gap := max(width-lipgloss.Width(left)-statusWidth, 1)
	row := left + strings.Repeat(" ", gap) + fmt.Sprintf("%-*s", statusWidth, status)
	row = truncate(row, width)
	if selected {
		return selectedStyle.Render(row)
	}
	return dimStyle.Render(row)
}

func (m *model) renderCockpit(s config.Server, width, height int) string {
	sections := []string{
		panelTitleStyle.Render("Cockpit"),
		cockpitMetric("Profile", m.profileName),
		cockpitMetric("VPN", vpnSummary(s.VPN, m.profile.VPN, m.profileName)),
		cockpitMetric("Key", keySummary(s, m.masked)),
		"",
		panelTitleStyle.Render("Actions"),
		actionLine("Enter", "connect now"),
		actionLine("e", "edit target"),
		actionLine("c", "duplicate"),
		actionLine("m", "move profile"),
		actionLine("d", "delete"),
		"",
		panelTitleStyle.Render("Flow"),
		dimStyle.Render("1. inspect target"),
		dimStyle.Render("2. verify VPN/key"),
		dimStyle.Render("3. press Enter"),
	}

	if m.masked {
		sections = append(sections, "", warnBadgeStyle.Render("masked"))
	}

	content := fitBlock(strings.Join(sections, "\n"), height-panelStyle.GetVerticalFrameSize())
	return panelStyle.Width(width - panelStyle.GetHorizontalFrameSize()).Height(height - panelStyle.GetVerticalFrameSize()).Render(content)
}

func cockpitMetric(label, value string) string {
	return fmt.Sprintf("%-8s %s", dimStyle.Render(label), value)
}

func (m *model) renderActionRail(width, height int) string {
	privacy := "off"
	if m.masked {
		privacy = "on"
	}

	vpn := "none"
	if m.profile.VPN != nil {
		vpn = vpnType(m.profile.VPN)
	}

	lines := []string{
		panelTitleStyle.Render("Workspace"),
		heroStyle.Render(m.profileName),
		dimStyle.Render(fmt.Sprintf("%d servers", len(m.servers))),
		badgeFor("vpn "+vpn, m.profile.VPN != nil),
		badgeFor("privacy "+privacy, m.masked),
		"",
		panelTitleStyle.Render("Do"),
		actionLine("Enter", "connect"),
		actionLine("/", "find server"),
		actionLine("^K", "command menu"),
		actionLine("a", "new server"),
		actionLine("e", "edit selected"),
		actionLine("p", "profiles"),
		actionLine("v", "vpn"),
		"",
		panelTitleStyle.Render("Trust"),
		dimStyle.Render("Preview before connect"),
		dimStyle.Render("VPN source visible"),
		dimStyle.Render("Mask with *"),
	}

	body := strings.Join(lines, "\n")
	body = fitBlock(body, max(height-panelStyle.GetVerticalFrameSize(), 1))
	return panelStyle.Width(width - panelStyle.GetHorizontalFrameSize()).Height(height - panelStyle.GetVerticalFrameSize()).Render(body)
}

func actionLine(key, label string) string {
	return fmt.Sprintf("%-6s %s", focusedInputStyle.Render(key), label)
}

func (m *model) renderServerLauncher(width, height int) string {
	innerHeight := max(height-panelStyle.GetVerticalFrameSize(), 1)
	header := panelTitleStyle.Render("Server Launcher")
	subtitle := dimStyle.Render(fmt.Sprintf("%d matches. Move to inspect, Enter to connect.", len(m.filtered)))
	rowsHeight := max(innerHeight-renderedHeight(header)-renderedHeight(subtitle)-2, 1)
	rows := strings.Join(m.visibleServerCards(rowsHeight, width-panelStyle.GetHorizontalFrameSize()), "\n")
	body := strings.Join([]string{header, subtitle, "", rows}, "\n")
	return panelStyle.Width(width - panelStyle.GetHorizontalFrameSize()).Height(innerHeight).Render(body)
}

func (m *model) visibleServerCards(height, width int) []string {
	cardHeight := 3
	numVisible := max(height/cardHeight, 1)
	start := max(m.index-numVisible+1, 0)
	end := min(start+numVisible, len(m.filtered))

	rows := make([]string, 0, numVisible*cardHeight)
	for i := start; i < end; i++ {
		rows = append(rows, renderServerCard(m.filtered[i], m.profile.VPN, i == m.index, m.masked, width))
	}
	return padRenderedRows(rows, height)
}

func renderServerCard(s config.Server, profileVPN *config.VPNConf, selected bool, masked bool, width int) string {
	cursor := inactiveCursor
	name := maskText(s.Name, masked, 14)
	if selected {
		cursor = activeCursor
		name = selectedStyle.Render(name)
	}

	badges := vpnBadge(s.VPN, profileVPN) + "  " + keyBadge(s)
	line1 := cursor + truncate(name, max(width-lipgloss.Width(badges)-4, 8))
	gap := width - lipgloss.Width(line1) - lipgloss.Width(badges)
	if gap > 0 {
		line1 += strings.Repeat(" ", gap) + dimStyle.Render(badges)
	}

	line2 := "    " + dimStyle.Render(truncate(serverAddress(s, masked, true), max(width-4, 1)))
	meta := ""
	if len(s.Tags) > 0 {
		meta = strings.Join(s.Tags, "  ")
	} else if strings.TrimSpace(s.Note) != "" && !masked {
		meta = s.Note
	}
	line3 := "    " + dimmedItalicStyle.Render(truncate(meta, max(width-4, 1)))
	return line1 + "\n" + line2 + "\n" + line3
}

func (m *model) renderConnectionDossier(s config.Server, width, height int) string {
	innerHeight := max(height-panelStyle.GetVerticalFrameSize(), 1)
	lines := []string{
		panelTitleStyle.Render("Connection Dossier"),
		heroStyle.Render(maskText(s.Name, m.masked, 14)),
		"",
		preflightLine("Target", serverAddress(s, m.masked, true)),
		preflightLine("VPN", vpnSummary(s.VPN, m.profile.VPN, m.profileName)),
		preflightLine("Key", keySummary(s, m.masked)),
	}

	if len(s.Tags) > 0 {
		lines = append(lines, preflightLine("Tags", strings.Join(s.Tags, ", ")))
	}
	if strings.TrimSpace(s.Note) != "" && !m.masked {
		lines = append(lines, "", panelTitleStyle.Render("Note"), dimStyle.Copy().Width(width-4).Render(s.Note))
	}

	lines = append(lines,
		"",
		panelTitleStyle.Render("Command"),
		helpStyle.Copy().Width(width-4).Render(commandPreview(s, m.masked)),
		"",
		goodBadgeStyle.Render("Enter to connect"),
	)

	body := strings.Join(lines, "\n")
	body = fitBlock(body, innerHeight)
	return panelStyle.Width(width - panelStyle.GetHorizontalFrameSize()).Height(innerHeight).Render(body)
}

func preflightLine(label, value string) string {
	return fmt.Sprintf("%-8s %s", dimStyle.Render(label+":"), value)
}

func (m *model) renderDashboardEmpty(width int) string {
	if strings.TrimSpace(m.search) != "" {
		lines := []string{
			heroStyle.Render("No matches"),
			fmt.Sprintf("Nothing matched %q in profile %s.", m.search, m.profileName),
			"",
			actionLine("Esc", "clear search"),
			actionLine("a", "add a new server"),
			actionLine("^K", "open command menu"),
		}
		return panelStyle.Width(width - panelStyle.GetHorizontalFrameSize()).Render(strings.Join(lines, "\n"))
	}

	lines := []string{
		heroStyle.Render("Build your first workspace"),
		"This profile has no servers yet.",
		"",
		"Start with one of these actions:",
		actionLine("a", "add a server manually"),
		actionLine("K", "prepare SSH public keys"),
		actionLine("v", "configure profile VPN"),
		actionLine("p", "switch workspace"),
	}
	return panelStyle.Width(width - panelStyle.GetHorizontalFrameSize()).Render(strings.Join(lines, "\n"))
}
