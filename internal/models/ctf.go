package models

import "time"

// PaymentStatus represents the Appwrite paymentStatus enum.
type PaymentStatus string

const (
	PaymentPending  PaymentStatus = "pending_verification"
	PaymentVerified PaymentStatus = "verified"
	PaymentRejected PaymentStatus = "rejected"
)

// CTFRegistration mirrors the ctf-registrations-new Appwrite collection.
// All member fields are stored as flat attributes — no JSON blobs.
type CTFRegistration struct {
	// Appwrite document meta
	ID        string `json:"$id"`
	CreatedAt string `json:"$createdAt"`
	UpdatedAt string `json:"$updatedAt"`

	// Registration type
	RegistrationType string `json:"registrationType"` // "Individual" | "Team"
	TeamName         string `json:"teamName,omitempty"`

	// Leader
	LeaderName       string `json:"leaderName"`
	LeaderUniversity string `json:"leaderUniversity"`
	LeaderContact    string `json:"leaderContact"`
	LeaderEmail      string `json:"leaderEmail"`
	LeaderWhatsapp   string `json:"leaderWhatsapp"`
	LeaderNIC        string `json:"leaderNIC"`
	LeaderRegNo      string `json:"leaderRegNo,omitempty"`

	// Member 2 (required for Team)
	Member2Name        string `json:"member2Name,omitempty"`
	Member2Contact     string `json:"member2Contact,omitempty"`
	Member2Email       string `json:"member2Email,omitempty"`
	Member2Whatsapp    string `json:"member2Whatsapp,omitempty"`
	Member2NIC         string `json:"member2NIC,omitempty"`
	Member2RegNo       string `json:"member2RegNo,omitempty"`
	Member2University  string `json:"member2University,omitempty"`

	// Member 3 (optional)
	Member3Name        string `json:"member3Name,omitempty"`
	Member3Contact     string `json:"member3Contact,omitempty"`
	Member3Email       string `json:"member3Email,omitempty"`
	Member3Whatsapp    string `json:"member3Whatsapp,omitempty"`
	Member3NIC         string `json:"member3NIC,omitempty"`
	Member3RegNo       string `json:"member3RegNo,omitempty"`
	Member3University  string `json:"member3University,omitempty"`

	// Member 4 (optional)
	Member4Name        string `json:"member4Name,omitempty"`
	Member4Contact     string `json:"member4Contact,omitempty"`
	Member4Email       string `json:"member4Email,omitempty"`
	Member4Whatsapp    string `json:"member4Whatsapp,omitempty"`
	Member4NIC         string `json:"member4NIC,omitempty"`
	Member4RegNo       string `json:"member4RegNo,omitempty"`
	Member4University  string `json:"member4University,omitempty"`

	// Extra fields
	ReferralSource      string `json:"referralSource"`
	AwarenessPreference string `json:"awarenessPreference"` // "Physical" | "Online"
	Agreement           bool   `json:"agreement"`

	// Payment slip (stored in CTF bucket)
	PaymentSlipFileId string `json:"paymentSlipFileId,omitempty"`
	PaymentSlipUrl    string `json:"paymentSlipUrl,omitempty"`

	// Payment status (set by Appwrite, managed by CLI)
	PaymentStatus PaymentStatus `json:"paymentStatus"`
	SubmittedAt   string        `json:"submittedAt"`
}

// DisplayName returns a short human-readable name for list views.
func (r *CTFRegistration) DisplayName() string {
	if r.RegistrationType == "Team" && r.TeamName != "" {
		return r.TeamName
	}
	return r.LeaderName
}

// PrimaryEmail returns the email to send confirmations to.
func (r *CTFRegistration) PrimaryEmail() string { return r.LeaderEmail }

// CreatedAtTime parses the Appwrite ISO timestamp.
func (r *CTFRegistration) CreatedAtTime() time.Time {
	t, _ := time.Parse(time.RFC3339, r.CreatedAt)
	return t
}
