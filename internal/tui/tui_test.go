package tui

import (
	"slices"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"

	"github.com/akunbeben/ssht/internal/config"
)

func testModel(t *testing.T) *model {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	cfg := &config.Config{LastProfile: "work", Profiles: map[string]config.Profile{
		"work": {Name: "work", VPN: &config.VPNConf{Type: "wg", ConfPath: "~/wg0.conf"}, Servers: []config.Server{
			{ID: "1", Name: "api-prod", Host: "10.0.1.12", User: "deploy", Tags: []string{"prod"}, Note: "Main API box with a long note that has to wrap."},
			{ID: "2", Name: "db", Host: "db.internal.example.com", Port: 2222, User: "postgres", KeyPath: "~/.ssh/db"},
		}},
		"home": {Name: "home", Servers: []config.Server{{ID: "9", Name: "db", Host: "old", User: "root"}}},
	}}
	return newModel(cfg, "work")
}

func press(m *model, keys ...string) {
	for _, k := range keys {
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(k)}
		switch k {
		case "enter":
			msg = tea.KeyMsg{Type: tea.KeyEnter}
		case "esc":
			msg = tea.KeyMsg{Type: tea.KeyEsc}
		case "tab":
			msg = tea.KeyMsg{Type: tea.KeyTab}
		case "ctrl+s":
			msg = tea.KeyMsg{Type: tea.KeyCtrlS}
		}
		m.Update(msg)
	}
}

func TestViewsFitTerminal(t *testing.T) {
	m := testModel(t)
	for _, open := range [][]string{nil, {"/", "a"}, {"e"}, {"a"}, {"v"}, {"S"}, {"d"}, {"m"}, {"p"}, {"?"}, {":"}, {"K"}} {
		for _, size := range [][2]int{{140, 30}, {80, 24}, {40, 12}} {
			m.Update(tea.WindowSizeMsg{Width: size[0], Height: size[1]})
			press(m, open...)
			lines := strings.Split(m.View(), "\n")
			if len(lines) != size[1] {
				t.Errorf("%v at %v: %d lines", open, size, len(lines))
			}
			for _, l := range lines {
				if w := ansi.StringWidth(l); w > size[0] {
					t.Errorf("%v at %v: line width %d", open, size, w)
				}
			}
			press(m, "esc")
			m.filter.SetValue("")
			m.applyFilter()
		}
	}
}

func TestDeleteSkipsFutureImport(t *testing.T) {
	m := testModel(t)
	press(m, "d", "y")
	p := m.profile()
	if len(p.Servers) != 1 || !slices.Contains(p.ImportExceptions, "deploy@10.0.1.12:22") {
		t.Fatalf("servers=%v exceptions=%v", p.Servers, p.ImportExceptions)
	}
}

func TestMoveReplacesSameNameAfterConfirm(t *testing.T) {
	m := testModel(t)
	press(m, "j", "m", "enter")
	if _, ok := m.dialog.(*confirmDialog); !ok {
		t.Fatalf("expected replace confirmation, got %T", m.dialog)
	}
	press(m, "y")
	home := m.cfg.Profiles["home"]
	if len(home.Servers) != 1 || home.Servers[0].ID != "2" {
		t.Fatalf("home servers = %v", home.Servers)
	}
	if len(m.profile().Servers) != 1 {
		t.Fatalf("work servers = %v", m.profile().Servers)
	}
}

func TestAddServerValidatesAndSaves(t *testing.T) {
	m := testModel(t)
	m.profile() // ensure profile exists
	p := m.profile()
	p.ImportExceptions = []string{"ops@new:22"}
	m.cfg.Profiles["work"] = p

	press(m, "a", "api-prod", "tab", "new", "tab", "ops", "ctrl+s")
	if f, ok := m.dialog.(*form); !ok || f.err == nil {
		t.Fatalf("expected duplicate name error, got %T", m.dialog)
	}
	f := m.dialog.(*form)
	f.fields[0].input.SetValue("new")
	press(m, "ctrl+s")
	if m.dialog != nil {
		t.Fatalf("form still open: %v", f.err)
	}
	p = m.profile()
	if len(p.Servers) != 3 || len(p.ImportExceptions) != 0 {
		t.Fatalf("servers=%d exceptions=%v", len(p.Servers), p.ImportExceptions)
	}
	if s, _ := m.selected(); s.Name != "new" || s.Port != 22 {
		t.Fatalf("selected %+v", s)
	}
}

func TestSearchEnterConnectsTopMatch(t *testing.T) {
	m := testModel(t)
	press(m, "/", "d", "b", "enter")
	if m.action.Type != ActionConnect || m.action.Server.ID != "2" {
		t.Fatalf("action = %+v", m.action)
	}
}

func TestVimKeysOnChoiceField(t *testing.T) {
	m := testModel(t)
	press(m, "S", "l")
	f := m.dialog.(*form)
	if f.value("Direction") != "upload" {
		t.Fatalf("l: direction = %s", f.value("Direction"))
	}
	press(m, "h")
	if f.value("Direction") != "download" {
		t.Fatalf("h: direction = %s", f.value("Direction"))
	}
	press(m, "j")
	if f.focus != 1 {
		t.Fatalf("j: focus = %d", f.focus)
	}
	press(m, "x", "k")
	if f.focus != 1 || f.value("Remote path") != "xk" {
		t.Fatalf("text field must keep j/k as input: focus=%d value=%q", f.focus, f.value("Remote path"))
	}
}

func TestCtrlJKMovesFilteredLists(t *testing.T) {
	m := testModel(t)
	press(m, "/")
	m.Update(tea.KeyMsg{Type: tea.KeyCtrlJ})
	if m.cursor != 1 || !m.filter.Focused() {
		t.Fatalf("ctrl+j in search: cursor=%d", m.cursor)
	}
	m.Update(tea.KeyMsg{Type: tea.KeyCtrlK})
	if m.cursor != 0 || m.dialog != nil {
		t.Fatalf("ctrl+k in search: cursor=%d dialog=%T", m.cursor, m.dialog)
	}
	press(m, "esc", ":")
	m.Update(tea.KeyMsg{Type: tea.KeyCtrlJ})
	if p := m.dialog.(*picker); p.cursor != 1 {
		t.Fatalf("ctrl+j in palette: cursor=%d", p.cursor)
	}
}

func TestSortCyclesAndPersists(t *testing.T) {
	m := testModel(t)
	p := m.profile()
	p.Servers = append(p.Servers, config.Server{ID: "3", Name: "Cache", Host: "a.example.com", User: "ops"})
	m.cfg.Profiles["work"] = p
	m.applyFilter()

	names := func() string {
		var out []string
		for _, s := range m.filtered {
			out = append(out, s.Name)
		}
		return strings.Join(out, ",")
	}
	if got := names(); got != "api-prod,db,Cache" {
		t.Fatalf("added order = %s", got)
	}
	press(m, "o")
	if got := names(); got != "api-prod,Cache,db" || m.cfg.SortBy != sortName {
		t.Fatalf("name sort = %s", got)
	}
	press(m, "O")
	if got := names(); got != "db,Cache,api-prod" {
		t.Fatalf("name desc = %s", got)
	}
	press(m, "o")
	if got := names(); got != "db,Cache,api-prod" || m.cfg.SortBy != sortAddress {
		t.Fatalf("address desc = %s", got)
	}
	if p := m.profile(); p.Servers[0].Name != "api-prod" {
		t.Fatalf("sorting must not reorder stored servers: %v", p.Servers)
	}
	loaded, err := config.Load()
	if err != nil || loaded.SortBy != sortAddress || !loaded.SortDesc {
		t.Fatalf("persisted sort = %q desc=%v err=%v", loaded.SortBy, loaded.SortDesc, err)
	}
}
