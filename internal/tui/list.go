package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/akunbeben/ssht/internal/config"
)

func renderListHeader(width int) string {
	if width < 55 {
		return ""
	}
	cols := calculateColumnWidths(width)
	parts := []string{
		fmt.Sprintf("%-*s", cols.name, "NAME"),
		fmt.Sprintf("%-*s", cols.host, "ADDRESS"),
		fmt.Sprintf("%-*s", cols.vpn, "VPN"),
	}
	if cols.key > 0 {
		parts = append(parts, fmt.Sprintf("%-*s", cols.key, "KEY"))
	}
	header := "  " + strings.Join(parts, " ")
	if cols.showTags {
		header += " TAGS"
	}
	return dimStyle.Render(truncate(header, width))
}

type columnWidths struct {
	name, host, vpn, key int
	showTags             bool
}

func calculateColumnWidths(width int) columnWidths {
	res := columnWidths{
		name:     20,
		host:     28,
		vpn:      11,
		key:      11,
		showTags: true,
	}

	if width < 110 {
		res.name = 18
		res.host = 26
	}
	if width < 80 {
		res.showTags = false
		res.name = 16
		res.host = 24
		res.vpn = 9
		res.key = 8
	}
	if width < 64 {
		res.key = 0
		res.vpn = 9
		res.host = 22
	}

	reserved := 2 + res.vpn + res.key + 4
	if res.key == 0 {
		reserved -= 1
	}
	available := width - reserved
	if res.showTags {
		available -= 14
	}
	if available < res.name+res.host {
		res.name = max(12, available/3)
		res.host = max(14, available-res.name)
	}

	return res
}

func renderServerRow(s config.Server, profileVPN *config.VPNConf, selected bool, masked bool, width int) string {
	if width < 55 {
		return renderMobileServerRow(s, profileVPN, selected, masked, width)
	}

	cols := calculateColumnWidths(width)

	tags := ""
	if cols.showTags && len(s.Tags) > 0 {
		tags = strings.Join(s.Tags, ",")
	}

	name := truncate(maskText(s.Name, masked, 14), cols.name)
	address := truncate(serverAddress(s, masked, false), cols.host)
	vpnDisplay := truncate(vpnBadge(s.VPN, profileVPN), cols.vpn)
	keyDisplay := truncate(keyBadge(s), cols.key)

	var rowParts []string
	if cols.name > 0 {
		rowParts = append(rowParts, fmt.Sprintf("%-*s", cols.name, name))
	}
	if cols.host > 0 {
		rowParts = append(rowParts, fmt.Sprintf("%-*s", cols.host, address))
	}
	if cols.vpn > 0 {
		rowParts = append(rowParts, fmt.Sprintf("%-*s", cols.vpn, vpnDisplay))
	}
	if cols.key > 0 {
		rowParts = append(rowParts, fmt.Sprintf("%-*s", cols.key, keyDisplay))
	}
	if cols.showTags {
		rowParts = append(rowParts, tags)
	}

	row := strings.Join(rowParts, " ")
	if selected {
		return selectedStyle.Render(activeCursor + truncate(row, width-2))
	}
	return inactiveCursor + truncate(row, width-2)
}

func renderMobileServerRow(s config.Server, profileVPN *config.VPNConf, selected bool, masked bool, width int) string {
	cursor := inactiveCursor
	name := maskText(s.Name, masked, 14)
	if selected {
		cursor = activeCursor
		name = selectedStyle.Render(name)
	}

	line1 := cursor + name
	vpnDisplay := vpnBadge(s.VPN, profileVPN)
	if vpnDisplay != "" {
		padding := width - lipgloss.Width(line1) - lipgloss.Width(vpnDisplay) - 2
		if padding > 0 {
			line1 += strings.Repeat(" ", padding) + dimStyle.Render(vpnDisplay)
		}
	}

	meta := serverAddress(s, masked, true) + "  " + keyBadge(s)
	if len(s.Tags) > 0 {
		meta += "  " + strings.Join(s.Tags, ",")
	}
	line2 := "    " + dimmedItalicStyle.Render(truncate(meta, width-4))

	return line1 + "\n" + line2
}

func truncate(s string, limit int) string {
	if len(s) <= limit {
		return s
	}
	if limit <= 3 {
		return s[:limit]
	}
	return s[:limit-3] + "..."
}
