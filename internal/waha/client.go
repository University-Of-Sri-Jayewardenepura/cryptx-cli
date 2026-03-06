// Package waha provides a thin HTTP client for the WAHA WhatsApp HTTP API.
// It is used to check whether a registrant's phone number is a member of
// the relevant WhatsApp group for their event.
package waha

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Client talks to a WAHA instance.
type Client struct {
	baseURL    string
	apiKey     string
	session    string
	httpClient *http.Client
}

// participant mirrors one element of the /participants/v2 response array.
type participant struct {
	ID   string `json:"id"`
	Role string `json:"role"`
}

// NewClient returns a Client.  baseURL must not have a trailing slash
// (e.g. "http://localhost:3000").  session is the WAHA session name
// (typically "default").
func NewClient(baseURL, apiKey, session string) *Client {
	if session == "" {
		session = "default"
	}
	httpClient := &http.Client{Timeout: 10 * time.Second}

	// When CRYPTX_DEBUG=1 is set, wrap the transport to log all WAHA
	// requests and responses into ~/cryptx-debug.log.
	if os.Getenv("CRYPTX_DEBUG") == "1" {
		if lt := wahaDebugLogger(); lt != nil {
			httpClient.Transport = &wahaLoggingTransport{
				wrapped: http.DefaultTransport,
				log:     lt,
			}
		}
	}

	return &Client{
		baseURL:    strings.TrimRight(baseURL, "/"),
		apiKey:     apiKey,
		session:    session,
		httpClient: httpClient,
	}
}

// ── debug HTTP logger ─────────────────────────────────────────────────────────

type wahaLoggingTransport struct {
	wrapped http.RoundTripper
	log     *os.File
}

func (l *wahaLoggingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	ts := time.Now().Format("15:04:05.000")
	fmt.Fprintf(l.log, "\n[%s] WAHA REQUEST  %s %s\n", ts, req.Method, req.URL.String())
	for k, vs := range req.Header {
		for _, v := range vs {
			fmt.Fprintf(l.log, "  > %s: %s\n", k, v)
		}
	}

	// Log request body if present.
	if req.Body != nil {
		body, _ := io.ReadAll(req.Body)
		req.Body = io.NopCloser(bytes.NewReader(body))
		fmt.Fprintf(l.log, "  > Body: %s\n", string(body))
	}

	wrapped := l.wrapped
	if wrapped == nil {
		wrapped = http.DefaultTransport
	}
	resp, err := wrapped.RoundTrip(req)
	if err != nil {
		fmt.Fprintf(l.log, "[%s] WAHA ERROR    %v\n", ts, err)
		return resp, err
	}

	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	resp.Body = io.NopCloser(bytes.NewReader(body))
	fmt.Fprintf(l.log, "[%s] WAHA RESPONSE %d\n", ts, resp.StatusCode)
	fmt.Fprintf(l.log, "  < Body: %s\n", string(body))
	return resp, nil
}

func wahaDebugLogger() *os.File {
	home, _ := os.UserHomeDir()
	path := filepath.Join(home, "cryptx-debug.log")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return nil
	}
	fmt.Fprintf(f, "\n=== waha debug session %s ===\n", time.Now().Format(time.RFC3339))
	return f
}

// IsEnabled returns false when no base URL has been configured, which means
// all group checks are silently skipped.
func (c *Client) IsEnabled() bool {
	return c.baseURL != ""
}

// GetParticipantSet fetches all participants of groupID and returns a set of
// normalised phone numbers (digits only, no @c.us suffix) for quick lookup.
func (c *Client) GetParticipantSet(groupID string) (map[string]bool, error) {
	// The @ character must be percent-encoded in the path segment.
	// WAHA group IDs look like "120363XXXXXXXXXX@g.us".
	encodedID := url.PathEscape(groupID)
	endpoint := fmt.Sprintf("%s/api/%s/groups/%s/participants/v2",
		c.baseURL, c.session, encodedID)

	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("waha: build request: %w", err)
	}
	if c.apiKey != "" {
		req.Header.Set("X-Api-Key", c.apiKey)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("waha: GET participants: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("waha: unexpected status %d for group %s", resp.StatusCode, groupID)
	}

	var participants []participant
	if err := json.NewDecoder(resp.Body).Decode(&participants); err != nil {
		return nil, fmt.Errorf("waha: decode participants: %w", err)
	}

	set := make(map[string]bool, len(participants))
	for _, p := range participants {
		// p.ID is like "94771234567@c.us" — keep only the digit prefix.
		digits := strings.SplitN(p.ID, "@", 2)[0]
		set[digits] = true
	}
	return set, nil
}

// IsInGroup normalises phone and checks whether it appears in the participant
// set for groupID.  Returns (false, nil) when the number is not present.
func (c *Client) IsInGroup(phone, groupID string) (bool, error) {
	set, err := c.GetParticipantSet(groupID)
	if err != nil {
		return false, err
	}
	normalised := NormalizePhone(phone)
	return set[normalised], nil
}

// CheckPhones fetches the participant set for groupID once and then checks
// every supplied phone number.  Returns a map from original phone string to
// in-group boolean.  If the lookup fails the error is returned; the map will
// be nil.
func (c *Client) CheckPhones(groupID string, phones []string) (map[string]bool, error) {
	set, err := c.GetParticipantSet(groupID)
	if err != nil {
		return nil, err
	}
	result := make(map[string]bool, len(phones))
	for _, p := range phones {
		if p == "" {
			continue
		}
		result[p] = set[NormalizePhone(p)]
	}
	return result, nil
}

// NormalizePhone converts a Sri Lankan (or already-international) phone number
// to the bare digit string used in WhatsApp JIDs (no leading +, no @c.us).
//
// Examples:
//
//	"0771234567"   → "94771234567"
//	"+94771234567" → "94771234567"
//	"94771234567"  → "94771234567"
func NormalizePhone(s string) string {
	// Strip whitespace, dashes, spaces.
	s = strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' || r == '+' {
			return r
		}
		return -1
	}, s)

	// Remove leading +
	s = strings.TrimPrefix(s, "+")

	// Sri Lankan local format: starts with 0 and is 10 digits → replace 0 with 94
	if strings.HasPrefix(s, "0") && len(s) == 10 {
		s = "94" + s[1:]
	}

	return s
}

// ResolveChatID calls GET /api/contacts/check-exists to get the canonical
// WhatsApp chatId for a normalised phone number. On newer WhatsApp protocols
// the chatId may be a LID rather than the traditional "digits@c.us" form.
// Returns ("", false, nil) when the number is not registered on WhatsApp.
func (c *Client) ResolveChatID(normalizedPhone string) (chatID string, exists bool, err error) {
	endpoint := fmt.Sprintf("%s/api/contacts/check-exists?phone=%s&session=%s",
		c.baseURL, url.QueryEscape(normalizedPhone), url.QueryEscape(c.session))

	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return "", false, fmt.Errorf("waha: build check-exists request: %w", err)
	}
	if c.apiKey != "" {
		req.Header.Set("X-Api-Key", c.apiKey)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", false, fmt.Errorf("waha: check-exists: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", false, fmt.Errorf("waha: check-exists status %d for %s", resp.StatusCode, normalizedPhone)
	}

	var result struct {
		NumberExists bool   `json:"numberExists"`
		ChatID       string `json:"chatId"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", false, fmt.Errorf("waha: decode check-exists: %w", err)
	}
	return result.ChatID, result.NumberExists, nil
}

// ResolveLID calls GET /api/{session}/lids/pn/{phoneNumber} to fetch the WhatsApp
// Linked ID (LID) for a phone number. The LID is required by newer WhatsApp protocol
// versions when adding participants to groups. Returns "" if the LID is not found.
func (c *Client) ResolveLID(normalizedPhone string) (string, error) {
	// The endpoint expects the JID form (digits@c.us), with @ percent-encoded.
	jid := url.PathEscape(normalizedPhone + "@c.us")
	endpoint := fmt.Sprintf("%s/api/%s/lids/pn/%s", c.baseURL, c.session, jid)

	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return "", fmt.Errorf("waha: build lids request: %w", err)
	}
	if c.apiKey != "" {
		req.Header.Set("X-Api-Key", c.apiKey)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("waha: lids lookup: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", nil // not found or unsupported — caller falls back
	}

	var result struct {
		LID string `json:"lid"`
		PN  string `json:"pn"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", nil
	}
	return result.LID, nil // may be "" (null in JSON) when not found
}

// RefreshGroups calls POST /api/{session}/groups/refresh to sync group metadata
// (including LID mappings) from the WhatsApp server. This must be called before
// LID lookups when the LID table is stale. Errors are soft — a failure here
// should not block the add attempt.
func (c *Client) RefreshGroups() error {
	endpoint := fmt.Sprintf("%s/api/%s/groups/refresh", c.baseURL, c.session)
	req, err := http.NewRequest(http.MethodPost, endpoint, nil)
	if err != nil {
		return fmt.Errorf("waha: build refresh request: %w", err)
	}
	if c.apiKey != "" {
		req.Header.Set("X-Api-Key", c.apiKey)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("waha: refresh groups: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("waha: refresh groups status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return nil
}

// AddParticipants adds phone numbers to groupID using the stable WAHA endpoint
// POST /api/{session}/groups/{id}/participants/add.
//
// Strategy for WEBJS engine:
//  1. Refresh group metadata (syncs the group's LID table for existing members).
//  2. For each phone, call check-exists — this forces the WEBJS session to look
//     up the contact and populate its internal chat table, which is required by
//     whatsapp-web.js's addParticipants before the add can succeed.
//  3. Use the chatId returned by check-exists (always @c.us format for WEBJS).
//     Skip numbers that are not registered on WhatsApp.
//
// Empty strings are silently skipped. A non-2xx HTTP status is returned as an error.
func (c *Client) AddParticipants(groupID string, phones []string) error {
	// Step 1: refresh group metadata.
	_ = c.RefreshGroups()

	type addEntry struct {
		ID string `json:"id"`
	}
	var entries []addEntry
	for _, p := range phones {
		if p == "" {
			continue
		}
		normalized := NormalizePhone(p)

		// Step 2+3: call check-exists to sync the contact into WEBJS's chat
		// table and get the canonical chatId (@c.us format).
		chatID, exists, err := c.ResolveChatID(normalized)
		if err != nil || !exists || chatID == "" {
			// Fall back to standard JID if lookup fails.
			chatID = normalized + "@c.us"
		}
		entries = append(entries, addEntry{ID: chatID})
	}
	if len(entries) == 0 {
		return nil
	}

	encodedID := url.PathEscape(groupID)
	endpoint := fmt.Sprintf("%s/api/%s/groups/%s/participants/add",
		c.baseURL, c.session, encodedID)

	payload, _ := json.Marshal(map[string]interface{}{"participants": entries})
	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("waha: build add request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if c.apiKey != "" {
		req.Header.Set("X-Api-Key", c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("waha: add participants: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("waha: add participants status %d: %s",
			resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return nil
}
