package tui

import (
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// LoginField identifies which text input is currently focused.
type LoginField int

const (
	fieldEmail LoginField = iota
	fieldPassword
)

// LoginModel is the Bubble Tea model for the login screen.
type LoginModel struct {
	email    textinput.Model
	password textinput.Model
	focused  LoginField
	err      string
	loading  bool
	width    int
	height   int
}

// LoginSuccessMsg is sent when authentication succeeds.
type LoginSuccessMsg struct {
	Email    string
	Password string
}

// LoginOAuthMsg is sent when the user requests OAuth login.
type LoginOAuthMsg struct{}

// NewLoginModel creates a fresh login screen model.
func NewLoginModel() LoginModel {
	email := textinput.New()
	email.Placeholder = "admin@cryptx.lk"
	email.CharLimit = 256
	email.SetWidth(38)
	_ = email.Focus()

	pw := textinput.New()
	pw.Placeholder = "••••••••"
	pw.EchoMode = textinput.EchoPassword
	pw.CharLimit = 128
	pw.SetWidth(38)

	return LoginModel{
		email:    email,
		password: pw,
		focused:  fieldEmail,
	}
}

func (m LoginModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m LoginModel) Update(msg tea.Msg) (LoginModel, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyPressMsg:
		switch msg.String() {
		case "tab", "down":
			m.focused = (m.focused + 1) % 2
			if m.focused == fieldEmail {
				cmds = append(cmds, m.email.Focus())
				m.password.Blur()
			} else {
				cmds = append(cmds, m.password.Focus())
				m.email.Blur()
			}

		case "shift+tab", "up":
			if m.focused == fieldEmail {
				m.focused = fieldPassword
				cmds = append(cmds, m.password.Focus())
				m.email.Blur()
			} else {
				m.focused = fieldEmail
				cmds = append(cmds, m.email.Focus())
				m.password.Blur()
			}

		case "enter":
			if m.focused == fieldEmail {
				// Move focus to password on enter in email field.
				m.focused = fieldPassword
				cmds = append(cmds, m.password.Focus())
				m.email.Blur()
			} else {
				return m, m.submitLogin()
			}

		case "ctrl+o":
			return m, func() tea.Msg { return LoginOAuthMsg{} }

		case "ctrl+n":
			return m, func() tea.Msg { return SwitchToSignupMsg{} }
		}

	case loginErrMsg:
		m.err = string(msg)
		m.loading = false
	}

	// Propagate key events to the active input.
	if m.focused == fieldEmail {
		updated, cmd := m.email.Update(msg)
		m.email = updated
		cmds = append(cmds, cmd)
	} else {
		updated, cmd := m.password.Update(msg)
		m.password = updated
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m LoginModel) submitLogin() tea.Cmd {
	email := strings.TrimSpace(m.email.Value())
	password := m.password.Value()
	if email == "" || password == "" {
		return func() tea.Msg { return loginErrMsg("Email and password are required") }
	}
	return func() tea.Msg {
		return LoginSuccessMsg{Email: email, Password: password}
	}
}

func (m LoginModel) View() string {
	var b strings.Builder

	title := LoginTitle.Render("  CryptX 2.0 — Admin CLI  ")
	subtitle := Subtle.Render("Sign in to manage registrations")

	emailLabel := labelFocused("Email", m.focused == fieldEmail)
	pwLabel := labelFocused("Password", m.focused == fieldPassword)

	emailInput := m.renderInput(m.email, m.focused == fieldEmail)
	pwInput := m.renderInput(m.password, m.focused == fieldPassword)

	b.WriteString(title)
	b.WriteString("\n")
	b.WriteString(subtitle)
	b.WriteString("\n\n")

	b.WriteString(emailLabel + "\n")
	b.WriteString(emailInput + "\n\n")

	b.WriteString(pwLabel + "\n")
	b.WriteString(pwInput + "\n\n")

	if m.err != "" {
		b.WriteString(Error.Render("✕ "+m.err) + "\n\n")
	}

	if m.loading {
		b.WriteString(Muted.Render("Signing in...") + "\n\n")
	}

	hints := Muted.Render("enter") + Subtle.Render(" sign in  ") +
		Muted.Render("ctrl+o") + Subtle.Render(" oauth  ") +
		Muted.Render("ctrl+n") + Subtle.Render(" new account  ") +
		Muted.Render("tab") + Subtle.Render(" next field")
	b.WriteString(hints)

	box := LoginBox.Render(b.String())

	// Center in terminal.
	if m.width > 0 && m.height > 0 {
		boxW := lipgloss.Width(box)
		boxH := lipgloss.Height(box)
		leftPad := max(0, (m.width-boxW)/2)
		topPad := max(0, (m.height-boxH)/2)
		box = strings.Repeat("\n", topPad) + strings.Repeat(" ", leftPad) + strings.ReplaceAll(box, "\n", "\n"+strings.Repeat(" ", leftPad))
	}
	return box
}

func (m LoginModel) renderInput(ti textinput.Model, focused bool) string {
	if focused {
		return LoginInputFocused.Render(ti.View())
	}
	return LoginInput.Render(ti.View())
}

func labelFocused(text string, active bool) string {
	if active {
		return Accent.Render(text)
	}
	return Muted.Render(text)
}

type loginErrMsg string

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
