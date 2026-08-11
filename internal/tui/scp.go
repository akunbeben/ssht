package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	scpFocusDirection = iota
	scpFocusRemote
	scpFocusLocal
)

type scpState struct {
	remoteInput textinput.Model
	localInput  textinput.Model
	focus       int
	direction   SCPDirection
}

func newSCPState() *scpState {
	remote := textinput.New()
	remote.Placeholder = "/var/log/app.log"
	remote.CharLimit = 4096
	remote.PromptStyle = blurredInputStyle
	remote.TextStyle = blurredInputStyle
	remote.PlaceholderStyle = placeholderStyle
	remote.Cursor.Style = cursorStyle

	local := textinput.New()
	local.Placeholder = "~/Downloads/"
	local.SetValue(".")
	local.CharLimit = 4096
	local.PromptStyle = blurredInputStyle
	local.TextStyle = blurredInputStyle
	local.PlaceholderStyle = placeholderStyle
	local.Cursor.Style = cursorStyle

	s := &scpState{
		remoteInput: remote,
		localInput:  local,
		direction:   SCPDownload,
	}
	s.focusField(scpFocusDirection)
	return s
}

func (m *model) handleSCPKey(key string, msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.scp == nil {
		m.mode = modeList
		return m, nil
	}

	s := m.scp
	switch key {
	case "esc":
		m.scp = nil
		m.err = nil
		m.mode = modeList
		return m, nil
	case "tab", "down":
		s.focusField((s.focus + 1) % 3)
		return m, nil
	case "shift+tab", "up":
		focus := s.focus - 1
		if focus < 0 {
			focus = 2
		}
		s.focusField(focus)
		return m, nil
	case "left", "right":
		if s.focus == scpFocusDirection {
			if s.direction == SCPDownload {
				s.direction = SCPUpload
			} else {
				s.direction = SCPDownload
			}
			return m, nil
		}
	case "enter":
		if s.focus < scpFocusLocal {
			s.focusField(s.focus + 1)
			return m, nil
		}
		return m.submitSCP()
	}

	var cmd tea.Cmd
	if s.focus == scpFocusRemote {
		s.remoteInput, cmd = s.remoteInput.Update(msg)
	} else if s.focus == scpFocusLocal {
		s.localInput, cmd = s.localInput.Update(msg)
	}
	return m, cmd
}

func (s *scpState) focusField(idx int) {
	if idx < scpFocusDirection {
		idx = scpFocusDirection
	}
	if idx > scpFocusLocal {
		idx = scpFocusLocal
	}
	s.focus = idx

	s.remoteInput.Blur()
	s.remoteInput.PromptStyle = blurredInputStyle
	s.remoteInput.TextStyle = blurredInputStyle
	s.localInput.Blur()
	s.localInput.PromptStyle = blurredInputStyle
	s.localInput.TextStyle = blurredInputStyle

	switch idx {
	case scpFocusRemote:
		s.remoteInput.Focus()
		s.remoteInput.PromptStyle = focusedInputStyle
		s.remoteInput.TextStyle = focusedInputStyle
	case scpFocusLocal:
		s.localInput.Focus()
		s.localInput.PromptStyle = focusedInputStyle
		s.localInput.TextStyle = focusedInputStyle
	}
}

func (m *model) submitSCP() (tea.Model, tea.Cmd) {
	if len(m.filtered) == 0 || m.scp == nil {
		return m, nil
	}

	remotePath := strings.TrimSpace(m.scp.remoteInput.Value())
	if remotePath == "" {
		m.err = fmt.Errorf("remote path is required")
		m.scp.focusField(scpFocusRemote)
		return m, nil
	}
	localPath := strings.TrimSpace(m.scp.localInput.Value())
	if localPath == "" {
		m.err = fmt.Errorf("local path is required")
		m.scp.focusField(scpFocusLocal)
		return m, nil
	}

	sel := m.filtered[m.index]
	m.action = Action{
		Type:       ActionSCP,
		Server:     &sel,
		Direction:  m.scp.direction,
		RemotePath: remotePath,
		LocalPath:  localPath,
	}
	m.scp = nil
	m.err = nil
	m.mode = modeList
	return m, tea.Quit
}

func (m *model) scpView(width, height int, helperWrapped bool) string {
	if m.scp == nil {
		return m.listView()
	}

	var body strings.Builder
	body.WriteString(titleStyle.Render("SCP Transfer") + "\n")
	body.WriteString(dimStyle.Copy().Width(width).Render("Transfer files through the selected server and its isolated VPN, if configured.") + "\n\n")

	directionCursor := inactiveCursor
	if m.scp.focus == scpFocusDirection {
		directionCursor = focusedInputStyle.Render(activeCursor)
	}
	download := "Download"
	upload := "Upload"
	if m.scp.direction == SCPDownload {
		download = selectedStyle.Render(download)
	} else {
		upload = selectedStyle.Render(upload)
	}
	body.WriteString(directionCursor + formLabelStyle.Render("Direction") + download + dimStyle.Render("  ") + upload + dimStyle.Render("  (Left/Right to change)") + "\n\n")

	isMobile := width < 45
	m.scp.remoteInput.Width = max(width-20, 10)
	m.scp.localInput.Width = max(width-20, 10)
	if isMobile {
		m.scp.remoteInput.Width = max(width-2, 10)
		m.scp.localInput.Width = max(width-2, 10)
	}

	if isMobile {
		remoteCursor := inactiveCursor
		if m.scp.focus == scpFocusRemote {
			remoteCursor = focusedInputStyle.Render(activeCursor)
		}
		localCursor := inactiveCursor
		if m.scp.focus == scpFocusLocal {
			localCursor = focusedInputStyle.Render(activeCursor)
		}
		body.WriteString(remoteCursor + formLabelStackedStyle.Render("Remote path") + "\n  " + m.scp.remoteInput.View() + "\n\n")
		body.WriteString(localCursor + formLabelStackedStyle.Render("Local path") + "\n  " + m.scp.localInput.View() + "\n")
	} else {
		remoteCursor := inactiveCursor
		if m.scp.focus == scpFocusRemote {
			remoteCursor = focusedInputStyle.Render(activeCursor)
		}
		localCursor := inactiveCursor
		if m.scp.focus == scpFocusLocal {
			localCursor = focusedInputStyle.Render(activeCursor)
		}
		body.WriteString(remoteCursor + formLabelStyle.Render("Remote path") + m.scp.remoteInput.View() + "\n")
		body.WriteString(localCursor + formLabelStyle.Render("Local path") + m.scp.localInput.View() + "\n")
	}

	if m.err != nil {
		errMsg := m.err.Error()
		errStyle := errorStyle.Copy().Width(width)
		if !helperWrapped {
			errMsg = truncate(errMsg, width)
			errStyle = errStyle.MaxHeight(1)
		}
		body.WriteString("\n" + errStyle.Render("✗ "+errMsg))
	}

	help := "Tab/Up/Down: navigate  Left/Right: direction  Enter: next/run  Esc: cancel"
	helpStyleWrap := helpStyle.Copy().Width(width)
	if !helperWrapped {
		help = truncate(help, width)
		helpStyleWrap = helpStyleWrap.MaxHeight(1)
	}
	renderedHelp := helpStyleWrap.Render(help)

	bodyContent := body.String()
	gap := height - lipgloss.Height(bodyContent) - lipgloss.Height(renderedHelp)
	if gap > 0 {
		bodyContent += strings.Repeat("\n", gap)
	} else {
		bodyContent += "\n\n"
	}
	return m.renderFullScreen(bodyContent + renderedHelp)
}
