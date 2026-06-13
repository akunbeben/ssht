package tui

import (
	"fmt"
	"strings"

	"github.com/akunbeben/ssht/internal/config"
	"github.com/sahilm/fuzzy"
)

func filterServers(servers []config.Server, query string) []config.Server {
	q := strings.TrimSpace(query)
	if q == "" {
		return servers
	}
	candidates := make([]string, 0, len(servers))
	for _, s := range servers {
		port := s.Port
		if port == 0 {
			port = 22
		}
		address := fmt.Sprintf("%s@%s:%d", s.User, s.Host, port)
		candidates = append(candidates, fmt.Sprintf("%s %s %s %s %v %s", s.Name, address, s.Host, s.User, s.Tags, s.Note))
	}
	matches := fuzzy.Find(q, candidates)
	result := make([]config.Server, 0, len(matches))
	for _, m := range matches {
		result = append(result, servers[m.Index])
	}
	return result
}
