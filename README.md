# CryptX CLI

A terminal UI (TUI) admin tool for managing CryptX 2.0 event registrations stored in Appwrite. Built with Go and Charm's Bubble Tea framework.

## Required Environment Variables

The following environment variables are required for the tool to function correctly. Ensure these are set in your `.env` file:

| Variable                                      | Description                                                              |
| --------------------------------------------- | ------------------------------------------------------------------------ |
| `APPWRITE_ENDPOINT`                           | Appwrite endpoint URL (e.g., `https://cloud.appwrite.io/v1`)             |
| `APPWRITE_PROJECT_ID`                         | Appwrite project ID                                                      |
| `APPWRITE_DATABASE_ID`                        | Appwrite database ID                                                     |
| `APPWRITE_CTF_COLLECTION_ID`                  | Collection ID for CTF registrations                                      |
| `APPWRITE_SCHOOL_HACKATHON_COLLECTION_ID`     | Collection ID for School Hackathon registrations                         |
| `APPWRITE_UNIVERSITY_HACKATHON_COLLECTION_ID` | Collection ID for University Hackathon registrations                     |
| `APPWRITE_DESIGNATHON_COLLECTION_ID`          | Collection ID for Designathon registrations                              |
| `APPWRITE_CTF_BUCKET_ID`                      | Storage bucket for CTF payment slips                                     |
| `APPWRITE_HACKATHON_SCHOOL_BUCKET_ID`         | Storage bucket for School Hackathon team logos                           |
| `APPWRITE_HACKATHON_UNIVERSITY_BUCKET_ID`     | Storage bucket for University Hackathon team logos                       |
| `APPWRITE_DESIGNATHON_BUCKET_ID`              | Storage bucket for Designathon team logos                                |
| `RESEND_API_KEY`                              | Resend API key for email confirmations                                   |
| `WAHA_BASE_URL`                               | Base URL for WhatsApp HTTP API (optional; leave empty to disable checks) |
| `WAHA_API_KEY`                                | API key for WhatsApp HTTP API                                            |
| `WAHA_SESSION`                                | WhatsApp session ID                                                      |
| `WAHA_CTF_GROUP_ID`                           | WhatsApp group ID for CTF                                                |
| `WAHA_SCHOOL_HACKATHON_GROUP_ID`              | WhatsApp group ID for School Hackathon                                   |
| `WAHA_UNIVERSITY_HACKATHON_GROUP_ID`          | WhatsApp group ID for University Hackathon                               |
| `WAHA_DESIGNATHON_GROUP_ID`                   | WhatsApp group ID for Designathon                                        |

---


## Features

- **Paginated list views** of CTF, School Hackathon, and Designathon registrations, merch store
- **Full detail view** with all fields, member breakdowns, and file info
- **Payment confirmation** — marks a registration confirmed in Appwrite and sends a styled HTML email to the registrant
- **Delete registrations** with confirmation prompt
- **Download payment slips / team logos** from Appwrite storage buckets
- **Status filter** — cycle between All / Pending / Confirmed registrations
- **Appwrite authentication** — email/password or OAuth (Google) with local session persistence (no re-login on every run)
- **Runs on macOS, Linux, and Windows**

---

## Table of Contents

1. [Prerequisites](#prerequisites)
2. [Installation — Pre-built Binary](#installation--pre-built-binary)
3. [Building from Source](#building-from-source)
   - [macOS](#macos)
   - [Linux](#linux)
   - [Windows](#windows)
   - [Cross-compilation](#cross-compilation)
4. [Configuration](#configuration)
5. [Appwrite Setup](#appwrite-setup)
6. [Running the Tool](#running-the-tool)
7. [TUI Navigation](#tui-navigation)
8. [Authentication](#authentication)
9. [Email Confirmation](#email-confirmation)
10. [Project Structure](#project-structure)

---

## Prerequisites

| Requirement     | Minimum version           | Notes                                                          |
| --------------- | ------------------------- | -------------------------------------------------------------- |
| **Go**          | 1.21+                     | [download](https://go.dev/dl/) — Go 1.25 recommended           |
| **Git**         | any                       | for cloning                                                    |
| **Appwrite**    | Cloud or self-hosted 1.6+ | project + API key required                                     |
| **SMTP server** | —                         | Gmail, Resend, Mailgun, etc. — for sending confirmation emails |

> **Windows note:** a modern terminal is strongly recommended — [Windows Terminal](https://apps.microsoft.com/store/detail/windows-terminal/9N0DX20HK701) or [WezTerm](https://wezfurlong.org/wezterm/). The legacy `cmd.exe` and older PowerShell windows do not render ANSI colours properly.

---

## Installation — Pre-built Binary

If a binary release is available:

1. Download the archive for your platform from the Releases page.
2. Extract and place `cryptx-cli` (or `cryptx-cli.exe` on Windows) somewhere on your `PATH`.
3. Copy `.env.example` to `.env` and fill in your credentials (see [Configuration](#configuration)).
4. Run `cryptx-cli`.

---

## Building from Source

### Clone the repository

```bash
git clone https://github.com/cryptx/cryptx-cli.git
cd cryptx-cli
```

### macOS

Requires Go 1.21+. Install via [Homebrew](https://brew.sh/) if needed:

```bash
brew install go
```

Build and install to `~/go/bin`:

```bash
CGO_ENABLED=0 go build -trimpath -tags "netgo osusergo timetzdata" -ldflags="-s -w" -o cryptx-cli .
```

Or install globally:

```bash
go install github.com/cryptx/cryptx-cli@latest
```

Run:

```bash
./cryptx-cli
```

### Linux

Install Go from [go.dev/dl](https://go.dev/dl/) or your package manager:

```bash
# Debian/Ubuntu
sudo apt install golang-go

# Fedora/RHEL
sudo dnf install golang

# Arch
sudo pacman -S go
```

Build:

```bash
CGO_ENABLED=0 go build -trimpath -tags "netgo osusergo timetzdata" -ldflags="-s -w" -o cryptx-cli .
chmod +x cryptx-cli
./cryptx-cli
```

To install system-wide:

```bash
sudo mv cryptx-cli /usr/local/bin/
```

### Windows

Install Go from [go.dev/dl](https://go.dev/dl/) — choose the Windows `.msi` installer.

Open **PowerShell** or **Windows Terminal** and run:

```powershell
$env:CGO_ENABLED="0"; go build -trimpath -tags "netgo osusergo timetzdata" -ldflags="-s -w" -o cryptx-cli.exe .
.\cryptx-cli.exe
```

> Tip: run inside [Windows Terminal](https://aka.ms/terminal) for full colour and Unicode support.

### Cross-compilation

Go's cross-compilation is built in — no additional tooling needed. Set `GOOS` and `GOARCH` before building:

| Target platform     | Command                                                                                                                                           |
| ------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------- |
| macOS Intel         | `CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -trimpath -tags "netgo osusergo timetzdata" -ldflags="-s -w" -o cryptx-cli-darwin-amd64 .`       |
| macOS Apple Silicon | `CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -trimpath -tags "netgo osusergo timetzdata" -ldflags="-s -w" -o cryptx-cli-darwin-arm64 .`       |
| Linux x86-64        | `CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -tags "netgo osusergo timetzdata" -ldflags="-s -w" -o cryptx-cli-linux-amd64 .`         |
| Linux ARM64         | `CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath -tags "netgo osusergo timetzdata" -ldflags="-s -w" -o cryptx-cli-linux-arm64 .`         |
| Windows x86-64      | `CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -trimpath -tags "netgo osusergo timetzdata" -ldflags="-s -w" -o cryptx-cli-windows-amd64.exe .` |
| Windows ARM64       | `CGO_ENABLED=0 GOOS=windows GOARCH=arm64 go build -trimpath -tags "netgo osusergo timetzdata" -ldflags="-s -w" -o cryptx-cli-windows-arm64.exe .` |

On Windows (PowerShell), prefix with environment variables:

```powershell
$env:GOOS="linux"; $env:GOARCH="amd64"; go build -o cryptx-cli-linux-amd64 .
```

**Build all platforms at once (bash/zsh):**

```bash
make release
```

This generates `dist/cryptx-cli-<os>-<arch>[.exe]` binaries with `CGO_ENABLED=0`.

Why these flags matter for portability:

- `CGO_ENABLED=0`: avoids libc/glibc runtime mismatch issues on target machines
- `-tags "netgo osusergo timetzdata"`: removes libc-based DNS/user lookups and embeds timezone data
- `-trimpath -ldflags "-s -w"`: strips path/debug metadata for smaller, reproducible artifacts

---

## Configuration

Copy the example env file and fill in your values:

```bash
cp .env.example .env
```

Edit `.env`:

```env
# ── Appwrite ──────────────────────────────────────────────────────────────────
APPWRITE_ENDPOINT=https://cloud.appwrite.io/v1
APPWRITE_PROJECT_ID=your_project_id_here
APPWRITE_API_KEY=your_server_api_key_here

# ── Database ──────────────────────────────────────────────────────────────────
DB_ID=your_database_id
CTF_COLLECTION_ID=ctf_registrations
HACKATHON_COLLECTION_ID=school_hackathon_registrations
DESIGNATHON_COLLECTION_ID=designathon_registrations

# ── Storage Buckets ───────────────────────────────────────────────────────────
PAYMENT_SLIPS_BUCKET_ID=payment_slips
TEAM_LOGOS_BUCKET_ID=team_logos

# ── Email (SMTP) ──────────────────────────────────────────────────────────────
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USER=your@gmail.com
SMTP_PASS=your_app_password      # Gmail: use an App Password, not your login password
SMTP_FROM=noreply@cryptx.lk
```

### Required variables

| Variable              | Description                                                   |
| --------------------- | ------------------------------------------------------------- |
| `APPWRITE_PROJECT_ID` | Found in Appwrite Console → Settings                          |
| `APPWRITE_API_KEY`    | Server-side API key with **Databases** + **Storage** scopes   |
| `DB_ID`               | The Appwrite database containing the registration collections |

### Optional variables

All other variables are optional — the tool will warn if collection IDs are missing when you try to access that event type.

### Gmail SMTP setup

1. Enable [2-Step Verification](https://myaccount.google.com/signinoptions/two-step-verification) on your Google account.
2. Generate an [App Password](https://myaccount.google.com/apppasswords) (select "Mail" + "Other device").
3. Use that 16-character App Password as `SMTP_PASS`.

---

## Appwrite Setup

### 1. Create the database

In the Appwrite Console, create a database and note its ID → set as `DB_ID`.

### 2. Create the collections

Create three collections with **any** IDs (set in `.env`):

#### `ctf_registrations`

| Attribute             | Type                | Required                   |
| --------------------- | ------------------- | -------------------------- |
| `registrationType`    | String              | Yes                        |
| `teamName`            | String              | No                         |
| `leaderName`          | String              | Yes                        |
| `leaderUniversity`    | String              | Yes                        |
| `leaderContact`       | String              | Yes                        |
| `leaderEmail`         | String              | Yes                        |
| `leaderWhatsapp`      | String              | Yes                        |
| `leaderNIC`           | String              | Yes                        |
| `leaderRegNo`         | String              | No                         |
| `member2Json`         | String (size: 2000) | No                         |
| `member3Json`         | String (size: 2000) | No                         |
| `member4Json`         | String (size: 2000) | No                         |
| `referralSource`      | String              | Yes                        |
| `awarenessPreference` | String              | Yes                        |
| `agreement`           | Boolean             | Yes                        |
| `paymentSlipFileId`   | String              | No                         |
| `paymentSlipName`     | String              | No                         |
| `paymentSlipMime`     | String              | No                         |
| `confirmationStatus`  | String              | Yes — default: `"pending"` |
| `confirmedAt`         | String              | No                         |

#### `school_hackathon_registrations`

| Attribute                     | Type                | Required                   |
| ----------------------------- | ------------------- | -------------------------- |
| `teamName`                    | String              | Yes                        |
| `teamMemberCount`             | Integer             | Yes                        |
| `leaderFullName`              | String              | Yes                        |
| `leaderGrade`                 | String              | Yes                        |
| `leaderContactNumber`         | String              | Yes                        |
| `leaderEmail`                 | String              | Yes                        |
| `leaderNIC`                   | String              | No                         |
| `leaderSchoolName`            | String              | Yes                        |
| `member2Json`                 | String (size: 2000) | No                         |
| `member3Json`                 | String (size: 2000) | No                         |
| `member4Json`                 | String (size: 2000) | No                         |
| `inChargePersonFullName`      | String              | Yes                        |
| `inChargePersonContactNumber` | String              | Yes                        |
| `referralSource`              | String              | Yes                        |
| `agreement`                   | Boolean             | Yes                        |
| `confirmationStatus`          | String              | Yes — default: `"pending"` |
| `confirmedAt`                 | String              | No                         |

#### `designathon_registrations`

| Attribute               | Type                | Required                   |
| ----------------------- | ------------------- | -------------------------- |
| `teamName`              | String              | Yes                        |
| `primaryContact`        | String              | Yes                        |
| `teamLogoFileId`        | String              | No                         |
| `teamLogoName`          | String              | No                         |
| `teamLogoMime`          | String              | No                         |
| `member1FullName`       | String              | Yes                        |
| `member1RegNo`          | String              | Yes                        |
| `member1NIC`            | String              | Yes                        |
| `member1Email`          | String              | Yes                        |
| `member1Phone`          | String              | Yes                        |
| `member1University`     | String              | Yes                        |
| `member2Json`           | String (size: 2000) | No                         |
| `member3Json`           | String (size: 2000) | No                         |
| `previousParticipation` | String              | Yes                        |
| `agreement`             | Boolean             | Yes                        |
| `confirmationStatus`    | String              | Yes — default: `"pending"` |
| `confirmedAt`           | String              | No                         |

### 3. Create storage buckets

Create two buckets:

- `payment_slips` — for CTF payment slip uploads
- `team_logos` — for Designathon team logo uploads

Note both bucket IDs → set as `PAYMENT_SLIPS_BUCKET_ID` and `TEAM_LOGOS_BUCKET_ID`.

### 4. Create an API key

In Appwrite Console → API Keys, create a **Server** API key with at minimum:

- `databases.read`, `databases.write`
- `documents.read`, `documents.write`, `documents.delete`
- `files.read`, `files.write`

Set this as `APPWRITE_API_KEY`.

---

## Running the Tool

```bash
# From the project directory (uses ./assets/ for email template)
./cryptx-cli

# Or if installed to PATH
cryptx-cli
```

On first run (or after session expiry), the login screen appears. Enter your Appwrite user credentials.

> The tool uses your **Appwrite user account** for login verification and your **API key** for all database/storage operations.

---

## TUI Navigation

### Login Screen

```
  CryptX 2.0 — Admin CLI
  Sign in to manage registrations

  Email
  ┌──────────────────────────────────────┐
  │ admin@cryptx.lk                      │
  └──────────────────────────────────────┘
  Password
  ┌──────────────────────────────────────┐
  │ ••••••••                             │
  └──────────────────────────────────────┘

  enter sign in  ctrl+o oauth  tab next field
```

| Key               | Action                           |
| ----------------- | -------------------------------- |
| `tab` / `↓`       | Move to next field               |
| `shift+tab` / `↑` | Move to previous field           |
| `enter`           | Submit login                     |
| `ctrl+o`          | Login with OAuth (opens browser) |

### Main Menu

| Key               | Action                   |
| ----------------- | ------------------------ |
| `↑` / `k`         | Move up                  |
| `↓` / `j`         | Move down                |
| `enter` / `space` | Select event             |
| `1`               | Jump to CTF              |
| `2`               | Jump to School Hackathon |
| `3`               | Jump to Designathon      |
| `q`               | Quit                     |

### Registration List

| Key       | Action                                  |
| --------- | --------------------------------------- |
| `↑` / `k` | Previous row                            |
| `↓` / `j` | Next row                                |
| `enter`   | Open detail view                        |
| `f`       | Cycle filter: All → Pending → Confirmed |
| `n`       | Next page (25 per page)                 |
| `p`       | Previous page                           |
| `r`       | Refresh data                            |
| `esc`     | Back to main menu                       |
| `q`       | Quit                                    |

### Detail View

| Key       | Action                                         |
| --------- | ---------------------------------------------- |
| `↑` / `k` | Scroll up                                      |
| `↓` / `j` | Scroll down                                    |
| `c`       | Confirm payment (opens confirmation modal)     |
| `d`       | Delete registration (opens confirmation modal) |
| `esc`     | Back to list                                   |
| `q`       | Quit                                           |

### Confirm / Delete Modal

| Key           | Action             |
| ------------- | ------------------ |
| `y` / `enter` | Confirm the action |
| `n` / `esc`   | Cancel             |

---

## Authentication

The tool uses a two-layer authentication approach:

1. **Identity verification** — Login with an Appwrite user account (email/password or OAuth). This verifies you are an authorised operator.
2. **Data access** — All database and storage operations use the server-side API key from `.env` (bypasses document-level permissions).

### Session persistence

After a successful login, the session is saved to:

```
~/.cryptx-cli/session.json   (macOS / Linux)
%USERPROFILE%\.cryptx-cli\session.json   (Windows)
```

The file is created with restricted permissions (`0600`). On subsequent runs the session is restored automatically — no login prompt until it expires (typically 30 days).

**Force re-login** by deleting the session file:

```bash
# macOS / Linux
rm ~/.cryptx-cli/session.json

# Windows (PowerShell)
Remove-Item "$env:USERPROFILE\.cryptx-cli\session.json"
```

### OAuth Login

Press `ctrl+o` on the login screen. The tool:

1. Starts a local HTTP server on port `19271`
2. Opens your default browser to the Appwrite OAuth URL (Google by default)
3. Waits up to 5 minutes for the OAuth callback
4. Exchanges the token for a session and saves it

If the browser does not open automatically, the OAuth URL is printed to the terminal — copy and paste it manually.

---

## Email Confirmation

When you press `c` on a registration and confirm the modal, the tool:

1. Updates `confirmationStatus` to `"confirmed"` and sets `confirmedAt` in Appwrite
2. Renders `assets/email_template.html` with the registrant's details
3. Sends the email via SMTP to the registrant's primary email address

The HTML template is loaded at runtime from `assets/email_template.html` — edit it to customise the email without recompiling.

**The `assets/` directory must be present alongside the binary when running.**

---

## Project Structure

```
cryptx-cli/
├── main.go                          Entry point
├── go.mod / go.sum                  Go module
├── .env.example                     Configuration template
├── README.md                        This file
├── AGENTS.md                        AI agent / developer guide
├── .cursor/
│   └── mcp.json                     Context7 MCP server config (Cursor IDE)
├── assets/
│   └── email_template.html          Confirmation email HTML template
├── config/
│   └── config.go                    Env loading, Config struct, validation
└── internal/
    ├── appwrite/
    │   ├── client.go                Appwrite service factory
    │   ├── auth.go                  Email login, OAuth, session validation
    │   ├── ctf.go                   CTF CRUD + storage download
    │   ├── hackathon.go             School Hackathon CRUD
    │   └── designathon.go           Designathon CRUD + storage download
    ├── email/
    │   └── mailer.go                SMTP email sender
    ├── models/
    │   ├── ctf.go                   CTF Go structs
    │   ├── hackathon.go             School Hackathon Go structs
    │   └── designathon.go           Designathon Go structs
    ├── session/
    │   └── store.go                 Session file management
    └── tui/
        ├── app.go                   Root Bubble Tea model & screen router
        ├── login.go                 Login screen
        ├── menu.go                  Main menu
        ├── list.go                  Paginated registration list
        ├── detail.go                Registration detail view
        ├── confirm.go               Confirm/delete modal dialogs
        └── styles.go                All Lipgloss styles
```

---

## Troubleshooting

### `configuration error: APPWRITE_PROJECT_ID is required`

You haven't created a `.env` file. Copy `.env.example` and fill in your values:

```bash
cp .env.example .env
```

### `login failed: ...`

- Check that the email/password match an existing Appwrite **user account** (not just an API key).
- Verify `APPWRITE_ENDPOINT` and `APPWRITE_PROJECT_ID` are correct.

### Email not sending

- Confirm `SMTP_*` values are correct.
- For Gmail, use an [App Password](https://myaccount.google.com/apppasswords), not your account password.
- Check that port `587` is not blocked by your firewall.

### Blank/corrupted terminal output on Windows

Use [Windows Terminal](https://aka.ms/terminal) instead of the legacy `cmd.exe`. Enable Virtual Terminal Processing if needed:

```powershell
Set-ItemProperty HKCU:\Console VirtualTerminalLevel -Type DWORD 1
```

### Session expired or invalid

Delete the local session file to force a fresh login:

```bash
rm ~/.cryptx-cli/session.json
```

### `assets/email_template.html: no such file or directory`

The binary looks for `assets/email_template.html` relative to the **current working directory**. Run the tool from the project root, or copy the `assets/` folder next to your binary.

---

## Development

```bash
# Run without building
go run .

# Run with race detector
go run -race .

# Format (Go's built-in formatter)
gofmt -w .

# Vet
go vet ./...

# Build with optimisations
go build -ldflags="-s -w" -o cryptx-cli .
```

---

## License

MIT — see [LICENSE](LICENSE) for details.

## Required Environment Variables

The following environment variables are required for the tool to function correctly. Ensure these are set in your `.env` file:

| Variable                                      | Description                                                              |
| --------------------------------------------- | ------------------------------------------------------------------------ |
| `APPWRITE_ENDPOINT`                           | Appwrite endpoint URL (e.g., `https://cloud.appwrite.io/v1`)             |
| `APPWRITE_PROJECT_ID`                         | Appwrite project ID                                                      |
| `APPWRITE_DATABASE_ID`                        | Appwrite database ID                                                     |
| `APPWRITE_CTF_COLLECTION_ID`                  | Collection ID for CTF registrations                                      |
| `APPWRITE_SCHOOL_HACKATHON_COLLECTION_ID`     | Collection ID for School Hackathon registrations                         |
| `APPWRITE_UNIVERSITY_HACKATHON_COLLECTION_ID` | Collection ID for University Hackathon registrations                     |
| `APPWRITE_DESIGNATHON_COLLECTION_ID`          | Collection ID for Designathon registrations                              |
| `APPWRITE_CTF_BUCKET_ID`                      | Storage bucket for CTF payment slips                                     |
| `APPWRITE_HACKATHON_SCHOOL_BUCKET_ID`         | Storage bucket for School Hackathon team logos                           |
| `APPWRITE_HACKATHON_UNIVERSITY_BUCKET_ID`     | Storage bucket for University Hackathon team logos                       |
| `APPWRITE_DESIGNATHON_BUCKET_ID`              | Storage bucket for Designathon team logos                                |
| `RESEND_API_KEY`                              | Resend API key for email confirmations                                   |
| `WAHA_BASE_URL`                               | Base URL for WhatsApp HTTP API (optional; leave empty to disable checks) |
| `WAHA_API_KEY`                                | API key for WhatsApp HTTP API                                            |
| `WAHA_SESSION`                                | WhatsApp session ID                                                      |
| `WAHA_CTF_GROUP_ID`                           | WhatsApp group ID for CTF                                                |
| `WAHA_SCHOOL_HACKATHON_GROUP_ID`              | WhatsApp group ID for School Hackathon                                   |
| `WAHA_UNIVERSITY_HACKATHON_GROUP_ID`          | WhatsApp group ID for University Hackathon                               |
| `WAHA_DESIGNATHON_GROUP_ID`                   | WhatsApp group ID for Designathon                                        |

---
