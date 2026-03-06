package main

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
	"github.com/cryptx/cryptx-cli/config"
	aw "github.com/cryptx/cryptx-cli/internal/appwrite"
	"github.com/cryptx/cryptx-cli/internal/session"
	"github.com/cryptx/cryptx-cli/internal/tui"
	"github.com/cryptx/cryptx-cli/internal/waha"
)

func main() {
	// ── Load configuration ────────────────────────────────────────────────
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "configuration error: %v\n\nCopy .env.example to .env and fill in your values.\n", err)
		os.Exit(1)
	}

	// ── Initialise Appwrite services (bare — no session yet) ─────────────
	svc := aw.NewWithSession(cfg, "")

	// ── Restore session (skip login if still valid) ───────────────────────
	var sess *session.Session
	if saved, loadErr := session.Load(); loadErr == nil {
		// Validate the session against Appwrite before trusting it.
		if validateErr := aw.ValidateSession(cfg, saved); validateErr == nil {
			sess = saved
			svc = aw.NewWithSession(cfg, saved.SessionSecret)
		}
	}

	// ── Initialise WAHA client (optional — nil when base URL not set) ─────
	var wahaClient *waha.Client
	if cfg.WAHABaseURL != "" {
		wahaClient = waha.NewClient(cfg.WAHABaseURL, cfg.WAHAAPIKey, cfg.WAHASession)
	}

	// ── Build and run the TUI ─────────────────────────────────────────────
	app := tui.NewApp(svc, wahaClient, sess)

	p := tea.NewProgram(app)

	if _, runErr := p.Run(); runErr != nil {
		fmt.Fprintf(os.Stderr, "TUI error: %v\n", runErr)
		os.Exit(1)
	}
}
