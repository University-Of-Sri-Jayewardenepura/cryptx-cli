package tui

import "charm.land/lipgloss/v2"

// Colour palette — cyan-on-dark consistent with CryptX brand.
var (
	colorBase      = lipgloss.Color("#0a0a0f")
	colorSurface   = lipgloss.Color("#12121a")
	colorBorder    = lipgloss.Color("#1e2033")
	colorAccent    = lipgloss.Color("#22d3ee")
	colorAccentDim = lipgloss.Color("#0ea5e9")
	colorMuted     = lipgloss.Color("#64748b")
	colorSubtle    = lipgloss.Color("#94a3b8")
	colorText      = lipgloss.Color("#e2e8f0")
	colorSuccess   = lipgloss.Color("#4ade80")
	colorWarning   = lipgloss.Color("#fbbf24")
	colorError     = lipgloss.Color("#f87171")
)

// ── Layout ────────────────────────────────────────────────────────────────────

var (
	// AppBorder wraps the outermost TUI container.
	AppBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorBorder).
			Padding(0, 1)

	// Header is the top title bar style.
	Header = lipgloss.NewStyle().
		Bold(true).
		Foreground(colorAccent).
		Background(colorSurface).
		Padding(0, 2).
		Width(0) // caller sets width

	// Footer is the help bar at the bottom.
	Footer = lipgloss.NewStyle().
		Foreground(colorMuted).
		Background(colorSurface).
		Padding(0, 2)
)

// ── Text ──────────────────────────────────────────────────────────────────────

var (
	Bold    = lipgloss.NewStyle().Bold(true)
	Muted   = lipgloss.NewStyle().Foreground(colorMuted)
	Subtle  = lipgloss.NewStyle().Foreground(colorSubtle)
	Accent  = lipgloss.NewStyle().Foreground(colorAccent)
	Error   = lipgloss.NewStyle().Foreground(colorError)
	Success = lipgloss.NewStyle().Foreground(colorSuccess)
	Warning = lipgloss.NewStyle().Foreground(colorWarning)

	Label = lipgloss.NewStyle().
		Foreground(colorMuted).
		Width(22)

	Value = lipgloss.NewStyle().
		Foreground(colorText)
)

// ── Badges ────────────────────────────────────────────────────────────────────

var (
	BadgePending = lipgloss.NewStyle().
			Foreground(colorWarning).
			Background(lipgloss.Color("#3b2a10")).
			Padding(0, 1).
			Bold(true)

	BadgeConfirmed = lipgloss.NewStyle().
			Foreground(colorSuccess).
			Background(lipgloss.Color("#0f2a1a")).
			Padding(0, 1).
			Bold(true)
)

// StatusBadge returns the appropriate styled badge for a payment status string.
func StatusBadge(status string) string {
	switch status {
	case "verified":
		return BadgeConfirmed.Render("✓ verified")
	case "rejected":
		return Error.Render("✕ rejected")
	default:
		return BadgePending.Render("⏳ pending")
	}
}

// ── Login ─────────────────────────────────────────────────────────────────────

var (
	LoginBox = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorAccent).
			Padding(2, 4).
			Width(52)

	LoginTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorAccent).
			MarginBottom(1)

	LoginInput = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(colorBorder).
			Padding(0, 1).
			Width(38)

	LoginInputFocused = lipgloss.NewStyle().
				Border(lipgloss.NormalBorder()).
				BorderForeground(colorAccent).
				Padding(0, 1).
				Width(38)
)

// ── Table ─────────────────────────────────────────────────────────────────────

var (
	TableHeader = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorAccent).
			Padding(0, 1)

	TableCell = lipgloss.NewStyle().
			Foreground(colorText).
			Padding(0, 1)

	TableSelected = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorBase).
			Background(colorAccent).
			Padding(0, 1)
)

// ── Detail ────────────────────────────────────────────────────────────────────

var (
	DetailSection = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorAccent).
			MarginTop(1).
			BorderBottom(true).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(colorBorder)

	DetailRow = lipgloss.NewStyle().
			PaddingLeft(2)
)

// ── Modal ─────────────────────────────────────────────────────────────────────

var (
	ModalBox = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorAccentDim).
			Padding(1, 3).
			Background(colorSurface)

	ModalTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorText).
			MarginBottom(1)

	ModalWarning = lipgloss.NewStyle().
			Foreground(colorWarning).
			MarginBottom(1)

	ModalDanger = lipgloss.NewStyle().
			Foreground(colorError).
			MarginBottom(1)

	ButtonYes = lipgloss.NewStyle().
			Foreground(colorBase).
			Background(colorSuccess).
			Padding(0, 2).
			Bold(true)

	ButtonNo = lipgloss.NewStyle().
			Foreground(colorBase).
			Background(colorError).
			Padding(0, 2).
			Bold(true)
)

// ── Menu ──────────────────────────────────────────────────────────────────────

var (
	MenuTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorAccent).
			MarginBottom(1)

	MenuItem = lipgloss.NewStyle().
			Foreground(colorSubtle).
			PaddingLeft(2)

	MenuItemSelected = lipgloss.NewStyle().
				Bold(true).
				Foreground(colorAccent).
				PaddingLeft(2)

	MenuItemPrefix = lipgloss.NewStyle().
			Foreground(colorAccent)
)
