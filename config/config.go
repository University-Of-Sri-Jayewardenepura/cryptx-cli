package config

import (
	"errors"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// Config holds all application configuration.
type Config struct {
	AppwriteEndpoint  string
	AppwriteProjectID string

	DatabaseID string

	// Collection IDs
	CTFCollectionID                 string
	SchoolHackathonCollectionID     string
	UniversityHackathonCollectionID string
	DesignathonCollectionID         string
	MerchCollectionID               string

	// Storage bucket IDs
	CTFBucketID          string // CTF payment slips
	DesignathonBucketID  string // Designathon team logos
	HackathonUniBucketID string // University hackathon logos
	HackathonSchBucketID string // School hackathon logos
	MerchBucketID        string // Merch payment slips

	// Merch event dispatch details (used in dispatch emails).
	MerchEventDate  string // e.g. "2026-04-05"
	MerchEventTime  string // e.g. "09:00 AM"
	MerchEventVenue string // e.g. "USJ Faculty of Technology"

	// Resend — primary email delivery provider.
	ResendAPIKey string

	// SMTP — optional fallback / legacy.
	SMTPHost string
	SMTPPort int
	SMTPUser string
	SMTPPass string
	SMTPFrom string

	// WAHA — WhatsApp HTTP API for group-membership checks.
	// All fields are optional; checks are skipped when WAHABaseURL is empty.
	WAHABaseURL             string
	WAHAAPIKey              string
	WAHASession             string
	WAHACTFGroupID          string
	WAHASchoolHackGroupID   string
	WAHAUniHackGroupID      string
	WAHADesignathonGroupID  string
}

// Load returns configuration from runtime environment variables.
// If a .env file exists in the current working directory it is loaded first.
func Load() (*Config, error) {
	// Best-effort .env load for local/dev usage; exported env vars still win.
	_ = godotenv.Load()

	cfg := &Config{
		AppwriteEndpoint:  getEnv("APPWRITE_ENDPOINT", "https://sgp.cloud.appwrite.io/v1"),
		AppwriteProjectID: getEnv("APPWRITE_PROJECT_ID", "6999c84e00381b7a5f0b"),

		DatabaseID: getEnv("APPWRITE_DATABASE_ID", "cryptx-db"),

		CTFCollectionID:                 getEnv("APPWRITE_CTF_COLLECTION_ID", "ctf-registrations-new"),
		SchoolHackathonCollectionID:     getEnv("APPWRITE_SCHOOL_HACKATHON_COLLECTION_ID", "hackathon-school-new"),
		UniversityHackathonCollectionID: getEnv("APPWRITE_UNIVERSITY_HACKATHON_COLLECTION_ID", "hackathon-university-new"),
		DesignathonCollectionID:         getEnv("APPWRITE_DESIGNATHON_COLLECTION_ID", "designathon-registrations-new"),
		MerchCollectionID:               getEnv("APPWRITE_MERCH_COLLECTION_ID", "merch-orders"),

		CTFBucketID:          getEnv("APPWRITE_CTF_BUCKET_ID", "cryptx-registrations"),
		DesignathonBucketID:  getEnv("APPWRITE_DESIGNATHON_BUCKET_ID", "cryptx-logos-designathon"),
		HackathonUniBucketID: getEnv("APPWRITE_HACKATHON_UNIVERSITY_BUCKET_ID", "cryptx-logos-hackathon-university"),
		HackathonSchBucketID: getEnv("APPWRITE_HACKATHON_SCHOOL_BUCKET_ID", "cryptx-logos-hackathon-school"),
		MerchBucketID:        getEnv("APPWRITE_MERCH_BUCKET_ID", "cryptx-merch-slips"),

		MerchEventDate:  getEnv("MERCH_EVENT_DATE", ""),
		MerchEventTime:  getEnv("MERCH_EVENT_TIME", ""),
		MerchEventVenue: getEnv("MERCH_EVENT_VENUE", ""),

		ResendAPIKey: getEnv("RESEND_API_KEY", ""),

		SMTPHost: getEnv("SMTP_HOST", "smtp.gmail.com"),
		SMTPPort: getEnvInt("SMTP_PORT", 587),
		SMTPUser: getEnv("SMTP_USER", ""),
		SMTPPass: getEnv("SMTP_PASS", ""),
		SMTPFrom: getEnv("SMTP_FROM", "noreply@cryptx.lk"),

		WAHABaseURL:            getEnv("WAHA_BASE_URL", ""),
		WAHAAPIKey:             getEnv("WAHA_API_KEY", ""),
		WAHASession:            getEnv("WAHA_SESSION", "default"),
		WAHACTFGroupID:         getEnv("WAHA_CTF_GROUP_ID", ""),
		WAHASchoolHackGroupID:  getEnv("WAHA_SCHOOL_HACKATHON_GROUP_ID", ""),
		WAHAUniHackGroupID:     getEnv("WAHA_UNIVERSITY_HACKATHON_GROUP_ID", ""),
		WAHADesignathonGroupID: getEnv("WAHA_DESIGNATHON_GROUP_ID", ""),
	}

	return cfg, cfg.validate()
}

func (c *Config) validate() error {
	if c.AppwriteProjectID == "" {
		return errors.New("AppwriteProjectID is required")
	}
	if c.DatabaseID == "" {
		return errors.New("DatabaseID is required")
	}
	return nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}
