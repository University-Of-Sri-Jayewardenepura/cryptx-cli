package tui

import (
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// VerifyModel is the TUI screen shown after signup to confirm the email address.
// After Appwrite sends the verification email, the user clicks the link in their
// inbox. The link URL contains userId and secret as query parameters. The user
// copies those values and pastes them here.
type VerifyModel struct {
	email    string // display only
	userID   textinput.Model
	secret   textinput.Model
	focused  int // 0 = userID, 1 = secret
	err      string
	loading  bool
	width    int
	height   int
}

// VerifyDoneMsg is sent when email verification succeeds.
type VerifyDoneMsg struct{}

// SkipVerifyMsg is sent when the user wants to skip verification for now.
type SkipVerifyMsg struct{}

// doVerifyMsg carries the userID+secret to app.go for Appwrite confirmation.
type doVerifyMsg struct {
	sessionID string
	userID    string
	secret    string
}

// NewVerifyModel creates a fresh email verification screen.
func NewVerifyModel(email string) VerifyModel {
	uid := textinput.New()
	uid.Placeholder = "userId from the link URL"
	uid.CharLimit = 64
	uid.SetWidth(42)
	_ = uid.Focus()

	sec := textinput.New()
	sec.Placeholder = "secret from the link URL"
	sec.CharLimit = 512
	sec.SetWidth(42)

	return VerifyModel{
		email:   email,
		userID:  uid,
		secret:  sec,
		focused: 0,
	}
}

func (m VerifyModel) Init() tea.Cmd { return textinput.Blink }

func (m VerifyModel) Update(msg tea.Msg) (VerifyModel, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case verifyErrMsg:
		m.err = string(msg)
		m.loading = false

	case tea.KeyPressMsg:
		switch msg.String() {
		case "tab", "down":
			m.focused = (m.focused + 1) % 2
			m.syncFocus(&cmds)

		case "shift+tab", "up":
			m.focused = (m.focused + 1) % 2
			m.syncFocus(&cmds)

		case "enter":
			if m.focused == 0 {
				m.focused = 1
				m.syncFocus(&cmds)
			} else {
				return m, m.submitVerify()
			}

		case "ctrl+s":
			return m, func() tea.Msg { return SkipVerifyMsg{} }
		}
	}

	if m.focused == 0 {
		u, cmd := m.userID.Update(msg)
		m.userID = u
		cmds = append(cmds, cmd)
	} else {
		u, cmd := m.secret.Update(msg)
		m.secret = u
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m *VerifyModel) syncFocus(cmds *[]tea.Cmd) {
	if m.focused == 0 {
		*cmds = append(*cmds, m.userID.Focus())
		m.secret.Blur()
	} else {
		m.userID.Blur()
		*cmds = append(*cmds, m.secret.Focus())
	}
}

func (m VerifyModel) submitVerify() tea.Cmd {
	uid := strings.TrimSpace(m.userID.Value())
	sec := strings.TrimSpace(m.secret.Value())
	if uid == "" || sec == "" {
		return func() tea.Msg { return verifyErrMsg("Both userId and secret are required") }
	}
	// sessionID is injected by app.go via SetSessionID before this screen appears.
	return func() tea.Msg {
		return doVerifyMsg{userID: uid, secret: sec}
	}
}

func (m VerifyModel) View() string {
	var b strings.Builder

	title := LoginTitle.Render("  Verify Your Email Address  ")
	b.WriteString(title + "\n\n")

	b.WriteString(Accent.Render("A verification email has been sent to:") + "\n")
	b.WriteString(Value.Render("  "+m.email) + "\n\n")

	b.WriteString(Muted.Render("1. Open the email and click the verification link.") + "\n")
	b.WriteString(Muted.Render("2. Copy the userId and secret from the URL in your browser.") + "\n")
	b.WriteString(Muted.Render("3. Paste them below and press Enter.") + "\n\n")

	b.WriteString(labelFocused("userId", m.focused == 0) + "\n")
	b.WriteString(renderInput(m.userID, m.focused == 0) + "\n\n")

	b.WriteString(labelFocused("secret", m.focused == 1) + "\n")
	b.WriteString(renderInput(m.secret, m.focused == 1) + "\n\n")

	if m.err != "" {
		b.WriteString(Error.Render("✕ "+m.err) + "\n\n")
	}
	if m.loading {
		b.WriteString(Muted.Render("Verifying...") + "\n\n")
	}

	hints := Muted.Render("enter") + Subtle.Render(" confirm  ") +
		Muted.Render("tab") + Subtle.Render(" next field  ") +
		Muted.Render("ctrl+s") + Subtle.Render(" skip for now")
	b.WriteString(hints)

	box := LoginBox.Render(b.String())

	if m.width > 0 && m.height > 0 {
		boxW := lipgloss.Width(box)
		boxH := lipgloss.Height(box)
		leftPad := max(0, (m.width-boxW)/2)
		topPad := max(0, (m.height-boxH)/2)
		box = strings.Repeat("\n", topPad) + strings.Repeat(" ", leftPad) +
			strings.ReplaceAll(box, "\n", "\n"+strings.Repeat(" ", leftPad))
	}
	return box
}

type verifyErrMsg string
