package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/user"
	"strings"
	"time"

	"cosmic-dashboard/collect"
	"cosmic-dashboard/ui"
)

func main() {
	// Skip if COSMIC_SKIP is set
	if os.Getenv("COSMIC_SKIP") != "" {
		return
	}

	// Only run for interactive shells
	if os.Getenv("PS1") == "" && os.Getenv("TERM") == "" {
		return
	}

	// Get current user
	u, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}

	// Load per-user state
	state := collect.LoadState()
	state.LastLogin = time.Now()

	// Collect data with 2s deadline
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	dashState := collect.CollectAll(ctx, state)
	dashState.Username = u.Username

	// Render dashboard
	err = ui.Run(dashState, u.Username)

	// Save state on exit (even on error)
	state.SaveState()

	// Only fatal on real errors, not no-TTY (which happens in scripts/tests)
	if err != nil && !strings.Contains(err.Error(), "not a terminal") {
		log.Fatal(err)
	}

	fmt.Println("\033[2K\rDropping to relay shell...")
	time.Sleep(500 * time.Millisecond)
}
