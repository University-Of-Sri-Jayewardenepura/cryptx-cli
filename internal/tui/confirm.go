package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// ModalKind distinguishes between confirm-payment and delete modals.
type ModalKind int

const (
	ModalConfirmPayment ModalKind = iota
	ModalDelete
)

// ConfirmModel is a centred modal dialog for destructive / important actions.
type ConfirmModel struct {
	kind    ModalKind
	event   EventType
	docID   string
	name    string
	email   string
	step    int // 0 = first prompt, 1 = second prompt (payment only)
	width   int
	height  int
}

// ConfirmedMsg is sent when the user confirms the action.
type ConfirmedMsg struct {
	Kind  ModalKind
	Event EventType
	DocID string
	Email string
}

// CancelledMsg is sent when the user presses n / Esc.
type CancelledMsg struct{}

// NewConfirmPaymentModal creates a modal for confirming a payment.
func NewConfirmPaymentModal(event EventType, docID, name, email string, width, height int) ConfirmModel {
	return ConfirmModel{
		kind:   ModalConfirmPayment,
		event:  event,
		docID:  docID,
		name:   name,
		email:  email,
		width:  width,
		height: height,
	}
}

// NewDeleteModal creates a modal for confirming deletion.
func NewDeleteModal(event EventType, docID, name string, width, height int) ConfirmModel {
	return ConfirmModel{
		kind:   ModalDelete,
		event:  event,
		docID:  docID,
		name:   name,
		width:  width,
		height: height,
	}
}

func (m ConfirmModel) Init() tea.Cmd { return nil }

func (m ConfirmModel) Update(msg tea.Msg) (ConfirmModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyPressMsg:
		switch msg.String() {
		case "y", "enter":
			// Payment confirmation requires two steps.
			if m.kind == ModalConfirmPayment && m.step == 0 {
				m.step = 1
				return m, nil
			}
			event := m.event
			docID := m.docID
			email := m.email
			kind := m.kind
			return m, func() tea.Msg {
				return ConfirmedMsg{Kind: kind, Event: event, DocID: docID, Email: email}
			}
		case "n", "esc", "q":
			return m, func() tea.Msg { return CancelledMsg{} }
		}
	}
	return m, nil
}

func (m ConfirmModel) View() string {
	box := m.renderBox()

	if m.width == 0 || m.height == 0 {
		return box
	}

	// Centre the modal overlay.
	boxW := lipgloss.Width(box)
	boxH := lipgloss.Height(box)
	leftPad := max(0, (m.width-boxW)/2)
	topPad := max(0, (m.height-boxH)/2)

	lines := strings.Split(box, "\n")
	padded := make([]string, 0, topPad+len(lines))
	for i := 0; i < topPad; i++ {
		padded = append(padded, "")
	}
	prefix := strings.Repeat(" ", leftPad)
	for _, l := range lines {
		padded = append(padded, prefix+l)
	}
	return strings.Join(padded, "\n")
}

func (m ConfirmModel) renderBox() string {
	var b strings.Builder

	switch m.kind {
	case ModalConfirmPayment:
		if m.step == 0 {
			b.WriteString(ModalTitle.Render("Confirm Payment") + "\n\n")
			b.WriteString(ModalWarning.Render(fmt.Sprintf(
				"Confirm payment for:  %s\nEmail:                %s",
				m.name, m.email,
			)) + "\n\n")
			b.WriteString(Subtle.Render("A confirmation email will be sent to the registrant.") + "\n\n")
		} else {
			b.WriteString(ModalTitle.Render("Are you sure?") + "\n\n")
			b.WriteString(ModalDanger.Render(fmt.Sprintf(
				"This will mark the payment as verified\nand send a confirmation email to:\n  %s",
				m.email,
			)) + "\n\n")
			b.WriteString(Error.Render("This action cannot be undone.") + "\n\n")
		}

	case ModalDelete:
		b.WriteString(ModalTitle.Render("Delete Registration") + "\n\n")
		b.WriteString(ModalDanger.Render(fmt.Sprintf(
			"This will permanently delete:\n  %s",
			m.name,
		)) + "\n\n")
		b.WriteString(Error.Render("This action cannot be undone.") + "\n\n")
	}

	yes := ButtonYes.Render("  Y  yes  ")
	no := ButtonNo.Render("  N  cancel  ")
	buttons := lipgloss.JoinHorizontal(lipgloss.Left, yes, "  ", no)
	b.WriteString(buttons)

	return ModalBox.Render(b.String())
}
