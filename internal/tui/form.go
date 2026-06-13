package tui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/google/uuid"

	"github.com/akunbeben/ssht/internal/config"
)

const (
	fieldName = iota
	fieldHost
	fieldPort
	fieldUser
	fieldKey
	fieldTags
	fieldVPNType
	fieldVPNConf
	fieldNote
	fieldCount
)

var fieldLabels = [fieldCount]string{
	"Name", "Host", "Port", "User", "Key Path", "Tags", "VPN Type", "VPN Config", "Note",
}

var fieldPlaceholders = [fieldCount]string{
	"api-prod",
	"10.0.1.12 or example.com",
	"22",
	"deploy",
	"~/.ssh/id_ed25519",
	"prod,api,customer-x",
	"wireguard, shadowsocks, trojan, openvpn",
	"~/wg0.conf, ss://..., trojan://..., or .ovpn",
	"why this server exists or how to use it",
}

var fieldHelp = [fieldCount]string{
	"Short name used for search and selection.",
	"Hostname or IP only. Do not include user or port here.",
	"Leave empty for SSH default port 22.",
	"SSH login user, for example deploy, ubuntu, root, or postgres.",
	"Optional. Leave empty to let ssh use its default identities.",
	"Optional comma-separated tags. Search uses these tags.",
	"Optional server VPN override. Leave blank to use the profile VPN. Blank type defaults to WireGuard when config is set.",
	"Required only when VPN Type is set. WireGuard sessions with the same config share one isolated userspace tunnel.",
	"Optional note. Search uses this text and details show it before connect.",
}

type formState struct {
	inputs  [fieldCount]textinput.Model
	focus   int
	editing bool
	editID  string
}

func newFormState(editing bool, s *config.Server) formState {
	f := formState{editing: editing}
	for i := 0; i < fieldCount; i++ {
		ti := textinput.New()
		ti.Placeholder = fieldPlaceholders[i]
		ti.CharLimit = 256
		ti.PromptStyle = blurredInputStyle
		ti.TextStyle = blurredInputStyle
		ti.PlaceholderStyle = placeholderStyle
		ti.Cursor.Style = cursorStyle
		f.inputs[i] = ti
	}
	f.inputs[fieldPort].Placeholder = "22"

	if s != nil {
		if editing {
			f.editID = s.ID
		}
		f.inputs[fieldName].SetValue(s.Name)
		f.inputs[fieldHost].SetValue(s.Host)
		if s.Port != 0 {
			f.inputs[fieldPort].SetValue(strconv.Itoa(s.Port))
		}
		f.inputs[fieldUser].SetValue(s.User)
		f.inputs[fieldKey].SetValue(s.KeyPath)
		f.inputs[fieldTags].SetValue(strings.Join(s.Tags, ","))
		if s.VPN != nil {
			f.inputs[fieldVPNType].SetValue(s.VPN.Type)
			f.inputs[fieldVPNConf].SetValue(s.VPN.ConfPath)
		}
		f.inputs[fieldNote].SetValue(s.Note)
	}

	f.focusField(0)
	return f
}

func (f *formState) focusField(idx int) {
	if idx < 0 {
		idx = 0
	}
	if idx >= fieldCount {
		idx = fieldCount - 1
	}
	for i := 0; i < fieldCount; i++ {
		if i == idx {
			f.inputs[i].Focus()
			f.inputs[i].PromptStyle = focusedInputStyle
			f.inputs[i].TextStyle = focusedInputStyle
		} else {
			f.inputs[i].Blur()
			f.inputs[i].PromptStyle = blurredInputStyle
			f.inputs[i].TextStyle = blurredInputStyle
		}
	}
	f.focus = idx
}

func (f *formState) updateInput(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	f.inputs[f.focus], cmd = f.inputs[f.focus].Update(msg)
	return cmd
}

func (f *formState) validate(existingNames map[string]bool) error {
	name := strings.TrimSpace(f.inputs[fieldName].Value())
	if name == "" {
		return fmt.Errorf("name is required")
	}
	if f.editing {
		if existingNames[name] {
			return fmt.Errorf("server %q already exists", name)
		}
	}
	if !f.editing {
		if existingNames[name] {
			return fmt.Errorf("server %q already exists", name)
		}
	}
	host := strings.TrimSpace(f.inputs[fieldHost].Value())
	if host == "" {
		return fmt.Errorf("host is required")
	}
	user := strings.TrimSpace(f.inputs[fieldUser].Value())
	if user == "" {
		return fmt.Errorf("user is required")
	}
	portStr := strings.TrimSpace(f.inputs[fieldPort].Value())
	if portStr != "" {
		p, err := strconv.Atoi(portStr)
		if err != nil || p <= 0 || p > 65535 {
			return fmt.Errorf("port must be 1-65535")
		}
	}
	vpnType := strings.TrimSpace(f.inputs[fieldVPNType].Value())
	vpnPath := strings.TrimSpace(f.inputs[fieldVPNConf].Value())
	if vpnType != "" && vpnPath == "" {
		return fmt.Errorf("vpn config is required when vpn type is set")
	}
	return nil
}

func (f *formState) toServer() config.Server {
	port := 22
	if p, err := strconv.Atoi(strings.TrimSpace(f.inputs[fieldPort].Value())); err == nil && p > 0 {
		port = p
	}
	tags := []string{}
	if raw := strings.TrimSpace(f.inputs[fieldTags].Value()); raw != "" {
		for _, t := range strings.Split(raw, ",") {
			t = strings.TrimSpace(t)
			if t != "" {
				tags = append(tags, t)
			}
		}
	}
	id := f.editID
	if id == "" {
		id = uuid.NewString()
	}

	var vpnConf *config.VPNConf
	vpnType := strings.TrimSpace(f.inputs[fieldVPNType].Value())
	vpnPath := strings.TrimSpace(f.inputs[fieldVPNConf].Value())

	if vpnType != "" || vpnPath != "" {
		vpnConf = &config.VPNConf{
			Type:     vpnType,
			ConfPath: vpnPath,
		}
	}

	return config.Server{
		ID:      id,
		Name:    strings.TrimSpace(f.inputs[fieldName].Value()),
		Host:    strings.TrimSpace(f.inputs[fieldHost].Value()),
		Port:    port,
		User:    strings.TrimSpace(f.inputs[fieldUser].Value()),
		KeyPath: strings.TrimSpace(f.inputs[fieldKey].Value()),
		VPN:     vpnConf,
		Tags:    tags,
		Note:    strings.TrimSpace(f.inputs[fieldNote].Value()),
	}
}

func (f *formState) view(width, height int, errMsg string, helperWrapped bool) string {
	var body strings.Builder

	title := "Add Server"
	if f.editing {
		title = "Edit Server"
	}
	body.WriteString(titleStyle.Render(title) + "\n\n")
	body.WriteString(dimStyle.Copy().Width(width).Render("Connection fields first; metadata and VPN override are optional.") + "\n\n")

	isMobile := width < 45
	for i := 0; i < fieldCount; i++ {
		f.inputs[i].Width = max(width-2, 10)
		if !isMobile {
			f.inputs[i].Width = max(width-20, 10)
		}

		labelStr := fieldLabels[i]
		cursor := inactiveCursor
		if i == f.focus {
			cursor = focusedInputStyle.Render(activeCursor)
		}

		if isMobile {
			label := formLabelStackedStyle.Render(labelStr)
			body.WriteString(cursor + label + "\n" + "  " + f.inputs[i].View() + "\n\n")
		} else {
			label := formLabelStyle.Render(labelStr)
			body.WriteString(cursor + label + f.inputs[i].View() + "\n")
		}
	}

	if errMsg != "" {
		errStyleWrap := errorStyle.Copy().Width(width)
		if !helperWrapped {
			errMsg = truncate(errMsg, width)
			errStyleWrap = errStyleWrap.MaxHeight(1)
		}
		body.WriteString("\n" + errStyleWrap.Render("✗ "+errMsg))
	}

	preview := f.preview()
	if preview != "" {
		previewStyle := helpStyle.Copy().Width(width)
		if !helperWrapped {
			preview = truncate(preview, width)
			previewStyle = previewStyle.MaxHeight(1)
		}
		body.WriteString("\n" + previewStyle.Render(preview))
	}

	hint := fieldHelp[f.focus]
	if hint != "" {
		hintStyle := dimStyle.Copy().Width(width)
		if !helperWrapped {
			hint = truncate(hint, width)
			hintStyle = hintStyle.MaxHeight(1)
		}
		body.WriteString("\n" + hintStyle.Render(hint))
	}

	help := "Tab/Up/Down: navigate  Enter: next/save  Esc: cancel"
	helpStyleWrap := helpStyle.Copy().Width(width)
	if !helperWrapped {
		help = truncate(help, width)
		helpStyleWrap = helpStyleWrap.MaxHeight(1)
	}
	renderedHelp := helpStyleWrap.Render(help)

	// Calculate vertical space to push help to the bottom
	bodyContent := body.String()
	gap := height - lipgloss.Height(bodyContent) - lipgloss.Height(renderedHelp)
	if gap > 0 {
		return bodyContent + strings.Repeat("\n", gap) + renderedHelp
	}
	return bodyContent + "\n\n" + renderedHelp
}

func (f *formState) preview() string {
	user := strings.TrimSpace(f.inputs[fieldUser].Value())
	host := strings.TrimSpace(f.inputs[fieldHost].Value())
	if user == "" && host == "" {
		return ""
	}
	if user == "" {
		user = "user"
	}
	if host == "" {
		host = "host"
	}

	port := strings.TrimSpace(f.inputs[fieldPort].Value())
	if port == "" {
		port = "22"
	}

	key := "key default"
	if strings.TrimSpace(f.inputs[fieldKey].Value()) != "" {
		key = "key set"
	}

	vpn := "vpn profile/default"
	if strings.TrimSpace(f.inputs[fieldVPNConf].Value()) != "" || strings.TrimSpace(f.inputs[fieldVPNType].Value()) != "" {
		vpn = "vpn server override"
	}

	return fmt.Sprintf("Preview: ssh %s@%s:%s  %s  %s", user, host, port, key, vpn)
}
