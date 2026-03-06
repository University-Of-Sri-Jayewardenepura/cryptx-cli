package models

import "time"

// DesignathonRegistration mirrors the designathon-registrations-new Appwrite collection.
// All member fields are stored as flat attributes — no JSON blobs.
type DesignathonRegistration struct {
	// Appwrite document meta
	ID        string `json:"$id"`
	CreatedAt string `json:"$createdAt"`
	UpdatedAt string `json:"$updatedAt"`

	TeamName       string `json:"teamName"`
	PrimaryContact string `json:"primaryContact"`

	// Team logo (stored in Designathon bucket)
	TeamLogoFileId string `json:"teamLogoFileId,omitempty"`
	TeamLogoUrl    string `json:"teamLogoUrl,omitempty"`

	// Member 1 — leader, always required
	Member1FullName   string `json:"member1FullName"`
	Member1RegNo      string `json:"member1RegNo"`
	Member1NIC        string `json:"member1NIC"`
	Member1Email      string `json:"member1Email"`
	Member1Phone      string `json:"member1Phone"`
	Member1University string `json:"member1University"`

	// Member 2 (optional)
	Member2FullName   string `json:"member2FullName,omitempty"`
	Member2RegNo      string `json:"member2RegNo,omitempty"`
	Member2NIC        string `json:"member2NIC,omitempty"`
	Member2Email      string `json:"member2Email,omitempty"`
	Member2Phone      string `json:"member2Phone,omitempty"`
	Member2University string `json:"member2University,omitempty"`

	// Member 3 (optional)
	Member3FullName   string `json:"member3FullName,omitempty"`
	Member3RegNo      string `json:"member3RegNo,omitempty"`
	Member3NIC        string `json:"member3NIC,omitempty"`
	Member3Email      string `json:"member3Email,omitempty"`
	Member3Phone      string `json:"member3Phone,omitempty"`
	Member3University string `json:"member3University,omitempty"`

	PreviousParticipation string        `json:"previousParticipation"` // "Yes" | "No"
	Agreement             bool          `json:"agreement"`
	PaymentStatus         PaymentStatus `json:"paymentStatus"`
	SubmittedAt           string        `json:"submittedAt"`
}

// DisplayName returns a short human-readable name for list views.
func (r *DesignathonRegistration) DisplayName() string { return r.TeamName }

// PrimaryEmail returns the email to send confirmations to.
func (r *DesignathonRegistration) PrimaryEmail() string { return r.Member1Email }

// CreatedAtTime parses the Appwrite ISO timestamp.
func (r *DesignathonRegistration) CreatedAtTime() time.Time {
	t, _ := time.Parse(time.RFC3339, r.CreatedAt)
	return t
}
