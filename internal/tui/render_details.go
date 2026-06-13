package tui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/akunbeben/ssht/internal/config"
)

func renderSelectedDetails(s config.Server, profileVPN *config.VPNConf, profileName string, masked bool, width int) string {
	if width < 24 {
		return ""
	}

	lines := []string{
		titleStyle.Render("Selected"),
		detailLine("Name", maskText(s.Name, masked, 14), width),
		detailLine("Address", serverAddress(s, masked, true), width),
		detailLine("VPN", vpnSummary(s.VPN, profileVPN, profileName), width),
		detailLine("Key", keySummary(s, masked), width),
	}

	if len(s.Tags) > 0 {
		lines = append(lines, detailLine("Tags", strings.Join(s.Tags, ", "), width))
	}
	if strings.TrimSpace(s.Note) != "" {
		lines = append(lines, detailLine("Note", maskText(s.Note, masked, 18), width))
	}
	lines = append(lines, detailLine("Command", commandPreview(s, masked), width))

	body := strings.Join(lines, "\n")
	return detailBoxStyle.Width(width - detailBoxStyle.GetHorizontalFrameSize()).Render(body)
}

func renderEmptyState(width int) string {
	lines := []string{
		titleStyle.Render("No servers yet"),
		"Add your first SSH target or switch to another profile.",
		"",
		"a  add server",
		"p  switch profile",
		"K  manage public keys",
		"v  configure profile VPN",
	}
	content := strings.Join(lines, "\n")
	if width < 40 {
		return dimStyle.Copy().Width(width).Render(content)
	}
	return detailBoxStyle.Width(width - detailBoxStyle.GetHorizontalFrameSize()).Render(content)
}

func detailLine(label, value string, width int) string {
	line := fmt.Sprintf("%-8s %s", label+":", value)
	return truncate(line, max(width-4, 1))
}

func serverAddress(s config.Server, masked bool, includeDefaultPort bool) string {
	user := s.User
	host := s.Host
	port := normalizedPort(s)
	if masked {
		user = maskText(user, true, 8)
		host = maskText(host, true, 12)
	}

	addr := fmt.Sprintf("%s@%s", user, host)
	if includeDefaultPort || port != 22 {
		portText := strconv.Itoa(port)
		if masked {
			portText = "*****"
		}
		addr += ":" + portText
	}
	return addr
}

func commandPreview(s config.Server, masked bool) string {
	if masked {
		return "ssh ********@********"
	}

	parts := []string{"ssh"}
	if s.KeyPath != "" {
		parts = append(parts, "-i", s.KeyPath)
	}
	if normalizedPort(s) != 22 {
		parts = append(parts, "-p", strconv.Itoa(normalizedPort(s)))
	}
	parts = append(parts, fmt.Sprintf("%s@%s", s.User, s.Host))
	return strings.Join(parts, " ")
}

func vpnSummary(serverVPN, profileVPN *config.VPNConf, profileName string) string {
	if serverVPN != nil {
		return vpnType(serverVPN) + " from server override"
	}
	if profileVPN != nil {
		return vpnType(profileVPN) + " from profile " + profileName
	}
	return "none"
}

func vpnBadge(serverVPN, profileVPN *config.VPNConf) string {
	if serverVPN != nil {
		return "vpn server"
	}
	if profileVPN != nil {
		return "vpn profile"
	}
	return "vpn none"
}

func vpnType(v *config.VPNConf) string {
	if v == nil {
		return "none"
	}
	if v.Type == "" {
		return "wireguard"
	}
	return v.Type
}

func keyBadge(s config.Server) string {
	if s.KeyPath == "" {
		return "key default"
	}
	return "key set"
}

func keySummary(s config.Server, masked bool) string {
	if s.KeyPath == "" {
		return "default ssh identity"
	}
	return maskText(s.KeyPath, masked, 18)
}

func normalizedPort(s config.Server) int {
	if s.Port == 0 {
		return 22
	}
	return s.Port
}

func maskText(s string, masked bool, limit int) string {
	if !masked {
		return s
	}
	if s == "" {
		return ""
	}
	return strings.Repeat("*", min(len(s), limit))
}

func fitBlock(s string, height int) string {
	if height <= 0 || s == "" {
		return ""
	}
	lines := strings.Split(s, "\n")
	if len(lines) <= height {
		return s
	}
	return strings.Join(lines[:height], "\n")
}

func joinPanels(left, right string, leftWidth, rightWidth int) string {
	return lipgloss.JoinHorizontal(lipgloss.Top,
		lipgloss.NewStyle().Width(leftWidth).Render(left),
		lipgloss.NewStyle().Width(2).Render(""),
		lipgloss.NewStyle().Width(rightWidth).Render(right),
	)
}
