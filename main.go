package main

import (
	"context"
	"fmt"
	"os"
	"os/user"
	"time"

	"cosmic-dashboard/collect"
	"cosmic-dashboard/ui"
)

func main() {
	// Skip if COSMIC_SKIP is set
	if os.Getenv("COSMIC_SKIP") != "" {
		return
	}

	// Get current user
	u, err := user.Current()
	if err != nil {
		return
	}

	// Load per-user state
	state := collect.LoadState()
	state.LastLogin = time.Now()

	// Collect data with 2s deadline
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	dashState := collect.CollectAll(ctx, state)
	dashState.Username = u.Username

	// Render dashboard - any error is non-fatal (never kill the shell)
	err = ui.Run(dashState, u.Username)

	// Save state on exit (even on error)
	state.SaveState()

	// Bail silently on any error
	if err != nil {
		return
	}

	fmt.Println("\033[2K\rContinue to relay shell...")
	time.Sleep(500 * time.Millisecond)
}
