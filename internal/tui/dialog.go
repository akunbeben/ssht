package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// dialog is a modal drawn over the server list. update returns the next
// dialog to show, or nil to close it.
type dialog interface {
	title() string
	update(m *model, msg tea.KeyMsg) (dialog, tea.Cmd)
	view(m *model, width, height int) string
	bindings() []key.Binding
}

// dialogView frames the active dialog. Short terminals get a compact frame,
// and the body is limited so the dialog never exceeds the screen.
func (m *model) dialogView() string {
	style, gap := dialogStyle, "\n"
	if m.height < 24 {
		style, gap = style.Padding(0, 1), ""
	}
	if c, ok := m.dialog.(*confirmDialog); ok && c.danger {
		style = style.BorderForeground(colorBad)
	}

	want := 60
	if _, ok := m.dialog.(*helpDialog); ok {
		want = 72
	}
	w := max(min(want, m.width-2-style.GetHorizontalFrameSize()), 16)
	h := max(m.height-style.GetVerticalFrameSize()-2-2*len(gap), 1)

	body := lipgloss.NewStyle().MaxHeight(h).Render(m.dialog.view(m, w, h))
	content := accentStyle.Render(m.dialog.title()) + "\n" + gap + body + "\n" + gap + m.shortHelp(w, m.dialog.bindings()...)
	return style.Width(w + style.GetHorizontalPadding()).Render(content)
}

// dim flattens the background so the dialog stands out.
func dim(s string) string {
	lines := strings.Split(ansi.Strip(s), "\n")
	for i, l := range lines {
		lines[i] = faintStyle.Render(l)
	}
	return strings.Join(lines, "\n")
}

// overlay draws fg centered on top of bg.
func overlay(bg, fg string, width int) string {
	bgLines := strings.Split(bg, "\n")
	fgLines := strings.Split(fg, "\n")
	fgW := lipgloss.Width(fg)
	x := max((width-fgW)/2, 0)
	y := max((len(bgLines)-len(fgLines))/2, 0)
	for i, line := range fgLines {
		row := y + i
		if row >= len(bgLines) {
			bgLines = append(bgLines, "")
		}
		left := fit(ansi.Truncate(bgLines[row], x, ""), x)
		right := ansi.TruncateLeft(bgLines[row], x+fgW, "")
		bgLines[row] = left + line + right
	}
	return strings.Join(bgLines, "\n")
}

type confirmDialog struct {
	heading string
	body    string
	danger  bool
	onYes   func(m *model) (dialog, tea.Cmd)
	onNo    func(m *model) dialog
}

func (d *confirmDialog) title() string { return d.heading }

func (d *confirmDialog) update(m *model, msg tea.KeyMsg) (dialog, tea.Cmd) {
	switch {
	case key.Matches(msg, keyYes):
		return d.onYes(m)
	case key.Matches(msg, keyNo):
		if d.onNo != nil {
			return d.onNo(m), nil
		}
		return nil, nil
	}
	return d, nil
}

func (d *confirmDialog) view(m *model, w, _ int) string {
	return textStyle.Width(w).Render(d.body)
}

func (d *confirmDialog) bindings() []key.Binding { return []key.Binding{keyYes, keyNo} }

type helpDialog struct{}

func (m *model) openHelp() tea.Cmd {
	m.dialog = &helpDialog{}
	return nil
}

func (d *helpDialog) title() string { return "Keyboard shortcuts" }

func (d *helpDialog) update(m *model, msg tea.KeyMsg) (dialog, tea.Cmd) {
	if key.Matches(msg, keyClose, keys.Help, keys.Quit) {
		return nil, nil
	}
	return d, nil
}

func (d *helpDialog) view(m *model, w, _ int) string {
	hm := m.help
	hm.Width = w
	return hm.FullHelpView([][]key.Binding{
		{keys.Up, keys.Down, keys.Top, keys.Bottom, keys.PageUp, keys.PageDown, keys.Search, keys.Clear},
		{keys.Connect, keys.SCP, keys.Add, keys.Edit, keys.Copy, keys.Move, keys.Delete},
		{keys.Profiles, keys.VPN, keys.VPNToggle, keys.Keys, keys.Sort, keys.SortDir, keys.Privacy, keys.Actions, keys.Quit},
	})
}

func (d *helpDialog) bindings() []key.Binding { return []key.Binding{bind("esc", "close", "esc")} }
