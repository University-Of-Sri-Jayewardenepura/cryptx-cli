// Package appwrite wraps the Appwrite Go SDK with helpers for this CLI.
package appwrite

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/appwrite/sdk-for-go/appwrite"
	"github.com/appwrite/sdk-for-go/client"
	"github.com/appwrite/sdk-for-go/databases"
	"github.com/appwrite/sdk-for-go/storage"
	"github.com/cryptx/cryptx-cli/config"
)

// Services aggregates Appwrite service clients for use throughout the app.
type Services struct {
	DB      *databases.Databases
	Storage *storage.Storage
	cfg     *config.Config
}

// NewWithSession creates a client authenticated as a specific user via their
// session ID. Pass an empty string before login to get a bare client that only
// carries the project header; swap it with the real session after login.
func NewWithSession(cfg *config.Config, sessionID string) *Services {
	var clt client.Client
	if sessionID != "" {
		clt = appwrite.NewClient(
			appwrite.WithEndpoint(cfg.AppwriteEndpoint),
			appwrite.WithProject(cfg.AppwriteProjectID),
			appwrite.WithSession(sessionID),
		)
	} else {
		clt = appwrite.NewClient(
			appwrite.WithEndpoint(cfg.AppwriteEndpoint),
			appwrite.WithProject(cfg.AppwriteProjectID),
		)
	}

	// When CRYPTX_DEBUG=1 is set, wrap the HTTP transport to log all
	// requests and responses to ~/cryptx-debug.log for inspection.
	if os.Getenv("CRYPTX_DEBUG") == "1" {
		if lt := newDebugLogger(); lt != nil {
			clt.Client.Transport = &loggingRoundTripper{
				wrapped: clt.Client.Transport,
				log:     lt,
			}
		}
	}

	return newServices(clt, cfg)
}

func newServices(clt client.Client, cfg *config.Config) *Services {
	return &Services{
		DB:      databases.New(clt),
		Storage: storage.New(clt),
		cfg:     cfg,
	}
}

// Config exposes the loaded configuration.
func (s *Services) Config() *config.Config {
	return s.cfg
}

// ── debug HTTP logger ─────────────────────────────────────────────────────────

type loggingRoundTripper struct {
	wrapped http.RoundTripper
	log     *os.File
}

func (l *loggingRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	ts := time.Now().Format("15:04:05.000")

	// Log request line + headers.
	fmt.Fprintf(l.log, "\n[%s] REQUEST  %s %s\n", ts, req.Method, req.URL.String())
	for k, vs := range req.Header {
		for _, v := range vs {
			fmt.Fprintf(l.log, "  > %s: %s\n", k, v)
		}
	}

	wrapped := l.wrapped
	if wrapped == nil {
		wrapped = http.DefaultTransport
	}
	resp, err := wrapped.RoundTrip(req)
	if err != nil {
		fmt.Fprintf(l.log, "[%s] ERROR    %v\n", ts, err)
		return resp, err
	}

	// Read, log, then restore the response body so the SDK can still parse it.
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	resp.Body = io.NopCloser(bytes.NewReader(body))

	fmt.Fprintf(l.log, "[%s] RESPONSE %d\n", ts, resp.StatusCode)
	fmt.Fprintf(l.log, "  < Body: %s\n", string(body))

	return resp, nil
}

func newDebugLogger() *os.File {
	home, _ := os.UserHomeDir()
	path := filepath.Join(home, "cryptx-debug.log")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return nil
	}
	fmt.Fprintf(f, "\n=== cryptx debug session %s ===\n", time.Now().Format(time.RFC3339))
	return f
}
