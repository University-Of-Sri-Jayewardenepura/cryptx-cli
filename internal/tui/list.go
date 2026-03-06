package tui

import (
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/cryptx/cryptx-cli/internal/models"
)

// RegistrationRow is a generic row stored alongside the table.
type RegistrationRow struct {
	ID                 string
	DisplayName        string
	RegistrationType   string
	Status             string
	CreatedAt          string
	HasPaymentFile     bool
}

// FilterMode cycles through All → Pending Verification → Verified → Rejected.
type FilterMode int

const (
	FilterAll FilterMode = iota
	FilterPending
	FilterVerified
	FilterRejected
)

func (f FilterMode) String() string {
	switch f {
	case FilterPending:
		return string(models.PaymentPending)
	case FilterVerified:
		return string(models.PaymentVerified)
	case FilterRejected:
		return string(models.PaymentRejected)
	default:
		return ""
	}
}

func (f FilterMode) Label() string {
	switch f {
	case FilterPending:
		return "⏳ Pending"
	case FilterVerified:
		return "✓ Verified"
	case FilterRejected:
		return "✕ Rejected"
	default:
		return "All"
	}
}

// ListModel is the paginated registration list screen.
type ListModel struct {
	event       EventType
	table       table.Model
	rows        []RegistrationRow
	page        int
	totalDocs   int
	filter      FilterMode
	search      string
	searchMode  bool
	debounceSeq int
	loading     bool
	err         string
	width       int
	height      int
}

// ListLoadMsg triggers a data fetch for the current page + filter + search.
type ListLoadMsg struct {
	Event  EventType
	Page   int
	Filter string
	Search string
}

// searchFireMsg fires after the debounce timer to trigger a search.
type searchFireMsg struct {
	seq   int
	query string
}

// ListDataMsg delivers fetched rows back to the model.
type ListDataMsg struct {
	Rows      []RegistrationRow
	TotalDocs int
	Err       error
}

// ListSelectMsg is sent when Enter is pressed on a row.
type ListSelectMsg struct {
	Event EventType
	DocID string
}

// NewListModel creates a list model for the given event type.
func NewListModel(event EventType, width, height int) ListModel {
	t := buildTable(nil, tableWidth(width))
	t.Focus()

	return ListModel{
		event:   event,
		table:   t,
		loading: true,
		width:   width,
		height:  height,
	}
}

func (m ListModel) Init() tea.Cmd {
	return func() tea.Msg {
		return ListLoadMsg{Event: m.event, Page: 0, Filter: m.filter.String(), Search: m.search}
	}
}

func (m ListModel) Update(msg tea.Msg) (ListModel, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.table = buildTable(m.rows, tableWidth(m.width))
		m.table.Focus()

	case ListDataMsg:
		m.loading = false
		if msg.Err != nil {
			m.err = msg.Err.Error()
			return m, nil
		}
		m.err = ""
		m.rows = msg.Rows
		m.totalDocs = msg.TotalDocs
		m.table = buildTable(m.rows, tableWidth(m.width))
		m.table.Focus()

	case searchFireMsg:
		if msg.seq == m.debounceSeq {
			m.page = 0
			m.loading = true
			cmds = append(cmds, m.loadCmd())
		}

	case tea.KeyPressMsg:
		if m.searchMode {
			switch msg.String() {
			case "esc":
				m.searchMode = false
				if m.search != "" {
					m.search = ""
					m.debounceSeq++
					m.page = 0
					m.loading = true
					cmds = append(cmds, m.loadCmd())
				}
			case "enter":
				m.searchMode = false
			case "backspace", "ctrl+h":
				if len(m.search) > 0 {
					m.search = m.search[:len(m.search)-1]
					m.debounceSeq++
					seq := m.debounceSeq
					q := m.search
					cmds = append(cmds, tea.Tick(350*time.Millisecond, func(_ time.Time) tea.Msg {
						return searchFireMsg{seq: seq, query: q}
					}))
				}
			default:
				if r := msg.String(); len(r) == 1 {
					m.search += r
					m.debounceSeq++
					seq := m.debounceSeq
					q := m.search
					cmds = append(cmds, tea.Tick(350*time.Millisecond, func(_ time.Time) tea.Msg {
						return searchFireMsg{seq: seq, query: q}
					}))
				}
			}
			return m, tea.Batch(cmds...)
		}

		switch msg.String() {
		case "enter":
			row := m.table.SelectedRow()
			if len(row) > 0 {
				docID := m.rows[m.table.Cursor()].ID
				return m, func() tea.Msg {
					return ListSelectMsg{Event: m.event, DocID: docID}
				}
			}

		case "/":
			m.searchMode = true
			return m, nil

		case "n":
			maxPage := (m.totalDocs - 1) / 25
			if m.page < maxPage {
				m.page++
				m.loading = true
				cmds = append(cmds, m.loadCmd())
			}

		case "p":
			if m.page > 0 {
				m.page--
				m.loading = true
				cmds = append(cmds, m.loadCmd())
			}

		case "f":
			// Payment status filter only applies to CTF.
			if m.event == EventCTF {
				m.filter = (m.filter + 1) % 4
				m.page = 0
				m.loading = true
				cmds = append(cmds, m.loadCmd())
			}

		case "r":
			m.loading = true
			cmds = append(cmds, m.loadCmd())

		case "esc", "backspace":
			return m, func() tea.Msg { return BackMsg{} }

		case "q", "ctrl+c":
			return m, tea.Quit
		}
	}

	// Forward navigation keys to the table when not in search mode.
	if !m.searchMode {
		updated, cmd := m.table.Update(msg)
		m.table = updated
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m ListModel) View() string {
	var b strings.Builder

	title := eventTitle(m.event)
	pageInfo := Muted.Render(fmt.Sprintf("page %d / %d  (%d total)", m.page+1, maxPage(m.totalDocs), m.totalDocs))

	var headerLine string
	if m.event == EventCTF {
		filterBadge := filterStyle(m.filter)
		headerLine = lipgloss.JoinHorizontal(lipgloss.Left,
			Accent.Bold(true).Render(title),
			"  ", filterBadge, "  ", pageInfo,
		)
	} else {
		headerLine = lipgloss.JoinHorizontal(lipgloss.Left,
			Accent.Bold(true).Render(title),
			"  ", pageInfo,
		)
	}
	b.WriteString(headerLine + "\n")

	// Search bar
	if m.searchMode {
		cursor := Accent.Render("█")
		b.WriteString(Muted.Render("  Search: ") + Value.Render(m.search) + cursor + "\n\n")
	} else if m.search != "" {
		b.WriteString(Muted.Render("  Search: ") + Accent.Render(m.search) +
			Muted.Render("  (esc to clear)") + "\n\n")
	} else {
		b.WriteString("\n")
	}

	if m.loading {
		b.WriteString(Muted.Render("  Loading registrations..."))
		return b.String()
	}
	if m.err != "" {
		b.WriteString(Error.Render("  Error: "+m.err) + "\n")
		if strings.Contains(m.err, "not authorized") || strings.Contains(m.err, "unauthorized") || strings.Contains(m.err, "Unauthorized") {
			b.WriteString(Muted.Render("  Possible causes:") + "\n")
			b.WriteString(Muted.Render("  1. Collection → Settings → Permissions: add Role 'users' Read+Write") + "\n")
			b.WriteString(Muted.Render("  2. Collection → Settings → 'Document Security' must be OFF") + "\n")
			b.WriteString(Muted.Render("     (When ON, each document needs its own permissions — collection rules are ignored)") + "\n")
		}
		b.WriteString(Muted.Render("  Press r to retry, Esc to go back."))
		return b.String()
	}
	if len(m.rows) == 0 {
		b.WriteString(Muted.Render("  No registrations found."))
	} else {
		b.WriteString(m.table.View())
	}

	b.WriteString("\n\n")
	hints := Muted.Render("↑↓") + Subtle.Render(" navigate  ") +
		Muted.Render("enter") + Subtle.Render(" view  ") +
		Muted.Render("/") + Subtle.Render(" search  ") +
		Muted.Render("n/p") + Subtle.Render(" page  ") +
		Muted.Render("r") + Subtle.Render(" refresh  ")
	if m.event == EventCTF {
		hints += Muted.Render("f") + Subtle.Render(" filter  ")
	}
	hints += Muted.Render("esc") + Subtle.Render(" back")
	b.WriteString(hints)

	return b.String()
}

func (m ListModel) loadCmd() tea.Cmd {
	event := m.event
	page := m.page
	filter := m.filter.String()
	search := m.search
	return func() tea.Msg {
		return ListLoadMsg{Event: event, Page: page, Filter: filter, Search: search}
	}
}

// ── helpers ───────────────────────────────────────────────────────────────────

func buildTable(rows []RegistrationRow, width int) table.Model {
	cols := []table.Column{
		{Title: "#", Width: 4},
		{Title: "Name / Team", Width: clampWidth(width-80, 24, 44)},
		{Title: "Type", Width: 14},
		{Title: "Status", Width: 13},
		{Title: "Registered", Width: 20},
	}

	var tRows []table.Row
	for i, r := range rows {
		tRows = append(tRows, table.Row{
			fmt.Sprintf("%d", i+1),
			truncate(r.DisplayName, cols[1].Width),
			r.RegistrationType,
			r.Status,
			r.CreatedAt,
		})
	}

	s := table.Styles{
		Header:   TableHeader,
		Cell:     TableCell,
		Selected: TableSelected,
	}

	t := table.New(
		table.WithColumns(cols),
		table.WithRows(tRows),
		table.WithHeight(20),
		table.WithWidth(width),
		table.WithStyles(s),
	)
	return t
}

func tableWidth(termWidth int) int {
	if termWidth < 80 {
		return 80
	}
	return termWidth - 4
}

func clampWidth(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

func maxPage(total int) int {
	if total == 0 {
		return 1
	}
	p := (total + 24) / 25
	return p
}

func eventTitle(e EventType) string {
	switch e {
	case EventSchoolHackathon:
		return "School Hackathon Registrations"
	case EventUniversityHackathon:
		return "University Hackathon Registrations"
	case EventDesignathon:
		return "Designathon Registrations"
	default:
		return "CTF Registrations"
	}
}

func filterStyle(f FilterMode) string {
	switch f {
	case FilterPending:
		return BadgePending.Render(" " + f.Label() + " ")
	case FilterVerified:
		return BadgeConfirmed.Render(" " + f.Label() + " ")
	case FilterRejected:
		return Error.Render(" " + f.Label() + " ")
	default:
		return Muted.Render("[" + f.Label() + "]")
	}
}

// RegistrationRowFromCTF converts a CTF model to a generic list row.
func RegistrationRowFromCTF(r *models.CTFRegistration) RegistrationRow {
	return RegistrationRow{
		ID:               r.ID,
		DisplayName:      r.DisplayName(),
		RegistrationType: r.RegistrationType,
		Status:           string(r.PaymentStatus),
		CreatedAt:        r.CreatedAtTime().Format("02 Jan 2006 15:04"),
		HasPaymentFile:   r.PaymentSlipFileId != "",
	}
}

// RegistrationRowFromSchoolHackathon converts a School Hackathon model to a list row.
func RegistrationRowFromSchoolHackathon(r *models.SchoolHackathonRegistration) RegistrationRow {
	return RegistrationRow{
		ID:               r.ID,
		DisplayName:      r.DisplayName(),
		RegistrationType: "Team (School)",
		Status:           "—",
		CreatedAt:        r.CreatedAtTime().Format("02 Jan 2006 15:04"),
	}
}

// RegistrationRowFromUniversityHackathon converts a University Hackathon model to a list row.
func RegistrationRowFromUniversityHackathon(r *models.UniversityHackathonRegistration) RegistrationRow {
	return RegistrationRow{
		ID:               r.ID,
		DisplayName:      r.DisplayName(),
		RegistrationType: "Team (Uni)",
		Status:           "—",
		CreatedAt:        r.CreatedAtTime().Format("02 Jan 2006 15:04"),
	}
}

// RegistrationRowFromDesignathon converts a Designathon model to a list row.
func RegistrationRowFromDesignathon(r *models.DesignathonRegistration) RegistrationRow {
	return RegistrationRow{
		ID:               r.ID,
		DisplayName:      r.DisplayName(),
		RegistrationType: "Team (Design)",
		Status:           "—",
		CreatedAt:        r.CreatedAtTime().Format("02 Jan 2006 15:04"),
		HasPaymentFile:   r.TeamLogoFileId != "",
	}
}

// BackMsg signals navigation back to the previous screen.
type BackMsg struct{}
