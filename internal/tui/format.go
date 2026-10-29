package tui

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/charmbracelet/x/ansi"

	"github.com/akunbeben/ssht/internal/config"
)

var vpnTypes = []string{"wireguard", "shadowsocks", "trojan", "openvpn"}

func port(s config.Server) int {
	if s.Port == 0 {
		return 22
	}
	return s.Port
}

func mask(s string, masked bool) string {
	if !masked || s == "" {
		return s
	}
	return strings.Repeat("•", min(len([]rune(s)), 8))
}

func address(s config.Server, masked bool) string {
	addr := mask(s.User, masked) + "@" + mask(s.Host, masked)
	if p := port(s); p != 22 {
		addr += ":" + mask(strconv.Itoa(p), masked)
	}
	return addr
}

func sshCommand(s config.Server, masked bool) string {
	parts := []string{"ssh"}
	if s.KeyPath != "" {
		parts = append(parts, "-i", mask(s.KeyPath, masked))
	}
	if p := port(s); p != 22 {
		parts = append(parts, "-p", mask(strconv.Itoa(p), masked))
	}
	parts = append(parts, mask(s.User, masked)+"@"+mask(s.Host, masked))
	return strings.Join(parts, " ")
}

// normalizeVPNType maps config aliases to the names shown in the TUI.
func normalizeVPNType(t string) string {
	switch strings.ToLower(strings.TrimSpace(t)) {
	case "", "wg", "wireguard":
		return "wireguard"
	case "ss", "shadowsocks":
		return "shadowsocks"
	case "ovpn", "openvpn":
		return "openvpn"
	}
	return t
}

// effectiveVPN returns the VPN a connection uses and where it comes from.
func effectiveVPN(s config.Server, profile *config.VPNConf) (string, string) {
	switch {
	case s.VPN != nil:
		return normalizeVPNType(s.VPN.Type), "server"
	case profile != nil:
		return normalizeVPNType(profile.Type), "profile"
	}
	return "", ""
}

func shortPath(p string) string {
	if home, err := os.UserHomeDir(); err == nil && strings.HasPrefix(p, home) {
		return "~" + strings.TrimPrefix(p, home)
	}
	return p
}

// fit truncates s to width cells and pads it with spaces to exactly width.
func fit(s string, width int) string {
	if width <= 0 {
		return ""
	}
	s = ansi.Truncate(s, width, "…")
	return s + strings.Repeat(" ", max(width-ansi.StringWidth(s), 0))
}

func plural(n int, word string) string {
	if n == 1 {
		return fmt.Sprintf("1 %s", word)
	}
	return fmt.Sprintf("%d %ss", n, word)
}
