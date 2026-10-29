package tui

import (
	"time"

	"github.com/charmbracelet/bubbles/cursor"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/akunbeben/ssht/internal/config"
)

type ActionType string

const (
	ActionNone      ActionType = "none"
	ActionConnect   ActionType = "connect"
	ActionSCP       ActionType = "scp"
	ActionToggleVPN ActionType = "toggle_vpn"
)

type SCPDirection string

const (
	SCPDownload SCPDirection = "download"
	SCPUpload   SCPDirection = "upload"
)

// Action is what the caller should do after the TUI exits.
type Action struct {
	Type        ActionType
	Server      *config.Server
	ProfileName string
	Direction   SCPDirection
	RemotePath  string
	LocalPath   string
}

type statusKind int

const (
	statusInfo statusKind = iota
	statusOK
	statusErr
)

type clearStatusMsg int

type model struct {
	cfg         *config.Config
	profileName string
	filtered    []config.Server
	cursor      int
	offset      int
	filter      textinput.Model
	dialog      dialog
	help        help.Model
	masked      bool
	action      Action

	status     string
	statusKind statusKind
	statusSeq  int

	width, height int
}

// Run starts the TUI on the given profile and returns the chosen action.
func Run(cfg *config.Config, profileName string) (Action, error) {
	m := newModel(cfg, profileName)
	if _, err := tea.NewProgram(m, tea.WithAltScreen()).Run(); err != nil {
		return Action{}, err
	}
	m.action.ProfileName = m.profileName
	return m.action, nil
}

func newModel(cfg *config.Config, profileName string) *model {
	m := &model{
		cfg:         cfg,
		profileName: profileName,
		filter:      newInput("name, address, tag, note", ""),
		help:        newHelp(),
		masked:      cfg.PrivacyMode,
		action:      Action{Type: ActionNone},
	}
	m.applyFilter()
	return m
}

func newInput(placeholder, value string) textinput.Model {
	ti := textinput.New()
	ti.Prompt = ""
	ti.Placeholder = placeholder
	ti.CharLimit = 4096
	ti.TextStyle = textStyle
	ti.PlaceholderStyle = faintStyle
	ti.Cursor.Style = accentStyle
	ti.Cursor.SetMode(cursor.CursorStatic)
	ti.SetValue(value)
	return ti
}

func newHelp() help.Model {
	h := help.New()
	h.ShortSeparator = faintStyle.Render(" · ")
	h.Styles.ShortKey = textStyle
	h.Styles.ShortDesc = mutedStyle
	h.Styles.FullKey = textStyle
	h.Styles.FullDesc = mutedStyle
	h.Styles.Ellipsis = faintStyle
	return h
}

func (m *model) Init() tea.Cmd { return nil }

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.clampScroll()
	case clearStatusMsg:
		if int(msg) == m.statusSeq {
			m.status = ""
		}
	case keyGeneratedMsg:
		if msg.err != nil {
			return m, m.setStatus(statusErr, msg.err.Error())
		}
		return m, m.setStatus(statusOK, "Created "+shortPath(msg.path)+".pub")
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		if m.dialog != nil {
			var cmd tea.Cmd
			m.dialog, cmd = m.dialog.update(m, msg)
			return m, cmd
		}
		return m, m.updateBrowse(msg)
	}
	return m, nil
}

func (m *model) View() string {
	if m.width == 0 {
		return ""
	}
	base := m.browseView()
	if m.dialog == nil {
		return base
	}
	return overlay(dim(base), m.dialogView(), m.width)
}

func (m *model) setStatus(kind statusKind, text string) tea.Cmd {
	m.status, m.statusKind = text, kind
	m.statusSeq++
	seq := m.statusSeq
	wait := 3 * time.Second
	if kind == statusErr {
		wait = 8 * time.Second
	}
	return tea.Tick(wait, func(time.Time) tea.Msg { return clearStatusMsg(seq) })
}
