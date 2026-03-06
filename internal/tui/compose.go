package tui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"charm.land/bubbles/v2/textarea"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/cryptx/cryptx-cli/internal/email"
)

// ── Compose screen state machine ──────────────────────────────────────────────

type composeField int

const (
	cFieldTo composeField = iota
	cFieldSubject
	cFieldBody
	cFieldAttach // virtual: attachment path input
	cFieldMax    // sentinel — keep last
)

// ComposeModel is the custom email compose screen.
type ComposeModel struct {
	width  int
	height int

	toInput      textinput.Model
	subjectInput textinput.Model
	bodyArea     textarea.Model
	attachInput  textinput.Model // path-entry for adding attachments

	focusedField composeField
	showAttach   bool // whether attachment panel is visible

	attachments []email.Attachment

	status    string
	statusErr bool
}

// composeSendViaResendMsg is handled by App.Update so it can access cfg.
type composeSendViaResendMsg struct {
	to          string
	subject     string
	body        string
	attachments []email.Attachment
}

// composeSendViaPopMsg signals that we should hand off to pop.
type composeSendViaPopMsg struct {
	to      string
	subject string
	body    string
	attach  []email.Attachment
}

// composeSendDoneMsg is returned after a Resend attempt.
type composeSendDoneMsg struct {
	err error
}

// NewComposeModel creates a fresh compose model.
func NewComposeModel(width, height int) ComposeModel {
	fieldW := inputWidth(width)

	to := textinput.New()
	to.Placeholder = "recipient@example.com  (comma-separated)"
	to.CharLimit = 512
	to.SetWidth(fieldW)
	to.Focus()

	sub := textinput.New()
	sub.Placeholder = "Email subject"
	sub.CharLimit = 256
	sub.SetWidth(fieldW)

	body := textarea.New()
	body.Placeholder = "Write your message here…"
	body.SetWidth(fieldW)
	body.SetHeight(12)

	att := textinput.New()
	att.Placeholder = "Drag & drop a file here, or paste/type its path, then press Enter"
	att.CharLimit = 1024
	att.SetWidth(fieldW)

	return ComposeModel{
		width:        width,
		height:       height,
		toInput:      to,
		subjectInput: sub,
		bodyArea:     body,
		attachInput:  att,
		focusedField: cFieldTo,
	}
}

func (m ComposeModel) Init() tea.Cmd { return textinput.Blink }

func (m ComposeModel) Update(msg tea.Msg) (ComposeModel, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		w := inputWidth(msg.Width)
		m.toInput.SetWidth(w)
		m.subjectInput.SetWidth(w)
		m.bodyArea.SetWidth(w)
		m.attachInput.SetWidth(w)

	case tea.KeyPressMsg:
		switch msg.String() {

		case "tab", "down":
			// Body is multi-line; 'down' inside body moves the cursor, not focus.
			if m.focusedField == cFieldBody {
				break
			}
			m = m.nextField()
			return m, nil

		case "shift+tab", "up":
			if m.focusedField == cFieldBody {
				break
			}
			m = m.prevField()
			return m, nil

		case "ctrl+a":
			// Toggle attachment input panel.
			m.showAttach = !m.showAttach
			if m.showAttach {
				m.focusedField = cFieldAttach
				m = m.applyFocus()
			} else if m.focusedField == cFieldAttach {
				m.focusedField = cFieldBody
				m = m.applyFocus()
			}
			return m, nil

		case "enter":
			if m.focusedField == cFieldAttach {
				path := strings.TrimSpace(m.attachInput.Value())
				if path != "" {
					a, err := email.LoadAttachment(path)
					if err != nil {
						m.status = "Attachment error: " + err.Error()
						m.statusErr = true
					} else {
						m.attachments = append(m.attachments, a)
						m.status = fmt.Sprintf("✓ Attached: %s (%.1f KB)", a.Filename, float64(len(a.Content))/1024)
						m.statusErr = false
						m.attachInput.SetValue("")
					}
				}
				return m, nil
			}

		case "ctrl+d":
			// Remove last attachment (when in attach panel).
			if m.focusedField == cFieldAttach && len(m.attachments) > 0 {
				removed := m.attachments[len(m.attachments)-1]
				m.attachments = m.attachments[:len(m.attachments)-1]
				m.status = "Removed: " + removed.Filename
				m.statusErr = false
				return m, nil
			}

		case "ctrl+s":
			return m, m.sendViaResendCmd()

		case "ctrl+r":
			return m, m.sendViaPopCmd()

		case "esc":
			return m, func() tea.Msg { return BackMsg{} }

		case "ctrl+c":
			return m, tea.Quit
		}

	case composeSendDoneMsg:
		if msg.err != nil {
			m.status = "Send failed: " + msg.err.Error()
			m.statusErr = true
		} else {
			m.status = "✓ Email sent via Resend!"
			m.statusErr = false
		}
		return m, nil
	}

	// Route events to focused sub-model.
	switch m.focusedField {
	case cFieldTo:
		updated, cmd := m.toInput.Update(msg)
		m.toInput = updated
		cmds = append(cmds, cmd)
	case cFieldSubject:
		updated, cmd := m.subjectInput.Update(msg)
		m.subjectInput = updated
		cmds = append(cmds, cmd)
	case cFieldBody:
		updated, cmd := m.bodyArea.Update(msg)
		m.bodyArea = updated
		cmds = append(cmds, cmd)
	case cFieldAttach:
		updated, cmd := m.attachInput.Update(msg)
		m.attachInput = updated
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m ComposeModel) View() string {
	var b strings.Builder

	b.WriteString(Accent.Bold(true).Render("  ✉  Compose Email") + "\n\n")

	// To
	b.WriteString(composeLabel("To", m.focusedField == cFieldTo))
	b.WriteString(composeInputBox(m.toInput.View(), m.toInput.Width(), m.focusedField == cFieldTo) + "\n\n")

	// Subject
	b.WriteString(composeLabel("Subject", m.focusedField == cFieldSubject))
	b.WriteString(composeInputBox(m.subjectInput.View(), m.subjectInput.Width(), m.focusedField == cFieldSubject) + "\n\n")

	// Body
	b.WriteString(composeLabel("Body (HTML or plain text)", m.focusedField == cFieldBody))
	b.WriteString(composeBoxRaw(m.bodyArea.View(), m.focusedField == cFieldBody))
	b.WriteString(Muted.Render("  Tab to move focus / Ctrl+↑↓ to resize") + "\n\n")

	// Attachments section
	b.WriteString(composeAttachSection(&m))

	// Status
	if m.status != "" {
		st := Success
		if m.statusErr {
			st = Error
		}
		b.WriteString(st.Render("  "+m.status) + "\n\n")
	}

	// Footer keybinds
	b.WriteString(composeFoot())

	return b.String()
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func inputWidth(termWidth int) int {
	w := termWidth - 8
	if w < 40 {
		return 40
	}
	return w
}

func composeLabel(label string, active bool) string {
	style := Muted
	if active {
		style = Accent
	}
	return style.Render("  "+label+":") + "\n"
}

func composeInputBox(content string, width int, focused bool) string {
	bfg := colorBorder
	if focused {
		bfg = colorAccent
	}
	box := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(bfg).
		Padding(0, 1).
		Width(width + 2).
		Render(content)
	return "  " + box
}

func composeBoxRaw(content string, focused bool) string {
	bfg := colorBorder
	if focused {
		bfg = colorAccent
	}
	box := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(bfg).
		Padding(0, 1).
		Render(content)
	return "  " + box + "\n"
}

func composeAttachSection(m *ComposeModel) string {
	var b strings.Builder

	headerStyle := Muted
	if m.focusedField == cFieldAttach {
		headerStyle = Accent
	}

	b.WriteString(
		headerStyle.Render("  Attachments:") + "  " +
			Subtle.Render(fmt.Sprintf("(%d attached)", len(m.attachments))) +
			"  " + Muted.Render("ctrl+a toggle  ctrl+d remove last") + "\n",
	)

	for i, a := range m.attachments {
		sizeKB := float64(len(a.Content)) / 1024
		icon := attachIcon(a.Filename)
		line := fmt.Sprintf("    %s  %s  (%.1f KB)  [%d]", icon, a.Filename, sizeKB, i+1)
		b.WriteString(Subtle.Render(line) + "\n")
	}

	if m.showAttach {
		b.WriteString("\n")
		bfg := colorBorder
		if m.focusedField == cFieldAttach {
			bfg = colorAccent
		}
		box := lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(bfg).
			Padding(0, 1).
			Width(m.attachInput.Width()+2).
			Render(m.attachInput.View())
		b.WriteString("  " + box + "\n")
		b.WriteString(Muted.Render("  Drag & drop a file here, or type a path — press Enter to add\n"))
		b.WriteString(Muted.Render("  ctrl+d to remove the last attachment\n"))
	}

	b.WriteString("\n")
	return b.String()
}

func attachIcon(name string) string {
	ext := strings.ToLower(filepath.Ext(name))
	switch ext {
	case ".pdf":
		return "📄"
	case ".jpg", ".jpeg", ".png", ".gif", ".webp":
		return "🖼"
	case ".zip", ".gz", ".tar", ".tar.gz":
		return "📦"
	case ".csv", ".xlsx", ".xls":
		return "📊"
	default:
		return "📎"
	}
}

func composeFoot() string {
	return "  " +
		Muted.Render("tab") + Subtle.Render(" next  ") +
		Muted.Render("ctrl+a") + Subtle.Render(" attach  ") +
		Muted.Render("ctrl+d") + Subtle.Render(" rm attach  ") +
		Muted.Render("ctrl+s") + Subtle.Render(" send via Resend  ") +
		Muted.Render("ctrl+r") + Subtle.Render(" send via pop  ") +
		Muted.Render("esc") + Subtle.Render(" back")
}

// ── Focus helpers ─────────────────────────────────────────────────────────────

func (m ComposeModel) nextField() ComposeModel {
	next := m.focusedField + 1
	if next == cFieldAttach && !m.showAttach {
		next = cFieldTo // wrap
	}
	if next >= cFieldMax {
		next = cFieldTo
	}
	m.focusedField = next
	return m.applyFocus()
}

func (m ComposeModel) prevField() ComposeModel {
	prev := m.focusedField - 1
	if prev == cFieldAttach && !m.showAttach {
		prev = cFieldBody
	}
	if prev < 0 {
		prev = composeField(cFieldMax - 1)
		if prev == cFieldAttach && !m.showAttach {
			prev = cFieldBody
		}
	}
	m.focusedField = prev
	return m.applyFocus()
}

func (m ComposeModel) applyFocus() ComposeModel {
	m.toInput.Blur()
	m.subjectInput.Blur()
	m.bodyArea.Blur()
	m.attachInput.Blur()

	switch m.focusedField {
	case cFieldTo:
		m.toInput.Focus()
	case cFieldSubject:
		m.subjectInput.Focus()
	case cFieldBody:
		m.bodyArea.Focus()
	case cFieldAttach:
		m.attachInput.Focus()
	}
	return m
}

// ── Send commands ─────────────────────────────────────────────────────────────

func (m ComposeModel) sendViaResendCmd() tea.Cmd {
	to := m.toInput.Value()
	subject := m.subjectInput.Value()
	body := m.bodyArea.Value()
	attachments := make([]email.Attachment, len(m.attachments))
	copy(attachments, m.attachments)

	return func() tea.Msg {
		return composeSendViaResendMsg{
			to:          to,
			subject:     subject,
			body:        body,
			attachments: attachments,
		}
	}
}

func (m ComposeModel) sendViaPopCmd() tea.Cmd {
	to := m.toInput.Value()
	subject := m.subjectInput.Value()
	body := m.bodyArea.Value()
	attachments := make([]email.Attachment, len(m.attachments))
	copy(attachments, m.attachments)

	return func() tea.Msg {
		return composeSendViaPopMsg{
			to:      to,
			subject: subject,
			body:    body,
			attach:  attachments,
		}
	}
}

// buildPopCmd constructs the pop *exec.Cmd and writes attachment bytes to temp
// files. The caller is responsible for removing the returned tmpFiles after the
// process exits. pop reads RESEND_API_KEY from the environment automatically.
func buildPopCmd(to, subject, body string, attachments []email.Attachment) (
	cmd *exec.Cmd, tmpFiles []string, err error,
) {
	popBin := findPop()
	if popBin == "" {
		return nil, nil, fmt.Errorf("`pop` not found — run: go install github.com/charmbracelet/pop@latest")
	}

	// Write each attachment to a temp file so pop can read it by path.
	for _, a := range attachments {
		tmpA, terr := os.CreateTemp("", "cryptx-attach-*-"+a.Filename)
		if terr != nil {
			cleanupTempFiles(tmpFiles)
			return nil, nil, fmt.Errorf("create temp attachment %q: %w", a.Filename, terr)
		}
		tmpFiles = append(tmpFiles, tmpA.Name())
		if _, werr := tmpA.Write(a.Content); werr != nil {
			tmpA.Close()
			cleanupTempFiles(tmpFiles)
			return nil, nil, fmt.Errorf("write attachment %q: %w", a.Filename, werr)
		}
		tmpA.Close()
	}

	// Build args. pop accepts --to as a repeatable single-value flag.
	args := []string{"--subject", subject, "--body", body}
	for _, t := range strings.Split(to, ",") {
		t = strings.TrimSpace(t)
		if t != "" {
			args = append(args, "--to", t)
		}
	}
	for _, p := range tmpFiles {
		args = append(args, "--attach", p)
	}

	cmd = exec.Command(popBin, args...)
	// Stdin/Stdout/Stderr are left nil so bubbletea's ExecProcess sets them
	// to the terminal's I/O when it suspends the TUI.
	return cmd, tmpFiles, nil
}

// findPop returns the absolute path to the pop binary.
// It checks PATH first, then falls back to $(go env GOPATH)/bin/pop,
// which is where `go install` places it.
func findPop() string {
	if p, err := exec.LookPath("pop"); err == nil {
		return p
	}
	// Try GOPATH/bin — common when GOPATH/bin is not on shell PATH.
	if gopath, err := exec.Command("go", "env", "GOPATH").Output(); err == nil {
		candidate := strings.TrimSpace(string(gopath)) + "/bin/pop"
		if _, serr := os.Stat(candidate); serr == nil {
			return candidate
		}
	}
	return ""
}

func cleanupTempFiles(paths []string) {
	for _, p := range paths {
		os.Remove(p)
	}
}
