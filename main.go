package main

import (
	"fmt"
	"os"
	"path/filepath"

	tea "charm.land/bubbletea/v2"
	"github.com/cryptx/cryptx-cli/config"
	aw "github.com/cryptx/cryptx-cli/internal/appwrite"
	"github.com/cryptx/cryptx-cli/internal/session"
	"github.com/cryptx/cryptx-cli/internal/tui"
	"github.com/cryptx/cryptx-cli/internal/upgrade"
	"github.com/cryptx/cryptx-cli/internal/waha"
)

// Version is injected at build time via -ldflags "-X main.Version=vX.Y.Z".
// When running with `go run .` it falls back to "dev".
var Version = "dev"

func main() {
	// ── Self-upgrade handover ─────────────────────────────────────────────
	// When launched as the replacement binary by HandoverTo, the first two
	// arguments are  --upgrade-from <old-binary-path>.  Handle this before
	// loading config so the operation is as fast and isolated as possible.
	if len(os.Args) == 3 && os.Args[1] == "--upgrade-from" {
		oldPath := os.Args[2]
		if err := upgrade.ApplyFrom(oldPath); err != nil {
			fmt.Fprintf(os.Stderr, "upgrade error: %v\n", err)
			os.Exit(1)
		}
		// ApplyFrom never returns on success (exec replacement).
		os.Exit(0)
	}

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
	app := tui.NewApp(svc, wahaClient, sess, Version)

	p := tea.NewProgram(app)

	finalModel, runErr := p.Run()
	if runErr != nil {
		fmt.Fprintf(os.Stderr, "TUI error: %v\n", runErr)
		os.Exit(1)
	}

	// ── Post-TUI: perform pending self-upgrade handover ───────────────────
	// The TUI calls tea.Quit after successfully downloading a new binary and
	// stores its temp path in the model.  We resolve the current executable
	// path here (after the TUI has cleaned up the terminal) and hand off.
	if a, ok := finalModel.(tui.App); ok {
		if pending := a.PendingUpgrade(); pending != "" {
			current, resolveErr := os.Executable()
			if resolveErr == nil {
				current, resolveErr = filepath.EvalSymlinks(current)
			}
			if resolveErr != nil {
				fmt.Fprintf(os.Stderr, "upgrade: cannot resolve own path: %v\n", resolveErr)
				os.Exit(1)
			}
			fmt.Printf("Installing update — replacing %s …\n", current)
			if handoverErr := upgrade.HandoverTo(pending, current); handoverErr != nil {
				fmt.Fprintf(os.Stderr, "upgrade handover failed: %v\n", handoverErr)
				os.Exit(1)
			}
		}
	}
}
