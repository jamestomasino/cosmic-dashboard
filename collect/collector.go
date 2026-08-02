package collect

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"
)

// DashboardState holds all data for the dashboard panels
type DashboardState struct {
	System     SystemInfo
	Users      []string
	Newsgroups []NewsgroupInfo
	Changelog  []string
	Fortune    string
	SolarWind   string
	MailCount  int
	Username   string
}

type SystemInfo struct {
	Uptime      string
	LoadAvg     string
	MemoryPct   float64
	Disk        string
	DiskPct     float64
	CPU         string
	RelayStatus string
}

type NewsgroupInfo struct {
	Name     string
	NewCount int
	Low      int
}

// CollectAll runs all collectors concurrently with context timeout
func CollectAll(ctx context.Context, userState *State) *DashboardState {
	state := &DashboardState{}

	collectors := []struct {
		name string
		fn   func(ctx context.Context, state *DashboardState, done chan struct{})
	}{
		{"system", collectSystem},
		{"users", collectUsers},
		{"fortune", collectFortune},
		{"newsgroups", func(ctx context.Context, state *DashboardState, done chan struct{}) { collectNewsgroups(ctx, state, userState) }},
		{"changelog", collectChangelog},
		{"solar", collectSolarWind},
		{"mail", collectMail},
	}

	done := make(chan struct{})
	for _, c := range collectors {
		go c.fn(ctx, state, done)
	}

	// Wait for context timeout (collectors will exit when ctx is done)
	<-ctx.Done()

	return state
}

func collectSystem(ctx context.Context, state *DashboardState, done chan struct{}) {
	// Uptime
	if out, err := exec.CommandContext(ctx, "uptime", "-p").Output(); err == nil {
		state.System.Uptime = strings.TrimSpace(string(out))
	}

	// Load average
	if out, err := exec.CommandContext(ctx, "cat", "/proc/loadavg").Output(); err == nil {
		parts := strings.Fields(string(out))
		if len(parts) >= 3 {
			state.System.LoadAvg = fmt.Sprintf("%s / %s / %s", parts[0], parts[1], parts[2])
		}
	}

	// Memory
	if out, err := exec.CommandContext(ctx, "free").Output(); err == nil {
		lines := strings.Split(string(out), "\n")
		if len(lines) >= 2 {
			parts := strings.Fields(lines[1])
			if len(parts) >= 3 {
				var totalMB, usedMB int
				fmt.Sscanf(parts[1], "%d", &totalMB)
				fmt.Sscanf(parts[2], "%d", &usedMB)
				if totalMB > 0 {
					state.System.MemoryPct = float64(usedMB) / float64(totalMB) * 100
				}
			}
		}
	}

	// Disk
	if out, err := exec.CommandContext(ctx, "df", "-h", "/").Output(); err == nil {
		lines := strings.Split(string(out), "\n")
		if len(lines) >= 2 {
			state.System.Disk = lines[1]
			// Extract percentage (e.g., "45%")
			parts := strings.Fields(lines[1])
			for _, p := range parts {
				if strings.HasSuffix(p, "%") {
					var pct float64
					fmt.Sscanf(p, "%f%%", &pct)
					state.System.DiskPct = pct / 100.0
					break
				}
			}
		}
	}

	// CPU
	if out, err := exec.CommandContext(ctx, "grep", "-c", "^processor", "/proc/cpuinfo").Output(); err == nil {
		state.System.CPU = strings.TrimSpace(string(out)) + " cores"
	}

	state.System.RelayStatus = "operational"
}

func collectUsers(ctx context.Context, state *DashboardState, done chan struct{}) {
	seen := make(map[string]bool)
	if out, err := exec.CommandContext(ctx, "who").Output(); err == nil {
		for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
			if line != "" {
				parts := strings.Fields(line)
				if len(parts) >= 1 {
					name := strings.ToUpper(parts[0])
					if !seen[name] {
						seen[name] = true
						state.Users = append(state.Users, name)
					}
				}
			}
		}
	}
	sort.Strings(state.Users)
}

func collectFortune(ctx context.Context, state *DashboardState, done chan struct{}) {
	if out, err := exec.CommandContext(ctx, "fortune").Output(); err == nil {
		state.Fortune = strings.TrimSpace(string(out))
	} else {
		state.Fortune = "No transmissions available."
	}
}

func collectNewsgroups(ctx context.Context, state *DashboardState, userState *State) {
	groups, err := fetchNewsgroups(ctx)
	if err != nil {
		state.Newsgroups = []NewsgroupInfo{
			{Name: "nntp-unavailable", NewCount: 0},
		}
		return
	}

	// Calculate new messages since last login and update high watermarks
	for i := range groups {
		lastHigh := userState.LastNewsgroupHigh[groups[i].Name]
		currentHigh := groups[i].NewCount + groups[i].Low - 1
		if lastHigh > 0 && currentHigh > lastHigh {
			groups[i].NewCount = currentHigh - lastHigh
		} else if lastHigh > 0 {
			groups[i].NewCount = 0
		}
		groups[i].Low = 0 // no longer needed, NewCount now means "new since last login"
		userState.LastNewsgroupHigh[groups[i].Name] = currentHigh
	}
	state.Newsgroups = groups
}

func collectChangelog(ctx context.Context, state *DashboardState, done chan struct{}) {
	entries, err := parseChangelog(ctx)
	if err != nil {
		state.Changelog = []string{"Relay systems operational"}
		return
	}
	if len(entries) == 0 {
		state.Changelog = []string{"No recent relay updates"}
		return
	}
	state.Changelog = entries
}

// parseChangelog reads /var/wiki/changelog.html and returns recent entries
func parseChangelog(ctx context.Context) ([]string, error) {
	data, err := exec.CommandContext(ctx, "cat", "/var/wiki/changelog.html").Output()
	if err != nil {
		return nil, err
	}

	text := string(data)
	// Strip <pre> tags
	text = strings.ReplaceAll(text, "<pre>", "")
	text = strings.ReplaceAll(text, "</pre>", "")

	var entries []string
	var currentEntry strings.Builder
	lines := strings.Split(text, "\n")

	for _, line := range lines {
		// Date lines start with whitespace + month abbreviation
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || trimmed == "<pre>" || trimmed == "</pre>" {
			continue
		}
		// Check if this is a date line (e.g., "Feb 17, 2026:")
		if len(trimmed) > 5 {
			month := trimmed[:3]
			if isMonth(month) {
				// Save previous entry
				if currentEntry.Len() > 0 {
					entries = append(entries, strings.TrimSpace(currentEntry.String()))
					currentEntry.Reset()
				}
				currentEntry.WriteString(trimmed)
			} else if currentEntry.Len() > 0 {
				// Continuation line (bullet point)
				currentEntry.WriteString(" " + strings.TrimPrefix(trimmed, "- "))
			}
		}
	}
	// Save last entry
	if currentEntry.Len() > 0 {
		entries = append(entries, strings.TrimSpace(currentEntry.String()))
	}

	// Return only the 5 most recent entries
	if len(entries) > 5 {
		entries = entries[:5]
	}
	return entries, nil
}

func isMonth(s string) bool {
	months := []string{"Jan", "Feb", "Mar", "Apr", "May", "Jun",
		"Jul", "Aug", "Sep", "Oct", "Nov", "Dec"}
	for _, m := range months {
		if s == m {
			return true
		}
	}
	return false
}

func collectSolarWind(ctx context.Context, state *DashboardState, done chan struct{}) {
	cacheFile := os.Getenv("HOME") + "/cache/solar-wind-mag-field.txt"
	data, err := os.ReadFile(cacheFile)
	if err != nil {
		state.SolarWind = "Solar data unavailable"
		return
	}
	state.SolarWind = strings.TrimSpace(string(data))
}

func collectMail(ctx context.Context, state *DashboardState, done chan struct{}) {
	count, err := countUnreadMail(ctx)
	if err != nil {
		state.MailCount = -1 // indicates error
		return
	}
	state.MailCount = count
}

// countUnreadMail counts messages in the user's mbox at /var/mail/<username>
func countUnreadMail(ctx context.Context) (int, error) {
	username := os.Getenv("USER")
	if username == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return 0, err
		}
		username = strings.TrimPrefix(home, "/home/")
	}
	mboxPath := "/var/mail/" + username
	data, err := os.ReadFile(mboxPath)
	if err != nil {
		return -1, err
	}
	// Each message in mbox starts with a "From " line
	count := strings.Count(string(data), "\nFrom ")
	if strings.HasPrefix(string(data), "From ") {
		count++
	}
	return count, nil
}
