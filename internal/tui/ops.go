package tui

import (
	"fmt"
	"slices"
	"sort"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/akunbeben/ssht/internal/config"
)

func (m *model) profile() config.Profile { return m.cfg.Profiles[m.profileName] }

func (m *model) selected() (config.Server, bool) {
	if len(m.filtered) == 0 {
		return config.Server{}, false
	}
	return m.filtered[m.cursor], true
}

func (m *model) applyFilter() {
	id := ""
	if s, ok := m.selected(); ok {
		id = s.ID
	}
	if m.filter.Value() == "" {
		m.filtered = sortServers(m.profile().Servers, m.cfg.SortBy, m.cfg.SortDesc)
	} else {
		m.filtered = filterServers(m.profile().Servers, m.filter.Value())
	}
	m.selectID(id)
}

// selectID moves the cursor to the server with id, or clamps it when absent.
func (m *model) selectID(id string) {
	for i, s := range m.filtered {
		if s.ID == id {
			m.cursor = i
			m.clampScroll()
			return
		}
	}
	m.cursor = min(m.cursor, max(len(m.filtered)-1, 0))
	m.clampScroll()
}

func (m *model) moveCursor(delta int) {
	m.cursor = max(min(m.cursor+delta, len(m.filtered)-1), 0)
	m.clampScroll()
}

func (m *model) clampScroll() {
	rows := m.listRows()
	if m.cursor < m.offset {
		m.offset = m.cursor
	}
	if m.cursor >= m.offset+rows {
		m.offset = m.cursor - rows + 1
	}
	m.offset = max(min(m.offset, len(m.filtered)-rows), 0)
}

func (m *model) profileNames() []string {
	names := make([]string, 0, len(m.cfg.Profiles))
	for n := range m.cfg.Profiles {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

// commit stores p as the current profile, saves config, and refreshes the list.
func (m *model) commit(p config.Profile, ok string) tea.Cmd {
	m.cfg.Profiles[m.profileName] = p
	m.applyFilter()
	return m.save(ok)
}

func (m *model) save(ok string) tea.Cmd {
	if err := config.Save(m.cfg); err != nil {
		return m.setStatus(statusErr, "Couldn't save config: "+err.Error())
	}
	return m.setStatus(statusOK, ok)
}

func endpointKey(s config.Server) string {
	return fmt.Sprintf("%s@%s:%d", s.User, s.Host, port(s))
}

func (m *model) upsertServer(s config.Server) tea.Cmd {
	p := m.profile()
	i := slices.IndexFunc(p.Servers, func(x config.Server) bool { return x.ID == s.ID })
	verb := "Updated"
	if i >= 0 {
		p.Servers = slices.Clone(p.Servers)
		p.Servers[i] = s
	} else {
		p.Servers = append(slices.Clone(p.Servers), s)
		verb = "Added"
	}
	p.ImportExceptions = slices.DeleteFunc(slices.Clone(p.ImportExceptions), func(k string) bool { return k == endpointKey(s) })
	cmd := m.commit(p, fmt.Sprintf("%s %s", verb, s.Name))
	m.selectID(s.ID)
	return cmd
}

// removeServer drops s from p and records its endpoint so history import skips it.
func removeServer(p config.Profile, s config.Server) config.Profile {
	p.Servers = slices.DeleteFunc(slices.Clone(p.Servers), func(x config.Server) bool { return x.ID == s.ID })
	if k := endpointKey(s); !slices.Contains(p.ImportExceptions, k) {
		p.ImportExceptions = append(slices.Clone(p.ImportExceptions), k)
	}
	return p
}

func (m *model) deleteServer(s config.Server) tea.Cmd {
	return m.commit(removeServer(m.profile(), s), "Deleted "+s.Name)
}

func (m *model) moveServer(s config.Server, target string) tea.Cmd {
	t := m.cfg.Profiles[target]
	t.Name = target
	t.Servers = append(slices.DeleteFunc(slices.Clone(t.Servers), func(x config.Server) bool { return x.Name == s.Name }), s)
	t.ImportExceptions = slices.DeleteFunc(slices.Clone(t.ImportExceptions), func(k string) bool { return k == endpointKey(s) })
	m.cfg.Profiles[target] = t
	return m.commit(removeServer(m.profile(), s), fmt.Sprintf("Moved %s to %s", s.Name, target))
}

func (m *model) switchProfile(name string) tea.Cmd {
	m.profileName = name
	m.cfg.LastProfile = name
	m.filter.SetValue("")
	m.cursor, m.offset = 0, 0
	m.applyFilter()
	return m.save(fmt.Sprintf("Switched to %s", name))
}

func (m *model) createProfile(name string) error {
	if err := config.ValidateProfileName(name); err != nil {
		return err
	}
	if _, ok := m.cfg.Profiles[name]; ok {
		return fmt.Errorf("profile %q already exists", name)
	}
	m.cfg.Profiles[name] = config.Profile{Name: name, Servers: []config.Server{}}
	return nil
}
