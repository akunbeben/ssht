package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/akunbeben/ssht/internal/vpn"
)

const detailsWidth = 42

// command is a browse action. The same list drives key handling and the palette.
type command struct {
	binding     key.Binding
	name        string
	needsServer bool
	run         func(m *model) tea.Cmd
}

func (m *model) commands() []command {
	return []command{
		{keys.Connect, "Connect", true, (*model).connect},
		{keys.SCP, "Transfer files (SCP)", true, (*model).openSCP},
		{keys.Search, "Search", false, (*model).focusSearch},
		{keys.Add, "Add server", false, (*model).openAdd},
		{keys.Edit, "Edit server", true, (*model).openEdit},
		{keys.Copy, "Duplicate server", true, (*model).openCopy},
		{keys.Move, "Move server to profile", true, (*model).openMove},
		{keys.Delete, "Delete server", true, (*model).openDelete},
		{keys.Profiles, "Switch profile", false, (*model).openProfiles},
		{keys.VPN, "Profile VPN settings", false, (*model).openVPN},
		{keys.VPNToggle, "Toggle system VPN", false, (*model).toggleSystemVPN},
		{keys.Keys, "SSH public keys", false, (*model).openKeys},
		{keys.Sort, "Change sort", false, (*model).cycleSort},
		{keys.SortDir, "Reverse sort", false, (*model).reverseSort},
		{keys.Privacy, "Mask sensitive details", false, (*model).togglePrivacy},
		{keys.Help, "Keyboard shortcuts", false, (*model).openHelp},
	}
}

func (m *model) run(c command) tea.Cmd {
	if c.needsServer && len(m.filtered) == 0 {
		return m.setStatus(statusInfo, "No server selected")
	}
	return c.run(m)
}

func (m *model) updateBrowse(msg tea.KeyMsg) tea.Cmd {
	if m.filter.Focused() {
		return m.updateFilter(msg)
	}
	switch {
	case key.Matches(msg, keys.Clear):
		if m.filter.Value() != "" {
			m.filter.SetValue("")
			m.applyFilter()
			return nil
		}
		return tea.Quit
	case key.Matches(msg, keys.Quit):
		return tea.Quit
	case key.Matches(msg, keys.Actions):
		return m.openPalette()
	case key.Matches(msg, keys.Up):
		m.moveCursor(-1)
	case key.Matches(msg, keys.Down):
		m.moveCursor(1)
	case key.Matches(msg, keys.PageUp):
		m.moveCursor(-m.listRows())
	case key.Matches(msg, keys.PageDown):
		m.moveCursor(m.listRows())
	case key.Matches(msg, keys.Top):
		m.moveCursor(-len(m.filtered))
	case key.Matches(msg, keys.Bottom):
		m.moveCursor(len(m.filtered))
	default:
		for _, c := range m.commands() {
			if key.Matches(msg, c.binding) {
				return m.run(c)
			}
		}
	}
	return nil
}

func (m *model) updateFilter(msg tea.KeyMsg) tea.Cmd {
	switch {
	case key.Matches(msg, keys.Clear):
		m.filter.SetValue("")
		m.filter.Blur()
		m.applyFilter()
		return nil
	case key.Matches(msg, keys.KeepFilter):
		m.filter.Blur()
		return nil
	case key.Matches(msg, keys.Connect):
		m.filter.Blur()
		return m.run(m.commands()[0])
	case key.Matches(msg, keyListUp):
		m.moveCursor(-1)
		return nil
	case key.Matches(msg, keyListDn):
		m.moveCursor(1)
		return nil
	}
	before := m.filter.Value()
	var cmd tea.Cmd
	m.filter, cmd = m.filter.Update(msg)
	if m.filter.Value() != before {
		m.cursor = 0
		m.applyFilter()
		m.cursor, m.offset = 0, 0
	}
	return cmd
}

func (m *model) connect() tea.Cmd {
	s, _ := m.selected()
	m.action = Action{Type: ActionConnect, Server: &s}
	return tea.Quit
}

func (m *model) focusSearch() tea.Cmd {
	return m.filter.Focus()
}

func (m *model) togglePrivacy() tea.Cmd {
	m.masked = !m.masked
	m.cfg.PrivacyMode = m.masked
	if m.masked {
		return m.save("Sensitive details masked")
	}
	return m.save("Sensitive details visible")
}

func (m *model) toggleSystemVPN() tea.Cmd {
	if m.profile().VPN == nil {
		return m.setStatus(statusInfo, "This profile has no VPN. Press v to set one.")
	}
	if !vpn.HasWgQuick() {
		return m.setStatus(statusErr, "wg-quick not found. Install wireguard-tools first.")
	}
	m.action = Action{Type: ActionToggleVPN}
	return tea.Quit
}

// Layout: header, gap, filter, column header, rows, [summary], gap, footer.
func (m *model) wide() bool { return m.width >= 100 }

func (m *model) listRows() int {
	rows := m.height - 6
	if !m.wide() {
		rows--
	}
	return max(rows, 1)
}

func (m *model) browseView() string {
	w := max(m.width-2, 10)
	bodyH := max(m.height-5, 2)
	view := strings.Join([]string{
		m.headerView(w),
		"",
		m.filterView(w),
		m.bodyView(w, bodyH),
		"",
		m.footerView(w),
	}, "\n")
	return lipgloss.NewStyle().Padding(0, 1).Render(view)
}

func (m *model) headerView(w int) string {
	total := len(m.profile().Servers)
	count := plural(total, "server")
	if m.filter.Value() != "" {
		count = fmt.Sprintf("%d of %d", len(m.filtered), total)
	}
	left := accentStyle.Render("ssht") + "  " + boldStyle.Render(m.profileName) + mutedStyle.Render("  "+count)

	var badges []string
	if v := m.profile().VPN; v != nil {
		badges = append(badges, goodStyle.Render("● vpn "+normalizeVPNType(v.Type)))
	}
	if m.masked {
		badges = append(badges, warnStyle.Render("● masked"))
	}
	if m.cfg.SyncEnabled {
		badges = append(badges, mutedStyle.Render("● sync"))
	}
	right := strings.Join(badges, "  ")

	gap := w - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 2 {
		return ansi.Truncate(left, w, "…")
	}
	return left + strings.Repeat(" ", gap) + right
}

func (m *model) filterView(w int) string {
	prompt := faintStyle.Render("/ ")
	switch {
	case m.filter.Focused():
		m.filter.Width = max(w-4, 1)
		return accentStyle.Render("/ ") + m.filter.View()
	case m.filter.Value() != "":
		return fit(prompt+textStyle.Render(m.filter.Value())+faintStyle.Render("  esc to clear"), w)
	}
	return prompt + faintStyle.Render("search")
}

func (m *model) bodyView(w, h int) string {
	if len(m.filtered) == 0 {
		return lipgloss.Place(w, h, lipgloss.Center, lipgloss.Center, m.emptyView())
	}
	if !m.wide() {
		s, _ := m.selected()
		return m.tableView(w, h-1) + "\n" + faintStyle.Render(fit(sshCommand(s, m.masked), w))
	}
	listW := w - detailsWidth - 2
	return lipgloss.JoinHorizontal(lipgloss.Top,
		lipgloss.NewStyle().Width(listW+2).Render(m.tableView(listW, h)),
		m.detailsView(detailsWidth, h),
	)
}

func (m *model) emptyView() string {
	if q := m.filter.Value(); q != "" {
		return lipgloss.JoinVertical(lipgloss.Center,
			boldStyle.Render(fmt.Sprintf("No match for %q", q)),
			mutedStyle.Render("esc to clear"),
		)
	}
	return lipgloss.JoinVertical(lipgloss.Center,
		boldStyle.Render("No servers in "+m.profileName),
		"",
		mutedStyle.Render("a add a server   p switch profile"),
	)
}

type columns struct{ name, addr, vpn, tags int }

func (m *model) columns(w int) columns {
	c := columns{name: 4, addr: 7}
	for _, s := range m.filtered {
		c.name = max(c.name, ansi.StringWidth(mask(s.Name, m.masked)))
		c.addr = max(c.addr, ansi.StringWidth(address(s, m.masked)))
	}
	c.name = min(c.name, 28) + 2
	c.addr = min(c.addr, 36) + 2
	w -= 2 // cursor marker
	if w >= c.name+c.addr+12+8 {
		c.vpn = 12
		c.tags = w - c.name - c.addr - c.vpn
		return c
	}
	if w >= c.name+c.addr+12 {
		c.vpn = w - c.name - c.addr
		return c
	}
	c.name = min(c.name, max(w/2, 8))
	c.addr = max(w-c.name, 0)
	return c
}

func (m *model) tableView(w, h int) string {
	c := m.columns(w)
	nameHead, addrHead := "NAME", "ADDRESS"
	if m.filter.Value() == "" {
		arrow := " ↑"
		if m.cfg.SortDesc {
			arrow = " ↓"
		}
		switch m.cfg.SortBy {
		case sortName:
			nameHead += arrow
		case sortAddress:
			addrHead += arrow
		}
	}
	head := "  " + fit(nameHead, c.name) + fit(addrHead, c.addr) + fit("VPN", c.vpn) + fit("TAGS", c.tags)
	lines := []string{columnStyle.Render(fit(head, w))}

	vpnConf := m.profile().VPN
	end := min(m.offset+h-1, len(m.filtered))
	for i := m.offset; i < end; i++ {
		s := m.filtered[i]
		vpnName, source := effectiveVPN(s, vpnConf)
		if vpnName == "" {
			vpnName = "–"
		}
		cells := []string{
			fit(mask(s.Name, m.masked), c.name),
			fit(address(s, m.masked), c.addr),
			fit(vpnName, c.vpn),
			fit(strings.Join(s.Tags, ", "), c.tags),
		}
		if i == m.cursor {
			lines = append(lines, accentStyle.Background(colorSelBg).Render("▌ ")+selectedStyle.Render(fit(strings.Join(cells, ""), w-2)))
			continue
		}
		vpnStyle := faintStyle
		if source == "server" {
			vpnStyle = mutedStyle
		}
		lines = append(lines, "  "+textStyle.Render(cells[0])+mutedStyle.Render(cells[1])+vpnStyle.Render(cells[2])+faintStyle.Render(cells[3]))
	}
	return lipgloss.NewStyle().Height(h).Render(strings.Join(lines, "\n"))
}

func (m *model) detailsView(w, h int) string {
	s, _ := m.selected()
	inner := w - detailsStyle.GetHorizontalFrameSize()
	row := func(label, value string) string {
		return lipgloss.JoinHorizontal(lipgloss.Top, labelStyle.Render(fit(label, 6)), lipgloss.NewStyle().Width(inner-6).Render(value))
	}

	vpnText := "none"
	if name, source := effectiveVPN(s, m.profile().VPN); name != "" {
		vpnText = name + mutedStyle.Render(" · "+source)
	}
	keyText := "default identity"
	if s.KeyPath != "" {
		keyText = mask(s.KeyPath, m.masked)
	}

	lines := []string{
		accentStyle.Render(ansi.Truncate(mask(s.Name, m.masked), inner, "…")),
		mutedStyle.Render(ansi.Truncate(address(s, m.masked), inner, "…")),
		"",
		row("VPN", vpnText),
		row("Key", keyText),
	}
	if len(s.Tags) > 0 {
		lines = append(lines, row("Tags", strings.Join(s.Tags, ", ")))
	}
	if s.Note != "" && !m.masked {
		lines = append(lines, row("Note", s.Note))
	}
	lines = append(lines, "", faintStyle.Width(inner).Render(sshCommand(s, m.masked)))

	body := lipgloss.NewStyle().MaxHeight(h).Render(strings.Join(lines, "\n"))
	return detailsStyle.Width(w - detailsStyle.GetHorizontalBorderSize()).Height(h).Render(body)
}

func (m *model) footerView(w int) string {
	if m.status != "" {
		style := map[statusKind]lipgloss.Style{statusInfo: mutedStyle, statusOK: goodStyle, statusErr: badStyle}[m.statusKind]
		icon := map[statusKind]string{statusInfo: "• ", statusOK: "✓ ", statusErr: "✗ "}[m.statusKind]
		return style.Render(ansi.Truncate(icon+m.status, w, "…"))
	}
	if m.filter.Focused() {
		return m.shortHelp(w, keys.Connect, keyListUp, keyListDn, keys.KeepFilter, keys.Clear)
	}
	if len(m.filtered) == 0 {
		return m.shortHelp(w, keys.Add, keys.Search, keys.Profiles, keys.Actions, keys.Help, keys.Quit)
	}
	return m.shortHelp(w, keys.Connect, keys.Search, keys.Add, keys.Edit, keys.SCP, keys.Actions, keys.Help, keys.Quit)
}

// shortHelp renders one line of key hints that never exceeds w.
// bubbles/help can overflow when the ellipsis itself does not fit.
func (m *model) shortHelp(w int, bindings ...key.Binding) string {
	m.help.Width = w
	return ansi.Truncate(m.help.ShortHelpView(bindings), w, "")
}
