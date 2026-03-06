# CryptX CLI — Agent Guide

## Purpose

This is a Go TUI (terminal UI) admin tool for managing CryptX 2.0 event registrations stored in Appwrite. It allows the operations team to:

- View paginated lists of CTF, School Hackathon, and Designathon registrations
- Read full registration details including team/member breakdowns
- Confirm payments (updates status in DB, sends confirmation email)
- Delete registrations
- Download payment slips and team logos from Appwrite storage buckets

## Tech Stack

| Layer            | Library                                          |
| ---------------- | ------------------------------------------------ |
| Language         | Go 1.25                                          |
| TUI framework    | `charm.land/bubbletea/v2` v2.0.1                 |
| UI components    | `charm.land/bubbles/v2` v2.0.0                   |
| Styling          | `charm.land/lipgloss/v2` v2.0.0                  |
| Database/Storage | `github.com/appwrite/sdk-for-go` v1.0.0          |
| Config           | `github.com/joho/godotenv` v1.5.1                |
| Email            | Resend HTTP API (fallback: Go stdlib `net/smtp`) |

## Project Layout

```
cryptx-cli/
├── main.go                          entry point
├── .env / .env.example              environment configuration
├── AGENTS.md                        this file
├── .cursor/mcp.json                 Context7 MCP config
├── assets/
│   └── email_template.html          HTML email template (legacy, unused by Resend path)
├── config/
│   └── config.go                    env loading, Config struct
└── internal/
    ├── appwrite/
    │   ├── client.go                Appwrite client factory
    │   ├── auth.go                  email login + OAuth flow
    │   ├── ctf.go                   CTF CRUD + storage download
    │   ├── hackathon.go             School Hackathon CRUD
    │   └── designathon.go           Designathon CRUD + storage download
    ├── email/
    │   └── mailer.go                Resend HTTP API sender + event-specific HTML templates
    ├── models/
    │   ├── ctf.go                   CTF Go structs
    │   ├── hackathon.go             Hackathon Go structs
    │   └── designathon.go           Designathon Go structs
    ├── session/
    │   └── store.go                 ~/.cryptx-cli/session.json management
    └── tui/
        ├── app.go                   root model, screen router, business logic
        ├── compose.go               custom email compose screen (Resend + pop)
        ├── login.go                 login screen
        ├── menu.go                  main menu
        ├── list.go                  paginated registration list
        ├── detail.go                registration detail view
        ├── confirm.go               confirm/delete modal dialogs
        └── styles.go                all Lipgloss styles
```

## Running the Tool

```bash
# First-time setup
cp .env.example .env
# Fill in your Appwrite + SMTP credentials in .env

# Run
go run .

# Build a binary
go build -o cryptx-cli .
./cryptx-cli
```

## Environment Variables

| Variable                                      | Required | Description                                                          |
| --------------------------------------------- | -------- | -------------------------------------------------------------------- |
| `APPWRITE_ENDPOINT`                           | No       | Defaults to `https://cloud.appwrite.io/v1`                           |
| `APPWRITE_PROJECT_ID`                         | **Yes**  | Your Appwrite project ID                                             |
| `APPWRITE_DATABASE_ID`                        | **Yes**  | Appwrite database ID                                                 |
| `APPWRITE_CTF_COLLECTION_ID`                  | No       | Collection ID for CTF registrations                                  |
| `APPWRITE_SCHOOL_HACKATHON_COLLECTION_ID`     | No       | Collection ID for School Hackathon registrations                     |
| `APPWRITE_UNIVERSITY_HACKATHON_COLLECTION_ID` | No       | Collection ID for University Hackathon registrations                 |
| `APPWRITE_DESIGNATHON_COLLECTION_ID`          | No       | Collection ID for Designathon registrations                          |
| `APPWRITE_CTF_BUCKET_ID`                      | No       | Storage bucket for CTF payment slips                                 |
| `APPWRITE_HACKATHON_SCHOOL_BUCKET_ID`         | No       | Storage bucket for School Hackathon team logos                       |
| `APPWRITE_HACKATHON_UNIVERSITY_BUCKET_ID`     | No       | Storage bucket for University Hackathon team logos                   |
| `APPWRITE_DESIGNATHON_BUCKET_ID`              | No       | Storage bucket for Designathon team logos                            |
| `RESEND_API_KEY`                              | **Yes**  | Resend API key — used for all confirmation emails and custom compose |
| `SMTP_HOST`                                   | No       | SMTP host (default: `smtp.gmail.com`) — fallback only                |
| `SMTP_PORT`                                   | No       | SMTP port (default: `587`)                                           |
| `SMTP_USER`                                   | No       | SMTP username                                                        |
| `SMTP_PASS`                                   | No       | SMTP password / app password                                         |
| `SMTP_FROM`                                   | No       | Sender address (default: `noreply@cryptx.lk`)                        |

## Appwrite Collections

### `ctf_registrations`

| Attribute             | Type    | Notes                        |
| --------------------- | ------- | ---------------------------- |
| `registrationType`    | string  | `"Individual"` or `"Team"`   |
| `teamName`            | string  | optional                     |
| `leaderName`          | string  |                              |
| `leaderUniversity`    | string  |                              |
| `leaderContact`       | string  | 10-digit phone               |
| `leaderEmail`         | string  |                              |
| `leaderWhatsapp`      | string  |                              |
| `leaderNIC`           | string  |                              |
| `leaderRegNo`         | string  | optional                     |
| `member2Json`         | string  | JSON-serialised `CTFMember`  |
| `member3Json`         | string  | optional                     |
| `member4Json`         | string  | optional                     |
| `referralSource`      | string  |                              |
| `awarenessPreference` | string  | `"Physical"` or `"Online"`   |
| `agreement`           | boolean |                              |
| `paymentSlipFileId`   | string  | Appwrite storage file ID     |
| `paymentSlipName`     | string  | original filename            |
| `paymentSlipMime`     | string  | MIME type                    |
| `confirmationStatus`  | string  | `"pending"` or `"confirmed"` |
| `confirmedAt`         | string  | ISO 8601 timestamp           |

### `school_hackathon_registrations`

| Attribute                     | Type    | Notes                          |
| ----------------------------- | ------- | ------------------------------ |
| `teamName`                    | string  |                                |
| `teamMemberCount`             | integer | 2–4                            |
| `leaderFullName`              | string  |                                |
| `leaderGrade`                 | string  |                                |
| `leaderContactNumber`         | string  |                                |
| `leaderEmail`                 | string  |                                |
| `leaderNIC`                   | string  | optional                       |
| `leaderSchoolName`            | string  |                                |
| `member2Json`                 | string  | JSON-serialised `SchoolMember` |
| `member3Json`                 | string  | optional                       |
| `member4Json`                 | string  | optional                       |
| `inChargePersonFullName`      | string  |                                |
| `inChargePersonContactNumber` | string  |                                |
| `referralSource`              | string  |                                |
| `agreement`                   | boolean |                                |
| `confirmationStatus`          | string  | `"pending"` or `"confirmed"`   |
| `confirmedAt`                 | string  | ISO 8601                       |

### `designathon_registrations`

| Attribute               | Type    | Notes                               |
| ----------------------- | ------- | ----------------------------------- |
| `teamName`              | string  |                                     |
| `primaryContact`        | string  |                                     |
| `teamLogoFileId`        | string  | Appwrite storage file ID            |
| `teamLogoName`          | string  |                                     |
| `teamLogoMime`          | string  |                                     |
| `member1FullName`       | string  |                                     |
| `member1RegNo`          | string  |                                     |
| `member1NIC`            | string  |                                     |
| `member1Email`          | string  |                                     |
| `member1Phone`          | string  |                                     |
| `member1University`     | string  |                                     |
| `member2Json`           | string  | JSON-serialised `DesignathonMember` |
| `member3Json`           | string  | optional                            |
| `previousParticipation` | string  | `"Yes"` or `"No"`                   |
| `agreement`             | boolean |                                     |
| `confirmationStatus`    | string  | `"pending"` or `"confirmed"`        |
| `confirmedAt`           | string  | ISO 8601                            |

## TUI Keybindings

### Login Screen

| Key               | Action                       |
| ----------------- | ---------------------------- |
| `tab` / `↓`       | Next field                   |
| `shift+tab` / `↑` | Previous field               |
| `enter`           | Submit (or move to password) |
| `ctrl+o`          | Start OAuth login (Google)   |

### Main Menu

| Key               | Action             |
| ----------------- | ------------------ |
| `↑↓` / `jk`       | Navigate items     |
| `enter` / `space` | Select event       |
| `1` / `2` / `3`   | Quick-select event |
| `q`               | Quit               |

### Compose Email (Screen 4)

| Key         | Action                                                 |
| ----------- | ------------------------------------------------------ |
| `tab`       | Next field (To → Subject → Body → Attachment)          |
| `shift+tab` | Previous field                                         |
| `ctrl+a`    | Toggle attachment input panel                          |
| `ctrl+d`    | Remove last attachment (when attachment panel is open) |
| `enter`     | Confirm attachment path (add file)                     |
| `ctrl+s`    | Send email via Resend API                              |
| `ctrl+r`    | Hand off to `pop` CLI composer                         |
| `esc`       | Back to menu                                           |

### Registration List

| Key         | Action                                  |
| ----------- | --------------------------------------- |
| `↑↓` / `jk` | Navigate rows                           |
| `enter`     | Open registration detail                |
| `f`         | Cycle filter: All → Pending → Confirmed |
| `n`         | Next page                               |
| `p`         | Previous page                           |
| `r`         | Refresh                                 |
| `esc`       | Back to menu                            |

### Registration Detail

| Key         | Action                            |
| ----------- | --------------------------------- |
| `↑↓` / `jk` | Scroll content                    |
| `c`         | Confirm payment (shows modal)     |
| `d`         | Delete registration (shows modal) |
| `esc`       | Back to list                      |

### Confirm / Delete Modal

| Key           | Action         |
| ------------- | -------------- |
| `y` / `enter` | Confirm action |
| `n` / `esc`   | Cancel         |

## Session Storage

Sessions are stored at `~/.cryptx-cli/session.json`. The file is created with permissions `0600` and contains:

```json
{
  "sessionId": "<appwrite session id>",
  "userId": "<appwrite user id>",
  "userEmail": "admin@cryptx.lk",
  "expiresAt": "2026-04-06T12:00:00Z"
}
```

Delete this file to force re-login: `rm ~/.cryptx-cli/session.json`

## OAuth Flow

OAuth login opens the system browser to the Appwrite OAuth2 token URL. A local HTTP server listens on `localhost:19271` for the callback. After the provider authenticates the user, Appwrite redirects back with `userId` + `secret` parameters. The CLI exchanges these for a full Appwrite session.

## Commit Conventions

- Lower-case, imperative subjects (`fix: handle empty member json`)
- First line under 72 characters
- Preferred prefixes: `feat:`, `fix:`, `update:`, `refactor:`
- Explain the "why" in the commit body when the change is not obvious

## Notes for AI Agents

- The CLI uses **pure session-based auth** — no server-side API key. The operator logs in via the TUI (email/password or Google OAuth), which creates an Appwrite session. All DB and storage operations run under that session. Appwrite collection/bucket permissions must grant read/write access to the relevant Appwrite team or user role.
- Member sub-objects (member2, member3, member4) are stored as **JSON strings** in string attributes (`member2Json`, etc.) because Appwrite does not natively support nested object arrays without relationship collections.
- The email template at `assets/email_template.html` is loaded at runtime by `internal/email/mailer.go`. Changes take effect immediately without recompiling.
- To add a new event type: add a model in `internal/models/`, CRUD methods in `internal/appwrite/`, a render function in `internal/tui/detail.go`, update `EventType` in `internal/tui/menu.go`, and wire it in `internal/tui/app.go`.
