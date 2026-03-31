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
	var parts []string
	if cols.name > 0 {
		parts = append(parts, fmt.Sprintf("%-*s", cols.name, "NAME"))
	}
	if cols.host > 0 {
		parts = append(parts, fmt.Sprintf("%-*s", cols.host, "HOST"))
	}
	if cols.user > 0 {
		parts = append(parts, fmt.Sprintf("%-*s", cols.user, "USER"))
	}
	if cols.port > 0 {
		parts = append(parts, fmt.Sprintf("%-*s", cols.port, "PORT"))
	}
	if cols.vpn > 0 {
		parts = append(parts, fmt.Sprintf("%-*s", cols.vpn, "VPN"))
	}
	if cols.showTags {
		parts = append(parts, "TAGS")
	}
	header := "  " + strings.Join(parts, " ")
	return dimStyle.Render(truncate(header, width))
}

type columnWidths struct {
	name, host, user, port, vpn int
	showTags                    bool
}

func calculateColumnWidths(width int) columnWidths {
	// Base widths: 20, 20, 10, 5, 15
	// Total with spaces: 2 + 20 + 1 + 20 + 1 + 10 + 1 + 5 + 1 + 15 + 1 = 77
	w := width - 2 // left padding

	res := columnWidths{
		name:     20,
		host:     20,
		user:     10,
		port:     5,
		vpn:      15,
		showTags: true,
	}

	if width < 110 {
		res.showTags = false
	}
	if width < 80 {
		res.vpn = 0 // Hide VPN or make it very small? Let's try 0 for now
	}
	if width < 60 {
		res.port = 0
		res.user = 8
	}
	if width < 45 {
		res.host = 15
		res.name = 15
	}

	// Dynamic scaling to fit width
	total := res.name + res.host + res.user + res.port + res.vpn + 5 // 5 spaces
	if total > w && w > 20 {
		scale := float64(w-5) / float64(total-5)
		res.name = max(int(float64(res.name)*scale), 10)
		res.host = max(int(float64(res.host)*scale), 10)
		if res.user > 0 {
			res.user = max(int(float64(res.user)*scale), 5)
		}
		if res.vpn > 0 {
			res.vpn = max(int(float64(res.vpn)*scale), 5)
		}
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
		tags = "[" + strings.Join(s.Tags, ",") + "]"
	}
	port := s.Port
	if port == 0 {
		port = 22
	}

	vpnDisplay := "-"
	isOverride := false
	activeVPN := s.VPN
	if activeVPN != nil {
		isOverride = true
	} else if profileVPN != nil {
		activeVPN = profileVPN
	}

	if activeVPN != nil {
		vType := activeVPN.Type
		if vType == "" {
			vType = "wg"
		}
		vpnDisplay = vType
		if isOverride {
			vpnDisplay = "*" + vpnDisplay
		}
	}

	host := s.Host
	user := s.User
	portDisplay := fmt.Sprintf("%d", port)
	if masked {
		host = strings.Repeat("*", min(len(host), 12))
		user = strings.Repeat("*", min(len(user), 8))
		portDisplay = "*****"
		if vpnDisplay != "-" {
			vpnDisplay = "****"
		}
	}

	name := truncate(s.Name, cols.name)
	hostDisplay := truncate(host, cols.host)
	userDisplay := truncate(user, cols.user)
	vpnDisplay = truncate(vpnDisplay, cols.vpn)

	var rowParts []string
	if cols.name > 0 {
		rowParts = append(rowParts, fmt.Sprintf("%-*s", cols.name, name))
	}
	if cols.host > 0 {
		rowParts = append(rowParts, fmt.Sprintf("%-*s", cols.host, hostDisplay))
	}
	if cols.user > 0 {
		rowParts = append(rowParts, fmt.Sprintf("%-*s", cols.user, userDisplay))
	}
	if cols.port > 0 {
		rowParts = append(rowParts, fmt.Sprintf("%-*s", cols.port, portDisplay))
	}
	if cols.vpn > 0 {
		rowParts = append(rowParts, fmt.Sprintf("%-*s", cols.vpn, vpnDisplay))
	}
	if cols.showTags {
		rowParts = append(rowParts, tags)
	}

	row := strings.Join(rowParts, " ")
	if selected {
		return selectedStyle.Render("> " + truncate(row, width-2))
	}
	return "  " + truncate(row, width-2)
}

func renderMobileServerRow(s config.Server, profileVPN *config.VPNConf, selected bool, masked bool, width int) string {
	port := s.Port
	if port == 0 {
		port = 22
	}

	vpnDisplay := ""
	activeVPN := s.VPN
	if activeVPN == nil {
		activeVPN = profileVPN
	}
	if activeVPN != nil {
		vType := activeVPN.Type
		if vType == "" {
			vType = "wg"
		}
		vpnDisplay = vType
		if s.VPN != nil {
			vpnDisplay = "*" + vpnDisplay
		}
	}

	host := s.Host
	user := s.User
	if masked {
		host = strings.Repeat("*", min(len(host), 12))
		user = strings.Repeat("*", min(len(user), 8))
	}

	cursor := "  "
	name := s.Name
	if selected {
		cursor = "> "
		name = selectedStyle.Render(name)
	}

	line1 := cursor + name
	if vpnDisplay != "" {
		padding := width - lipgloss.Width(line1) - lipgloss.Width(vpnDisplay) - 2
		if padding > 0 {
			line1 += strings.Repeat(" ", padding) + dimStyle.Render(vpnDisplay)
		}
	}

	meta := fmt.Sprintf("%s@%s:%d", user, host, port)
	if len(s.Tags) > 0 {
		meta += " [" + strings.Join(s.Tags, ",") + "]"
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
