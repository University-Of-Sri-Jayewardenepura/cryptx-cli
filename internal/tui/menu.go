package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// EventType identifies which registration collection to show.
type EventType int

const (
	EventCTF                EventType = iota
	EventSchoolHackathon              // previously EventHackathon
	EventUniversityHackathon          // university hackathon
	EventDesignathon
	EventCompose // custom email compose
)

// menuItems defines the display info for each menu option.
var menuItems = []struct {
	event       EventType
	key         string
	label       string
	description string
}{
	{EventCTF, "1", "CTF Registrations", "Capture The Flag — Individual & Team"},
	{EventSchoolHackathon, "2", "School Hackathon", "School teams competing in hacking challenges"},
	{EventUniversityHackathon, "3", "University Hackathon", "University teams competing in hacking challenges"},
	{EventDesignathon, "4", "Designathon", "University design competition teams"},
	{EventCompose, "5", "Compose Email", "Send a custom email via Resend or pop"},
}

// MenuModel is the main menu screen.
type MenuModel struct {
	cursor    int
	width     int
	height    int
	userEmail string
}

// MenuSelectMsg is sent when the user picks an event type.
type MenuSelectMsg struct {
	Event EventType
}

// NewMenuModel creates a fresh menu model.
func NewMenuModel(userEmail string) MenuModel {
	return MenuModel{userEmail: userEmail}
}

func (m MenuModel) Init() tea.Cmd { return nil }

func (m MenuModel) Update(msg tea.Msg) (MenuModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyPressMsg:
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(menuItems)-1 {
				m.cursor++
			}
		case "enter", " ":
			return m, func() tea.Msg {
				return MenuSelectMsg{Event: menuItems[m.cursor].event}
			}
		case "1":
			return m, func() tea.Msg { return MenuSelectMsg{Event: EventCTF} }
		case "2":
			return m, func() tea.Msg { return MenuSelectMsg{Event: EventSchoolHackathon} }
		case "3":
			return m, func() tea.Msg { return MenuSelectMsg{Event: EventUniversityHackathon} }
		case "4":
			return m, func() tea.Msg { return MenuSelectMsg{Event: EventDesignathon} }
		case "5":
			return m, func() tea.Msg { return MenuSelectMsg{Event: EventCompose} }
		case "q", "ctrl+c":
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m MenuModel) View() string {
	var b strings.Builder

	// Header strip
	headerText := "  ◈  CryptX 2.0 — Registration Manager  "
	header := lipgloss.NewStyle().
		Bold(true).
		Foreground(colorAccent).
		Background(colorSurface).
		PaddingLeft(1).PaddingRight(1).
		Render(headerText)
	b.WriteString(header)
	b.WriteString("\n\n")

	// Operator info
	if m.userEmail != "" {
		b.WriteString(Muted.Render("  Signed in as: ") + Subtle.Render(m.userEmail) + "\n\n")
	}

	b.WriteString(Accent.Render("  Select an event to manage:") + "\n\n")

	// Menu items
	for i, item := range menuItems {
		var row string
		prefix := fmt.Sprintf("  [%s]  ", item.key)
		if i == m.cursor {
			row = MenuItemSelected.Render(prefix+item.label) +
				"  " + Muted.Render(item.description)
		} else {
			row = MenuItem.Render(prefix+item.label) +
				"  " + Muted.Render(item.description)
		}
		b.WriteString(row + "\n")
	}

	b.WriteString("\n")
	b.WriteString(Muted.Render("  ↑↓/jk") + Subtle.Render(" navigate  ") +
		Muted.Render("enter") + Subtle.Render(" select  ") +
		Muted.Render("1-5") + Subtle.Render(" quick select  ") +
		Muted.Render("q") + Subtle.Render(" quit"))

	return b.String()
}
