package tui

import (
	"strings"
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
	return m.visibleServerRowsFor(height, m.width)
}

func (m *model) visibleServerRowsFor(height, width int) []string {
	lineHeight := 1
	if width < 55 {
		lineHeight = 2
	}

	numVisible := height / lineHeight
	if numVisible == 0 {
		numVisible = 1
	}

	if len(m.filtered) == 0 {
		return padRenderedRows([]string{dimStyle.Render("  no servers - press a to add")}, height)
	}

	// Calculate viewport
	start := max(m.index-numVisible+1, 0)
	end := min(start+numVisible, len(m.filtered))

	rows := make([]string, 0, height)
	for i := start; i < end; i++ {
		rows = append(rows, renderServerRow(m.filtered[i], m.profile.VPN, i == m.index, m.masked, width))
	}

	return padRenderedRows(rows, height)
}

func (m *model) visiblePubkeyRows(height int) []string {
	items := make([]string, 0, len(m.pubkeys)+1)
	if len(m.pubkeys) == 0 {
		items = append(items, dimStyle.Render(inactiveCursor+"no pubkeys in ~/.ssh"))
	} else {
		for i, path := range m.pubkeys {
			label := inactiveCursor + path
			if i == m.pubIndex {
				label = selectedStyle.Render(activeCursor + path)
			}
			items = append(items, label)
		}
	}
	gen := inactiveCursor + "+ Generate new ed25519 keypair"
	if m.pubIndex == len(m.pubkeys) {
		gen = selectedStyle.Render(activeCursor + "+ Generate new ed25519 keypair")
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

func padRenderedRows(rows []string, height int) []string {
	for lipgloss.Height(strings.Join(rows, "\n")) < height {
		rows = append(rows, "")
	}
	return rows
}

func pinFooter(content, footer string, height int) string {
	content = strings.TrimRight(content, "\n")
	footer = strings.TrimRight(footer, "\n")
	gap := height - lipgloss.Height(content) - lipgloss.Height(footer)
	if gap < 1 {
		gap = 1
	}
	return content + strings.Repeat("\n", gap) + footer
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
