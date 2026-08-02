package ui

import (
	"fmt"
	"strings"
	"time"

	"cosmic-dashboard/collect"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	amber   = lipgloss.Color("#FFD700")
	cyan    = lipgloss.Color("#00FFFF")
	dim     = lipgloss.Color("#666666")
	green   = lipgloss.Color("#00FF00")
	red     = lipgloss.Color("#FF4444")
	borderC = lipgloss.Color("#334455")
)

const panelWidth = 78

var (
	topTitleStyle = lipgloss.NewStyle().
		Foreground(amber).
		Bold(true)

	topSubStyle = lipgloss.NewStyle().
		Foreground(dim)

	dimStyle = lipgloss.NewStyle().
		Foreground(dim)

	statusOk = lipgloss.NewStyle().
		Foreground(green).
		Render("●")

	statusWarn = lipgloss.NewStyle().
		Foreground(red).
		Render("●")

	sectionStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(cyan)

	containerStyle = lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(borderC).
		Padding(0, 1).
		Width(panelWidth).
		Align(lipgloss.Left)

	containerNoPadStyle = lipgloss.NewStyle()

	tabActiveStyle = lipgloss.NewStyle().
		Foreground(cyan).
		Background(lipgloss.Color("#2a4a6a")).
		Bold(true).
		Padding(0, 1)

	tabInactiveStyle = lipgloss.NewStyle().
		Foreground(dim).
		Background(lipgloss.Color("#1a2a3a")).
		Padding(0, 1)

	tabBarContainerStyle = lipgloss.NewStyle().
		Background(lipgloss.Color("#0a1520"))

	footerStyle = lipgloss.NewStyle().
		Foreground(dim)
)

func panel(content string, borderColor lipgloss.Color) string {
	return lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(borderColor).
		Padding(0, 1).
		Width(panelWidth).
		Render(content)
}

type connectionTick struct{}

func connectionTickCmd() tea.Cmd {
	return tea.Tick(300*time.Millisecond, func(t time.Time) tea.Msg {
		return connectionTick{}
	})
}

func Run(state *collect.DashboardState, username string) error {
	p := tea.NewProgram(&model{
		state:    state,
		username: username,
		tab:      0,
		connStep: 0,
	}, tea.WithAltScreen())
	_, err := p.Run()
	return err
}

type model struct {
	state      *collect.DashboardState
	username   string
	tab        int
	connStep   int
	relaysStep int
	scroll     int
	termH      int
}

func (m model) Init() tea.Cmd {
	return connectionTickCmd()
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.termH = msg.Height
	case connectionTick:
		if m.tab == 0 {
			if m.connStep < len(connectionLines) {
				m.connStep++
				return m, connectionTickCmd()
			}
			// Connection done — animate relays in
			if m.relaysStep < len(m.state.Users)+1 {
				m.relaysStep++
				return m, connectionTickCmd()
			}
		}
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			return m, tea.Quit
		case "tab", "right":
			m.tab = (m.tab + 1) % 4
			m.scroll = 0 // reset scroll on tab change
		case "shift+tab", "left":
			m.tab = (m.tab + 3) % 4
			m.scroll = 0
		case "up", "k":
			if m.scroll > 0 {
				m.scroll--
			}
		case "down", "j":
			m.scroll++
		}
	}
	return m, nil
}

func (m model) View() string {
	var b strings.Builder

	// Fixed header (6 lines)
	b.WriteString(lipgloss.NewStyle().Foreground(amber).Render(strings.Repeat("═", 80)))
	b.WriteString("\n")
	b.WriteString(topTitleStyle.Render("QEC RELAY HUB — SOL SECTOR — RS.001@L4"))
	b.WriteString("\n")
	b.WriteString(topSubStyle.Render("REMOTE RELAY: " + strings.ToUpper(m.username)))
	b.WriteString("\n")
	b.WriteString(lipgloss.NewStyle().Foreground(amber).Render(strings.Repeat("═", 80)))
	b.WriteString("\n")
	b.WriteString(tabBarContainerStyle.Render(m.renderTabBar()))
	b.WriteString("\n")

	// Calculate available panel height: total - header(6) - footer(1) - spacing(1)
	availH := m.termH - 8
	if availH < 4 {
		availH = 4
	}

	// Render full panel content, then clip
	fullContent := m.renderActivePanel()
	lines := strings.Split(fullContent, "\n")
	totalLines := len(lines)

	// Clamp scroll offset
	if m.scroll < 0 {
		m.scroll = 0
	}
	maxScroll := totalLines - availH
	if maxScroll < 0 {
		maxScroll = 0
	}
	if m.scroll > maxScroll {
		m.scroll = maxScroll
	}

	// Show scroll-up indicator if content above
	if m.scroll > 0 {
		b.WriteString(lipgloss.NewStyle().Foreground(dim).Render(" ▲ "))
		b.WriteString("\n")
		availH-- // use one less line for the indicator
	}

	// Clip and render visible lines
	start := m.scroll
	end := start + availH
	if end > totalLines {
		end = totalLines
	}
	for _, line := range lines[start:end] {
		b.WriteString(line)
		b.WriteString("\n")
	}

	// Show scroll-down indicator if content below
	if m.scroll+availH < totalLines {
		b.WriteString(lipgloss.NewStyle().Foreground(dim).Render(" ▼ "))
		b.WriteString("\n")
	}

	// Fixed footer
	b.WriteString(footerStyle.Render("q drop to relay shell  tab/→ next  shift+tab/← prev  j/k scroll"))

	return b.String()
}

func (m model) renderTabBar() string {
	tabs := []string{"CONNECTION", "DIAGNOSTICS", "QEC SIDE COMMS", "MAINTENANCE"}
	var parts []string
	for i, t := range tabs {
		if i == m.tab {
			parts = append(parts, tabActiveStyle.Render(t))
		} else {
			parts = append(parts, tabInactiveStyle.Render(t))
		}
	}
	return strings.Join(parts, " ")
}

func (m model) renderActivePanel() string {
	switch m.tab {
	case 0:
		return containerNoPadStyle.Render(m.renderConnection())
	case 1:
		return containerNoPadStyle.Render(m.renderDiagnostics())
	case 2:
		return containerNoPadStyle.Render(m.renderSideComms())
	case 3:
		return containerStyle.Render(m.renderMaintenance())
	default:
		return ""
	}
}

var connectionLines = []string{
	"[SYS]  QEC relay hub initializing .... OK",
	"[SYS]  Quantum entanglement link ..... establishing",
	"[NET]  Phase alignment ............... OK",
	"[NET]  Qubit synchronization ......... OK",
	"[NET]  Entropy pool .................. seeded",
	"[AUTH] Relay handshake ............... complete",
	"[OK]   Link established. All systems nominal.",
}

func (m model) renderConnection() string {
	var linkStatus strings.Builder
	linkStatus.WriteString(sectionStyle.Render("LINK STATUS"))
	linkStatus.WriteString("\n\n")

	maxStep := m.connStep
	if maxStep > len(connectionLines) {
		maxStep = len(connectionLines)
	}

	for i := 0; i < maxStep; i++ {
		line := connectionLines[i]
		if strings.HasPrefix(line, "[OK]") {
			linkStatus.WriteString(lipgloss.NewStyle().Foreground(green).Render(line))
		} else if strings.HasPrefix(line, "[AUTH]") {
			linkStatus.WriteString(lipgloss.NewStyle().Foreground(amber).Render(line))
		} else if strings.HasPrefix(line, "[NET]") {
			linkStatus.WriteString(lipgloss.NewStyle().Foreground(cyan).Render(line))
		} else {
			linkStatus.WriteString(line)
		}
		linkStatus.WriteString("\n")
	}

	if m.connStep < len(connectionLines) {
		linkStatus.WriteString(dimStyle.Render("▌"))
	} else {
		linkStatus.WriteString("\n")
		linkStatus.WriteString(dimStyle.Render("────────────────────────────────────────"))
		linkStatus.WriteString("\n")
		linkStatus.WriteString(lipgloss.NewStyle().Foreground(amber).Render("WELCOME MESSAGE:"))
		linkStatus.WriteString("\n")

		wrapped := wrapText(m.state.Fortune, 52)
		for _, line := range strings.Split(wrapped, "\n") {
			linkStatus.WriteString(line)
			linkStatus.WriteString("\n")
		}
	}

	top := panel(linkStatus.String(), cyan)

	// Only show ACTIVE REMOTE RELAYS after connection animation completes
	if m.connStep >= len(connectionLines) {
		var relays strings.Builder
		relays.WriteString(sectionStyle.Render(fmt.Sprintf("ACTIVE REMOTE RELAYS (%d linked)", len(m.state.Users))))
		relays.WriteString("\n\n")

		// Animate relays in: relaysStep 0 = header only, then each user appears one at a time
		visibleUsers := m.relaysStep
		if visibleUsers > len(m.state.Users) {
			visibleUsers = len(m.state.Users)
		}
		for i, u := range m.state.Users {
			if i >= visibleUsers {
				break
			}
			relays.WriteString("  ")
			relays.WriteString(statusOk)
			relays.WriteString(" " + u)
			relays.WriteString("\n")
		}

		bottom := panel(relays.String(), amber)
		return lipgloss.JoinVertical(lipgloss.Left, top, bottom)
	}

	return top
}

// wrapText wraps text to maxWidth, breaking on spaces
func wrapText(text string, maxWidth int) string {
	words := strings.Fields(text)
	if len(words) == 0 {
		return text
	}

	var lines []string
	var current string

	for _, word := range words {
		if len(current)+len(word)+1 <= maxWidth {
			if current == "" {
				current = word
			} else {
				current += " " + word
			}
		} else {
			if current != "" {
				lines = append(lines, current)
			}
			current = word
		}
	}
	if current != "" {
		lines = append(lines, current)
	}
	return strings.Join(lines, "\n")
}

// progressBar renders a simple progress bar: [████░░░░░░] 62%
// pct is 0.0–1.0, returns colored bar string
func progressBar(pct float64, width int) string {
	if pct > 1.0 {
		pct = 1.0
	}
	if pct < 0 {
		pct = 0
	}
	filled := int(pct * float64(width))
	empty := width - filled

	// Choose color based on usage level
	var fillColor lipgloss.Color
	if pct >= 0.9 {
		fillColor = red
	} else if pct >= 0.7 {
		fillColor = amber
	} else {
		fillColor = green
	}

	fill := lipgloss.NewStyle().Foreground(fillColor).Render(strings.Repeat("█", filled))
	emptyStr := dimStyle.Render(strings.Repeat("░", empty))
	return "[" + fill + emptyStr + "]"
}

func (m model) renderDiagnostics() string {
	var b strings.Builder
	b.WriteString(sectionStyle.Render("RELAY DIAGNOSTICS"))
	b.WriteString("\n\n")
	b.WriteString("Status:   ")
	b.WriteString(statusOk)
	b.WriteString(" ")
	b.WriteString(lipgloss.NewStyle().Foreground(green).Bold(true).Render(m.state.System.RelayStatus))
	b.WriteString("\nUptime:   ")
	b.WriteString(dimStyle.Render(m.state.System.Uptime))
	b.WriteString("\nLoad:     ")
	b.WriteString(lipgloss.NewStyle().Foreground(cyan).Render(m.state.System.LoadAvg))
	b.WriteString("\nCPU:      ")
	b.WriteString(dimStyle.Render(m.state.System.CPU))

	// Memory bar
	b.WriteString("\n\nMemory:")
	memBar := progressBar(m.state.System.MemoryPct / 100.0, 30)
	b.WriteString("\n  " + memBar)
	b.WriteString(" ")
	b.WriteString(dimStyle.Render(fmt.Sprintf("%.0f%%", m.state.System.MemoryPct)))

	// Disk bar
	b.WriteString("\n\nDisk:")
	diskBar := progressBar(m.state.System.DiskPct, 30)
	b.WriteString("\n  " + diskBar)
	if m.state.System.Disk != "" {
		parts := strings.Fields(m.state.System.Disk)
		for _, p := range parts {
			if strings.HasSuffix(p, "%") {
				b.WriteString(" ")
				b.WriteString(dimStyle.Render(p))
				break
			}
		}
	}

	// Solar wind
	b.WriteString("\n\nSolar Wind:")
	for _, line := range strings.Split(m.state.SolarWind, "\n") {
		b.WriteString("\n  " + dimStyle.Render(line))
	}

	return panel(b.String(), cyan)
}

func (m model) renderSideComms() string {
	// Private messages subpanel
	var mail strings.Builder
	mail.WriteString(sectionStyle.Render("PRIVATE MESSAGES"))
	mail.WriteString("\n\n")
	if m.state.MailCount < 0 {
		mail.WriteString("No private messages\n")
	} else if m.state.MailCount > 0 {
		mail.WriteString(fmt.Sprintf("%d messages waiting\n", m.state.MailCount))
	} else {
		mail.WriteString("Inbox clear\n")
	}

	// Channels subpanel
	var channels strings.Builder
	channels.WriteString(sectionStyle.Render("NET NEWS GROUPS"))
	channels.WriteString("\n")
	channels.WriteString(lipgloss.NewStyle().Foreground(dim).Render("  New messages since last login"))
	channels.WriteString("\n\n")

	for _, ng := range m.state.Newsgroups {
		channels.WriteString(lipgloss.NewStyle().Bold(true).Foreground(amber).Render(ng.Name))
		channels.WriteString(fmt.Sprintf(" %d messages", ng.NewCount))
		channels.WriteString("\n")
	}

	if len(m.state.Newsgroups) == 0 {
		channels.WriteString("No relay channels available.\n")
	}

	top := panel(mail.String(), amber)
	bottom := panel(channels.String(), cyan)

	return lipgloss.JoinVertical(lipgloss.Left, top, bottom)
}

func (m model) renderMaintenance() string {
	var b strings.Builder

	for _, entry := range m.state.Changelog {
		b.WriteString("◆ ")
		b.WriteString(entry)
		b.WriteString("\n\n")
	}

	return b.String()
}
