package tui

import (
	"sort"
	"strings"

	"github.com/akunbeben/ssht/internal/config"
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

func (ps *profileSwitchState) view() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Switch Profile") + "\n\n")

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
		b.WriteString(cursor + label + "\n")
	}

	b.WriteString("\n")
	b.WriteString(helpStyle.Render("j/k: navigate · Enter: switch · Esc: cancel"))

	return b.String()
}
