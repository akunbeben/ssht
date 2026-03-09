package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/akunbeben/ssht/internal/config"
	k "github.com/akunbeben/ssht/internal/key"
	"github.com/akunbeben/ssht/internal/vpn"
)

type ActionType string

const (
	ActionNone      ActionType = "none"
	ActionConnect   ActionType = "connect"
	ActionToggleVPN ActionType = "toggle_vpn"
)

type Action struct {
	Type        ActionType
	Server      *config.Server
	ProfileName string
}

type viewMode int

const (
	modeList viewMode = iota
	modeSearch
	modeHelp
	modePubkey
	modeForm
	modeConfirmDelete
	modeProfileSwitch
	modeMoveServer
	modeMoveNewProfile
	modeMoveConfirmOverwrite
	modeVPNConfig
)

type model struct {
	cfg         *config.Config
	profileName string
	profile     config.Profile
	servers     []config.Server
	filtered    []config.Server
	index       int
	mode        viewMode
	pendingG    bool
	search      string
	err         error
	status      string
	statusStyle lipgloss.Style
	statusSeq   int
	action      Action

	pubkeys  []string
	pubIndex int

	form              *formState
	profileSwitch     *profileSwitchState
	deleteTarget      *config.Server
	moveTarget        *config.Server
	moveProfiles      *profileSwitchState
	moveTargetProfile string
	moveNewName       textinput.Model
	vpnInput          textinput.Model
	masked            bool

	width  int
	height int
	ready  bool
}

func Run(cfg *config.Config, profileName string, profile config.Profile) (Action, error) {
	m := model{
		cfg:         cfg,
		profileName: profileName,
		profile:     profile,
		servers:     profile.Servers,
		filtered:    profile.Servers,
		action:      Action{Type: ActionNone},
		statusStyle: helpStyle,
		masked:      cfg.PrivacyMode,
	}
	prog := tea.NewProgram(&m, tea.WithAltScreen())
	if _, err := prog.Run(); err != nil {
		return Action{}, err
	}
	m.action.ProfileName = m.profileName
	return m.action, nil
}

func (m *model) Init() tea.Cmd { return nil }

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true
		return m, nil
	case clearStatusMsg:
		if int(msg) == m.statusSeq {
			m.status = ""
		}
		return m, nil
	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m *model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	switch m.mode {
	case modeSearch:
		return m.handleSearchKey(key, msg)
	case modePubkey:
		return m.handlePubkeyKey(key)
	case modeForm:
		return m.handleFormKey(key, msg)
	case modeConfirmDelete:
		return m.handleDeleteKey(key)
	case modeProfileSwitch:
		return m.handleProfileSwitchKey(key)
	case modeMoveServer:
		return m.handleMoveKey(key)
	case modeMoveNewProfile:
		return m.handleMoveNewProfileKey(key, msg)
	case modeMoveConfirmOverwrite:
		return m.handleMoveConfirmOverwriteKey(key)
	case modeVPNConfig:
		return m.handleVPNConfigKey(key, msg)
	case modeHelp:
		if key == "?" || key == "q" || key == "esc" {
			m.mode = modeList
		}
		return m, nil
	}

	return m.handleListKey(key)
}

func (m *model) handleListKey(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "q", "esc", "ctrl+c":
		return m, tea.Quit
	case "?":
		m.mode = modeHelp
		m.pendingG = false
		return m, nil
	case "/":
		m.pendingG = false
		m.mode = modeSearch
		m.search = ""
		return m, nil
	case "*":
		m.masked = !m.masked
		m.cfg.PrivacyMode = m.masked
		_ = config.Save(m.cfg)
		status := "off"
		if m.masked {
			status = "on"
		}
		return m, m.setStatus("✓ sensitive masking "+status, successStyle)
	}

	if cmd := m.handleNavigationKey(key); cmd != nil {
		return m, cmd
	}

	if m.isNavigationKey(key) {
		return m, nil
	}

	return m.handleActionKey(key)
}

func (m *model) isNavigationKey(key string) bool {
	switch key {
	case "j", "down", "k", "up", "home", "end", "pgdown", "ctrl+d", "pgup", "ctrl+u", "g", "G":
		return true
	}
	return false
}

func (m *model) handleNavigationKey(key string) tea.Cmd {
	switch key {
	case "j", "down":
		m.moveList(1)
	case "k", "up":
		m.moveList(-1)
	case "home":
		m.index = 0
		m.pendingG = false
	case "end":
		m.index = max(len(m.filtered)-1, 0)
		m.pendingG = false
	case "pgdown", "ctrl+d":
		m.moveList(10)
		m.pendingG = false
	case "pgup", "ctrl+u":
		m.moveList(-10)
		m.pendingG = false
	case "g":
		if m.pendingG {
			m.index = 0
			m.pendingG = false
		} else {
			m.pendingG = true
		}
	case "G":
		if len(m.filtered) > 0 {
			m.index = len(m.filtered) - 1
		}
		m.pendingG = false
	default:
		return nil
	}
	return nil
}

func (m *model) handleActionKey(key string) (tea.Model, tea.Cmd) {
	m.pendingG = false
	switch key {
	case "K":
		m.mode = modePubkey
		m.refreshPubkeys()
	case "v":
		if m.profile.VPN == nil {
			m.vpnInput = newVPNInput()
			m.mode = modeVPNConfig
			return m, nil
		}
		if !vpn.HasWgQuick() {
			return m, m.setStatus("✗ wireguard-tools not found. Run: brew install wireguard-tools", errorStyle)
		}
		m.action = Action{Type: ActionToggleVPN}
		return m, tea.Quit
	case "enter":
		if len(m.filtered) == 0 {
			return m, nil
		}
		sel := m.filtered[m.index]
		m.action = Action{Type: ActionConnect, Server: &sel}
		return m, tea.Quit
	case "a":
		m.clearStatus()
		f := newFormState(false, nil)
		m.form = &f
		m.mode = modeForm
	case "e":
		if len(m.filtered) == 0 {
			return m, m.setStatus("no server to edit", errorStyle)
		}
		m.clearStatus()
		sel := m.filtered[m.index]
		f := newFormState(true, &sel)
		m.form = &f
		m.mode = modeForm
	case "d":
		if len(m.filtered) == 0 {
			return m, m.setStatus("no server to delete", errorStyle)
		}
		sel := m.filtered[m.index]
		m.deleteTarget = &sel
		m.mode = modeConfirmDelete
	case "p":
		if len(m.cfg.Profiles) <= 1 {
			return m, m.setStatus("only one profile available", dimStyle)
		}
		ps := newProfileSwitchState(m.cfg, m.profileName)
		m.profileSwitch = &ps
		m.mode = modeProfileSwitch
	case "m":
		if len(m.filtered) == 0 {
			return m, m.setStatus("no server to move", errorStyle)
		}
		sel := m.filtered[m.index]
		m.moveTarget = &sel
		if len(m.cfg.Profiles) <= 1 {
			m.moveNewName = newMoveProfileInput()
			m.mode = modeMoveNewProfile
		} else {
			ps := newProfileSwitchState(m.cfg, m.profileName)
			m.moveProfiles = &ps
			m.mode = modeMoveServer
		}
	}
	return m, nil
}

func (m *model) handleSearchKey(key string, msg tea.Msg) (tea.Model, tea.Cmd) {
	m.pendingG = false
	switch key {
	case "enter":
		m.mode = modeList
	case "esc":
		m.search = ""
		m.filtered = m.servers
		m.index = 0
		m.mode = modeList
	case "backspace":
		if len(m.search) > 0 {
			m.search = m.search[:len(m.search)-1]
			m.applyFilter()
		}
	default:
		if len(key) == 1 {
			m.search += key
			m.applyFilter()
		}
	}
	return m, nil
}

func (m *model) handlePubkeyKey(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "q", "esc":
		m.mode = modeList
		m.pendingG = false
	case "j", "down":
		m.movePubkey(1)
	case "k", "up":
		m.movePubkey(-1)
	case "home":
		m.pubIndex = 0
	case "end", "G":
		m.pubIndex = len(m.pubkeys)
	case "pgdown", "ctrl+d":
		m.movePubkey(10)
	case "pgup", "ctrl+u":
		m.movePubkey(-10)
	case "g":
		if m.pendingG {
			m.pubIndex = 0
			m.pendingG = false
		} else {
			m.pendingG = true
		}
		return m, nil
	case "enter":
		m.pendingG = false
		if len(m.pubkeys) == 0 || m.pubIndex == len(m.pubkeys) {
			if err := k.Generate("", "ssht"); err != nil {
				m.err = err
				return m, nil
			}
			m.refreshPubkeys()
			cmd := m.setStatus("✓ generated new ed25519 keypair", successStyle)
			m.err = nil
			return m, cmd
		}
		content, err := k.CopyToClipboard(m.pubkeys[m.pubIndex])
		if err != nil {
			m.err = err
			return m, nil
		}
		cmd := m.setStatus("✓ copied: "+trim(content, 72), successStyle)
		m.err = nil
		return m, cmd
	default:
		m.pendingG = false
	}
	return m, nil
}

func (m *model) handleFormKey(key string, msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.form == nil {
		m.mode = modeList
		return m, nil
	}

	switch key {
	case "esc":
		m.form = nil
		m.mode = modeList
		return m, nil
	case "tab", "down":
		next := m.form.focus + 1
		if next >= fieldCount {
			next = 0
		}
		m.form.focusField(next)
		return m, nil
	case "shift+tab", "up":
		prev := m.form.focus - 1
		if prev < 0 {
			prev = fieldCount - 1
		}
		m.form.focusField(prev)
		return m, nil
	case "enter":

		if m.form.focus < fieldCount-1 {
			m.form.focusField(m.form.focus + 1)
			return m, nil
		}

		return m.submitForm()
	}

	cmd := m.form.updateInput(msg)
	return m, cmd
}

func (m *model) submitForm() (tea.Model, tea.Cmd) {
	existingNames := make(map[string]bool, len(m.servers))
	for _, s := range m.servers {
		if m.form.editing && s.ID == m.form.editID {
			continue
		}
		existingNames[s.Name] = true
	}

	if err := m.form.validate(existingNames); err != nil {
		m.err = err
		return m, nil
	}

	newServer := m.form.toServer()

	var cmd tea.Cmd
	if m.form.editing {
		for i, s := range m.profile.Servers {
			if s.ID == m.form.editID {
				m.profile.Servers[i] = newServer
				break
			}
		}

		port := newServer.Port
		if port == 0 {
			port = 22
		}
		exKey := fmt.Sprintf("%s@%s:%d", newServer.User, newServer.Host, port)
		m.profile.ImportExceptions = removeString(m.profile.ImportExceptions, exKey)

		cmd = m.setStatus(fmt.Sprintf("✓ updated %q", newServer.Name), successStyle)
	} else {
		m.profile.Servers = append(m.profile.Servers, newServer)

		port := newServer.Port
		if port == 0 {
			port = 22
		}
		exKey := fmt.Sprintf("%s@%s:%d", newServer.User, newServer.Host, port)
		m.profile.ImportExceptions = removeString(m.profile.ImportExceptions, exKey)

		cmd = m.setStatus(fmt.Sprintf("✓ added %q", newServer.Name), successStyle)
	}
	m.err = nil
	m.form = nil
	m.mode = modeList
	m.syncServers()
	return m, cmd
}

func (m *model) handleProfileSwitchKey(key string) (tea.Model, tea.Cmd) {
	if m.profileSwitch == nil {
		m.mode = modeList
		return m, nil
	}

	switch key {
	case "esc", "q":
		m.profileSwitch = nil
		m.mode = modeList
	case "j", "down":
		m.profileSwitch.move(1)
	case "k", "up":
		m.profileSwitch.move(-1)
	case "enter":
		selected := m.profileSwitch.selected()
		if selected == "" || selected == m.profileName {
			m.profileSwitch = nil
			m.mode = modeList
			return m, nil
		}
		profile, ok := m.cfg.Profiles[selected]
		if !ok {
			m.err = fmt.Errorf("profile %q not found", selected)
			m.profileSwitch = nil
			m.mode = modeList
			return m, nil
		}

		m.saveProfile()

		m.profileName = selected
		m.profile = profile
		m.cfg.LastProfile = selected
		m.syncServers()
		m.profileSwitch = nil
		m.mode = modeList
		cmd := m.setStatus(fmt.Sprintf("✓ switched to %q (%d servers)", selected, len(profile.Servers)), successStyle)

		_ = config.Save(m.cfg)
		return m, cmd
	}
	return m, nil
}

func (m *model) View() string {
	if !m.ready {
		return "loading..."
	}
	switch m.mode {
	case modeHelp:
		return m.renderFullScreen(titleStyle.Render("ssht help") + "\n\n" + helpText)
	case modePubkey:
		return m.pubkeyView()
	case modeForm:
		if m.form != nil {
			content := m.form.view()
			if m.err != nil {
				content += "\n" + errorStyle.Render("✗ "+m.err.Error())
			}
			return m.renderFullScreen(content)
		}
	case modeConfirmDelete:
		return m.confirmDeleteView()
	case modeProfileSwitch:
		if m.profileSwitch != nil {
			content := m.profileSwitch.view()
			return m.renderFullScreen(content)
		}
	case modeMoveServer:
		return m.moveServerView()
	case modeMoveNewProfile:
		return m.moveNewProfileView()
	case modeMoveConfirmOverwrite:
		return m.moveConfirmOverwriteView()
	case modeVPNConfig:
		return m.vpnConfigView()
	}

	return m.listView()
}

func (m *model) listView() string {
	vpnState := "off"
	if m.profile.VPN != nil {
		vpnState = "configured"
	}

	searchLine := dimStyle.Render("/ to search")
	if m.mode == modeSearch {
		searchLine = focusedInputStyle.Render("/ ") + m.search + focusedInputStyle.Render("▏")
	}

	status := m.status
	if m.err != nil {
		status = errorStyle.Render("✗ " + m.err.Error())
	}
	if status != "" {
		status = m.statusStyle.Render(status)
	} else {
		status = helpStyle.Render("Enter: connect · a: add · e: edit · d: del · p: profile · v: system vpn · *: mask · K: keys · ?: help")
	}

	head := fmt.Sprintf("ssht · profile: %s", m.profileName)
	foot := fmt.Sprintf("%d servers · VPN: %s", len(m.servers), vpnState)
	_, innerH := m.innerSize()
	bodyHeight := max(innerH-5, 1)
	rows := m.visibleServerRows(bodyHeight)

	content := strings.Join([]string{
		titleStyle.Render(head),
		renderListHeader(),
		strings.Join(rows, "\n"),
		searchLine,
		dimStyle.Render(foot),
		status,
	}, "\n")
	return m.renderFullScreen(content)
}

func (m *model) pubkeyView() string {
	_, innerH := m.innerSize()
	bodyHeight := max(innerH-3, 1)
	lines := m.visiblePubkeyRows(bodyHeight)
	lines = append(lines, helpStyle.Render("Enter: copy/generate · q: back"))
	if m.status != "" {
		lines = append(lines, m.statusStyle.Render(m.status))
	}
	if m.err != nil {
		lines = append(lines, errorStyle.Render("✗ "+m.err.Error()))
	}
	content := titleStyle.Render("Pubkey Manager") + "\n" + strings.Join(lines, "\n")
	return m.renderFullScreen(content)
}
