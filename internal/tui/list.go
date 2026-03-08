package tui

import (
	"fmt"
	"strings"

	"github.com/akunbeben/ssht/internal/config"
)

func renderListHeader() string {
	header := fmt.Sprintf("  %-20s %-20s %-10s %-5s %-15s %s", "NAME", "HOST", "USER", "PORT", "VPN", "TAGS")
	return dimStyle.Render(header)
}

func renderServerRow(s config.Server, profileVPN *config.VPNConf, selected bool, masked bool) string {
	tags := ""
	if len(s.Tags) > 0 {
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

	name := truncate(s.Name, 20)
	hostDisplay := truncate(host, 20)
	userDisplay := truncate(user, 10)
	vpnDisplay = truncate(vpnDisplay, 15)

	row := fmt.Sprintf("%-20s %-20s %-10s %-5s %-15s %s", name, hostDisplay, userDisplay, portDisplay, vpnDisplay, tags)
	if selected {
		return selectedStyle.Render("> " + row)
	}
	return "  " + row
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
