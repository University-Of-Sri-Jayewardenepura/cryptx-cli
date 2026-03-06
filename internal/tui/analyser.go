package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
)

// AnalyserModel shows the WhatsApp group membership analysis screen.
type AnalyserModel struct {
	loading  bool
	report   string
	logPath  string
	err      string
	viewport viewport.Model
	tick     int
	width    int
	height   int
}

// analyserDoneMsg is returned when the background analysis goroutine completes.
type analyserDoneMsg struct {
	report string
	err    error
}

// analyserTickMsg drives the loading-spinner animation while analysis runs.
type analyserTickMsg struct{}

// analyserSaveReportDoneMsg is returned after the log file is written.
type analyserSaveReportDoneMsg struct {
	path string
	err  error
}

// NewAnalyserModel creates the analyser screen model.
func NewAnalyserModel(width, height int) AnalyserModel {
	vp := viewport.New(
		viewport.WithWidth(width-4),
		viewport.WithHeight(height-8),
	)
	return AnalyserModel{
		loading:  true,
		viewport: vp,
		width:    width,
		height:   height,
	}
}

func (m AnalyserModel) Init() tea.Cmd {
	return analyserTickCmd()
}

func analyserTickCmd() tea.Cmd {
	return tea.Tick(150*time.Millisecond, func(time.Time) tea.Msg {
		return analyserTickMsg{}
	})
}

func (m AnalyserModel) Update(msg tea.Msg) (AnalyserModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.viewport.SetWidth(msg.Width - 4)
		m.viewport.SetHeight(msg.Height - 8)

	case analyserTickMsg:
		if m.loading {
			m.tick++
			return m, analyserTickCmd()
		}

	case analyserDoneMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err.Error()
		} else {
			m.report = msg.report
			m.viewport.SetContent(msg.report)
		}

	case analyserSaveReportDoneMsg:
		if msg.err != nil {
			m.err = "Save failed: " + msg.err.Error()
		} else {
			m.logPath = msg.path
		}

	case tea.KeyPressMsg:
		switch msg.String() {
		case "s":
			if !m.loading && m.err == "" && m.report != "" && m.logPath == "" {
				report := m.report
				return m, func() tea.Msg {
					return saveAnalyserReport(report)
				}
			}
		case "esc", "q", "backspace":
			return m, func() tea.Msg { return BackMsg{} }
		}
	}

	if !m.loading && m.err == "" {
		updated, cmd := m.viewport.Update(msg)
		m.viewport = updated
		return m, cmd
	}

	return m, nil
}

func (m AnalyserModel) View() string {
	var b strings.Builder

	title := Accent.Bold(true).Render("  ◈  CryptX 2.0 — Group Membership Analyser")
	b.WriteString(title + "\n\n")

	if m.loading {
		spinners := []string{"⣾", "⣽", "⣻", "⢿", "⡿", "⣟", "⣯", "⣷"}
		spin := spinners[m.tick%len(spinners)]
		b.WriteString(Accent.Render("  "+spin+"  Analysing WhatsApp group memberships...") + "\n\n")
		b.WriteString(Muted.Render("  Fetching all registrations and checking each member against") + "\n")
		b.WriteString(Muted.Render("  the group participant list. This may take a few moments."))
		return b.String()
	}

	if m.err != "" {
		b.WriteString(Error.Render("  Error: "+m.err) + "\n\n")
		b.WriteString(Muted.Render("  Press esc to go back."))
		return b.String()
	}

	if m.logPath != "" {
		b.WriteString(Success.Render("  ✓ Report saved → "+m.logPath) + "\n\n")
	}

	b.WriteString(m.viewport.View())
	b.WriteString("\n\n")

	hints := Muted.Render("↑↓/jk") + Subtle.Render(" scroll  ")
	if m.logPath == "" {
		hints += Warning.Render("s") + Subtle.Render(" save report  ")
	}
	hints += Muted.Render("esc") + Subtle.Render(" back")
	b.WriteString(hints)

	return b.String()
}

// saveAnalyserReport writes the report to ~/Downloads and returns a message.
func saveAnalyserReport(report string) tea.Msg {
	home, _ := os.UserHomeDir()
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	filename := "cryptx_group_report_" + timestamp + ".txt"
	savePath := filepath.Join(home, "Downloads", filename)
	if err := os.WriteFile(savePath, []byte(report), 0o644); err != nil {
		return analyserSaveReportDoneMsg{err: err}
	}
	return analyserSaveReportDoneMsg{path: savePath}
}

// ── Report builder ────────────────────────────────────────────────────────────

const reportSep = "══════════════════════════════════════════════════════════════════════"
const reportThin = "──────────────────────────────────────────────────────────────────────"

// BuildGroupReport generates the full membership report string.
// It is called as a goroutine in app.go via doAnalyse().
func BuildGroupReport(entries []EventGroupReport) string {
	var b strings.Builder

	b.WriteString("CryptX 2.0 — WhatsApp Group Membership Report\n")
	b.WriteString(fmt.Sprintf("Generated : %s\n", time.Now().Format("2006-01-02 15:04:05 MST")))
	b.WriteString(reportSep + "\n\n")

	for _, ev := range entries {
		b.WriteString(fmt.Sprintf("%s\n", ev.Label))
		if ev.GroupID != "" {
			b.WriteString(fmt.Sprintf("Group     : %s\n", ev.GroupID))
		}
		b.WriteString(reportThin + "\n")

		if ev.Err != "" {
			b.WriteString(fmt.Sprintf("SKIPPED   : %s\n\n", ev.Err))
			continue
		}

		totalTeams := len(ev.Teams)
		totalPhones := 0
		totalMissing := 0
		for _, t := range ev.Teams {
			for _, m := range t.Members {
				totalPhones++
				if !m.InGroup {
					totalMissing++
				}
			}
		}
		b.WriteString(fmt.Sprintf("Registrations : %d\n", totalTeams))
		b.WriteString(fmt.Sprintf("Phone numbers : %d checked\n", totalPhones))
		if totalMissing == 0 {
			b.WriteString(fmt.Sprintf("Status        : ✓ All %d numbers are in the group\n", totalPhones))
		} else {
			b.WriteString(fmt.Sprintf("Status        : ⚠  %d / %d numbers NOT in group\n", totalMissing, totalPhones))
		}
		b.WriteString("\n")

		// Teams with at least one missing member
		b.WriteString("  TEAMS WITH MISSING MEMBERS\n")
		b.WriteString("  " + reportThin + "\n")
		anyMissing := false
		for _, t := range ev.Teams {
			missing := 0
			for _, m := range t.Members {
				if !m.InGroup {
					missing++
				}
			}
			if missing == 0 {
				continue
			}
			anyMissing = true
			b.WriteString(fmt.Sprintf("  %-42s [%s]\n", t.TeamName, t.DocID))
			for _, m := range t.Members {
				statusIcon := "✓"
				statusTxt := "in group"
				if !m.InGroup {
					statusIcon = "✗"
					statusTxt = "NOT IN GROUP"
				}
				b.WriteString(fmt.Sprintf("    %s  %-28s  %-16s  %s\n",
					statusIcon, truncateStr(m.Name, 28), m.Phone, statusTxt))
			}
			b.WriteString("\n")
		}
		if !anyMissing {
			b.WriteString("  (none — all teams fully in group)\n")
		}
		b.WriteString("\n")

		// Teams fully in group
		b.WriteString("  FULLY IN GROUP\n")
		b.WriteString("  " + reportThin + "\n")
		anyFull := false
		for _, t := range ev.Teams {
			allIn := true
			for _, m := range t.Members {
				if !m.InGroup {
					allIn = false
					break
				}
			}
			if !allIn {
				continue
			}
			anyFull = true
			b.WriteString(fmt.Sprintf("  %-42s [%s]\n", t.TeamName, t.DocID))
		}
		if !anyFull {
			b.WriteString("  (none)\n")
		}
		b.WriteString("\n\n")
	}

	// Summary table
	b.WriteString(reportSep + "\n")
	b.WriteString("SUMMARY\n")
	b.WriteString(reportThin + "\n")
	for _, ev := range entries {
		if ev.Err != "" {
			b.WriteString(fmt.Sprintf("  %-32s  SKIPPED — %s\n", ev.Label+":", ev.Err))
			continue
		}
		totalPhones := 0
		totalMissing := 0
		for _, t := range ev.Teams {
			for _, m := range t.Members {
				totalPhones++
				if !m.InGroup {
					totalMissing++
				}
			}
		}
		pct := 0.0
		if totalPhones > 0 {
			pct = float64(totalMissing) / float64(totalPhones) * 100
		}
		icon := "✓"
		if totalMissing > 0 {
			icon = "⚠"
		}
		b.WriteString(fmt.Sprintf("  %s  %-32s  %d / %d not in group  (%.1f%%)\n",
			icon, ev.Label+":", totalMissing, totalPhones, pct))
	}
	b.WriteString("\n")

	return b.String()
}

// EventGroupReport holds the data for one event section in the report.
type EventGroupReport struct {
	Label   string
	GroupID string
	Teams   []TeamGroupResult
	Err     string // non-empty → event was skipped
}

// TeamGroupResult holds one registration's membership data.
type TeamGroupResult struct {
	DocID    string
	TeamName string
	Members  []MemberGroupResult
}

// MemberGroupResult holds one member's phone and group status.
type MemberGroupResult struct {
	Name    string
	Phone   string
	InGroup bool
}

func truncateStr(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-1] + "…"
}
