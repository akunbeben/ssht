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
		candidates = append(candidates, fmt.Sprintf("%s %s %s %v", s.Name, s.Host, s.User, s.Tags))
	}
	matches := fuzzy.Find(q, candidates)
	result := make([]config.Server, 0, len(matches))
	for _, m := range matches {
		result = append(result, servers[m.Index])
	}
	return result
}
