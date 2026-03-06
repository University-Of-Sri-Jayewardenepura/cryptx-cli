package models

import "time"

// SchoolHackathonRegistration mirrors the hackathon-school-new Appwrite collection.
// All member fields are stored as flat attributes.
type SchoolHackathonRegistration struct {
	// Appwrite document meta
	ID        string `json:"$id"`
	CreatedAt string `json:"$createdAt"`
	UpdatedAt string `json:"$updatedAt"`

	TeamName        string `json:"teamName"`
	TeamMemberCount int    `json:"teamMemberCount"`

	// Team logo
	TeamLogoFileId string `json:"teamLogoFileId,omitempty"`
	TeamLogoUrl    string `json:"teamLogoUrl,omitempty"`

	// Leader (mandatory)
	LeaderFullName      string `json:"leaderFullName"`
	LeaderGrade         string `json:"leaderGrade"`
	LeaderContactNumber string `json:"leaderContactNumber"`
	LeaderEmail         string `json:"leaderEmail"`
	LeaderNIC           string `json:"leaderNIC,omitempty"`
	LeaderSchoolName    string `json:"leaderSchoolName"`

	// Member 2 (mandatory)
	Member2FullName      string `json:"member2FullName"`
	Member2Grade         string `json:"member2Grade"`
	Member2ContactNumber string `json:"member2ContactNumber"`
	Member2Email         string `json:"member2Email"`
	Member2NIC           string `json:"member2NIC,omitempty"`
	Member2SchoolName    string `json:"member2SchoolName"`

	// Member 3 (optional)
	Member3FullName      string `json:"member3FullName,omitempty"`
	Member3Grade         string `json:"member3Grade,omitempty"`
	Member3ContactNumber string `json:"member3ContactNumber,omitempty"`
	Member3Email         string `json:"member3Email,omitempty"`
	Member3NIC           string `json:"member3NIC,omitempty"`
	Member3SchoolName    string `json:"member3SchoolName,omitempty"`

	// Member 4 (optional)
	Member4FullName      string `json:"member4FullName,omitempty"`
	Member4Grade         string `json:"member4Grade,omitempty"`
	Member4ContactNumber string `json:"member4ContactNumber,omitempty"`
	Member4Email         string `json:"member4Email,omitempty"`
	Member4NIC           string `json:"member4NIC,omitempty"`
	Member4SchoolName    string `json:"member4SchoolName,omitempty"`

	// Person-in-charge (mandatory)
	InChargePersonFullName      string `json:"inChargePersonFullName"`
	InChargePersonContactNumber string `json:"inChargePersonContactNumber"`

	ReferralSource string        `json:"referralSource"`
	PaymentStatus  PaymentStatus `json:"paymentStatus"`
	SubmittedAt    string        `json:"submittedAt"`
}

// DisplayName returns a short human-readable name for list views.
func (r *SchoolHackathonRegistration) DisplayName() string { return r.TeamName }

// PrimaryEmail returns the email to send confirmations to.
func (r *SchoolHackathonRegistration) PrimaryEmail() string { return r.LeaderEmail }

// CreatedAtTime parses the Appwrite ISO timestamp.
func (r *SchoolHackathonRegistration) CreatedAtTime() time.Time {
	t, _ := time.Parse(time.RFC3339, r.CreatedAt)
	return t
}

// ──────────────────────────────────────────────────────────────────────────────

// UniversityHackathonRegistration mirrors the hackathon-university-new Appwrite collection.
type UniversityHackathonRegistration struct {
	// Appwrite document meta
	ID        string `json:"$id"`
	CreatedAt string `json:"$createdAt"`
	UpdatedAt string `json:"$updatedAt"`

	TeamName string `json:"teamName"`

	// Team logo
	TeamLogoFileId string `json:"teamLogoFileId,omitempty"`
	TeamLogoUrl    string `json:"teamLogoUrl,omitempty"`

	// Leader
	LeaderName       string `json:"leaderName"`
	LeaderUniversity string `json:"leaderUniversity"`
	LeaderContact    string `json:"leaderContact"`
	LeaderEmail      string `json:"leaderEmail"`
	LeaderWhatsapp   string `json:"leaderWhatsapp"`
	LeaderNIC        string `json:"leaderNIC"`
	LeaderRegNo      string `json:"leaderRegNo"`

	// Member 2 (optional)
	Member2Name       string `json:"member2Name,omitempty"`
	Member2Contact    string `json:"member2Contact,omitempty"`
	Member2Email      string `json:"member2Email,omitempty"`
	Member2Whatsapp   string `json:"member2Whatsapp,omitempty"`
	Member2NIC        string `json:"member2NIC,omitempty"`
	Member2RegNo      string `json:"member2RegNo,omitempty"`
	Member2University string `json:"member2University,omitempty"`

	// Member 3 (optional)
	Member3Name       string `json:"member3Name,omitempty"`
	Member3Contact    string `json:"member3Contact,omitempty"`
	Member3Email      string `json:"member3Email,omitempty"`
	Member3Whatsapp   string `json:"member3Whatsapp,omitempty"`
	Member3NIC        string `json:"member3NIC,omitempty"`
	Member3RegNo      string `json:"member3RegNo,omitempty"`
	Member3University string `json:"member3University,omitempty"`

	// Member 4 (optional)
	Member4Name       string `json:"member4Name,omitempty"`
	Member4Contact    string `json:"member4Contact,omitempty"`
	Member4Email      string `json:"member4Email,omitempty"`
	Member4Whatsapp   string `json:"member4Whatsapp,omitempty"`
	Member4NIC        string `json:"member4NIC,omitempty"`
	Member4RegNo      string `json:"member4RegNo,omitempty"`
	Member4University string `json:"member4University,omitempty"`

	ReferralSource string `json:"referralSource"`
	SubmittedAt    string `json:"submittedAt"`
}

// DisplayName returns a short human-readable name for list views.
func (r *UniversityHackathonRegistration) DisplayName() string { return r.TeamName }

// PrimaryEmail returns the email to send confirmations to.
func (r *UniversityHackathonRegistration) PrimaryEmail() string { return r.LeaderEmail }

// CreatedAtTime parses the Appwrite ISO timestamp.
func (r *UniversityHackathonRegistration) CreatedAtTime() time.Time {
	t, _ := time.Parse(time.RFC3339, r.CreatedAt)
	return t
}
