package tui

import (
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// SwitchToSignupMsg is sent when the user requests to go to the signup screen.
type SwitchToSignupMsg struct{}

// SignupSuccessMsg is sent when an account has been successfully created.
type SignupSuccessMsg struct {
	Email     string
	Password  string
	SessionID string
}

// signupField identifies which input is focused.
type signupField int

const (
	signupFieldName signupField = iota
	signupFieldEmail
	signupFieldPassword
	signupFieldCount
)

// SignupModel is the Bubble Tea model for the sign-up screen.
type SignupModel struct {
	name     textinput.Model
	email    textinput.Model
	password textinput.Model
	focused  signupField
	err      string
	loading  bool
	width    int
	height   int
}

// NewSignupModel creates a fresh signup screen model.
func NewSignupModel() SignupModel {
	name := textinput.New()
	name.Placeholder = "Your Name"
	name.CharLimit = 128
	name.SetWidth(38)
	_ = name.Focus()

	email := textinput.New()
	email.Placeholder = "you@example.com"
	email.CharLimit = 256
	email.SetWidth(38)

	pw := textinput.New()
	pw.Placeholder = "••••••••  (min 8 chars)"
	pw.EchoMode = textinput.EchoPassword
	pw.CharLimit = 128
	pw.SetWidth(38)

	return SignupModel{
		name:     name,
		email:    email,
		password: pw,
		focused:  signupFieldName,
	}
}

func (m SignupModel) Init() tea.Cmd { return textinput.Blink }

func (m SignupModel) Update(msg tea.Msg) (SignupModel, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case signupErrMsg:
		m.err = string(msg)
		m.loading = false

	case tea.KeyPressMsg:
		switch msg.String() {
		case "tab", "down", "enter":
			if msg.String() == "enter" && m.focused == signupFieldPassword {
				return m, m.submitSignup()
			}
			m.focused = (m.focused + 1) % signupFieldCount
			m.syncFocus(&cmds)

		case "shift+tab", "up":
			m.focused = signupField((int(m.focused) - 1 + int(signupFieldCount)) % int(signupFieldCount))
			m.syncFocus(&cmds)

		case "ctrl+l":
			// Switch back to login screen.
			return m, func() tea.Msg { return SwitchToLoginMsg{} }
		}
	}

	// Propagate key events to the focused input.
	switch m.focused {
	case signupFieldName:
		u, cmd := m.name.Update(msg)
		m.name = u
		cmds = append(cmds, cmd)
	case signupFieldEmail:
		u, cmd := m.email.Update(msg)
		m.email = u
		cmds = append(cmds, cmd)
	case signupFieldPassword:
		u, cmd := m.password.Update(msg)
		m.password = u
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m *SignupModel) syncFocus(cmds *[]tea.Cmd) {
	m.name.Blur()
	m.email.Blur()
	m.password.Blur()
	switch m.focused {
	case signupFieldName:
		*cmds = append(*cmds, m.name.Focus())
	case signupFieldEmail:
		*cmds = append(*cmds, m.email.Focus())
	case signupFieldPassword:
		*cmds = append(*cmds, m.password.Focus())
	}
}

func (m SignupModel) submitSignup() tea.Cmd {
	name := strings.TrimSpace(m.name.Value())
	email := strings.TrimSpace(m.email.Value())
	password := m.password.Value()

	if name == "" {
		return func() tea.Msg { return signupErrMsg("Name is required") }
	}
	if email == "" {
		return func() tea.Msg { return signupErrMsg("Email is required") }
	}
	if len(password) < 8 {
		return func() tea.Msg { return signupErrMsg("Password must be at least 8 characters") }
	}
	return func() tea.Msg {
		return doSignupMsg{name: name, email: email, password: password}
	}
}

func (m SignupModel) View() string {
	var b strings.Builder

	title := LoginTitle.Render("  CryptX 2.0 — Create Account  ")
	subtitle := Subtle.Render("Already have an account? ctrl+l to sign in")

	b.WriteString(title + "\n")
	b.WriteString(subtitle + "\n\n")

	b.WriteString(labelFocused("Name", m.focused == signupFieldName) + "\n")
	b.WriteString(renderInput(m.name, m.focused == signupFieldName) + "\n\n")

	b.WriteString(labelFocused("Email", m.focused == signupFieldEmail) + "\n")
	b.WriteString(renderInput(m.email, m.focused == signupFieldEmail) + "\n\n")

	b.WriteString(labelFocused("Password", m.focused == signupFieldPassword) + "\n")
	b.WriteString(renderInput(m.password, m.focused == signupFieldPassword) + "\n\n")

	if m.err != "" {
		b.WriteString(Error.Render("✕ "+m.err) + "\n\n")
	}
	if m.loading {
		b.WriteString(Muted.Render("Creating account...") + "\n\n")
	}

	hints := Muted.Render("tab") + Subtle.Render(" next field  ") +
		Muted.Render("enter") + Subtle.Render(" sign up  ") +
		Muted.Render("ctrl+l") + Subtle.Render(" back to login")
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

// renderInput is a package-level helper (used by both login.go and signup.go).
func renderInput(ti textinput.Model, focused bool) string {
	if focused {
		return LoginInputFocused.Render(ti.View())
	}
	return LoginInput.Render(ti.View())
}

type signupErrMsg string

// doSignupMsg carries name/email/password to app.go for processing.
type doSignupMsg struct {
	name, email, password string
}

// SwitchToLoginMsg is sent when the user wants to return to the login screen.
type SwitchToLoginMsg struct{}
