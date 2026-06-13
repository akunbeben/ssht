package tui

import (
	"fmt"
	"strings"
)

func (m *model) renderMainHeader(width int) string {
	vpnState := "vpn none"
	if m.profile.VPN != nil {
		vpnState = "vpn " + vpnType(m.profile.VPN)
	}

	privacy := "privacy off"
	if m.masked {
		privacy = "privacy on"
	}

	syncState := "sync off"
	if m.cfg.SyncEnabled {
		syncState = "sync on"
	}

	count := fmt.Sprintf("%d/%d servers", len(m.filtered), len(m.servers))
	if strings.TrimSpace(m.search) == "" {
		count = fmt.Sprintf("%d servers", len(m.servers))
	}

	parts := []string{
		titleStyle.Render("ssht"),
		profileActiveStyle.Render(m.profileName),
		dimStyle.Render(count),
		badgeFor(vpnState, m.profile.VPN != nil),
		badgeFor(privacy, m.masked),
	}
	if width >= 72 {
		parts = append(parts, dimStyle.Render(syncState))
	}

	header := strings.Join(parts, "  ")
	return header
}

func (m *model) renderSearchLine(width int) string {
	if m.mode == modeSearch {
		query := m.search
		if query == "" {
			query = "type to filter"
		}
		line := fmt.Sprintf("Search: %s  %d matches  Enter keep  Esc clear", query, len(m.filtered))
		return focusedInputStyle.Render(truncate(line, width))
	}

	line := "Search: / to filter by name, address, user, tag, or note"
	return dimStyle.Render(truncate(line, width))
}

func badgeFor(label string, active bool) string {
	if active {
		return goodBadgeStyle.Render(label)
	}
	return badgeStyle.Render(label)
}
