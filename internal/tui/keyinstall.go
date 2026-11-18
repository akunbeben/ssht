package tui

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/akunbeben/ssht/internal/config"
	k "github.com/akunbeben/ssht/internal/key"
	sshx "github.com/akunbeben/ssht/internal/ssh"
)

// keyJob identifies one key install: the target server, its profile, and the private key.
type keyJob struct {
	profile string
	server  config.Server
	keyPath string
}

type keyInstalledMsg struct {
	job keyJob
	err string
}

type keyVerifiedMsg struct {
	job keyJob
	err string
}

func (m *model) openInstallKey() tea.Cmd {
	s, _ := m.selected()
	pubs, err := k.ScanPubkeys()
	if err != nil {
		return m.setStatus(statusErr, err.Error())
	}
	keys := slices.DeleteFunc(pubs, func(p string) bool {
		_, err := os.Stat(strings.TrimSuffix(p, ".pub"))
		return err != nil
	})

	items := make([]pickItem, 0, len(keys)+1)
	for _, p := range keys {
		meta := shortPath(filepath.Dir(p))
		if priv := strings.TrimSuffix(p, ".pub"); s.KeyPath != "" && sameFile(s.KeyPath, priv) {
			meta += " · current"
		}
		items = append(items, pickItem{title: filepath.Base(p), meta: meta})
	}
	items = append(items, pickItem{title: "+ Generate ed25519 key"})

	p := newPicker("Install key on "+mask(s.Name, m.masked), items, func(m *model, i int) (dialog, tea.Cmd) {
		if i == len(keys) {
			return nil, generateKey(&s)
		}
		return nil, m.installKey(s, strings.TrimSuffix(keys[i], ".pub"))
	})
	p.note = "You enter the server password once. After that, ssht logs in with this key."
	m.dialog = p
	return nil
}

// installKey hands the terminal to ssh so the user can type the password.
func (m *model) installKey(s config.Server, keyPath string) tea.Cmd {
	job := keyJob{profile: m.profileName, server: s, keyPath: keyPath}
	cmd, err := sshx.InstallKeyCommand(s, m.cfg.Profiles[job.profile].VPN, keyPath+".pub")
	if err != nil {
		return m.setStatus(statusErr, err.Error())
	}
	var stderr bytes.Buffer
	cmd.Stdout = io.Discard
	cmd.Stderr = io.MultiWriter(os.Stderr, &stderr)
	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		if err != nil {
			return keyInstalledMsg{job: job, err: lastLine(stderr.String(), err)}
		}
		return keyInstalledMsg{job: job}
	})
}

func (m *model) keyInstalled(msg keyInstalledMsg) tea.Cmd {
	if msg.err != "" {
		return m.setStatus(statusErr, "Key not installed: "+msg.err)
	}
	vpn := m.cfg.Profiles[msg.job.profile].VPN
	verify := func() tea.Msg {
		cmd, err := sshx.VerifyKeyCommand(msg.job.server, vpn, msg.job.keyPath)
		if err != nil {
			return keyVerifiedMsg{job: msg.job, err: err.Error()}
		}
		if out, err := cmd.CombinedOutput(); err != nil {
			return keyVerifiedMsg{job: msg.job, err: lastLine(string(out), err)}
		}
		return keyVerifiedMsg{job: msg.job}
	}
	return tea.Batch(m.setStatus(statusInfo, "Key installed. Checking login…"), verify)
}

func (m *model) keyVerified(msg keyVerifiedMsg) tea.Cmd {
	if msg.err != "" {
		return m.setStatus(statusErr, "Key installed, but the server rejected it: "+msg.err)
	}
	return m.setServerKey(msg.job.profile, msg.job.server.ID, shortPath(msg.job.keyPath))
}

func sameFile(a, b string) bool {
	a, errA := config.ExpandHome(a)
	b, errB := config.ExpandHome(b)
	return errA == nil && errB == nil && filepath.Clean(a) == filepath.Clean(b)
}

// lastLine returns the last non-empty line of ssh output, or err when there is none.
func lastLine(out string, err error) string {
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if l := strings.TrimSpace(lines[len(lines)-1]); l != "" {
		return l
	}
	return err.Error()
}
