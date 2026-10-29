package tui

import (
	"errors"
	"fmt"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/google/uuid"

	"github.com/akunbeben/ssht/internal/config"
	k "github.com/akunbeben/ssht/internal/key"
)

const (
	vpnInherit = "profile default"
	vpnNone    = "none"
)

func (m *model) openAdd() tea.Cmd {
	m.dialog = m.serverForm("Add server", config.Server{})
	return nil
}

func (m *model) openEdit() tea.Cmd {
	s, _ := m.selected()
	m.dialog = m.serverForm("Edit server", s)
	return nil
}

func (m *model) openCopy() tea.Cmd {
	s, _ := m.selected()
	s.ID = ""
	s.Name += " copy"
	m.dialog = m.serverForm("Duplicate server", s)
	return nil
}

func (m *model) serverForm(heading string, s config.Server) *form {
	portText := ""
	if s.Port != 0 && s.Port != 22 {
		portText = strconv.Itoa(s.Port)
	}
	vpnType, vpnConf := vpnInherit, ""
	if s.VPN != nil {
		vpnType, vpnConf = normalizeVPNType(s.VPN.Type), s.VPN.ConfPath
	}
	hasVPN := func(f *form) bool { return f.value("VPN") != vpnInherit }

	id := s.ID
	submit := func(m *model, f *form) (dialog, tea.Cmd, error) {
		srv, err := serverFromForm(f, id)
		if err != nil {
			return nil, nil, err
		}
		for _, other := range m.profile().Servers {
			if other.ID != srv.ID && other.Name == srv.Name {
				return nil, nil, fmt.Errorf("%q already exists in this profile", srv.Name)
			}
		}
		return nil, m.upsertServer(srv), nil
	}

	return newForm(heading, submit,
		textField("Name", "api-prod", s.Name, "Shown in the list and used by search."),
		textField("Host", "10.0.1.12 or example.com", s.Host, "Hostname or IP only."),
		textField("User", "deploy", s.User, ""),
		textField("Port", "22", portText, ""),
		textField("Key", "~/.ssh/id_ed25519", s.KeyPath, "Leave blank to use your default SSH identities."),
		textField("Tags", "prod, api", strings.Join(s.Tags, ", "), "Comma separated."),
		textField("Note", "", s.Note, ""),
		choiceField("VPN", vpnType, "Profile default uses this profile's VPN, if any.", append([]string{vpnInherit}, vpnTypes...)...),
		withShowIf(textField("VPN config", "~/wg0.conf, ss://…, trojan://…", vpnConf, vpnConfHint), hasVPN),
	)
}

const vpnConfHint = "WireGuard .conf path, ss:// or trojan:// URL, or .ovpn path."

func withShowIf(fd *field, showIf func(f *form) bool) *field {
	fd.showIf = showIf
	return fd
}

func serverFromForm(f *form, id string) (config.Server, error) {
	s := config.Server{
		ID:      id,
		Name:    f.value("Name"),
		Host:    f.value("Host"),
		User:    f.value("User"),
		Port:    22,
		KeyPath: f.value("Key"),
		Note:    f.value("Note"),
		Tags:    []string{},
	}
	switch {
	case s.Name == "":
		return s, errors.New("Name is required")
	case s.Host == "":
		return s, errors.New("Host is required")
	case s.User == "":
		return s, errors.New("User is required")
	}
	if p := f.value("Port"); p != "" {
		n, err := strconv.Atoi(p)
		if err != nil || n < 1 || n > 65535 {
			return s, errors.New("Port must be a number from 1 to 65535")
		}
		s.Port = n
	}
	for _, t := range strings.Split(f.value("Tags"), ",") {
		if t = strings.TrimSpace(t); t != "" {
			s.Tags = append(s.Tags, t)
		}
	}
	if t := f.value("VPN"); t != vpnInherit {
		conf := f.value("VPN config")
		if conf == "" {
			return s, errors.New("VPN config is required")
		}
		s.VPN = &config.VPNConf{Type: t, ConfPath: conf}
	}
	if s.ID == "" {
		s.ID = uuid.NewString()
	}
	return s, nil
}

func (m *model) openVPN() tea.Cmd {
	current := m.profile().VPN
	vpnType, conf := vpnNone, ""
	if current != nil {
		vpnType, conf = normalizeVPNType(current.Type), current.ConfPath
	}
	submit := func(m *model, f *form) (dialog, tea.Cmd, error) {
		p := m.profile()
		t := f.value("Type")
		if t == vpnNone {
			p.VPN = nil
			return nil, m.commit(p, "Profile VPN removed"), nil
		}
		c := f.value("Config")
		if c == "" {
			return nil, nil, errors.New("Config is required")
		}
		next := &config.VPNConf{Type: t, ConfPath: c, AutoUp: true}
		if p.VPN != nil {
			next.AutoUp, next.AutoDown = p.VPN.AutoUp, p.VPN.AutoDown
		}
		p.VPN = next
		return nil, m.commit(p, "Profile VPN saved"), nil
	}
	m.dialog = newForm("VPN for "+m.profileName, submit,
		choiceField("Type", vpnType, "Servers in this profile use this VPN unless they set their own. Only SSH traffic goes through it.", append([]string{vpnNone}, vpnTypes...)...),
		withShowIf(textField("Config", "~/wg0.conf, ss://…, trojan://…", conf, vpnConfHint), func(f *form) bool { return f.value("Type") != vpnNone }),
	)
	return nil
}

func (m *model) openSCP() tea.Cmd {
	s, _ := m.selected()
	submit := func(m *model, f *form) (dialog, tea.Cmd, error) {
		remote, local := f.value("Remote path"), f.value("Local path")
		if remote == "" {
			return nil, nil, errors.New("Remote path is required")
		}
		if local == "" {
			return nil, nil, errors.New("Local path is required")
		}
		dir := SCPDownload
		if f.value("Direction") == "upload" {
			dir = SCPUpload
		}
		m.action = Action{Type: ActionSCP, Server: &s, Direction: dir, RemotePath: remote, LocalPath: local}
		return nil, tea.Quit, nil
	}
	m.dialog = newForm("Transfer files · "+mask(s.Name, m.masked), submit,
		choiceField("Direction", "download", "Download copies from the server. Upload copies to it.", "download", "upload"),
		textField("Remote path", "/var/log/app.log", "", ""),
		textField("Local path", "~/Downloads/", ".", ""),
	)
	return nil
}

func (m *model) openDelete() tea.Cmd {
	s, _ := m.selected()
	m.dialog = &confirmDialog{
		heading: "Delete server",
		body:    fmt.Sprintf("Delete %s (%s)?\nHistory import will skip it from now on.", mask(s.Name, m.masked), address(s, m.masked)),
		danger:  true,
		onYes:   func(m *model) (dialog, tea.Cmd) { return nil, m.deleteServer(s) },
	}
	return nil
}

func (m *model) openProfiles() tea.Cmd {
	names := m.profileNames()
	items := make([]pickItem, 0, len(names)+1)
	cursor := 0
	for i, n := range names {
		p := m.cfg.Profiles[n]
		meta := plural(len(p.Servers), "server")
		if p.VPN != nil {
			meta += " · " + normalizeVPNType(p.VPN.Type)
		}
		if n == m.profileName {
			meta += " · current"
			cursor = i
		}
		items = append(items, pickItem{title: n, meta: meta})
	}
	items = append(items, pickItem{title: "+ New profile"})

	p := newPicker("Switch profile", items, func(m *model, i int) (dialog, tea.Cmd) {
		if i == len(names) {
			return m.newProfileForm(func(m *model, name string) tea.Cmd { return m.switchProfile(name) }), nil
		}
		if names[i] == m.profileName {
			return nil, nil
		}
		return nil, m.switchProfile(names[i])
	})
	p.move(cursor)
	m.dialog = p
	return nil
}

func (m *model) newProfileForm(then func(m *model, name string) tea.Cmd) *form {
	return newForm("New profile", func(m *model, f *form) (dialog, tea.Cmd, error) {
		name := f.value("Name")
		if err := m.createProfile(name); err != nil {
			return nil, nil, err
		}
		return nil, then(m, name), nil
	}, textField("Name", "work", "", "Letters, numbers, dashes, and underscores."))
}

func (m *model) openMove() tea.Cmd {
	s, _ := m.selected()
	others := slices.DeleteFunc(m.profileNames(), func(n string) bool { return n == m.profileName })
	moveTo := func(m *model, name string) tea.Cmd { return m.moveServer(s, name) }

	if len(others) == 0 {
		m.dialog = m.newProfileForm(moveTo)
		return nil
	}

	items := make([]pickItem, 0, len(others)+1)
	for _, n := range others {
		items = append(items, pickItem{title: n, meta: plural(len(m.cfg.Profiles[n].Servers), "server")})
	}
	items = append(items, pickItem{title: "+ New profile"})

	var p *picker
	p = newPicker("Move "+mask(s.Name, m.masked), items, func(m *model, i int) (dialog, tea.Cmd) {
		if i == len(others) {
			return m.newProfileForm(moveTo), nil
		}
		target := others[i]
		clash := slices.ContainsFunc(m.cfg.Profiles[target].Servers, func(x config.Server) bool { return x.Name == s.Name })
		if !clash {
			return nil, moveTo(m, target)
		}
		return &confirmDialog{
			heading: "Replace server",
			body:    fmt.Sprintf("%s already has a server named %s. Replace it?", target, mask(s.Name, m.masked)),
			danger:  true,
			onYes:   func(m *model) (dialog, tea.Cmd) { return nil, moveTo(m, target) },
			onNo:    func(*model) dialog { return p },
		}, nil
	})
	p.note = "Choose the destination profile."
	m.dialog = p
	return nil
}

type keyGeneratedMsg struct {
	path string
	err  error
}

func (m *model) openKeys() tea.Cmd {
	paths, err := k.ScanPubkeys()
	if err != nil {
		return m.setStatus(statusErr, err.Error())
	}
	items := make([]pickItem, 0, len(paths)+1)
	for _, p := range paths {
		items = append(items, pickItem{title: filepath.Base(p), meta: shortPath(filepath.Dir(p))})
	}
	items = append(items, pickItem{title: "+ Generate ed25519 key"})

	p := newPicker("SSH public keys", items, func(m *model, i int) (dialog, tea.Cmd) {
		if i == len(paths) {
			return nil, generateKey
		}
		if _, err := k.CopyToClipboard(paths[i]); err != nil {
			return nil, m.setStatus(statusErr, err.Error())
		}
		return nil, m.setStatus(statusOK, "Copied "+filepath.Base(paths[i]))
	})
	p.note = "Enter copies the key to the clipboard."
	m.dialog = p
	return nil
}

func generateKey() tea.Msg {
	path, err := k.FreeKeyPath()
	if err != nil {
		return keyGeneratedMsg{err: err}
	}
	cmd, err := k.GenerateCommand(path, "ssht")
	if err != nil {
		return keyGeneratedMsg{err: err}
	}
	if out, err := cmd.CombinedOutput(); err != nil {
		return keyGeneratedMsg{err: fmt.Errorf("ssh-keygen: %s", strings.TrimSpace(string(out)))}
	}
	return keyGeneratedMsg{path: path}
}

func (m *model) openPalette() tea.Cmd {
	cmds := m.commands()
	items := make([]pickItem, len(cmds))
	for i, c := range cmds {
		items[i] = pickItem{title: c.name, meta: c.binding.Help().Key, disabled: c.needsServer && len(m.filtered) == 0}
	}
	m.dialog = newPicker("Actions", items, func(m *model, i int) (dialog, tea.Cmd) {
		m.dialog = nil
		cmd := m.run(cmds[i])
		return m.dialog, cmd
	}).withFilter()
	return nil
}
