// Package email handles sending confirmation emails via the Resend HTTP API.
// If RESEND_API_KEY is not configured it falls back to the legacy SMTP path.
package email

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"mime"
	"net/http"
	"net/smtp"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cryptx/cryptx-cli/config"
)

const resendEndpoint = "https://api.resend.com/emails"

// ── Data types ────────────────────────────────────────────────────────────────

// ConfirmationData holds template variables for the confirmation email.
type ConfirmationData struct {
	EventName        string // "CTF", "School Hackathon", "University Hackathon", "Designathon"
	RecipientName    string
	RecipientEmail   string
	TeamName         string // empty for individual registrations
	RegistrationType string // "Individual", "Team", etc.
	ConfirmedAt      string // human-readable timestamp
	DocID            string // Appwrite document ID / reference
}

// CustomEmailData holds the fields for an all-in operator-composed email.
type CustomEmailData struct {
	From        string // e.g. "CryptX Team <info@cryptx.lk>"
	To          []string
	Subject     string
	HTML        string
	Attachments []Attachment
}

// Attachment represents a file to attach to an outgoing email.
type Attachment struct {
	Filename string
	Content  []byte // raw file bytes
}

// ── Resend wire types ─────────────────────────────────────────────────────────

type resendAttachment struct {
	Filename string `json:"filename"`
	Content  string `json:"content"` // base64
}

type resendPayload struct {
	From        string             `json:"from"`
	To          []string           `json:"to"`
	Subject     string             `json:"subject"`
	HTML        string             `json:"html"`
	Attachments []resendAttachment `json:"attachments,omitempty"`
}

// ── Public API ────────────────────────────────────────────────────────────────

// SendConfirmation sends a payment-confirmed email to the registrant.
// It uses Resend if RESEND_API_KEY is set, otherwise falls back to SMTP.
func SendConfirmation(cfg *config.Config, data ConfirmationData) error {
	if data.ConfirmedAt == "" {
		data.ConfirmedAt = time.Now().Format("02 Jan 2006, 15:04 MST")
	}

	htmlBody, from, subject := buildConfirmationEmail(data)

	if cfg.ResendAPIKey != "" {
		return sendViaResend(cfg.ResendAPIKey, resendPayload{
			From:    from,
			To:      []string{data.RecipientEmail},
			Subject: subject,
			HTML:    htmlBody,
		})
	}

	// Fallback: SMTP
	return sendViaSMTP(cfg, data.RecipientEmail, subject, htmlBody)
}

// SendCustomEmail delivers an operator-composed email with optional attachments.
// It uses Resend if RESEND_API_KEY is set, otherwise errors (pop handles that path).
func SendCustomEmail(cfg *config.Config, data CustomEmailData) error {
	if cfg.ResendAPIKey == "" {
		return fmt.Errorf("RESEND_API_KEY not configured; use pop to send custom emails")
	}

	var attachments []resendAttachment
	for _, a := range data.Attachments {
		attachments = append(attachments, resendAttachment{
			Filename: a.Filename,
			Content:  base64.StdEncoding.EncodeToString(a.Content),
		})
	}

	return sendViaResend(cfg.ResendAPIKey, resendPayload{
		From:        data.From,
		To:          data.To,
		Subject:     data.Subject,
		HTML:        data.HTML,
		Attachments: attachments,
	})
}

// LoadAttachment reads a file from disk and returns an Attachment.
func LoadAttachment(path string) (Attachment, error) {
	path = strings.TrimSpace(path)

	// Strip any terminal drag-and-drop file:// prefix or surrounding quotes.
	path = strings.TrimPrefix(path, "file://")
	path = strings.Trim(path, "'\"")

	data, err := os.ReadFile(path)
	if err != nil {
		return Attachment{}, fmt.Errorf("read attachment %q: %w", path, err)
	}
	return Attachment{
		Filename: filepath.Base(path),
		Content:  data,
	}, nil
}

// ── Internal helpers ──────────────────────────────────────────────────────────

func sendViaResend(apiKey string, payload resendPayload) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal resend payload: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, resendEndpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create resend request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("resend http: %w", err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("resend API error %d: %s", resp.StatusCode, string(respBody))
	}
	return nil
}

func sendViaSMTP(cfg *config.Config, to, subject, htmlBody string) error {
	addr := fmt.Sprintf("%s:%d", cfg.SMTPHost, cfg.SMTPPort)

	var auth smtp.Auth
	if cfg.SMTPUser != "" && cfg.SMTPPass != "" {
		auth = smtp.PlainAuth("", cfg.SMTPUser, cfg.SMTPPass, cfg.SMTPHost)
	}

	mimeHeader := "MIME-version: 1.0;\r\nContent-Type: text/html; charset=\"UTF-8\";\r\n"
	msg := "From: CryptX 2.0 <" + cfg.SMTPFrom + ">\r\n" +
		"To: " + to + "\r\n" +
		"Subject: " + subject + "\r\n" +
		mimeHeader + "\r\n" +
		htmlBody

	return smtp.SendMail(addr, auth, cfg.SMTPFrom, []string{to}, []byte(msg))
}

// MIMEType infers a MIME type from a filename extension.
func MIMEType(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	t := mime.TypeByExtension(ext)
	if t == "" {
		return "application/octet-stream"
	}
	return t
}

// ── Confirmation email builders (event-specific themes) ───────────────────────

func buildConfirmationEmail(data ConfirmationData) (html, from, subject string) {
	switch strings.ToLower(data.EventName) {
	case "ctf", "capture the flag":
		return buildCTFConfirmEmail(data)
	case "school hackathon", "hackathon":
		return buildHackathonConfirmEmail(data)
	case "university hackathon":
		return buildUniversityHackathonConfirmEmail(data)
	case "designathon":
		return buildDesignathonConfirmEmail(data)
	default:
		return buildGenericConfirmEmail(data)
	}
}

// ── CTF — rose/red theme ──────────────────────────────────────────────────────

func buildCTFConfirmEmail(data ConfirmationData) (string, string, string) {
	const tmplStr = `
<div style="font-family:sans-serif;max-width:600px;margin:0 auto;background-color:#0f0a0b;color:#f0e6e8;padding:0;border:1px solid #3b1520;">
  <!-- Header -->
  <div style="background:linear-gradient(135deg,#1a0510 0%,#2d0918 100%);padding:32px 32px 24px;border-bottom:1px solid #e11d4833;">
    <h1 style="margin:0 0 4px;font-size:26px;letter-spacing:0.08em;color:#e11d48;font-weight:800;text-transform:uppercase;">CryptX 2.0</h1>
    <p style="margin:0;font-size:12px;color:#94a3b8;letter-spacing:0.1em;text-transform:uppercase;">Capture The Flag</p>
    <div style="display:inline-block;margin-top:14px;padding:5px 16px;background:#e11d4822;border:1px solid #e11d4855;border-radius:20px;font-size:11px;color:#e11d48;letter-spacing:0.14em;text-transform:uppercase;">✓ Registration Confirmed</div>
  </div>

  <!-- Body -->
  <div style="padding:32px;">
    <p style="margin:0 0 12px;font-size:17px;font-weight:600;color:#f1e0e3;">Hi {{.RecipientName}},</p>
    <p style="margin:0 0 24px;font-size:14px;color:#c09ca0;line-height:1.7;">
      Great news! Your registration for <strong style="color:#f1e0e3;">CryptX 2.0 — Capture The Flag</strong>
      has been reviewed and your payment has been <strong style="color:#e11d48;">confirmed</strong>.
      You're officially in!
    </p>

    <!-- Details card -->
    <div style="background:#1a0d10;border:1px solid #3b1520;border-radius:8px;padding:20px 24px;margin-bottom:24px;">
      <p style="margin:0 0 14px;font-size:11px;text-transform:uppercase;letter-spacing:0.1em;color:#64748b;">Registration Details</p>
      {{if .TeamName}}
      <div style="display:flex;justify-content:space-between;padding:8px 0;border-bottom:1px solid #3b152066;font-size:13px;">
        <span style="color:#875568;">Team Name</span><span style="color:#f0e6e8;font-weight:500;">{{.TeamName}}</span>
      </div>
      {{end}}
      <div style="display:flex;justify-content:space-between;padding:8px 0;border-bottom:1px solid #3b152066;font-size:13px;">
        <span style="color:#875568;">Name</span><span style="color:#f0e6e8;font-weight:500;">{{.RecipientName}}</span>
      </div>
      <div style="display:flex;justify-content:space-between;padding:8px 0;border-bottom:1px solid #3b152066;font-size:13px;">
        <span style="color:#875568;">Type</span><span style="color:#f0e6e8;font-weight:500;">{{.RegistrationType}}</span>
      </div>
      {{if .DocID}}
      <div style="display:flex;justify-content:space-between;padding:8px 0;border-bottom:1px solid #3b152066;font-size:13px;">
        <span style="color:#875568;">Reference ID</span><span style="color:#f0e6e8;font-family:monospace;font-size:11px;">{{.DocID}}</span>
      </div>
      {{end}}
      <div style="display:flex;justify-content:space-between;padding:8px 0;font-size:13px;">
        <span style="color:#875568;">Confirmed At</span><span style="color:#f0e6e8;font-weight:500;">{{.ConfirmedAt}}</span>
      </div>
    </div>

    <p style="margin:0 0 24px;font-size:13px;color:#875568;line-height:1.6;padding:14px;background:#160909;border-left:3px solid #e11d4844;">
      Please keep this email as proof of your registration. Further details about
      the event venue, schedule, and requirements will be shared closer to the event date.
    </p>

    <!-- CTA -->
    <div style="text-align:center;margin:28px 0;">
      <a href="https://link.cryptx.lk/whatsapp"
         style="display:inline-block;background-color:#e11d48;color:#ffffff;text-decoration:none;padding:12px 28px;font-weight:bold;font-size:14px;letter-spacing:0.04em;">
        Join WhatsApp Channel
      </a>
    </div>
  </div>

  <!-- Footer -->
  <div style="padding:20px 32px;border-top:1px solid #3b1520;text-align:center;">
    <p style="margin:0;font-size:13px;color:#c09ca0;line-height:1.7;">Best regards,<br/><strong style="color:#f0e6e8;">The CryptX Team</strong></p>
    <p style="margin:10px 0 0;font-size:11px;color:#4a3036;">© 2026 ICTS — University of Sri Jayewardenepura</p>
  </div>
</div>`

	html := renderSimpleTemplate(tmplStr, data)
	return html,
		"CryptX Registration <registrations@cryptx.lk>",
		"[CryptX 2.0] Your CTF Registration is Confirmed!"
}

// ── School Hackathon — dark green theme ───────────────────────────────────────

func buildHackathonConfirmEmail(data ConfirmationData) (string, string, string) {
	const tmplStr = `
<div style="font-family:sans-serif;max-width:600px;margin:0 auto;background-color:#001a0e;color:#e2e8f0;padding:0;border:1px solid #00542e44;">
  <!-- Header -->
  <div style="background:linear-gradient(135deg,#00120a 0%,#001f10 100%);padding:32px 32px 24px;border-bottom:1px solid #00542e33;">
    <h1 style="margin:0 0 4px;font-size:26px;letter-spacing:0.08em;color:#00c46a;font-weight:800;text-transform:uppercase;">CryptX 2.0</h1>
    <p style="margin:0;font-size:12px;color:#94a3b8;letter-spacing:0.1em;text-transform:uppercase;">School Hackathon</p>
    <div style="display:inline-block;margin-top:14px;padding:5px 16px;background:#00542e22;border:1px solid #00542e55;border-radius:20px;font-size:11px;color:#00c46a;letter-spacing:0.14em;text-transform:uppercase;">✓ Registration Confirmed</div>
  </div>

  <div style="padding:32px;">
    <p style="margin:0 0 12px;font-size:17px;font-weight:600;color:#e2e8f0;">Hi {{.RecipientName}},</p>
    <p style="margin:0 0 24px;font-size:14px;color:#94a3b8;line-height:1.7;">
      Your registration for <strong style="color:#e2e8f0;">CryptX 2.0 — School Hackathon</strong>
      has been reviewed and your payment has been <strong style="color:#00c46a;">confirmed</strong>.
      You're officially in — get ready to hack!
    </p>

    <div style="background:#001208;border:1px solid #00542e44;border-radius:8px;padding:20px 24px;margin-bottom:24px;">
      <p style="margin:0 0 14px;font-size:11px;text-transform:uppercase;letter-spacing:0.1em;color:#64748b;">Registration Details</p>
      {{if .TeamName}}
      <div style="display:flex;justify-content:space-between;padding:8px 0;border-bottom:1px solid #00542e33;font-size:13px;">
        <span style="color:#4a7a5e;">Team Name</span><span style="color:#e2e8f0;font-weight:500;">{{.TeamName}}</span>
      </div>
      {{end}}
      <div style="display:flex;justify-content:space-between;padding:8px 0;border-bottom:1px solid #00542e33;font-size:13px;">
        <span style="color:#4a7a5e;">Leader</span><span style="color:#e2e8f0;font-weight:500;">{{.RecipientName}}</span>
      </div>
      {{if .DocID}}
      <div style="display:flex;justify-content:space-between;padding:8px 0;border-bottom:1px solid #00542e33;font-size:13px;">
        <span style="color:#4a7a5e;">Reference ID</span><span style="color:#e2e8f0;font-family:monospace;font-size:11px;">{{.DocID}}</span>
      </div>
      {{end}}
      <div style="display:flex;justify-content:space-between;padding:8px 0;font-size:13px;">
        <span style="color:#4a7a5e;">Confirmed At</span><span style="color:#e2e8f0;font-weight:500;">{{.ConfirmedAt}}</span>
      </div>
    </div>

    <div style="text-align:center;margin:28px 0;">
      <a href="https://link.cryptx.lk/whatsapp"
         style="display:inline-block;background-color:#00542e;color:#ffffff;text-decoration:none;padding:12px 28px;font-weight:bold;font-size:14px;">
        Join WhatsApp Channel
      </a>
    </div>
  </div>

  <div style="padding:20px 32px;border-top:1px solid #00542e44;text-align:center;">
    <p style="margin:0;font-size:13px;color:#94a3b8;line-height:1.7;">Best regards,<br/><strong style="color:#e2e8f0;">The CryptX 2.0 Organizing Committee</strong></p>
    <p style="margin:10px 0 0;font-size:11px;color:#2a4a36;">© 2026 ICTS — University of Sri Jayewardenepura</p>
  </div>
</div>`

	html := renderSimpleTemplate(tmplStr, data)
	return html,
		"CryptX Registration <registrations@cryptx.lk>",
		"[CryptX 2.0] Your School Hackathon Registration is Confirmed!"
}

// ── University Hackathon — dark green theme ───────────────────────────────────

func buildUniversityHackathonConfirmEmail(data ConfirmationData) (string, string, string) {
	// Same colour palette as school hackathon; different wording.
	const tmplStr = `
<div style="font-family:sans-serif;max-width:600px;margin:0 auto;background-color:#001a0e;color:#e2e8f0;padding:0;border:1px solid #00542e44;">
  <div style="background:linear-gradient(135deg,#00120a 0%,#001f10 100%);padding:32px 32px 24px;border-bottom:1px solid #00542e33;">
    <h1 style="margin:0 0 4px;font-size:26px;letter-spacing:0.08em;color:#00c46a;font-weight:800;text-transform:uppercase;">CryptX 2.0</h1>
    <p style="margin:0;font-size:12px;color:#94a3b8;letter-spacing:0.1em;text-transform:uppercase;">University Hackathon</p>
    <div style="display:inline-block;margin-top:14px;padding:5px 16px;background:#00542e22;border:1px solid #00542e55;border-radius:20px;font-size:11px;color:#00c46a;letter-spacing:0.14em;text-transform:uppercase;">✓ Registration Confirmed</div>
  </div>

  <div style="padding:32px;">
    <p style="margin:0 0 12px;font-size:17px;font-weight:600;color:#e2e8f0;">Hi {{.RecipientName}},</p>
    <p style="margin:0 0 24px;font-size:14px;color:#94a3b8;line-height:1.7;">
      Your registration for <strong style="color:#e2e8f0;">CryptX 2.0 — University Hackathon</strong>
      has been reviewed and your payment has been <strong style="color:#00c46a;">confirmed</strong>.
      You're in — start preparing!
    </p>

    <div style="background:#001208;border:1px solid #00542e44;border-radius:8px;padding:20px 24px;margin-bottom:24px;">
      <p style="margin:0 0 14px;font-size:11px;text-transform:uppercase;letter-spacing:0.1em;color:#64748b;">Registration Details</p>
      {{if .TeamName}}
      <div style="display:flex;justify-content:space-between;padding:8px 0;border-bottom:1px solid #00542e33;font-size:13px;">
        <span style="color:#4a7a5e;">Team Name</span><span style="color:#e2e8f0;font-weight:500;">{{.TeamName}}</span>
      </div>
      {{end}}
      <div style="display:flex;justify-content:space-between;padding:8px 0;border-bottom:1px solid #00542e33;font-size:13px;">
        <span style="color:#4a7a5e;">Leader</span><span style="color:#e2e8f0;font-weight:500;">{{.RecipientName}}</span>
      </div>
      {{if .DocID}}
      <div style="display:flex;justify-content:space-between;padding:8px 0;border-bottom:1px solid #00542e33;font-size:13px;">
        <span style="color:#4a7a5e;">Reference ID</span><span style="color:#e2e8f0;font-family:monospace;font-size:11px;">{{.DocID}}</span>
      </div>
      {{end}}
      <div style="display:flex;justify-content:space-between;padding:8px 0;font-size:13px;">
        <span style="color:#4a7a5e;">Confirmed At</span><span style="color:#e2e8f0;font-weight:500;">{{.ConfirmedAt}}</span>
      </div>
    </div>

    <div style="text-align:center;margin:28px 0;">
      <a href="https://link.cryptx.lk/whatsapp"
         style="display:inline-block;background-color:#00542e;color:#ffffff;text-decoration:none;padding:12px 28px;font-weight:bold;font-size:14px;">
        Join WhatsApp Channel
      </a>
    </div>
  </div>

  <div style="padding:20px 32px;border-top:1px solid #00542e44;text-align:center;">
    <p style="margin:0;font-size:13px;color:#94a3b8;line-height:1.7;">Best regards,<br/><strong style="color:#e2e8f0;">The CryptX 2.0 Organizing Committee</strong></p>
    <p style="margin:10px 0 0;font-size:11px;color:#2a4a36;">© 2026 ICTS — University of Sri Jayewardenepura</p>
  </div>
</div>`

	html := renderSimpleTemplate(tmplStr, data)
	return html,
		"CryptX Registration <registrations@cryptx.lk>",
		"[CryptX 2.0] Your University Hackathon Registration is Confirmed!"
}

// ── Designathon — deep navy/blue theme ───────────────────────────────────────

func buildDesignathonConfirmEmail(data ConfirmationData) (string, string, string) {
	const tmplStr = `
<div style="font-family:sans-serif;max-width:600px;margin:0 auto;background-color:#001223;color:#e2e8f0;padding:0;border:1px solid #004e9c44;">
  <div style="background:linear-gradient(135deg,#000d1a 0%,#001a33 100%);padding:32px 32px 24px;border-bottom:1px solid #004e9c33;">
    <h1 style="margin:0 0 4px;font-size:26px;letter-spacing:0.08em;color:#3b9eff;font-weight:800;text-transform:uppercase;">CryptX 2.0</h1>
    <p style="margin:0;font-size:12px;color:#94a3b8;letter-spacing:0.1em;text-transform:uppercase;">Designathon</p>
    <div style="display:inline-block;margin-top:14px;padding:5px 16px;background:#004e9c22;border:1px solid #004e9c55;border-radius:20px;font-size:11px;color:#3b9eff;letter-spacing:0.14em;text-transform:uppercase;">✓ Registration Confirmed</div>
  </div>

  <div style="padding:32px;">
    <p style="margin:0 0 12px;font-size:17px;font-weight:600;color:#e2e8f0;">Hi {{.RecipientName}},</p>
    <p style="margin:0 0 24px;font-size:14px;color:#94a3b8;line-height:1.7;">
      Your registration for <strong style="color:#e2e8f0;">CryptX 2.0 — Designathon</strong>
      has been reviewed and your payment has been <strong style="color:#3b9eff;">confirmed</strong>.
      You're officially in — time to design something extraordinary!
    </p>

    <div style="background:#000e1d;border:1px solid #004e9c44;border-radius:8px;padding:20px 24px;margin-bottom:24px;">
      <p style="margin:0 0 14px;font-size:11px;text-transform:uppercase;letter-spacing:0.1em;color:#64748b;">Registration Details</p>
      {{if .TeamName}}
      <div style="display:flex;justify-content:space-between;padding:8px 0;border-bottom:1px solid #004e9c33;font-size:13px;">
        <span style="color:#3a6a9c;">Team Name</span><span style="color:#e2e8f0;font-weight:500;">{{.TeamName}}</span>
      </div>
      {{end}}
      <div style="display:flex;justify-content:space-between;padding:8px 0;border-bottom:1px solid #004e9c33;font-size:13px;">
        <span style="color:#3a6a9c;">Leader</span><span style="color:#e2e8f0;font-weight:500;">{{.RecipientName}}</span>
      </div>
      {{if .DocID}}
      <div style="display:flex;justify-content:space-between;padding:8px 0;border-bottom:1px solid #004e9c33;font-size:13px;">
        <span style="color:#3a6a9c;">Reference ID</span><span style="color:#e2e8f0;font-family:monospace;font-size:11px;">{{.DocID}}</span>
      </div>
      {{end}}
      <div style="display:flex;justify-content:space-between;padding:8px 0;font-size:13px;">
        <span style="color:#3a6a9c;">Confirmed At</span><span style="color:#e2e8f0;font-weight:500;">{{.ConfirmedAt}}</span>
      </div>
    </div>

    <div style="text-align:center;margin:28px 0;">
      <a href="https://link.cryptx.lk/whatsapp"
         style="display:inline-block;background-color:#004e9c;color:#ffffff;text-decoration:none;padding:12px 28px;font-weight:bold;font-size:14px;">
        Join WhatsApp Channel
      </a>
    </div>
  </div>

  <div style="padding:20px 32px;border-top:1px solid #004e9c44;text-align:center;">
    <p style="margin:0;font-size:13px;color:#94a3b8;line-height:1.7;">Best regards,<br/><strong style="color:#e2e8f0;">The CryptX 2.0 Organizing Committee</strong></p>
    <p style="margin:10px 0 0;font-size:11px;color:#1a3a5c;">© 2026 ICTS — University of Sri Jayewardenepura</p>
  </div>
</div>`

	html := renderSimpleTemplate(tmplStr, data)
	return html,
		"CryptX Registration <registrations@cryptx.lk>",
		"[CryptX 2.0] Your Designathon Registration is Confirmed!"
}

// ── Generic fallback — cyan/dark theme ───────────────────────────────────────

func buildGenericConfirmEmail(data ConfirmationData) (string, string, string) {
	const tmplStr = `
<div style="font-family:sans-serif;max-width:600px;margin:0 auto;background-color:#0a0a0f;color:#e2e8f0;padding:0;border:1px solid #1e2033;">
  <div style="background:linear-gradient(135deg,#0f172a 0%,#1a1a2e 100%);padding:32px 32px 24px;border-bottom:1px solid #22d3ee33;">
    <h1 style="margin:0 0 4px;font-size:26px;letter-spacing:0.08em;color:#22d3ee;font-weight:800;text-transform:uppercase;">CryptX 2.0</h1>
    <p style="margin:0;font-size:12px;color:#94a3b8;letter-spacing:0.1em;text-transform:uppercase;">{{.EventName}}</p>
    <div style="display:inline-block;margin-top:14px;padding:5px 16px;background:#22d3ee22;border:1px solid #22d3ee55;border-radius:20px;font-size:11px;color:#22d3ee;letter-spacing:0.14em;text-transform:uppercase;">✓ Registration Confirmed</div>
  </div>

  <div style="padding:32px;">
    <p style="margin:0 0 12px;font-size:17px;font-weight:600;color:#f1f5f9;">Hi {{.RecipientName}},</p>
    <p style="margin:0 0 24px;font-size:14px;color:#94a3b8;line-height:1.7;">
      Your registration for <strong style="color:#f1f5f9;">CryptX 2.0 — {{.EventName}}</strong>
      has been reviewed and your payment has been confirmed. You're officially in!
    </p>

    <div style="background:#0f0f1a;border:1px solid #1e2033;border-radius:8px;padding:20px 24px;margin-bottom:24px;">
      <p style="margin:0 0 14px;font-size:11px;text-transform:uppercase;letter-spacing:0.1em;color:#64748b;">Registration Details</p>
      {{if .TeamName}}
      <div style="display:flex;justify-content:space-between;padding:8px 0;border-bottom:1px solid #1e203366;font-size:13px;">
        <span style="color:#64748b;">Team Name</span><span style="color:#e2e8f0;font-weight:500;">{{.TeamName}}</span>
      </div>
      {{end}}
      <div style="display:flex;justify-content:space-between;padding:8px 0;border-bottom:1px solid #1e203366;font-size:13px;">
        <span style="color:#64748b;">Name</span><span style="color:#e2e8f0;font-weight:500;">{{.RecipientName}}</span>
      </div>
      {{if .DocID}}
      <div style="display:flex;justify-content:space-between;padding:8px 0;border-bottom:1px solid #1e203366;font-size:13px;">
        <span style="color:#64748b;">Reference ID</span><span style="color:#e2e8f0;font-family:monospace;font-size:11px;">{{.DocID}}</span>
      </div>
      {{end}}
      <div style="display:flex;justify-content:space-between;padding:8px 0;font-size:13px;">
        <span style="color:#64748b;">Confirmed At</span><span style="color:#e2e8f0;font-weight:500;">{{.ConfirmedAt}}</span>
      </div>
    </div>

    <div style="text-align:center;margin:28px 0;">
      <a href="https://link.cryptx.lk/whatsapp"
         style="display:inline-block;background:linear-gradient(135deg,#22d3ee,#0ea5e9);color:#0a0a0f;text-decoration:none;padding:12px 28px;font-weight:700;font-size:14px;border-radius:6px;">
        Join WhatsApp Channel
      </a>
    </div>
  </div>

  <div style="padding:20px 32px;border-top:1px solid #1e2033;text-align:center;">
    <p style="margin:0;font-size:13px;color:#94a3b8;line-height:1.7;">Best regards,<br/><strong style="color:#e2e8f0;">The CryptX Team</strong></p>
    <p style="margin:10px 0 0;font-size:11px;color:#334155;">© 2026 ICTS — University of Sri Jayewardenepura</p>
  </div>
</div>`

	html := renderSimpleTemplate(tmplStr, data)
	return html,
		"CryptX Registration <registrations@cryptx.lk>",
		fmt.Sprintf("[CryptX 2.0] Your %s Registration is Confirmed!", data.EventName)
}

// renderSimpleTemplate executes a Go html/template string with the given data.
func renderSimpleTemplate(tmplStr string, data ConfirmationData) string {
	t, err := template.New("email").Parse(tmplStr)
	if err != nil {
		return "<p>Email render error: " + err.Error() + "</p>"
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "<p>Email render error: " + err.Error() + "</p>"
	}
	return buf.String()
}

// ── Merch Store emails ─────────────────────────────────────────────────────────

// MerchEmailData holds template variables for all merch-related emails.
type MerchEmailData struct {
	RecipientName  string
	RecipientEmail string
	ProductName    string
	Size           string
	Quantity       int
	TotalPrice     int
	PaymentOption  string // "pre-order" | "full"
	DeliveryMethod string // "Event Day Collection" | "Courier"
	DocID          string
	ConfirmedAt    string
	// Dispatch-specific — Event Day Collection
	EventDate string
	EventTime string
	Venue     string
	// Dispatch-specific — Courier
	TrackingNumber string
}

// SendMerchPreOrderConfirm sends a pre-order payment received email.
func SendMerchPreOrderConfirm(cfg *config.Config, data MerchEmailData) error {
	if data.ConfirmedAt == "" {
		data.ConfirmedAt = time.Now().Format("02 Jan 2006, 15:04 MST")
	}
	html, from, subject := buildMerchPreOrderEmail(data)
	if cfg.ResendAPIKey != "" {
		return sendViaResend(cfg.ResendAPIKey, resendPayload{
			From:    from,
			To:      []string{data.RecipientEmail},
			Subject: subject,
			HTML:    html,
		})
	}
	return sendViaSMTP(cfg, data.RecipientEmail, subject, html)
}

// SendMerchFullPaymentConfirm sends a full payment confirmed email.
func SendMerchFullPaymentConfirm(cfg *config.Config, data MerchEmailData) error {
	if data.ConfirmedAt == "" {
		data.ConfirmedAt = time.Now().Format("02 Jan 2006, 15:04 MST")
	}
	html, from, subject := buildMerchFullPaymentEmail(data)
	if cfg.ResendAPIKey != "" {
		return sendViaResend(cfg.ResendAPIKey, resendPayload{
			From:    from,
			To:      []string{data.RecipientEmail},
			Subject: subject,
			HTML:    html,
		})
	}
	return sendViaSMTP(cfg, data.RecipientEmail, subject, html)
}

// SendMerchDispatch sends a dispatch notification email.
// The template adapts based on DeliveryMethod: collection details for
// "Event Day Collection", tracking info for "Courier".
func SendMerchDispatch(cfg *config.Config, data MerchEmailData) error {
	if data.ConfirmedAt == "" {
		data.ConfirmedAt = time.Now().Format("02 Jan 2006, 15:04 MST")
	}
	html, from, subject := buildMerchDispatchEmail(data)
	if cfg.ResendAPIKey != "" {
		return sendViaResend(cfg.ResendAPIKey, resendPayload{
			From:    from,
			To:      []string{data.RecipientEmail},
			Subject: subject,
			HTML:    html,
		})
	}
	return sendViaSMTP(cfg, data.RecipientEmail, subject, html)
}

// ── Merch email builders — amber/gold theme ────────────────────────────────────

func buildMerchPreOrderEmail(data MerchEmailData) (string, string, string) {
	const tmplStr = `
<div style="font-family:sans-serif;max-width:600px;margin:0 auto;background-color:#0d0a02;color:#f0e8cc;padding:0;border:1px solid #4a3800;">
  <!-- Header -->
  <div style="background:linear-gradient(135deg,#1a1200 0%,#2a1e00 100%);padding:32px 32px 24px;border-bottom:1px solid #f59e0b33;">
    <h1 style="margin:0 0 4px;font-size:26px;letter-spacing:0.08em;color:#f59e0b;font-weight:800;text-transform:uppercase;">CryptX 2.0</h1>
    <p style="margin:0;font-size:12px;color:#94a3b8;letter-spacing:0.1em;text-transform:uppercase;">Merch Store</p>
    <div style="display:inline-block;margin-top:14px;padding:5px 16px;background:#f59e0b22;border:1px solid #f59e0b55;border-radius:20px;font-size:11px;color:#f59e0b;letter-spacing:0.14em;text-transform:uppercase;">✓ Pre-Order Payment Received</div>
  </div>

  <!-- Body -->
  <div style="padding:32px;">
    <p style="margin:0 0 12px;font-size:17px;font-weight:600;color:#fef3c7;">Hi {{.RecipientName}},</p>
    <p style="margin:0 0 24px;font-size:14px;color:#c09c4a;line-height:1.7;">
      We've received your <strong style="color:#fef3c7;">pre-order payment</strong> for your CryptX 2.0 merch!
      We'll verify it shortly. Once your full payment is received and confirmed, we'll process your order.
    </p>

    <!-- Order card -->
    <div style="background:#1a1200;border:1px solid #4a3800;border-radius:8px;padding:20px 24px;margin-bottom:24px;">
      <p style="margin:0 0 14px;font-size:11px;text-transform:uppercase;letter-spacing:0.1em;color:#64748b;">Order Details</p>
      <div style="display:flex;justify-content:space-between;padding:8px 0;border-bottom:1px solid #4a380066;font-size:13px;">
        <span style="color:#8a7040;">Product</span><span style="color:#f0e8cc;font-weight:500;">{{.ProductName}}</span>
      </div>
      <div style="display:flex;justify-content:space-between;padding:8px 0;border-bottom:1px solid #4a380066;font-size:13px;">
        <span style="color:#8a7040;">Size</span><span style="color:#f0e8cc;font-weight:500;">{{.Size}}</span>
      </div>
      <div style="display:flex;justify-content:space-between;padding:8px 0;border-bottom:1px solid #4a380066;font-size:13px;">
        <span style="color:#8a7040;">Quantity</span><span style="color:#f0e8cc;font-weight:500;">{{.Quantity}}</span>
      </div>
      <div style="display:flex;justify-content:space-between;padding:8px 0;border-bottom:1px solid #4a380066;font-size:13px;">
        <span style="color:#8a7040;">Total Price</span><span style="color:#f0e8cc;font-weight:500;">LKR {{.TotalPrice}}</span>
      </div>
      <div style="display:flex;justify-content:space-between;padding:8px 0;border-bottom:1px solid #4a380066;font-size:13px;">
        <span style="color:#8a7040;">Delivery</span><span style="color:#f0e8cc;font-weight:500;">{{.DeliveryMethod}}</span>
      </div>
      {{if .DocID}}
      <div style="display:flex;justify-content:space-between;padding:8px 0;border-bottom:1px solid #4a380066;font-size:13px;">
        <span style="color:#8a7040;">Reference ID</span><span style="color:#f0e8cc;font-family:monospace;font-size:11px;">{{.DocID}}</span>
      </div>
      {{end}}
      <div style="display:flex;justify-content:space-between;padding:8px 0;font-size:13px;">
        <span style="color:#8a7040;">Pre-order Confirmed At</span><span style="color:#f0e8cc;font-weight:500;">{{.ConfirmedAt}}</span>
      </div>
    </div>

    <p style="margin:0 0 24px;font-size:13px;color:#8a7040;line-height:1.6;padding:14px;background:#120e00;border-left:3px solid #f59e0b44;">
      <strong style="color:#fef3c7;">Next step:</strong> Please submit your full payment slip via the merch store.
      Your order will be processed only after the full payment is confirmed.
    </p>
  </div>

  <!-- Footer -->
  <div style="padding:20px 32px;border-top:1px solid #4a3800;text-align:center;">
    <p style="margin:0;font-size:13px;color:#c09c4a;line-height:1.7;">Best regards,<br/><strong style="color:#f0e8cc;">The CryptX 2.0 Merch Team</strong></p>
    <p style="margin:10px 0 0;font-size:11px;color:#3a2c00;">© 2026 ICTS — University of Sri Jayewardenepura</p>
  </div>
</div>`

	t, _ := template.New("merch-preorder").Parse(tmplStr)
	var buf bytes.Buffer
	_ = t.Execute(&buf, data)
	return buf.String(),
		"CryptX Merch <merch@cryptx.lk>",
		"[CryptX 2.0] Pre-Order Payment Received — " + data.ProductName
}

func buildMerchFullPaymentEmail(data MerchEmailData) (string, string, string) {
	const tmplStr = `
<div style="font-family:sans-serif;max-width:600px;margin:0 auto;background-color:#0d0a02;color:#f0e8cc;padding:0;border:1px solid #4a3800;">
  <!-- Header -->
  <div style="background:linear-gradient(135deg,#1a1200 0%,#2a1e00 100%);padding:32px 32px 24px;border-bottom:1px solid #f59e0b33;">
    <h1 style="margin:0 0 4px;font-size:26px;letter-spacing:0.08em;color:#f59e0b;font-weight:800;text-transform:uppercase;">CryptX 2.0</h1>
    <p style="margin:0;font-size:12px;color:#94a3b8;letter-spacing:0.1em;text-transform:uppercase;">Merch Store</p>
    <div style="display:inline-block;margin-top:14px;padding:5px 16px;background:#22c55e22;border:1px solid #22c55e55;border-radius:20px;font-size:11px;color:#22c55e;letter-spacing:0.14em;text-transform:uppercase;">✓ Full Payment Confirmed</div>
  </div>

  <!-- Body -->
  <div style="padding:32px;">
    <p style="margin:0 0 12px;font-size:17px;font-weight:600;color:#fef3c7;">Hi {{.RecipientName}},</p>
    <p style="margin:0 0 24px;font-size:14px;color:#c09c4a;line-height:1.7;">
      Great news! Your <strong style="color:#fef3c7;">full payment</strong> for your CryptX 2.0 merch has been
      <strong style="color:#22c55e;">confirmed</strong>. Your order is now being prepared!
    </p>

    <!-- Order card -->
    <div style="background:#1a1200;border:1px solid #4a3800;border-radius:8px;padding:20px 24px;margin-bottom:24px;">
      <p style="margin:0 0 14px;font-size:11px;text-transform:uppercase;letter-spacing:0.1em;color:#64748b;">Order Details</p>
      <div style="display:flex;justify-content:space-between;padding:8px 0;border-bottom:1px solid #4a380066;font-size:13px;">
        <span style="color:#8a7040;">Product</span><span style="color:#f0e8cc;font-weight:500;">{{.ProductName}}</span>
      </div>
      <div style="display:flex;justify-content:space-between;padding:8px 0;border-bottom:1px solid #4a380066;font-size:13px;">
        <span style="color:#8a7040;">Size</span><span style="color:#f0e8cc;font-weight:500;">{{.Size}}</span>
      </div>
      <div style="display:flex;justify-content:space-between;padding:8px 0;border-bottom:1px solid #4a380066;font-size:13px;">
        <span style="color:#8a7040;">Quantity</span><span style="color:#f0e8cc;font-weight:500;">{{.Quantity}}</span>
      </div>
      <div style="display:flex;justify-content:space-between;padding:8px 0;border-bottom:1px solid #4a380066;font-size:13px;">
        <span style="color:#8a7040;">Total Price</span><span style="color:#f0e8cc;font-weight:500;">LKR {{.TotalPrice}}</span>
      </div>
      <div style="display:flex;justify-content:space-between;padding:8px 0;border-bottom:1px solid #4a380066;font-size:13px;">
        <span style="color:#8a7040;">Delivery Method</span><span style="color:#f0e8cc;font-weight:500;">{{.DeliveryMethod}}</span>
      </div>
      {{if .DocID}}
      <div style="display:flex;justify-content:space-between;padding:8px 0;border-bottom:1px solid #4a380066;font-size:13px;">
        <span style="color:#8a7040;">Reference ID</span><span style="color:#f0e8cc;font-family:monospace;font-size:11px;">{{.DocID}}</span>
      </div>
      {{end}}
      <div style="display:flex;justify-content:space-between;padding:8px 0;font-size:13px;">
        <span style="color:#8a7040;">Confirmed At</span><span style="color:#f0e8cc;font-weight:500;">{{.ConfirmedAt}}</span>
      </div>
    </div>

    <p style="margin:0 0 24px;font-size:13px;color:#8a7040;line-height:1.6;padding:14px;background:#120e00;border-left:3px solid #f59e0b44;">
      We will notify you once your order has been dispatched or is ready for collection.
      Keep this email as proof of your purchase.
    </p>
  </div>

  <!-- Footer -->
  <div style="padding:20px 32px;border-top:1px solid #4a3800;text-align:center;">
    <p style="margin:0;font-size:13px;color:#c09c4a;line-height:1.7;">Best regards,<br/><strong style="color:#f0e8cc;">The CryptX 2.0 Merch Team</strong></p>
    <p style="margin:10px 0 0;font-size:11px;color:#3a2c00;">© 2026 ICTS — University of Sri Jayewardenepura</p>
  </div>
</div>`

	t, _ := template.New("merch-fullpay").Parse(tmplStr)
	var buf bytes.Buffer
	_ = t.Execute(&buf, data)
	return buf.String(),
		"CryptX Merch <merch@cryptx.lk>",
		"[CryptX 2.0] Full Payment Confirmed — Your Order is Being Prepared!"
}

func buildMerchDispatchEmail(data MerchEmailData) (string, string, string) {
	// Two templates — one per delivery method.
	collectionTmpl := `
<div style="font-family:sans-serif;max-width:600px;margin:0 auto;background-color:#0d0a02;color:#f0e8cc;padding:0;border:1px solid #4a3800;">
  <div style="background:linear-gradient(135deg,#1a1200 0%,#2a1e00 100%);padding:32px 32px 24px;border-bottom:1px solid #f59e0b33;">
    <h1 style="margin:0 0 4px;font-size:26px;letter-spacing:0.08em;color:#f59e0b;font-weight:800;text-transform:uppercase;">CryptX 2.0</h1>
    <p style="margin:0;font-size:12px;color:#94a3b8;letter-spacing:0.1em;text-transform:uppercase;">Merch Store — Collection Details</p>
    <div style="display:inline-block;margin-top:14px;padding:5px 16px;background:#f59e0b22;border:1px solid #f59e0b55;border-radius:20px;font-size:11px;color:#f59e0b;letter-spacing:0.14em;text-transform:uppercase;">📦 Ready for Collection</div>
  </div>

  <div style="padding:32px;">
    <p style="margin:0 0 12px;font-size:17px;font-weight:600;color:#fef3c7;">Hi {{.RecipientName}},</p>
    <p style="margin:0 0 24px;font-size:14px;color:#c09c4a;line-height:1.7;">
      Your CryptX 2.0 merch order is ready for collection at the event!
      Please bring this email or your order reference ID on the day.
    </p>

    <div style="background:#1a1200;border:1px solid #4a3800;border-radius:8px;padding:20px 24px;margin-bottom:24px;">
      <p style="margin:0 0 14px;font-size:11px;text-transform:uppercase;letter-spacing:0.1em;color:#64748b;">Collection Details</p>
      <div style="display:flex;justify-content:space-between;padding:8px 0;border-bottom:1px solid #4a380066;font-size:13px;">
        <span style="color:#8a7040;">Date</span><span style="color:#f59e0b;font-weight:600;">{{.EventDate}}</span>
      </div>
      <div style="display:flex;justify-content:space-between;padding:8px 0;border-bottom:1px solid #4a380066;font-size:13px;">
        <span style="color:#8a7040;">Time</span><span style="color:#f59e0b;font-weight:600;">{{.EventTime}}</span>
      </div>
      <div style="display:flex;justify-content:space-between;padding:8px 0;border-bottom:1px solid #4a380066;font-size:13px;">
        <span style="color:#8a7040;">Venue</span><span style="color:#f0e8cc;font-weight:500;">{{.Venue}}</span>
      </div>
      <div style="display:flex;justify-content:space-between;padding:8px 0;border-bottom:1px solid #4a380066;font-size:13px;">
        <span style="color:#8a7040;">Product</span><span style="color:#f0e8cc;font-weight:500;">{{.ProductName}} ({{.Size}} × {{.Quantity}})</span>
      </div>
      {{if .DocID}}
      <div style="display:flex;justify-content:space-between;padding:8px 0;font-size:13px;">
        <span style="color:#8a7040;">Reference ID</span><span style="color:#f0e8cc;font-family:monospace;font-size:11px;">{{.DocID}}</span>
      </div>
      {{end}}
    </div>

    <p style="margin:0;font-size:13px;color:#8a7040;line-height:1.6;padding:14px;background:#120e00;border-left:3px solid #f59e0b44;">
      Please be on time. If you are unable to collect on the day, contact us in advance.
    </p>
  </div>

  <div style="padding:20px 32px;border-top:1px solid #4a3800;text-align:center;">
    <p style="margin:0;font-size:13px;color:#c09c4a;line-height:1.7;">See you at the event!<br/><strong style="color:#f0e8cc;">The CryptX 2.0 Merch Team</strong></p>
    <p style="margin:10px 0 0;font-size:11px;color:#3a2c00;">© 2026 ICTS — University of Sri Jayewardenepura</p>
  </div>
</div>`

	courierTmpl := `
<div style="font-family:sans-serif;max-width:600px;margin:0 auto;background-color:#0d0a02;color:#f0e8cc;padding:0;border:1px solid #4a3800;">
  <div style="background:linear-gradient(135deg,#1a1200 0%,#2a1e00 100%);padding:32px 32px 24px;border-bottom:1px solid #f59e0b33;">
    <h1 style="margin:0 0 4px;font-size:26px;letter-spacing:0.08em;color:#f59e0b;font-weight:800;text-transform:uppercase;">CryptX 2.0</h1>
    <p style="margin:0;font-size:12px;color:#94a3b8;letter-spacing:0.1em;text-transform:uppercase;">Merch Store — Shipped!</p>
    <div style="display:inline-block;margin-top:14px;padding:5px 16px;background:#3b82f622;border:1px solid #3b82f655;border-radius:20px;font-size:11px;color:#60a5fa;letter-spacing:0.14em;text-transform:uppercase;">🚚 Order Dispatched</div>
  </div>

  <div style="padding:32px;">
    <p style="margin:0 0 12px;font-size:17px;font-weight:600;color:#fef3c7;">Hi {{.RecipientName}},</p>
    <p style="margin:0 0 24px;font-size:14px;color:#c09c4a;line-height:1.7;">
      Your CryptX 2.0 merch order has been <strong style="color:#60a5fa;">dispatched</strong>!
      It's on its way to you. Use the tracking number below to follow your delivery.
    </p>

    <div style="background:#1a1200;border:1px solid #4a3800;border-radius:8px;padding:20px 24px;margin-bottom:24px;">
      <p style="margin:0 0 14px;font-size:11px;text-transform:uppercase;letter-spacing:0.1em;color:#64748b;">Shipment Details</p>
      <div style="display:flex;justify-content:space-between;padding:8px 0;border-bottom:1px solid #4a380066;font-size:13px;">
        <span style="color:#8a7040;">Product</span><span style="color:#f0e8cc;font-weight:500;">{{.ProductName}} ({{.Size}} × {{.Quantity}})</span>
      </div>
      {{if .TrackingNumber}}
      <div style="display:flex;justify-content:space-between;padding:8px 0;border-bottom:1px solid #4a380066;font-size:13px;">
        <span style="color:#8a7040;">Tracking #</span><span style="color:#60a5fa;font-family:monospace;font-weight:600;">{{.TrackingNumber}}</span>
      </div>
      {{end}}
      <div style="display:flex;justify-content:space-between;padding:8px 0;border-bottom:1px solid #4a380066;font-size:13px;">
        <span style="color:#8a7040;">Dispatched On</span><span style="color:#f0e8cc;font-weight:500;">{{.ConfirmedAt}}</span>
      </div>
      {{if .DocID}}
      <div style="display:flex;justify-content:space-between;padding:8px 0;font-size:13px;">
        <span style="color:#8a7040;">Reference ID</span><span style="color:#f0e8cc;font-family:monospace;font-size:11px;">{{.DocID}}</span>
      </div>
      {{end}}
    </div>

    <p style="margin:0;font-size:13px;color:#8a7040;line-height:1.6;padding:14px;background:#120e00;border-left:3px solid #60a5fa44;">
      If you have any issues with your delivery, please contact us with your reference ID.
    </p>
  </div>

  <div style="padding:20px 32px;border-top:1px solid #4a3800;text-align:center;">
    <p style="margin:0;font-size:13px;color:#c09c4a;line-height:1.7;">Best regards,<br/><strong style="color:#f0e8cc;">The CryptX 2.0 Merch Team</strong></p>
    <p style="margin:10px 0 0;font-size:11px;color:#3a2c00;">© 2026 ICTS — University of Sri Jayewardenepura</p>
  </div>
</div>`

	var chosenTmpl string
	var subject string
	if data.DeliveryMethod == "Event Day Collection" {
		chosenTmpl = collectionTmpl
		subject = "[CryptX 2.0] Your Merch is Ready for Collection!"
	} else {
		chosenTmpl = courierTmpl
		subject = "[CryptX 2.0] Your Merch Order Has Been Dispatched!"
	}

	t, _ := template.New("merch-dispatch").Parse(chosenTmpl)
	var buf bytes.Buffer
	_ = t.Execute(&buf, data)
	return buf.String(),
		"CryptX Merch <merch@cryptx.lk>",
		subject
}

