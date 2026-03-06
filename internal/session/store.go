// Package session manages the local operator session for the CLI tool.
// The session is stored in ~/.cryptx-cli/session.json and holds the
// Appwrite session ID and expiry so the operator is not prompted to
// log in on every invocation.
package session

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"
)

const sessionFileName = "session.json"

// Session is the locally persisted operator session.
type Session struct {
	// SessionID is the Appwrite session $id returned after login.
	SessionID string `json:"sessionId"`
	// SessionSecret is the session secret (from the a_session_* cookie).
	// This is the value required for X-Appwrite-Session authenticated requests.
	SessionSecret string `json:"sessionSecret"`
	// UserID is the Appwrite user $id.
	UserID string `json:"userId"`
	// UserEmail is the logged-in operator's email.
	UserEmail string `json:"userEmail"`
	// ExpiresAt is when the session expires (RFC3339).
	ExpiresAt time.Time `json:"expiresAt"`
}

// IsExpired returns true if the session has passed its expiry time.
func (s *Session) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}

// dir returns the ~/.cryptx-cli directory path, creating it if needed.
func dir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	d := filepath.Join(home, ".cryptx-cli")
	if err := os.MkdirAll(d, 0o700); err != nil {
		return "", err
	}
	return d, nil
}

// Load reads and returns the persisted session. Returns an error if the
// file is absent, unreadable, or the session is expired.
func Load() (*Session, error) {
	d, err := dir()
	if err != nil {
		return nil, err
	}
	path := filepath.Join(d, sessionFileName)

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, errors.New("no saved session found")
		}
		return nil, err
	}

	var s Session
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, errors.New("corrupt session file")
	}

	if s.IsExpired() {
		return nil, errors.New("session has expired")
	}

	return &s, nil
}

// Save persists the session to disk.
func Save(s *Session) error {
	d, err := dir()
	if err != nil {
		return err
	}
	path := filepath.Join(d, sessionFileName)

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

// Delete removes the local session file (logout).
func Delete() error {
	d, err := dir()
	if err != nil {
		return err
	}
	path := filepath.Join(d, sessionFileName)
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}
