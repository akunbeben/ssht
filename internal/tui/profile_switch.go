package tui

import (
	"sort"
	"strings"

	"github.com/akunbeben/ssht/internal/config"
	"github.com/charmbracelet/lipgloss"
)

type profileSwitchState struct {
	names   []string
	current string
	index   int
}

func newProfileSwitchState(cfg *config.Config, currentName string) profileSwitchState {
	names := make([]string, 0, len(cfg.Profiles))
	for n := range cfg.Profiles {
		names = append(names, n)
	}
	sort.Strings(names)

	idx := 0
	for i, n := range names {
		if n == currentName {
			idx = i
			break
		}
	}

	return profileSwitchState{
		names:   names,
		current: currentName,
		index:   idx,
	}
}

func (ps *profileSwitchState) move(delta int) {
	ps.moveWithMax(delta, len(ps.names)-1)
}

func (ps *profileSwitchState) moveWithMax(delta int, maxIdx int) {
	if maxIdx < 0 {
		return
	}
	next := ps.index + delta
	if next < 0 {
		next = 0
	}
	if next > maxIdx {
		next = maxIdx
	}
	ps.index = next
}

func (ps *profileSwitchState) selected() string {
	if len(ps.names) == 0 {
		return ""
	}
	return ps.names[ps.index]
}

func (ps *profileSwitchState) view(width, height int, helperWrapped bool) string {
	var body strings.Builder

	body.WriteString(titleStyle.Render("Switch Profile") + "\n\n")

	for i, name := range ps.names {
		cursor := "  "
		active := ""
		if name == ps.current {
			active = " ✓"
		}

		label := name + active
		if i == ps.index {
			cursor = focusedInputStyle.Render("▸ ")
			label = selectedStyle.Render(label)
		} else if name == ps.current {
			label = profileActiveStyle.Render(label)
		} else {
			label = dimStyle.Render(label)
		}
		body.WriteString(cursor + label + "\n")
	}

	help := "j/k: navigate · Enter: switch · Esc: cancel"
	helpStyleWrap := helpStyle.Copy().Width(width)
	if !helperWrapped {
		help = truncate(help, width)
		helpStyleWrap = helpStyleWrap.MaxHeight(1)
	}
	renderedHelp := helpStyleWrap.Render(help)

	bodyContent := body.String()
	gap := height - lipgloss.Height(bodyContent) - lipgloss.Height(renderedHelp)
	if gap > 0 {
		return bodyContent + strings.Repeat("\n", gap) + renderedHelp
	}
	return bodyContent + "\n\n" + renderedHelp
}
