package tui

import (
	"path/filepath"

	k "github.com/akunbeben/ssht/internal/key"
)

func (m *model) refreshPubkeys() {
	keys, err := k.ScanPubkeys()
	if err != nil {
		m.err = err
		m.pubkeys = nil
		m.pubIndex = 0
		return
	}
	m.pubkeys = keys
	if m.pubIndex >= len(m.pubkeys) {
		m.pubIndex = 0
	}
}

func (m *model) currentPubkeyLabel() string {
	if len(m.pubkeys) == 0 {
		return ""
	}
	return filepath.Base(m.pubkeys[m.pubIndex])
}
