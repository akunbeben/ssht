package tui

import (
	"fmt"
	"strings"

	"github.com/akunbeben/ssht/internal/config"
)

func renderListHeader() string {
	header := fmt.Sprintf("  %-20s %-20s %-10s %-5s %s", "NAME", "HOST", "USER", "PORT", "TAGS")
	return dimStyle.Render(header)
}

func renderServerRow(s config.Server, selected bool, masked bool) string {
	tags := ""
	if len(s.Tags) > 0 {
		tags = "[" + strings.Join(s.Tags, ",") + "]"
	}
	port := s.Port
	if port == 0 {
		port = 22
	}

	host := s.Host
	user := s.User
	portDisplay := fmt.Sprintf("%d", port)
	if masked {
		host = strings.Repeat("*", min(len(host), 12))
		user = strings.Repeat("*", min(len(user), 8))
		portDisplay = "*****"
	}

	name := truncate(s.Name, 20)
	hostDisplay := truncate(host, 20)
	userDisplay := truncate(user, 10)

	row := fmt.Sprintf("%-20s %-20s %-10s %-5s %s", name, hostDisplay, userDisplay, portDisplay, tags)
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
