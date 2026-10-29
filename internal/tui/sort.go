package tui

import (
	"cmp"
	"slices"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/akunbeben/ssht/internal/config"
)

// Sort modes. The empty mode keeps the order servers were added in.
const (
	sortAdded   = ""
	sortName    = "name"
	sortAddress = "address"
)

var sortModes = []string{sortAdded, sortName, sortAddress}

// sortServers returns a sorted copy. Search results keep relevance order instead.
func sortServers(servers []config.Server, by string, desc bool) []config.Server {
	out := slices.Clone(servers)
	switch by {
	case sortName:
		slices.SortStableFunc(out, func(a, b config.Server) int {
			return cmp.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name))
		})
	case sortAddress:
		slices.SortStableFunc(out, func(a, b config.Server) int {
			return cmp.Or(
				cmp.Compare(strings.ToLower(a.Host), strings.ToLower(b.Host)),
				cmp.Compare(strings.ToLower(a.User), strings.ToLower(b.User)),
				cmp.Compare(port(a), port(b)),
			)
		})
	}
	if desc {
		slices.Reverse(out)
	}
	return out
}

func sortLabel(by string, desc bool) string {
	label := map[string]string{sortAdded: "date added", sortName: "name", sortAddress: "address"}[by]
	if label == "" {
		label = by
	}
	if desc {
		return label + " ↓"
	}
	return label + " ↑"
}

func (m *model) cycleSort() tea.Cmd {
	i := slices.Index(sortModes, m.cfg.SortBy)
	m.cfg.SortBy = sortModes[(i+1)%len(sortModes)]
	return m.applySort()
}

func (m *model) reverseSort() tea.Cmd {
	m.cfg.SortDesc = !m.cfg.SortDesc
	return m.applySort()
}

func (m *model) applySort() tea.Cmd {
	m.applyFilter()
	msg := "Sorted by " + sortLabel(m.cfg.SortBy, m.cfg.SortDesc)
	if m.filter.Value() != "" {
		msg += ". Search results stay ranked by match."
	}
	return m.save(msg)
}
