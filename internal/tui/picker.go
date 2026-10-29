package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/sahilm/fuzzy"
)

type pickItem struct {
	title    string
	meta     string
	disabled bool
}

// picker is a selectable list, optionally filtered by typing.
type picker struct {
	heading    string
	note       string
	items      []pickItem
	cursor     int
	offset     int
	filterable bool
	query      textinput.Model
	matches    []int
	rows       int
	onPick     func(m *model, index int) (dialog, tea.Cmd)
}

func newPicker(heading string, items []pickItem, onPick func(m *model, index int) (dialog, tea.Cmd)) *picker {
	p := &picker{heading: heading, items: items, rows: 12, onPick: onPick}
	p.refilter()
	return p
}

func (p *picker) withFilter() *picker {
	p.filterable = true
	p.query = newInput("type to filter", "")
	p.query.Focus()
	return p
}

func (p *picker) refilter() {
	p.matches = p.matches[:0]
	q := ""
	if p.filterable {
		q = strings.TrimSpace(p.query.Value())
	}
	if q == "" {
		for i := range p.items {
			p.matches = append(p.matches, i)
		}
	} else {
		titles := make([]string, len(p.items))
		for i, it := range p.items {
			titles[i] = it.title
		}
		for _, r := range fuzzy.Find(q, titles) {
			p.matches = append(p.matches, r.Index)
		}
	}
	p.cursor, p.offset = 0, 0
}

func (p *picker) move(delta int) {
	p.cursor = max(min(p.cursor+delta, len(p.matches)-1), 0)
	if p.cursor < p.offset {
		p.offset = p.cursor
	}
	if p.cursor >= p.offset+p.rows {
		p.offset = p.cursor - p.rows + 1
	}
}

func (p *picker) title() string { return p.heading }

func (p *picker) update(m *model, msg tea.KeyMsg) (dialog, tea.Cmd) {
	switch {
	case key.Matches(msg, keyClose):
		return nil, nil
	case key.Matches(msg, keySelect):
		if len(p.matches) == 0 || p.items[p.matches[p.cursor]].disabled {
			return p, nil
		}
		return p.onPick(m, p.matches[p.cursor])
	case key.Matches(msg, keyListUp), !p.filterable && key.Matches(msg, keys.Up):
		p.move(-1)
		return p, nil
	case key.Matches(msg, keyListDn), !p.filterable && key.Matches(msg, keys.Down):
		p.move(1)
		return p, nil
	}
	if !p.filterable {
		return p, nil
	}
	var cmd tea.Cmd
	before := p.query.Value()
	p.query, cmd = p.query.Update(msg)
	if p.query.Value() != before {
		p.refilter()
	}
	return p, cmd
}

func (p *picker) view(m *model, w, h int) string {
	var rows []string
	if p.note != "" {
		rows = append(rows, mutedStyle.Width(w).Render(p.note), "")
	}
	if p.filterable {
		p.query.Width = max(w-3, 1)
		rows = append(rows, accentStyle.Render("› ")+p.query.View(), "")
	}
	if len(p.matches) == 0 {
		rows = append(rows, faintStyle.Render("No matches"))
	}
	p.rows = max(min(12, h-len(rows)), 1)
	if len(p.matches) > p.rows {
		p.rows = max(p.rows-1, 1) // room for the position line
	}
	p.move(0)

	metaW := 0
	for _, i := range p.matches {
		metaW = max(metaW, len([]rune(p.items[i].meta)))
	}
	metaW = min(metaW, w/2)

	end := min(p.offset+p.rows, len(p.matches))
	for row := p.offset; row < end; row++ {
		it := p.items[p.matches[row]]
		titleW := w - 2 - metaW - 1
		line := fit(it.title, titleW) + " " + fit(it.meta, metaW)
		switch {
		case row == p.cursor:
			rows = append(rows, accentStyle.Background(colorSelBg).Render("▌ ")+selectedStyle.Render(line))
		case it.disabled:
			rows = append(rows, faintStyle.Render("  "+line))
		default:
			rows = append(rows, "  "+textStyle.Render(fit(it.title, titleW))+" "+faintStyle.Render(fit(it.meta, metaW)))
		}
	}
	if len(p.matches) > p.rows {
		rows = append(rows, faintStyle.Render(fmt.Sprintf("  %d of %d", p.cursor+1, len(p.matches))))
	}
	return strings.Join(rows, "\n")
}

func (p *picker) bindings() []key.Binding {
	return []key.Binding{keySelect, keyListUp, keyListDn, keyClose}
}
