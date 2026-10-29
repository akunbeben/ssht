package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// field is a text input, or a choice when options is set.
type field struct {
	label   string
	hint    string
	input   textinput.Model
	options []string
	choice  int
	showIf  func(f *form) bool
}

func textField(label, placeholder, value, hint string) *field {
	return &field{label: label, hint: hint, input: newInput(placeholder, value)}
}

// choiceField selects value in options, keeping unknown values selectable.
func choiceField(label, value, hint string, options ...string) *field {
	f := &field{label: label, hint: hint, options: options}
	for i, o := range options {
		if o == value {
			f.choice = i
			return f
		}
	}
	if value != "" {
		f.options = append(f.options, value)
		f.choice = len(f.options) - 1
	}
	return f
}

type form struct {
	heading  string
	fields   []*field
	focus    int
	err      error
	onSubmit func(m *model, f *form) (dialog, tea.Cmd, error)
}

func newForm(heading string, onSubmit func(m *model, f *form) (dialog, tea.Cmd, error), fields ...*field) *form {
	f := &form{heading: heading, fields: fields, onSubmit: onSubmit}
	f.setFocus(0)
	return f
}

// value returns the trimmed value of the field with label.
func (f *form) value(label string) string {
	for _, fd := range f.fields {
		if fd.label != label {
			continue
		}
		if fd.options != nil {
			return fd.options[fd.choice]
		}
		return strings.TrimSpace(fd.input.Value())
	}
	return ""
}

func (f *form) visible(i int) bool {
	return f.fields[i].showIf == nil || f.fields[i].showIf(f)
}

func (f *form) setFocus(i int) {
	f.focus = i
	for j, fd := range f.fields {
		if fd.options != nil {
			continue
		}
		if j == i {
			fd.input.Focus()
		} else {
			fd.input.Blur()
		}
	}
}

// step moves focus to the next visible field in dir, and reports false at the edge.
func (f *form) step(dir int) bool {
	for i := f.focus + dir; i >= 0 && i < len(f.fields); i += dir {
		if f.visible(i) {
			f.setFocus(i)
			return true
		}
	}
	return false
}

func (f *form) title() string { return f.heading }

func (f *form) update(m *model, msg tea.KeyMsg) (dialog, tea.Cmd) {
	fd := f.fields[f.focus]
	switch {
	case key.Matches(msg, keyClose):
		return nil, nil
	case key.Matches(msg, keySubmit):
		return f.submit(m)
	case key.Matches(msg, keyEnter):
		if !f.step(1) {
			return f.submit(m)
		}
		return f, nil
	case key.Matches(msg, keyNext):
		f.step(1)
		return f, nil
	case key.Matches(msg, keyPrev):
		f.step(-1)
		return f, nil
	case fd.options != nil && key.Matches(msg, keyFieldD):
		f.step(1)
		return f, nil
	case fd.options != nil && key.Matches(msg, keyFieldU):
		f.step(-1)
		return f, nil
	case fd.options != nil && key.Matches(msg, keyChoose):
		n := len(fd.options)
		if key.Matches(msg, keyLeft) {
			fd.choice = (fd.choice + n - 1) % n
		} else {
			fd.choice = (fd.choice + 1) % n
		}
		return f, nil
	case fd.options != nil:
		return f, nil
	}
	var cmd tea.Cmd
	fd.input, cmd = fd.input.Update(msg)
	return f, cmd
}

func (f *form) submit(m *model) (dialog, tea.Cmd) {
	next, cmd, err := f.onSubmit(m, f)
	if err != nil {
		f.err = err
		return f, nil
	}
	return next, cmd
}

func (f *form) view(m *model, w, h int) string {
	labelW := 0
	for _, fd := range f.fields {
		labelW = max(labelW, lipgloss.Width(fd.label))
	}
	labelW += 2
	stacked := w-labelW < 24

	var rows []string
	focusRow := 0
	for i, fd := range f.fields {
		if !f.visible(i) {
			continue
		}
		focused := i == f.focus
		if focused {
			focusRow = len(rows)
		}
		label := labelStyle.Render(fit(fd.label, labelW))
		if focused {
			label = accentStyle.Render(fit(fd.label, labelW))
		}
		inputW := w - labelW
		if stacked {
			inputW = w
		}
		value := fd.render(focused, inputW)
		if stacked {
			rows = append(rows, label, value)
		} else {
			rows = append(rows, label+value)
		}
	}

	// The error always shows. The hint shows when it fits.
	tail := ""
	if f.err != nil {
		tail = "\n\n" + badStyle.Width(w).Render("✗ "+f.err.Error())
	}
	if hint := f.fields[f.focus].hint; hint != "" {
		withHint := "\n\n" + mutedStyle.Width(w).Render(hint) + tail
		if len(rows)+lipgloss.Height(withHint)-1 <= h {
			tail = withHint
		}
	}

	// Scroll the fields so the focused one stays visible.
	room := max(h-lipgloss.Height(tail)+1, 1)
	if len(rows) > room {
		start := min(max(focusRow-room/2, 0), len(rows)-room)
		rows = rows[start : start+room]
	}
	return strings.Join(rows, "\n") + tail
}

func (fd *field) render(focused bool, w int) string {
	if fd.options == nil {
		fd.input.Width = max(w-1, 1)
		return fd.input.View()
	}
	value := textStyle.Render(fd.options[fd.choice])
	if !focused {
		return value
	}
	return faintStyle.Render("‹ ") + value + faintStyle.Render(" ›")
}

func (f *form) bindings() []key.Binding {
	b := []key.Binding{keySubmit, keyNext}
	if f.fields[f.focus].options != nil {
		b = append(b, keyChoose)
	}
	return append(b, keyClose)
}
