package cmd

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"os/exec"
	stduser "os/user"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/spf13/cobra"

	"github.com/akunbeben/ssht/internal/config"
)

var (
	importHistoryFile  string
	importHistoryLimit int
	importHistoryDry   bool
)

const defaultHistoryImportLimit = 10000

var serverImportHistoryCmd = &cobra.Command{
	Use:   "import-history",
	Short: "Import SSH targets from shell history",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, profileName, profile, err := loadActiveProfile()
		if err != nil {
			return err
		}

		historyPath, err := resolveHistoryPath(importHistoryFile)
		if err != nil {
			return err
		}
		summary, err := importServersFromHistory(cfg, profileName, profile, historyPath, importHistoryLimit, !importHistoryDry)
		if err != nil {
			return err
		}

		fmt.Fprintf(cmd.OutOrStdout(), "history: %s\n", historyPath)
		fmt.Fprintf(cmd.OutOrStdout(), "parsed: %d, added: %d, skipped: %d\n", summary.Parsed, summary.Added, summary.Skipped)
		if importHistoryDry || summary.Added == 0 {
			if importHistoryDry {
				fmt.Fprintln(cmd.OutOrStdout(), "dry-run enabled; no changes saved")
			}
			return nil
		}
		return nil
	},
}

type importedServer struct {
	Host    string
	User    string
	Port    int
	KeyPath string
}

func init() {
	serverCmd.AddCommand(serverImportHistoryCmd)
	serverImportHistoryCmd.Flags().StringVar(&importHistoryFile, "file", "", "history file path (default: auto detect)")
	serverImportHistoryCmd.Flags().IntVar(&importHistoryLimit, "limit", defaultHistoryImportLimit, "max recent history lines to parse")
	serverImportHistoryCmd.Flags().BoolVar(&importHistoryDry, "dry-run", false, "parse and show summary without saving")
}

type historyImportSummary struct {
	Parsed  int
	Added   int
	Skipped int
}

func autoImportHistory(cfg *config.Config, profileName string) (historyImportSummary, error) {
	profile, ok := cfg.Profiles[profileName]
	if !ok {
		return historyImportSummary{}, fmt.Errorf("profile %q not found", profileName)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return historyImportSummary{}, nil
	}

	allEntries := make([]importedServer, 0)
	seen := map[string]struct{}{}
	for _, path := range allHistoryCandidates(home) {
		if _, err := os.Stat(path); err != nil {
			continue
		}
		entries, err := parseHistorySSH(path, 0)
		if err != nil {
			continue
		}
		for _, e := range entries {
			key := endpointKey(e.User, e.Host, e.Port)
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			allEntries = append(allEntries, e)
		}
	}

	if len(allEntries) == 0 {
		return historyImportSummary{}, nil
	}

	return mergeImportedServers(cfg, profileName, profile, allEntries, true)
}

func importServersFromHistory(
	cfg *config.Config,
	profileName string,
	profile config.Profile,
	historyPath string,
	limit int,
	save bool,
) (historyImportSummary, error) {
	entries, err := parseHistorySSH(historyPath, limit)
	if err != nil {
		return historyImportSummary{}, err
	}
	if len(entries) == 0 {
		return historyImportSummary{}, nil
	}

	return mergeImportedServers(cfg, profileName, profile, entries, save)
}

func mergeImportedServers(
	cfg *config.Config,
	profileName string,
	profile config.Profile,
	entries []importedServer,
	save bool,
) (historyImportSummary, error) {
	existingByName := make(map[string]struct{}, len(profile.Servers))
	existingByEndpoint := make(map[string]struct{}, len(profile.Servers))
	for _, s := range profile.Servers {
		existingByName[s.Name] = struct{}{}
		port := s.Port
		if port == 0 {
			port = 22
		}
		key := endpointKey(s.User, s.Host, port)
		existingByEndpoint[key] = struct{}{}
	}

	exceptions := make(map[string]struct{}, len(profile.ImportExceptions))
	for _, ex := range profile.ImportExceptions {
		exceptions[ex] = struct{}{}
	}

	summary := historyImportSummary{Parsed: len(entries)}
	for _, e := range entries {
		key := endpointKey(e.User, e.Host, e.Port)
		if _, ok := existingByEndpoint[key]; ok {
			summary.Skipped++
			continue
		}
		if _, ok := exceptions[key]; ok {
			summary.Skipped++
			continue
		}
		name := uniqueServerName(deriveServerName(e), existingByName)
		s := config.Server{
			ID:      uuid.NewString(),
			Name:    name,
			Host:    e.Host,
			Port:    e.Port,
			User:    e.User,
			KeyPath: e.KeyPath,
			Tags:    []string{"imported", "history"},
			Note:    "imported from shell history",
		}
		profile.Servers = append(profile.Servers, s)
		existingByName[name] = struct{}{}
		existingByEndpoint[key] = struct{}{}
		summary.Added++
	}

	if save && summary.Added > 0 {
		cfg.Profiles[profileName] = profile
		if err := config.Save(cfg); err != nil {
			return historyImportSummary{}, err
		}
	}
	return summary, nil
}

func resolveHistoryPath(custom string) (string, error) {
	if custom != "" {
		return config.ExpandHome(custom)
	}

	if histFile := strings.TrimSpace(os.Getenv("HISTFILE")); histFile != "" {
		expanded, err := config.ExpandHome(histFile)
		if err == nil {
			if _, err := os.Stat(expanded); err == nil {
				return expanded, nil
			}
		}
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home dir: %w", err)
	}

	shell := detectCurrentShell()
	if shell != "" {
		for _, p := range pathsForShell(home, shell) {
			if _, err := os.Stat(p); err == nil {
				return p, nil
			}
		}
	}

	return mostRecentHistory(home)
}

func detectCurrentShell() string {
	ppid := os.Getppid()
	if ppid > 0 {
		out, err := exec.Command("ps", "-p", strconv.Itoa(ppid), "-o", "comm=").Output()
		if err == nil {
			name := filepath.Base(strings.TrimSpace(string(out)))
			name = strings.TrimPrefix(name, "-")
			if name != "" {
				return name
			}
		}
	}

	if shell := strings.TrimSpace(os.Getenv("SHELL")); shell != "" {
		return filepath.Base(shell)
	}
	return ""
}

func pathsForShell(home, shell string) []string {
	switch shell {
	case "fish":
		return []string{filepath.Join(home, ".local", "share", "fish", "fish_history")}
	case "zsh":
		return []string{
			filepath.Join(home, ".zsh_history"),
			filepath.Join(home, ".zhistory"),
		}
	case "bash":
		return []string{filepath.Join(home, ".bash_history")}
	case "nu", "nushell":
		return []string{filepath.Join(home, ".config", "nushell", "history.txt")}
	case "xonsh":
		return []string{filepath.Join(home, ".xonsh_history")}
	}
	return nil
}

func allHistoryCandidates(home string) []string {
	return []string{
		filepath.Join(home, ".local", "share", "fish", "fish_history"),
		filepath.Join(home, ".zsh_history"),
		filepath.Join(home, ".zhistory"),
		filepath.Join(home, ".bash_history"),
		filepath.Join(home, ".config", "nushell", "history.txt"),
		filepath.Join(home, ".xonsh_history"),
	}
}

func mostRecentHistory(home string) (string, error) {
	type candidate struct {
		path    string
		modTime int64
	}

	var best candidate
	for _, p := range allHistoryCandidates(home) {
		info, err := os.Stat(p)
		if err != nil {
			continue
		}
		mt := info.ModTime().UnixNano()
		if best.path == "" || mt > best.modTime {
			best = candidate{path: p, modTime: mt}
		}
	}

	if best.path == "" {
		return "", fmt.Errorf("no shell history file found; use --file to set one")
	}
	return best.path, nil
}

func parseHistorySSH(path string, limit int) ([]importedServer, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open history: %w", err)
	}
	defer f.Close()

	lines := make([]string, 0, 4096)
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		lines = append(lines, sc.Text())
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("scan history: %w", err)
	}
	if limit <= 0 || limit > len(lines) {
		limit = len(lines)
	}
	lines = lines[len(lines)-limit:]

	parsed := make([]importedServer, 0)
	seen := map[string]struct{}{}
	for i := len(lines) - 1; i >= 0; i-- {
		cmdLine := normalizeHistoryLine(lines[i])
		if cmdLine == "" {
			continue
		}
		entry, ok := parseSSHCommand(cmdLine)
		if !ok {
			continue
		}
		key := endpointKey(entry.User, entry.Host, entry.Port)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		parsed = append(parsed, entry)
	}
	return parsed, nil
}

func normalizeHistoryLine(line string) string {
	line = strings.TrimSpace(line)
	if line == "" {
		return ""
	}
	if strings.HasPrefix(line, "- cmd:") {
		return strings.TrimSpace(strings.TrimPrefix(line, "- cmd:"))
	}
	if strings.HasPrefix(line, ": ") {
		if idx := strings.Index(line, ";"); idx >= 0 && idx+1 < len(line) {
			return strings.TrimSpace(line[idx+1:])
		}
	}
	return line
}

func parseSSHCommand(line string) (importedServer, bool) {
	fields := strings.Fields(line)
	if len(fields) == 0 {
		return importedServer{}, false
	}

	fields = stripCommandWrappers(fields)
	if len(fields) == 0 {
		return importedServer{}, false
	}

	if fields[0] == "sudo" {
		fields = fields[1:]
	}
	fields = stripCommandWrappers(fields)
	if len(fields) == 0 || fields[0] != "ssh" {
		return importedServer{}, false
	}
	fields = fields[1:]

	port := 22
	keyPath := ""
	target := ""
	forcedUser := ""
	nonInteractive := false
	flagsWithValue := map[string]struct{}{
		"-b": {}, "-c": {}, "-D": {}, "-E": {}, "-e": {}, "-F": {}, "-I": {}, "-J": {},
		"-L": {}, "-m": {}, "-O": {}, "-o": {}, "-Q": {}, "-R": {}, "-S": {}, "-W": {}, "-w": {},
	}
	for i := 0; i < len(fields); i++ {
		part := fields[i]
		switch part {
		case "-p", "-l", "-i":
			if i+1 >= len(fields) {
				return importedServer{}, false
			}
			next := fields[i+1]
			switch part {
			case "-p":
				p, err := strconv.Atoi(next)
				if err != nil {
					return importedServer{}, false
				}
				port = p
			case "-l":
				forcedUser = next
			case "-i":
				keyPath = next
			}
			i++
		default:
			if strings.HasPrefix(part, "-") {
				if _, ok := flagsWithValue[part]; ok && i+1 < len(fields) {
					i++
					continue
				}
				if part == "-T" || part == "-N" {
					nonInteractive = true
				}
				if strings.HasPrefix(part, "-p") && len(part) > 2 {
					p, err := strconv.Atoi(strings.TrimPrefix(part, "-p"))
					if err == nil {
						port = p
					}
				}
				continue
			}
			if target == "" {
				target = part
				continue
			}

		}
	}
	if target == "" || nonInteractive {
		return importedServer{}, false
	}

	user := ""
	host := target
	if at := strings.LastIndex(target, "@"); at > 0 && at+1 < len(target) {
		user = target[:at]
		host = target[at+1:]
	}

	if h, p, err := net.SplitHostPort(host); err == nil {
		host = h
		if parsed, convErr := strconv.Atoi(p); convErr == nil {
			port = parsed
		}
	}
	host = strings.Trim(host, "[]")
	if host == "" {
		return importedServer{}, false
	}
	if user == "" {
		user = forcedUser
	}
	if user == "" {
		user = defaultSSHUser()
	}
	return importedServer{Host: host, User: user, Port: port, KeyPath: keyPath}, true
}

func deriveServerName(e importedServer) string {
	host := e.Host
	if net.ParseIP(host) != nil {
		if e.KeyPath != "" {
			name := strings.ToLower(filepath.Base(e.KeyPath))
			name = strings.TrimSuffix(name, filepath.Ext(name))
			name = strings.TrimPrefix(name, "id_")
			// filter out generic default keys
			if name != "rsa" && name != "ed25519" && name != "ecdsa" && name != "dsa" && name != "" {
				name = strings.ReplaceAll(name, "_", "-")
				name = strings.ReplaceAll(name, ".", "-")
				return name
			}
		}
		return "imported-server"
	}

	name := strings.ToLower(strings.TrimSpace(host))
	name = strings.ReplaceAll(name, ".", "-")
	name = strings.ReplaceAll(name, "_", "-")
	name = strings.ReplaceAll(name, ":", "-")
	name = strings.ReplaceAll(name, "[", "")
	name = strings.ReplaceAll(name, "]", "")
	if name == "" {
		return "server"
	}
	return name
}

func uniqueServerName(base string, existing map[string]struct{}) string {
	if _, ok := existing[base]; !ok {
		return base
	}
	for i := 2; ; i++ {
		candidate := fmt.Sprintf("%s-%d", base, i)
		if _, ok := existing[candidate]; !ok {
			return candidate
		}
	}
}

func endpointKey(user, host string, port int) string {
	return fmt.Sprintf("%s@%s:%d", user, host, port)
}

func stripCommandWrappers(fields []string) []string {
	for len(fields) > 0 {
		head := fields[0]
		if head == "command" || head == "noglob" || head == "nocorrect" || head == "builtin" {
			fields = fields[1:]
			continue
		}
		if strings.Contains(head, "=") && !strings.HasPrefix(head, "-") {
			fields = fields[1:]
			continue
		}
		break
	}
	return fields
}

func defaultSSHUser() string {
	current, err := stduser.Current()
	if err != nil || current.Username == "" {
		return "root"
	}
	return current.Username
}
