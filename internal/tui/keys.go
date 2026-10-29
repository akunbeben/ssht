package tui

import "github.com/charmbracelet/bubbles/key"

func bind(help, desc string, keys ...string) key.Binding {
	return key.NewBinding(key.WithKeys(keys...), key.WithHelp(help, desc))
}

// Browse view bindings. Actions that appear in the palette live in commands().
var keys = struct {
	Up, Down, Top, Bottom, PageUp, PageDown key.Binding
	Connect, Search, Add, Copy, Edit        key.Binding
	Delete, Move, SCP, Profiles, VPN        key.Binding
	VPNToggle, Keys, Privacy, Actions, Help key.Binding
	Quit, Clear, KeepFilter, Sort, SortDir  key.Binding
}{
	Up:         bind("↑/k", "up", "up", "k"),
	Down:       bind("↓/j", "down", "down", "j"),
	Top:        bind("g", "first", "g", "home"),
	Bottom:     bind("G", "last", "G", "end"),
	PageUp:     bind("ctrl+u", "page up", "ctrl+u", "pgup"),
	PageDown:   bind("ctrl+d", "page down", "ctrl+d", "pgdown"),
	Connect:    bind("enter", "connect", "enter"),
	Search:     bind("/", "search", "/"),
	Add:        bind("a", "add", "a"),
	Copy:       bind("c", "duplicate", "c"),
	Edit:       bind("e", "edit", "e"),
	Delete:     bind("d", "delete", "d"),
	Move:       bind("m", "move", "m"),
	SCP:        bind("S", "scp", "S"),
	Profiles:   bind("p", "profiles", "p"),
	VPN:        bind("v", "vpn", "v"),
	VPNToggle:  bind("V", "system vpn", "V"),
	Keys:       bind("K", "ssh keys", "K"),
	Privacy:    bind("*", "mask", "*"),
	Sort:       bind("o", "sort", "o"),
	SortDir:    bind("O", "reverse sort", "O"),
	Actions:    bind("ctrl+k", "actions", "ctrl+k", ":"),
	Help:       bind("?", "help", "?"),
	Quit:       bind("q", "quit", "q"),
	Clear:      bind("esc", "clear", "esc"),
	KeepFilter: bind("tab", "keep filter", "tab"),
}

// Dialog bindings.
var (
	keyClose  = bind("esc", "cancel", "esc")
	keyNext   = bind("tab", "next", "tab", "down")
	keyPrev   = bind("shift+tab", "prev", "shift+tab", "up")
	keySubmit = bind("ctrl+s", "save", "ctrl+s")
	keyEnter  = bind("enter", "next", "enter")
	keyChoose = bind("←/→ h/l", "change", "left", "right", "h", "l")
	keyLeft   = bind("←/h", "previous option", "left", "h")
	keyFieldU = bind("k", "prev", "k")
	keyFieldD = bind("j", "next", "j")
	keySelect = bind("enter", "select", "enter")
	keyListUp = bind("↑/ctrl+k", "up", "up", "ctrl+k", "ctrl+p")
	keyListDn = bind("↓/ctrl+j", "down", "down", "ctrl+j", "ctrl+n")
	keyYes    = bind("y", "confirm", "y", "Y")
	keyNo     = bind("n", "cancel", "n", "N", "esc")
)
