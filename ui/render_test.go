package ui

import (
	"fmt"
	"regexp"
	"strings"
	"testing"
	"unicode/utf8"

	"cosmic-dashboard/collect"
)

func testState() *collect.DashboardState {
	return &collect.DashboardState{
		System: collect.SystemInfo{
			RelayStatus: "nominal",
			Uptime:      "3d 14h 22m",
			LoadAvg:     "0.42, 0.38, 0.31",
			CPU:         "AMD Ryzen 7 7700",
			MemoryPct:   62.0,
			Disk:        "/ 42% used",
			DiskPct:     0.42,
		},
		Users:      []string{"tomasino", "visitor1"},
		Fortune:    "The cosmos is within us. We are made of star-stuff. We are a way for the universe to know itself.",
		Newsgroups: []collect.NewsgroupInfo{
			{Name: "cosmic.general", NewCount: 14},
			{Name: "cosmic.tech", NewCount: 7},
			{Name: "cosmic.fiction", NewCount: 3},
		},
		Changelog: []string{"2026-08-02 Updated QEC relay firmware", "2026-08-01 Patched entropy pool", "2026-07-30 Added new relay node"},
		MailCount: 5,
		SolarWind:   "Bt: 13 nT\nBz: 7 nT",
	}
}

const testLayoutWidth = 80

var ansiRe = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

func stripANSI(s string) string {
	return ansiRe.ReplaceAllString(s, "")
}

func TestTabBarNoWrap(t *testing.T) {
	m := &model{state: testState(), username: "tomasino", tab: 0, connStep: 0, termH: 24}
	tabs := m.renderTabBar()
	lines := strings.Split(tabs, "\n")
	if len(lines) > 1 {
		t.Errorf("Tab bar wraps across %d lines, expected 1:\n%s", len(lines), tabs)
	}
}

func TestWelcomeMessageLeftAligned(t *testing.T) {
	m := &model{state: testState(), username: "tomasino", tab: 0, connStep: 7, termH: 24}
	view := m.View()
	lines := strings.Split(view, "\n")
	for i, line := range lines {
		// After the separator line, content should start at column 0 (no leading spaces before the container border)
		if strings.Contains(line, "WELCOME MESSAGE") {
			if strings.HasPrefix(line, " ") {
				t.Errorf("Line %d: WELCOME MESSAGE is indented (right-aligned), expected flush left:\n%q", i, line)
			}
		}
		if strings.Contains(line, "The cosmos") {
			if strings.HasPrefix(line, " ") && !strings.HasPrefix(line, " ╭") {
				t.Errorf("Line %d: fortune text is indented, expected flush left:\n%q", i, line)
			}
		}
	}
}

func TestNoLineLongerThanLayoutWidth(t *testing.T) {
	for tab := 0; tab < 4; tab++ {
		m := &model{state: testState(), username: "tomasino", tab: tab, connStep: 7, termH: 24}
		view := m.View()
		lines := strings.Split(view, "\n")
		for i, line := range lines {
			clean := stripANSI(line)
			runeCount := utf8.RuneCountInString(clean)
			if runeCount > testLayoutWidth {
				t.Errorf("Tab %d, line %d: %d visible runes exceeds layoutWidth %d:\n%q", tab, i, runeCount, testLayoutWidth, clean)
			}
		}
	}
}

func TestMain(m *testing.M) {
	// Quick render preview for each tab
	for tab := 0; tab < 4; tab++ {
		mm := &model{state: testState(), username: "tomasino", tab: tab, connStep: 7, termH: 24}
		view := mm.View()
		clean := stripANSI(view)
		fmt.Printf("\n=== TAB %d ===\n%s\n", tab, clean)
	}
	m.Run()
}
