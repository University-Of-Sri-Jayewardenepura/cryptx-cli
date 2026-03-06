package tui

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"github.com/cryptx/cryptx-cli/internal/models"
)

// DetailModel shows the full details of a single registration.
type DetailModel struct {
	event       EventType
	docID       string
	name        string
	email       string
	fileID      string
	teamName    string
	content     string
	phones      []string        // all phone numbers for this registration
	groupStatus map[string]bool // phone → in-group; nil when WAHA disabled
	viewport    viewport.Model
	loading     bool
	err         string
	width       int
	height      int
}

// DetailLoadMsg triggers fetching a registration document.
type DetailLoadMsg struct {
	Event EventType
	DocID string
}

// DetailDataMsg delivers the rendered detail content to the model.
type DetailDataMsg struct {
	Event       EventType
	DocID       string
	Content     string
	Name        string
	Email       string
	FileID      string
	TeamName    string
	Phones      []string        // all phones for this registration (original format)
	GroupStatus map[string]bool // phone → in-group; nil when WAHA disabled
	Err         error
}

// AddToGroupMsg requests adding phones-not-in-group to the relevant WhatsApp group.
type AddToGroupMsg struct {
	Event  EventType
	Phones []string // original phone strings that are NOT in the group
}

// ConfirmActionMsg triggers the confirm-payment modal for the current doc.
type ConfirmActionMsg struct {
	Event EventType
	DocID string
	Name  string
	Email string
}

// DeleteActionMsg triggers the delete-confirmation modal.
type DeleteActionMsg struct {
	Event EventType
	DocID string
	Name  string
}

// DownloadFileMsg triggers downloading a file from storage.
type DownloadFileMsg struct {
	Event    EventType
	FileID   string
	TeamName string
}

// NewDetailModel creates a detail model for a given registration.
func NewDetailModel(event EventType, docID string, width, height int) DetailModel {
	vp := viewport.New(viewport.WithWidth(width-4), viewport.WithHeight(height-6))
	return DetailModel{
		event:    event,
		docID:    docID,
		viewport: vp,
		loading:  true,
		width:    width,
		height:   height,
	}
}

func (m DetailModel) Init() tea.Cmd {
	return func() tea.Msg {
		return DetailLoadMsg{Event: m.event, DocID: m.docID}
	}
}

func (m DetailModel) Update(msg tea.Msg) (DetailModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.viewport.SetWidth(msg.Width - 4)
		m.viewport.SetHeight(msg.Height - 6)

	case DetailDataMsg:
		m.loading = false
		if msg.Err != nil {
			m.err = msg.Err.Error()
			return m, nil
		}
		m.err = ""
		m.content = msg.Content
		m.name = msg.Name
		m.email = msg.Email
		m.fileID = msg.FileID
		m.teamName = msg.TeamName
		m.phones = msg.Phones
		m.groupStatus = msg.GroupStatus
		m.viewport.SetContent(msg.Content)

	case tea.KeyPressMsg:
		switch msg.String() {
		case "esc", "backspace":
			return m, func() tea.Msg { return BackMsg{} }

		case "c":
			if m.event == EventCTF {
				name, email := m.nameAndEmail()
				return m, func() tea.Msg {
					return ConfirmActionMsg{
						Event: m.event,
						DocID: m.docID,
						Name:  name,
						Email: email,
					}
				}
			}

		case "s":
			if m.fileID != "" {
				event := m.event
				fileID := m.fileID
				teamName := m.teamName
				return m, func() tea.Msg {
					return DownloadFileMsg{
						Event:    event,
						FileID:   fileID,
						TeamName: teamName,
					}
				}
			}

		case "a":
			missing := m.phonesNotInGroup()
			if len(missing) > 0 {
				event := m.event
				return m, func() tea.Msg {
					return AddToGroupMsg{Event: event, Phones: missing}
				}
			}

		case "d":
			name, _ := m.nameAndEmail()
			return m, func() tea.Msg {
				return DeleteActionMsg{
					Event: m.event,
					DocID: m.docID,
					Name:  name,
				}
			}

		case "q", "ctrl+c":
			return m, tea.Quit
		}
	}

	updated, cmd := m.viewport.Update(msg)
	m.viewport = updated
	return m, cmd
}

func (m DetailModel) View() string {
	var b strings.Builder

	title := Accent.Bold(true).Render(fmt.Sprintf("  %s  ›  Detail", eventTitle(m.event)))
	b.WriteString(title + "\n\n")

	if m.loading {
		b.WriteString(Muted.Render("  Loading..."))
		return b.String()
	}
	if m.err != "" {
		b.WriteString(Error.Render("  Error: "+m.err) + "\n")
		b.WriteString(Muted.Render("  Press Esc to go back."))
		return b.String()
	}

	b.WriteString(m.viewport.View())
	b.WriteString("\n\n")
	hints := Muted.Render("↑↓/jk") + Subtle.Render(" scroll  ")
	if m.event == EventCTF {
		hints += Muted.Render("c") + Subtle.Render(" confirm payment  ")
	}
	hints += Muted.Render("d") + Subtle.Render(" delete  ") +
		Muted.Render("esc") + Subtle.Render(" back")
	if m.fileID != "" {
		hints += "  " + Muted.Render("s") + Subtle.Render(" save file")
	}
	if len(m.phonesNotInGroup()) > 0 {
		hints += "  " + Warning.Render("a") + Subtle.Render(" add missing to WA group")
	}
	b.WriteString(hints)
	return b.String()
}

func (m DetailModel) nameAndEmail() (string, string) {
	return m.name, m.email
}

// phonesNotInGroup returns the phones from this registration that are not yet
// in the relevant WhatsApp group.  Returns nil when WAHA is not enabled or
// when all members are already in the group.
func (m DetailModel) phonesNotInGroup() []string {
	if m.groupStatus == nil || len(m.phones) == 0 {
		return nil
	}
	var out []string
	for _, p := range m.phones {
		if p != "" && !m.groupStatus[p] {
			out = append(out, p)
		}
	}
	return out
}

// ── Rendering helpers ─────────────────────────────────────────────────────────

// groupTag returns a compact badge showing whether a phone is in the group.
// If groupStatus is nil (WAHA disabled) or the phone is empty it returns the
// phone string unmodified.
func groupTag(phone string, groupStatus map[string]bool) string {
	if phone == "" {
		return phone
	}
	in, ok := groupStatus[phone]
	if !ok {
		return phone
	}
	if in {
		return phone + " " + Success.Render("✓ in group")
	}
	return phone + " " + Error.Render("✗ not in group")
}

// RenderCTFDetail builds a rich text view for a CTF registration.
func RenderCTFDetail(r *models.CTFRegistration, groupStatus map[string]bool) string {
	var b strings.Builder

	b.WriteString(DetailSection.Render("Payment Status") + "\n")
	b.WriteString(DetailRow.Render(row("Status", StatusBadge(string(r.PaymentStatus)))) + "\n")
	b.WriteString(DetailRow.Render(row("Submitted", r.SubmittedAt)) + "\n")
	b.WriteString(DetailRow.Render(row("Document ID", Muted.Render(r.ID))) + "\n\n")

	b.WriteString(DetailSection.Render("Registration Type") + "\n")
	b.WriteString(DetailRow.Render(row("Type", r.RegistrationType)) + "\n")
	if r.TeamName != "" {
		b.WriteString(DetailRow.Render(row("Team Name", r.TeamName)) + "\n")
	}
	b.WriteString("\n")

	b.WriteString(DetailSection.Render("Leader / Registrant") + "\n")
	b.WriteString(DetailRow.Render(row("Name", r.LeaderName)) + "\n")
	b.WriteString(DetailRow.Render(row("University", r.LeaderUniversity)) + "\n")
	b.WriteString(DetailRow.Render(row("Email", r.LeaderEmail)) + "\n")
	b.WriteString(DetailRow.Render(row("Contact", r.LeaderContact)) + "\n")
	b.WriteString(DetailRow.Render(row("WhatsApp", r.LeaderWhatsapp)) + "\n")
	b.WriteString(DetailRow.Render(row("NIC", r.LeaderNIC)) + "\n")
	if r.LeaderRegNo != "" {
		b.WriteString(DetailRow.Render(row("Reg. No", r.LeaderRegNo)) + "\n")
	}
	b.WriteString("\n")

	// Members 2–4 (flat fields)
	type memberFields struct {
		Name, Email, Contact, Whatsapp, NIC, RegNo, University string
	}
	members := []memberFields{
		{r.Member2Name, r.Member2Email, r.Member2Contact, r.Member2Whatsapp, r.Member2NIC, r.Member2RegNo, r.Member2University},
		{r.Member3Name, r.Member3Email, r.Member3Contact, r.Member3Whatsapp, r.Member3NIC, r.Member3RegNo, r.Member3University},
		{r.Member4Name, r.Member4Email, r.Member4Contact, r.Member4Whatsapp, r.Member4NIC, r.Member4RegNo, r.Member4University},
	}
	for i, m := range members {
		if m.Name == "" {
			continue
		}
		b.WriteString(DetailSection.Render(fmt.Sprintf("Member %d", i+2)) + "\n")
		b.WriteString(DetailRow.Render(row("Name", m.Name)) + "\n")
		b.WriteString(DetailRow.Render(row("Email", m.Email)) + "\n")
		b.WriteString(DetailRow.Render(row("Contact", m.Contact)) + "\n")
		b.WriteString(DetailRow.Render(row("WhatsApp", groupTag(m.Whatsapp, groupStatus))) + "\n")
		b.WriteString(DetailRow.Render(row("NIC", m.NIC)) + "\n")
		if m.RegNo != "" {
			b.WriteString(DetailRow.Render(row("Reg. No", m.RegNo)) + "\n")
		}
		if m.University != "" {
			b.WriteString(DetailRow.Render(row("University", m.University)) + "\n")
		}
		b.WriteString("\n")
	}

	b.WriteString(DetailSection.Render("Other Details") + "\n")
	b.WriteString(DetailRow.Render(row("Referral Source", r.ReferralSource)) + "\n")
	b.WriteString(DetailRow.Render(row("Awareness Pref.", r.AwarenessPreference)) + "\n\n")

	if r.PaymentSlipFileId != "" {
		b.WriteString(DetailSection.Render("Payment Slip") + "\n")
		b.WriteString(DetailRow.Render(row("File ID", Muted.Render(r.PaymentSlipFileId))) + "\n")
		if r.PaymentSlipUrl != "" {
			b.WriteString(DetailRow.Render(row("URL", r.PaymentSlipUrl)) + "\n")
		}
		b.WriteString(DetailRow.Render(Accent.Render("  Press [S] to download payment slip")) + "\n\n")
	}

	return b.String()
}

// RenderSchoolHackathonDetail builds a rich text view for a school hackathon registration.
func RenderSchoolHackathonDetail(r *models.SchoolHackathonRegistration, groupStatus map[string]bool) string {
	var b strings.Builder

	b.WriteString(DetailSection.Render("Registration Info") + "\n")
	b.WriteString(DetailRow.Render(row("Document ID", Muted.Render(r.ID))) + "\n")
	b.WriteString(DetailRow.Render(row("Submitted", r.SubmittedAt)) + "\n\n")

	b.WriteString(DetailSection.Render("Team Information") + "\n")
	b.WriteString(DetailRow.Render(row("Team Name", r.TeamName)) + "\n")
	b.WriteString(DetailRow.Render(row("Member Count", fmt.Sprintf("%d", r.TeamMemberCount))) + "\n\n")

	b.WriteString(DetailSection.Render("Team Leader") + "\n")
	b.WriteString(DetailRow.Render(row("Full Name", r.LeaderFullName)) + "\n")
	b.WriteString(DetailRow.Render(row("School", r.LeaderSchoolName)) + "\n")
	b.WriteString(DetailRow.Render(row("Grade", r.LeaderGrade)) + "\n")
	b.WriteString(DetailRow.Render(row("Email", r.LeaderEmail)) + "\n")
	b.WriteString(DetailRow.Render(row("Contact", groupTag(r.LeaderContactNumber, groupStatus))) + "\n")
	if r.LeaderNIC != "" {
		b.WriteString(DetailRow.Render(row("NIC", r.LeaderNIC)) + "\n")
	}
	b.WriteString("\n")

	type schoolMember struct {
		FullName, Grade, Contact, Email, NIC, SchoolName string
	}
	members := []schoolMember{
		{r.Member2FullName, r.Member2Grade, r.Member2ContactNumber, r.Member2Email, r.Member2NIC, r.Member2SchoolName},
		{r.Member3FullName, r.Member3Grade, r.Member3ContactNumber, r.Member3Email, r.Member3NIC, r.Member3SchoolName},
		{r.Member4FullName, r.Member4Grade, r.Member4ContactNumber, r.Member4Email, r.Member4NIC, r.Member4SchoolName},
	}
	for i, m := range members {
		if m.FullName == "" {
			continue
		}
		b.WriteString(DetailSection.Render(fmt.Sprintf("Member %d", i+2)) + "\n")
		b.WriteString(DetailRow.Render(row("Full Name", m.FullName)) + "\n")
		b.WriteString(DetailRow.Render(row("School", m.SchoolName)) + "\n")
		b.WriteString(DetailRow.Render(row("Grade", m.Grade)) + "\n")
		b.WriteString(DetailRow.Render(row("Email", m.Email)) + "\n")
		b.WriteString(DetailRow.Render(row("Contact", groupTag(m.Contact, groupStatus))) + "\n")
		b.WriteString("\n")
	}

	b.WriteString(DetailSection.Render("Person in Charge") + "\n")
	b.WriteString(DetailRow.Render(row("Full Name", r.InChargePersonFullName)) + "\n")
	b.WriteString(DetailRow.Render(row("Contact", r.InChargePersonContactNumber)) + "\n\n")

	b.WriteString(DetailSection.Render("Other Details") + "\n")
	b.WriteString(DetailRow.Render(row("Referral Source", r.ReferralSource)) + "\n\n")

	if r.TeamLogoFileId != "" {
		b.WriteString(DetailSection.Render("Team Logo") + "\n")
		b.WriteString(DetailRow.Render(row("File ID", Muted.Render(r.TeamLogoFileId))) + "\n")
		b.WriteString(DetailRow.Render(Accent.Render("  Press [S] to download team logo")) + "\n\n")
	}

	return b.String()
}

// RenderUniversityHackathonDetail builds a rich text view for a university hackathon registration.
func RenderUniversityHackathonDetail(r *models.UniversityHackathonRegistration, groupStatus map[string]bool) string {
	var b strings.Builder

	b.WriteString(DetailSection.Render("Registration Info") + "\n")
	b.WriteString(DetailRow.Render(row("Document ID", Muted.Render(r.ID))) + "\n")
	b.WriteString(DetailRow.Render(row("Submitted", r.SubmittedAt)) + "\n\n")

	b.WriteString(DetailSection.Render("Team Information") + "\n")
	b.WriteString(DetailRow.Render(row("Team Name", r.TeamName)) + "\n\n")

	b.WriteString(DetailSection.Render("Team Leader") + "\n")
	b.WriteString(DetailRow.Render(row("Name", r.LeaderName)) + "\n")
	b.WriteString(DetailRow.Render(row("University", r.LeaderUniversity)) + "\n")
	b.WriteString(DetailRow.Render(row("Email", r.LeaderEmail)) + "\n")
	b.WriteString(DetailRow.Render(row("Contact", r.LeaderContact)) + "\n")
	b.WriteString(DetailRow.Render(row("WhatsApp", groupTag(r.LeaderWhatsapp, groupStatus))) + "\n")
	b.WriteString(DetailRow.Render(row("NIC", r.LeaderNIC)) + "\n")
	if r.LeaderRegNo != "" {
		b.WriteString(DetailRow.Render(row("Reg. No", r.LeaderRegNo)) + "\n")
	}
	b.WriteString("\n")

	type uniMember struct {
		Name, Contact, Email, Whatsapp, NIC, RegNo, University string
	}
	members := []uniMember{
		{r.Member2Name, r.Member2Contact, r.Member2Email, r.Member2Whatsapp, r.Member2NIC, r.Member2RegNo, r.Member2University},
		{r.Member3Name, r.Member3Contact, r.Member3Email, r.Member3Whatsapp, r.Member3NIC, r.Member3RegNo, r.Member3University},
		{r.Member4Name, r.Member4Contact, r.Member4Email, r.Member4Whatsapp, r.Member4NIC, r.Member4RegNo, r.Member4University},
	}
	for i, m := range members {
		if m.Name == "" {
			continue
		}
		b.WriteString(DetailSection.Render(fmt.Sprintf("Member %d", i+2)) + "\n")
		b.WriteString(DetailRow.Render(row("Name", m.Name)) + "\n")
		b.WriteString(DetailRow.Render(row("University", m.University)) + "\n")
		b.WriteString(DetailRow.Render(row("Email", m.Email)) + "\n")
		b.WriteString(DetailRow.Render(row("Contact", m.Contact)) + "\n")
		b.WriteString(DetailRow.Render(row("WhatsApp", groupTag(m.Whatsapp, groupStatus))) + "\n")
		b.WriteString("\n")
	}

	b.WriteString(DetailSection.Render("Other Details") + "\n")
	b.WriteString(DetailRow.Render(row("Referral Source", r.ReferralSource)) + "\n\n")

	return b.String()
}

// RenderDesignathonDetail builds a rich text view for a designathon registration.
func RenderDesignathonDetail(r *models.DesignathonRegistration, groupStatus map[string]bool) string {
	var b strings.Builder

	b.WriteString(DetailSection.Render("Registration Info") + "\n")
	b.WriteString(DetailRow.Render(row("Document ID", Muted.Render(r.ID))) + "\n")
	b.WriteString(DetailRow.Render(row("Submitted", r.SubmittedAt)) + "\n\n")

	b.WriteString(DetailSection.Render("Team Information") + "\n")
	b.WriteString(DetailRow.Render(row("Team Name", r.TeamName)) + "\n")
	b.WriteString(DetailRow.Render(row("Primary Contact", r.PrimaryContact)) + "\n")
	b.WriteString(DetailRow.Render(row("Prev. Participation", r.PreviousParticipation)) + "\n\n")

	b.WriteString(DetailSection.Render("Member 1 (Leader)") + "\n")
	b.WriteString(DetailRow.Render(row("Full Name", r.Member1FullName)) + "\n")
	b.WriteString(DetailRow.Render(row("University", r.Member1University)) + "\n")
	b.WriteString(DetailRow.Render(row("Email", r.Member1Email)) + "\n")
	b.WriteString(DetailRow.Render(row("Phone", groupTag(r.Member1Phone, groupStatus))) + "\n")
	b.WriteString(DetailRow.Render(row("Reg. No", r.Member1RegNo)) + "\n")
	b.WriteString(DetailRow.Render(row("NIC", r.Member1NIC)) + "\n\n")

	type designMember struct {
		FullName, RegNo, NIC, Email, Phone, University string
	}
	members := []designMember{
		{r.Member2FullName, r.Member2RegNo, r.Member2NIC, r.Member2Email, r.Member2Phone, r.Member2University},
		{r.Member3FullName, r.Member3RegNo, r.Member3NIC, r.Member3Email, r.Member3Phone, r.Member3University},
	}
	for i, m := range members {
		if m.FullName == "" {
			continue
		}
		b.WriteString(DetailSection.Render(fmt.Sprintf("Member %d", i+2)) + "\n")
		b.WriteString(DetailRow.Render(row("Full Name", m.FullName)) + "\n")
		b.WriteString(DetailRow.Render(row("University", m.University)) + "\n")
		b.WriteString(DetailRow.Render(row("Email", m.Email)) + "\n")
		b.WriteString(DetailRow.Render(row("Phone", groupTag(m.Phone, groupStatus))) + "\n")
		b.WriteString(DetailRow.Render(row("Reg. No", m.RegNo)) + "\n")
		b.WriteString(DetailRow.Render(row("NIC", m.NIC)) + "\n\n")
	}

	if r.TeamLogoFileId != "" {
		b.WriteString(DetailSection.Render("Team Logo") + "\n")
		b.WriteString(DetailRow.Render(row("File ID", Muted.Render(r.TeamLogoFileId))) + "\n")
		if r.TeamLogoUrl != "" {
			b.WriteString(DetailRow.Render(row("URL", r.TeamLogoUrl)) + "\n")
		}
		b.WriteString(DetailRow.Render(Accent.Render("  Press [S] to download team logo")) + "\n\n")
	}

	return b.String()
}

// row renders a label+value pair aligned with fixed label width.
func row(label, value string) string {
	return Label.Render(label+":") + " " + Value.Render(value)
}
