package tui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/akunbeben/ssht/internal/config"
)

type clearStatusMsg int

const statusTimeout = 3 * time.Second

func (m *model) setStatus(msg string, style lipgloss.Style) tea.Cmd {
	m.status = msg
	m.statusStyle = style
	m.statusSeq++
	seq := m.statusSeq
	return tea.Tick(statusTimeout, func(_ time.Time) tea.Msg {
		return clearStatusMsg(seq)
	})
}

func (m *model) clearStatus() {
	m.status = ""
	m.err = nil
}

func (m *model) syncServers() {
	m.servers = m.profile.Servers
	m.applyFilter()

	m.cfg.Profiles[m.profileName] = m.profile
	_ = config.Save(m.cfg)
}

func (m *model) saveProfile() {
	m.cfg.Profiles[m.profileName] = m.profile
}

func (m *model) applyFilter() {
	m.filtered = filterServers(m.servers, m.search)
	if m.index >= len(m.filtered) {
		m.index = 0
	}
}

func (m *model) moveList(delta int) {
	if len(m.filtered) == 0 {
		m.index = 0
		return
	}
	next := m.index + delta
	if next < 0 {
		next = 0
	}
	if next >= len(m.filtered) {
		next = len(m.filtered) - 1
	}
	m.index = next
}

func (m *model) movePubkey(delta int) {
	maxIndex := len(m.pubkeys)
	next := m.pubIndex + delta
	if next < 0 {
		next = 0
	}
	if next > maxIndex {
		next = maxIndex
	}
	m.pubIndex = next
}

func (m *model) renderFullScreen(content string) string {
	w, h := m.innerSize()
	rendered := boxStyle.Width(w).Height(h).Render(content)
	return lipgloss.Place(m.width, m.height, lipgloss.Left, lipgloss.Top, rendered)
}

func (m *model) innerSize() (int, int) {
	frameW, frameH := boxStyle.GetFrameSize()
	w := max(m.width-frameW, 20)
	h := max(m.height-frameH, 8)
	return w, h
}

func (m *model) visibleServerRows(height int) []string {
	if len(m.filtered) == 0 {
		return padRows([]string{dimStyle.Render("  no servers — press a to add")}, height)
	}
	start := max(m.index-height+1, 0)
	end := min(start+height, len(m.filtered))
	rows := make([]string, 0, height)
	for i := start; i < end; i++ {
		rows = append(rows, renderServerRow(m.filtered[i], m.profile.VPN, i == m.index, m.masked, m.width))
	}
	return padRows(rows, height)
}

func (m *model) visiblePubkeyRows(height int) []string {
	items := make([]string, 0, len(m.pubkeys)+1)
	if len(m.pubkeys) == 0 {
		items = append(items, dimStyle.Render("  no pubkeys in ~/.ssh"))
	} else {
		for i, path := range m.pubkeys {
			label := "  " + path
			if i == m.pubIndex {
				label = selectedStyle.Render("> " + path)
			}
			items = append(items, label)
		}
	}
	gen := "  + Generate keypair baru"
	if m.pubIndex == len(m.pubkeys) {
		gen = selectedStyle.Render("> + Generate keypair baru")
	}
	items = append(items, gen)

	start := max(m.pubIndex-height+1, 0)
	end := min(start+height, len(items))
	return padRows(items[start:end], height)
}

func padRows(rows []string, height int) []string {
	for len(rows) < height {
		rows = append(rows, "")
	}
	return rows
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func trim(s string, limit int) string {
	if len(s) <= limit {
		return s
	}
	return s[:limit] + "..."
}

func containsString(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}

func removeString(slice []string, s string) []string {
	result := make([]string, 0, len(slice))
	for _, v := range slice {
		if v != s {
			result = append(result, v)
		}
	}
	return result
}
