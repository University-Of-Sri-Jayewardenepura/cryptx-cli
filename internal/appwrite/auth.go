package appwrite

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/appwrite/sdk-for-go/account"
	aw "github.com/appwrite/sdk-for-go/appwrite"
	appwriteModels "github.com/appwrite/sdk-for-go/models"
	"github.com/cryptx/cryptx-cli/config"
	"github.com/cryptx/cryptx-cli/internal/session"
)

// LoginResult is returned from a successful email login.
type LoginResult struct {
	SessionID     string
	SessionSecret string // value of the a_session_* cookie — used for WithSession
	UserID        string
	UserEmail     string
	ExpiresAt     time.Time
}

// SignUpResult is returned from a successful account creation.
type SignUpResult struct {
	UserID        string
	UserEmail     string
	SessionID     string // Session created immediately after signup.
	SessionSecret string // value of the a_session_* cookie — used for WithSession
	ExpiresAt     time.Time
}

// SignUp creates a new Appwrite account and immediately signs in to produce
// a session. The session is needed to send the verification email.
func SignUp(cfg *config.Config, name, email, password string) (*SignUpResult, error) {
	clt := aw.NewClient(
		aw.WithEndpoint(cfg.AppwriteEndpoint),
		aw.WithProject(cfg.AppwriteProjectID),
	)
	svc := account.New(clt)

	// Create the account ("unique()" lets Appwrite generate the ID).
	_, err := svc.Create("unique()", email, password, svc.WithCreateName(name))
	if err != nil {
		return nil, fmt.Errorf("account creation failed: %w", err)
	}

	// Immediately sign in so we have a session for verification.
	sess, err := svc.CreateEmailPasswordSession(email, password)
	if err != nil {
		return nil, fmt.Errorf("post-signup login failed: %w", err)
	}

	expiry, _ := time.Parse(time.RFC3339, sess.Expire)
	if expiry.IsZero() {
		expiry = time.Now().Add(30 * 24 * time.Hour)
	}

	return &SignUpResult{
		UserID:        sess.UserId,
		UserEmail:     email,
		SessionID:     sess.Id,
		SessionSecret: extractSessionSecret(clt.Client, cfg.AppwriteEndpoint),
		ExpiresAt:     expiry,
	}, nil
}

// SendEmailVerification sends a verification email to the logged-in user.
// The email will contain a link with userId and secret as query parameters.
// The user must copy those values and confirm via ConfirmEmailVerification.
// sessionSecret must be the a_session_* cookie value for the user's session.
func SendEmailVerification(cfg *config.Config, sessionSecret string) error {
	clt := aw.NewClient(
		aw.WithEndpoint(cfg.AppwriteEndpoint),
		aw.WithProject(cfg.AppwriteProjectID),
		aw.WithSession(sessionSecret),
	)
	svc := account.New(clt)

	// The URL is a placeholder — the user reads userId+secret from it directly.
	_, err := svc.CreateEmailVerification("http://localhost")
	if err != nil {
		return fmt.Errorf("send verification email: %w", err)
	}
	return nil
}

// ConfirmEmailVerification completes email verification using the userId and
// secret copied from the link the user received by email.
func ConfirmEmailVerification(cfg *config.Config, sessionSecret, userID, secret string) error {
	clt := aw.NewClient(
		aw.WithEndpoint(cfg.AppwriteEndpoint),
		aw.WithProject(cfg.AppwriteProjectID),
		aw.WithSession(sessionSecret),
	)
	svc := account.New(clt)

	_, err := svc.UpdateEmailVerification(userID, secret)
	if err != nil {
		return fmt.Errorf("confirm email verification: %w", err)
	}
	return nil
}

// LoginWithEmail authenticates against Appwrite using email and password.
func LoginWithEmail(cfg *config.Config, email, password string) (*LoginResult, error) {
	clt := aw.NewClient(
		aw.WithEndpoint(cfg.AppwriteEndpoint),
		aw.WithProject(cfg.AppwriteProjectID),
	)
	svc := account.New(clt)

	sess, err := svc.CreateEmailPasswordSession(email, password)
	if err != nil {
		return nil, fmt.Errorf("login failed: %w", err)
	}

	expiry, _ := time.Parse(time.RFC3339, sess.Expire)
	if expiry.IsZero() {
		expiry = time.Now().Add(30 * 24 * time.Hour)
	}

	return &LoginResult{
		SessionID:     sess.Id,
		SessionSecret: extractSessionSecret(clt.Client, cfg.AppwriteEndpoint),
		UserID:        sess.UserId,
		UserEmail:     email,
		ExpiresAt:     expiry,
	}, nil
}

// ValidateSession checks whether a stored session is still valid in Appwrite.
func ValidateSession(cfg *config.Config, s *session.Session) error {
	if s.SessionSecret == "" {
		return fmt.Errorf("no session secret stored; please log in again")
	}
	clt := aw.NewClient(
		aw.WithEndpoint(cfg.AppwriteEndpoint),
		aw.WithProject(cfg.AppwriteProjectID),
		aw.WithSession(s.SessionSecret),
	)
	svc := account.New(clt)

	_, err := svc.GetSession(s.SessionID)
	if err != nil {
		return fmt.Errorf("session no longer valid: %w", err)
	}
	return nil
}

// LogoutSession deletes the session from Appwrite and removes the local file.
func LogoutSession(cfg *config.Config, s *session.Session) error {
	clt := aw.NewClient(
		aw.WithEndpoint(cfg.AppwriteEndpoint),
		aw.WithProject(cfg.AppwriteProjectID),
		aw.WithSession(s.SessionSecret),
	)
	svc := account.New(clt)
	_, _ = svc.DeleteSession(s.SessionID)
	return session.Delete()
}

// OAuthProvider enumerates supported OAuth providers.
type OAuthProvider string

const (
	OAuthGoogle OAuthProvider = "google"
	OAuthGitHub OAuthProvider = "github"
)

const oauthCallbackPort = 19271

// OAuthResult is returned on a successful OAuth login.
type OAuthResult struct {
	UserID        string
	UserEmail     string
	SessionID     string
	SessionSecret string // value of the a_session_* cookie — used for WithSession
	ExpiresAt     time.Time
}

// LoginWithOAuth starts a local HTTP server, opens the Appwrite OAuth URL in
// the default browser, waits for the callback, and returns an OAuthResult.
func LoginWithOAuth(cfg *config.Config, provider OAuthProvider) (*OAuthResult, error) {
	successURL := fmt.Sprintf("http://localhost:%d/callback", oauthCallbackPort)
	failureURL := fmt.Sprintf("http://localhost:%d/failure", oauthCallbackPort)

	// Build the Appwrite OAuth token URL.
	oauthURL := fmt.Sprintf(
		"%s/account/tokens/oauth2/%s?project=%s&success=%s&failure=%s",
		cfg.AppwriteEndpoint, provider, cfg.AppwriteProjectID, successURL, failureURL,
	)

	type callbackData struct {
		userID string
		secret string
		err    error
	}
	ch := make(chan callbackData, 1)

	mux := http.NewServeMux()
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", oauthCallbackPort),
		Handler: mux,
	}

	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		userID := r.URL.Query().Get("userId")
		secret := r.URL.Query().Get("secret")
		if userID == "" || secret == "" {
			ch <- callbackData{err: errors.New("OAuth callback missing userId or secret")}
			fmt.Fprintln(w, "Authentication failed. You may close this tab.")
			return
		}
		ch <- callbackData{userID: userID, secret: secret}
		fmt.Fprintln(w, "Authentication successful! Return to the terminal.")
	})

	mux.HandleFunc("/failure", func(w http.ResponseWriter, r *http.Request) {
		ch <- callbackData{err: errors.New("OAuth provider returned failure")}
		fmt.Fprintln(w, "Authentication failed. You may close this tab.")
	})

	go func() { _ = srv.ListenAndServe() }()
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
	}()

	if err := openBrowser(oauthURL); err != nil {
		return nil, fmt.Errorf("failed to open browser: %w\n\nPlease open this URL manually:\n%s", err, oauthURL)
	}

	select {
	case cb := <-ch:
		if cb.err != nil {
			return nil, cb.err
		}
		return exchangeOAuthToken(cfg, cb.userID, cb.secret)
	case <-time.After(5 * time.Minute):
		return nil, errors.New("OAuth login timed out (5 minutes)")
	}
}

// exchangeOAuthToken calls account.CreateSession with the OAuth userId + secret.
func exchangeOAuthToken(cfg *config.Config, userID, secret string) (*OAuthResult, error) {
	clt := aw.NewClient(
		aw.WithEndpoint(cfg.AppwriteEndpoint),
		aw.WithProject(cfg.AppwriteProjectID),
	)
	svc := account.New(clt)

	sess, err := svc.CreateSession(userID, secret)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange OAuth token: %w", err)
	}

	expiry, _ := time.Parse(time.RFC3339, sess.Expire)
	if expiry.IsZero() {
		expiry = time.Now().Add(30 * 24 * time.Hour)
	}

	email, err := fetchUserEmail(cfg, sess)
	if err != nil {
		email = ""
	}

	return &OAuthResult{
		UserID:        userID,
		UserEmail:     email,
		SessionID:     sess.Id,
		SessionSecret: extractSessionSecret(clt.Client, cfg.AppwriteEndpoint),
		ExpiresAt:     expiry,
	}, nil
}

func fetchUserEmail(cfg *config.Config, sess *appwriteModels.Session) (string, error) {
	clt := aw.NewClient(
		aw.WithEndpoint(cfg.AppwriteEndpoint),
		aw.WithProject(cfg.AppwriteProjectID),
		aw.WithSession(sess.Secret),
	)
	svc := account.New(clt)
	user, err := svc.Get()
	if err != nil {
		return "", err
	}
	return user.Email, nil
}

// openBrowser opens the given URL in the system default browser.
func openBrowser(url string) error {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "darwin":
		cmd = "open"
		args = []string{url}
	case "windows":
		cmd = "rundll32"
		args = []string{"url.dll,FileProtocolHandler", url}
	default:
		cmd = "xdg-open"
		args = []string{url}
	}

	return exec.Command(cmd, args...).Start()
}

// extractSessionSecret reads the a_session_* cookie set by Appwrite after a
// successful CreateEmailPasswordSession or CreateSession call. The cookie value
// is the session secret required for X-Appwrite-Session authenticated requests.
func extractSessionSecret(httpClient *http.Client, endpoint string) string {
	if httpClient == nil || httpClient.Jar == nil {
		return ""
	}
	u, err := url.Parse(endpoint)
	if err != nil {
		return ""
	}
	for _, c := range httpClient.Jar.Cookies(u) {
		if strings.HasPrefix(c.Name, "a_session_") {
			return c.Value
		}
	}
	return ""
}
