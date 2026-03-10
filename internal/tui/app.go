// Package tui implements the Bubble Tea TUI application for CryptX CLI.
package tui

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	awclient "github.com/appwrite/sdk-for-go/client"
	aw "github.com/cryptx/cryptx-cli/internal/appwrite"
	"github.com/cryptx/cryptx-cli/internal/email"
	"github.com/cryptx/cryptx-cli/internal/models"
	"github.com/cryptx/cryptx-cli/internal/session"
	"github.com/cryptx/cryptx-cli/internal/waha"
)

// Screen identifies which screen is currently active.
type Screen int

const (
	ScreenLogin Screen = iota
	ScreenSignup
	ScreenVerify
	ScreenMenu
	ScreenList
	ScreenDetail
	ScreenModal
	ScreenCompose
	ScreenAnalyser
)

// App is the root Bubble Tea model that owns all child models and manages
// navigation between screens.
type App struct {
	screen Screen
	width  int
	height int

	// services
	svc  *aw.Services
	waha *waha.Client

	// child models
	login   LoginModel
	signup  SignupModel
	verify  VerifyModel
	menu    MenuModel
	list    ListModel
	detail  DetailModel
	modal   ConfirmModel
	compose ComposeModel
	analyser AnalyserModel

	// saved state for returning from detail back to list
	activeEvent EventType
	activeDocID string

	// toasts / status messages
	toast    string
	toastErr bool

	// operator session
	sess *session.Session
}

// toastClearMsg clears the toast after a brief delay.
type toastClearMsg struct{}

// NewApp creates a new root application model.
// If sess is non-nil the user is already authenticated and the menu is shown.
func NewApp(svc *aw.Services, wahaClient *waha.Client, sess *session.Session) App {
	app := App{
		svc:  svc,
		waha: wahaClient,
		sess: sess,
		login: NewLoginModel(),
	}
	if sess != nil {
		app.screen = ScreenMenu
		app.menu = NewMenuModel(sess.UserEmail)
	} else {
		app.screen = ScreenLogin
	}
	return app
}

func (a App) Init() tea.Cmd {
	if a.screen == ScreenLogin {
		return a.login.Init()
	}
	if a.screen == ScreenSignup {
		return a.signup.Init()
	}
	if a.screen == ScreenVerify {
		return a.verify.Init()
	}
	return nil
}

func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	// ── Global exit ───────────────────────────────────────────────────────
	if key, ok := msg.(tea.KeyPressMsg); ok && key.String() == "ctrl+c" {
		return a, tea.Quit
	}

	switch msg := msg.(type) {
	// ── Window resize ──────────────────────────────────────────────────────
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		// Propagate to active sub-model.
		switch a.screen {
		case ScreenLogin:
			updated, cmd := a.login.Update(msg)
			a.login = updated
			cmds = append(cmds, cmd)
		case ScreenSignup:
			updated, cmd := a.signup.Update(msg)
			a.signup = updated
			cmds = append(cmds, cmd)
		case ScreenVerify:
			updated, cmd := a.verify.Update(msg)
			a.verify = updated
			cmds = append(cmds, cmd)
		case ScreenList:
			updated, cmd := a.list.Update(msg)
			a.list = updated
			cmds = append(cmds, cmd)
		case ScreenDetail:
			updated, cmd := a.detail.Update(msg)
			a.detail = updated
			cmds = append(cmds, cmd)
		case ScreenModal:
			updated, cmd := a.modal.Update(msg)
			a.modal = updated
			cmds = append(cmds, cmd)
		case ScreenAnalyser:
			updated, cmd := a.analyser.Update(msg)
			a.analyser = updated
			cmds = append(cmds, cmd)
		}

	// ── Login success → authenticate + go to menu ─────────────────────────
	case LoginSuccessMsg:
		return a, a.doLogin(msg.Email, msg.Password)

	// ── Switch to signup screen ───────────────────────────────────────────
	case SwitchToSignupMsg:
		a.signup = NewSignupModel()
		a.screen = ScreenSignup
		return a, a.signup.Init()

	// ── Switch back to login screen ───────────────────────────────────────
	case SwitchToLoginMsg:
		a.login = NewLoginModel()
		a.screen = ScreenLogin
		return a, a.login.Init()

	// ── Signup form submitted → create account ────────────────────────────
	case doSignupMsg:
		return a, a.doSignup(msg)

	// ── Signup done (internal) ────────────────────────────────────────────
	case signupDoneMsg:
		if msg.err != nil {
			updated, cmd := a.signup.Update(signupErrMsg(msg.err.Error()))
			a.signup = updated
			cmds = append(cmds, cmd)
			return a, tea.Batch(cmds...)
		}
		// Save session, move to email verification screen.
		a.sess = msg.sess
		a.svc = aw.NewWithSession(a.svc.Config(), msg.sess.SessionSecret)
		a.verify = NewVerifyModel(msg.sess.UserEmail)
		a.screen = ScreenVerify
		return a, a.verify.Init()

	// ── Verify form submitted → confirm verification ──────────────────────
	case doVerifyMsg:
		return a, a.doVerify(msg)

	// ── Verification done (internal) ──────────────────────────────────────
	case verifyDoneMsg:
		if msg.err != nil {
			updated, cmd := a.verify.Update(verifyErrMsg(msg.err.Error()))
			a.verify = updated
			cmds = append(cmds, cmd)
			return a, tea.Batch(cmds...)
		}
		a.screen = ScreenMenu
		a.menu = NewMenuModel(a.sess.UserEmail)
		return a, nil

	// ── Skip verification → go to menu ────────────────────────────────────
	case SkipVerifyMsg:
		a.screen = ScreenMenu
		a.menu = NewMenuModel(a.sess.UserEmail)
		return a, nil

	// ── OAuth requested → start OAuth flow ───────────────────────────────
	case LoginOAuthMsg:
		return a, a.doOAuth()

	// ── loginDoneMsg (internal) ───────────────────────────────────────────
	case loginDoneMsg:
		if msg.err != nil {
			updated, cmd := a.login.Update(loginErrMsg(msg.err.Error()))
			a.login = updated
			cmds = append(cmds, cmd)
			return a, tea.Batch(cmds...)
		}
		a.sess = msg.sess
		a.svc = aw.NewWithSession(a.svc.Config(), msg.sess.SessionSecret)
		a.screen = ScreenMenu
		a.menu = NewMenuModel(a.sess.UserEmail)
		return a, nil

	// ── Menu selection ────────────────────────────────────────────────────
	case MenuSelectMsg:
		if msg.Event == EventCompose {
			a.compose = NewComposeModel(a.width, a.height)
			a.screen = ScreenCompose
			return a, a.compose.Init()
		}
		if msg.Event == EventGroupAnalyser {
			a.analyser = NewAnalyserModel(a.width, a.height)
			a.screen = ScreenAnalyser
			return a, tea.Batch(a.analyser.Init(), a.doAnalyse())
		}
		a.activeEvent = msg.Event
		a.list = NewListModel(msg.Event, a.width, a.height)
		a.screen = ScreenList
		return a, a.list.Init()

	// ── List: fetch data ──────────────────────────────────────────────────
	case ListLoadMsg:
		return a, a.loadListData(msg)

	// ── List: data arrived ────────────────────────────────────────────────
	case ListDataMsg:
		if msg.Err != nil {
			cmds = append(cmds, tea.Println("[list error] "+appwriteErrVerbose(msg.Err)))
		}
		updated, cmd := a.list.Update(msg)
		a.list = updated
		cmds = append(cmds, cmd)

	// ── List: row selected → show detail ─────────────────────────────────
	case ListSelectMsg:
		a.activeDocID = msg.DocID
		a.activeEvent = msg.Event
		a.detail = NewDetailModel(msg.Event, msg.DocID, a.width, a.height)
		a.screen = ScreenDetail
		return a, a.detail.Init()

	// ── Detail: fetch data ────────────────────────────────────────────────
	case DetailLoadMsg:
		return a, a.loadDetailData(msg)

	// ── Detail: data arrived ──────────────────────────────────────────────
	case DetailDataMsg:
		if msg.Err != nil {
			cmds = append(cmds, tea.Println("[detail error] "+appwriteErrVerbose(msg.Err)))
		}
		updated, cmd := a.detail.Update(msg)
		a.detail = updated
		cmds = append(cmds, cmd)

	// ── Confirm payment ───────────────────────────────────────────────────
	case ConfirmActionMsg:
		a.modal = NewConfirmPaymentModal(msg.Event, msg.DocID, msg.Name, msg.Email, a.width, a.height)
		a.screen = ScreenModal
		return a, nil

	// ── Delete ────────────────────────────────────────────────────────────
	case DeleteActionMsg:
		a.modal = NewDeleteModal(msg.Event, msg.DocID, msg.Name, a.width, a.height)
		a.screen = ScreenModal
		return a, nil

	// ── Modal confirmed ───────────────────────────────────────────────────
	case ConfirmedMsg:
		switch msg.Kind {
		case ModalConfirmPayment:
			return a, a.doConfirmPayment(msg.Event, msg.DocID)
		case ModalDelete:
			return a, a.doDelete(msg.Event, msg.DocID)
		}

	// ── Modal cancelled ───────────────────────────────────────────────────
	case CancelledMsg:
		a.screen = ScreenDetail
		return a, nil

	// ── Add-to-group action (from detail screen) ──────────────────────────
	case AddToGroupMsg:
		return a, a.doAddToGroup(msg)

	// ── Reject merch payment ───────────────────────────────────────────────
	case RejectActionMsg:
		if msg.Event == EventMerch {
			return a, a.doRejectMerch(msg.DocID)
		}

	// ── Dispatch merch order ───────────────────────────────────────────────
	case DispatchActionMsg:
		if msg.Event == EventMerch {
			return a, a.doDispatchMerch(msg.DocID)
		}

	case addToGroupDoneMsg:
		if msg.err != nil {
			a.toast = "Add to group failed: " + msg.err.Error()
			a.toastErr = true
		} else {
			a.toast = msg.ok
			a.toastErr = false
		}
		// Reload detail to refresh group-membership badges.
		if a.activeDocID != "" {
			a.detail = NewDetailModel(a.activeEvent, a.activeDocID, a.width, a.height)
			a.screen = ScreenDetail
			cmds = append(cmds, a.detail.Init())
		}
		cmds = append(cmds, clearToastCmd())

	// ── Back navigation ───────────────────────────────────────────────────
	case BackMsg:
		switch a.screen {
		case ScreenDetail:
			a.screen = ScreenList
		case ScreenList:
			a.screen = ScreenMenu
		case ScreenCompose:
			a.screen = ScreenMenu
		case ScreenAnalyser:
			a.screen = ScreenMenu
		case ScreenMenu:
			return a, tea.Quit
		}

	// ── Action results ────────────────────────────────────────────────────
	case actionDoneMsg:
		if msg.err != nil {
			a.toast = "Error: " + msg.err.Error()
			a.toastErr = true
		} else {
			a.toast = msg.ok
			a.toastErr = false
		}
		// Reload detail after confirm/delete.
		if msg.reloadDetail && a.activeDocID != "" {
			a.detail = NewDetailModel(a.activeEvent, a.activeDocID, a.width, a.height)
			a.screen = ScreenDetail
			cmds = append(cmds, a.detail.Init())
		} else if msg.goList {
			a.list = NewListModel(a.activeEvent, a.width, a.height)
			a.screen = ScreenList
			cmds = append(cmds, a.list.Init())
		}
		cmds = append(cmds, clearToastCmd())

	case toastClearMsg:
		a.toast = ""

	// ── Download file ─────────────────────────────────────────────────────
	case DownloadFileMsg:
		return a, a.doDownload(msg)

	case downloadDoneMsg:
		if msg.err != nil {
			a.toast = "Download failed: " + msg.err.Error()
			a.toastErr = true
		} else {
			a.toast = "Saved: " + msg.savedPath
			a.toastErr = false
		}
		cmds = append(cmds, clearToastCmd())

	// ── Compose: send via Resend ───────────────────────────────────────────
	case composeSendViaResendMsg:
		return a, a.doSendResend(msg)

	// ── Compose: send via pop ─────────────────────────────────────────────
	case composeSendViaPopMsg:
		return a, a.doSendPop(msg)

	// ── Compose: Resend send result ───────────────────────────────────────
	case composeSendDoneMsg:
		updated, cmd := a.compose.Update(msg)
		a.compose = updated
		cmds = append(cmds, cmd)
	}

	// ── Route key events to active sub-model ──────────────────────────────
	switch a.screen {
	case ScreenLogin:
		updated, cmd := a.login.Update(msg)
		a.login = updated
		cmds = append(cmds, cmd)
	case ScreenSignup:
		updated, cmd := a.signup.Update(msg)
		a.signup = updated
		cmds = append(cmds, cmd)
	case ScreenVerify:
		updated, cmd := a.verify.Update(msg)
		a.verify = updated
		cmds = append(cmds, cmd)
	case ScreenMenu:
		updated, cmd := a.menu.Update(msg)
		a.menu = updated
		cmds = append(cmds, cmd)
	case ScreenList:
		updated, cmd := a.list.Update(msg)
		a.list = updated
		cmds = append(cmds, cmd)
	case ScreenDetail:
		updated, cmd := a.detail.Update(msg)
		a.detail = updated
		cmds = append(cmds, cmd)
	case ScreenModal:
		updated, cmd := a.modal.Update(msg)
		a.modal = updated
		cmds = append(cmds, cmd)
	case ScreenCompose:
		updated, cmd := a.compose.Update(msg)
		a.compose = updated
		cmds = append(cmds, cmd)
	case ScreenAnalyser:
		updated, cmd := a.analyser.Update(msg)
		a.analyser = updated
		cmds = append(cmds, cmd)
	}

	return a, tea.Batch(cmds...)
}

func (a App) View() tea.View {
	var content string

	switch a.screen {
	case ScreenLogin:
		content = a.login.View()
	case ScreenSignup:
		content = a.signup.View()
	case ScreenVerify:
		content = a.verify.View()
	case ScreenMenu:
		content = a.menu.View()
	case ScreenList:
		content = a.list.View()
	case ScreenDetail:
		content = a.detail.View()
	case ScreenModal:
		content = a.modal.View()
	case ScreenCompose:
		content = a.compose.View()
	case ScreenAnalyser:
		content = a.analyser.View()
	}

	if a.toast != "" {
		toastStyle := Success
		if a.toastErr {
			toastStyle = Error
		}
		content += "\n" + toastStyle.Render("  "+a.toast)
	}

	v := tea.NewView(content)
	v.AltScreen = true
	return v
}

// ── Internal command helpers ──────────────────────────────────────────────────

type loginDoneMsg struct {
	sess *session.Session
	err  error
}

type signupDoneMsg struct {
	sess *session.Session
	err  error
}

type verifyDoneMsg struct {
	err error
}

type actionDoneMsg struct {
	ok           string
	err          error
	reloadDetail bool
	goList       bool
}

type downloadDoneMsg struct {
	savedPath string
	err       error
}

func (a App) doLogin(emailAddr, password string) tea.Cmd {
	cfg := a.svc.Config()
	return func() (tm tea.Msg) {
		defer func() {
			if r := recover(); r != nil {
				tm = loginDoneMsg{err: fmt.Errorf("unexpected error: %v", r)}
			}
		}()
		result, err := aw.LoginWithEmail(cfg, emailAddr, password)
		if err != nil {
			return loginDoneMsg{err: err}
		}
		sess := &session.Session{
			SessionID:     result.SessionID,
			SessionSecret: result.SessionSecret,
			UserID:        result.UserID,
			UserEmail:     result.UserEmail,
			ExpiresAt:     result.ExpiresAt,
		}
		_ = session.Save(sess)
		return loginDoneMsg{sess: sess}
	}
}

func (a App) doOAuth() tea.Cmd {
	cfg := a.svc.Config()
	return func() (tm tea.Msg) {
		defer func() {
			if r := recover(); r != nil {
				tm = loginDoneMsg{err: fmt.Errorf("unexpected error: %v", r)}
			}
		}()
		result, err := aw.LoginWithOAuth(cfg, aw.OAuthGoogle)
		if err != nil {
			return loginDoneMsg{err: err}
		}
		sess := &session.Session{
			SessionID:     result.SessionID,
			SessionSecret: result.SessionSecret,
			UserID:        result.UserID,
			UserEmail:     result.UserEmail,
			ExpiresAt:     result.ExpiresAt,
		}
		_ = session.Save(sess)
		return loginDoneMsg{sess: sess}
	}
}

func (a App) doSignup(msg doSignupMsg) tea.Cmd {
	cfg := a.svc.Config()
	return func() (tm tea.Msg) {
		defer func() {
			if r := recover(); r != nil {
				tm = signupDoneMsg{err: fmt.Errorf("unexpected error: %v", r)}
			}
		}()
		result, err := aw.SignUp(cfg, msg.name, msg.email, msg.password)
		if err != nil {
			return signupDoneMsg{err: err}
		}
		sess := &session.Session{
			SessionID:     result.SessionID,
			SessionSecret: result.SessionSecret,
			UserID:        result.UserID,
			UserEmail:     result.UserEmail,
			ExpiresAt:     result.ExpiresAt,
		}
		_ = session.Save(sess)
		// Best-effort: send verification email. Ignore error — user can retry from the verify screen.
		_ = aw.SendEmailVerification(cfg, result.SessionSecret)
		return signupDoneMsg{sess: sess}
	}
}

func (a App) doVerify(msg doVerifyMsg) tea.Cmd {
	cfg := a.svc.Config()
	sessionSecret := a.sess.SessionSecret
	return func() (tm tea.Msg) {
		defer func() {
			if r := recover(); r != nil {
				tm = verifyDoneMsg{err: fmt.Errorf("unexpected error: %v", r)}
			}
		}()
		err := aw.ConfirmEmailVerification(cfg, sessionSecret, msg.userID, msg.secret)
		return verifyDoneMsg{err: err}
	}
}

func (a App) loadListData(msg ListLoadMsg) tea.Cmd {
	svc := a.svc
	return func() (tm tea.Msg) {
		defer func() {
			if r := recover(); r != nil {
				tm = ListDataMsg{Err: fmt.Errorf("unexpected error: %v", r)}
			}
		}()
		var rows []RegistrationRow
		var total int
		var err error

		switch msg.Event {
		case EventCTF:
			regs, t, e := svc.ListCTF(msg.Page, msg.Filter, msg.Search)
			err = e
			total = t
			for _, r := range regs {
				rows = append(rows, RegistrationRowFromCTF(r))
			}
		case EventSchoolHackathon:
			regs, t, e := svc.ListSchoolHackathon(msg.Page, msg.Search)
			err = e
			total = t
			for _, r := range regs {
				rows = append(rows, RegistrationRowFromSchoolHackathon(r))
			}
		case EventUniversityHackathon:
			regs, t, e := svc.ListUniversityHackathon(msg.Page, msg.Search)
			err = e
			total = t
			for _, r := range regs {
				rows = append(rows, RegistrationRowFromUniversityHackathon(r))
			}
		case EventDesignathon:
			regs, t, e := svc.ListDesignathon(msg.Page, msg.Search)
			err = e
			total = t
			for _, r := range regs {
				rows = append(rows, RegistrationRowFromDesignathon(r))
			}
		case EventMerch:
			orders, t, e := svc.ListMerch(msg.Page, msg.Filter, msg.Search)
			err = e
			total = t
			for _, o := range orders {
				rows = append(rows, RegistrationRowFromMerch(o))
			}
		}

		return ListDataMsg{Rows: rows, TotalDocs: total, Err: err}
	}
}

func (a App) loadDetailData(msg DetailLoadMsg) tea.Cmd {
	svc := a.svc
	wahaClient := a.waha
	cfg := svc.Config()
	return func() (tm tea.Msg) {
		defer func() {
			if r := recover(); r != nil {
				tm = DetailDataMsg{Err: fmt.Errorf("unexpected error: %v", r)}
			}
		}()
		var content, name, email, fileID, teamName string
		var err error
		var groupStatus map[string]bool
		var phones []string

		switch msg.Event {
		case EventCTF:
			r, e := svc.GetCTF(msg.DocID)
			if e != nil {
				err = e
			} else {
				phones = nonEmpty(r.LeaderWhatsapp, r.Member2Whatsapp, r.Member3Whatsapp, r.Member4Whatsapp)
				if wahaClient != nil && wahaClient.IsEnabled() && cfg.WAHACTFGroupID != "" {
					groupStatus, _ = wahaClient.CheckPhones(cfg.WAHACTFGroupID, phones)
				}
				content = RenderCTFDetail(r, groupStatus)
				name = r.LeaderName
				email = r.LeaderEmail
				fileID = r.PaymentSlipFileId
				teamName = r.TeamName
				if teamName == "" {
					teamName = r.LeaderName
				}
			}
		case EventSchoolHackathon:
			r, e := svc.GetSchoolHackathon(msg.DocID)
			if e != nil {
				err = e
			} else {
				phones = nonEmpty(r.LeaderContactNumber, r.Member2ContactNumber, r.Member3ContactNumber, r.Member4ContactNumber)
				if wahaClient != nil && wahaClient.IsEnabled() && cfg.WAHASchoolHackGroupID != "" {
					groupStatus, _ = wahaClient.CheckPhones(cfg.WAHASchoolHackGroupID, phones)
				}
				content = RenderSchoolHackathonDetail(r, groupStatus)
				name = r.LeaderFullName
				email = r.LeaderEmail
				fileID = r.TeamLogoFileId
				teamName = r.TeamName
			}
		case EventUniversityHackathon:
			r, e := svc.GetUniversityHackathon(msg.DocID)
			if e != nil {
				err = e
			} else {
				phones = nonEmpty(r.LeaderWhatsapp, r.Member2Whatsapp, r.Member3Whatsapp, r.Member4Whatsapp)
				if wahaClient != nil && wahaClient.IsEnabled() && cfg.WAHAUniHackGroupID != "" {
					groupStatus, _ = wahaClient.CheckPhones(cfg.WAHAUniHackGroupID, phones)
				}
				content = RenderUniversityHackathonDetail(r, groupStatus)
				name = r.LeaderName
				email = r.LeaderEmail
				fileID = r.TeamLogoFileId
				teamName = r.TeamName
			}
		case EventDesignathon:
			r, e := svc.GetDesignathon(msg.DocID)
			if e != nil {
				err = e
			} else {
				phones = nonEmpty(r.Member1Phone, r.Member2Phone, r.Member3Phone)
				if wahaClient != nil && wahaClient.IsEnabled() && cfg.WAHADesignathonGroupID != "" {
					groupStatus, _ = wahaClient.CheckPhones(cfg.WAHADesignathonGroupID, phones)
				}
				content = RenderDesignathonDetail(r, groupStatus)
				name = r.Member1FullName
				email = r.Member1Email
				fileID = r.TeamLogoFileId
				teamName = r.TeamName
			}
		case EventMerch:
			o, e := svc.GetMerch(msg.DocID)
			if e != nil {
				err = e
			} else {
				content = RenderMerchDetail(o)
				name = o.FullName
				email = o.Email
				fileID = o.PaymentSlipFileId
				teamName = o.FullName + "_" + o.ProductName
			}
		}

		return DetailDataMsg{
			Event:       msg.Event,
			DocID:       msg.DocID,
			Content:     content,
			Name:        name,
			Email:       email,
			FileID:      fileID,
			TeamName:    teamName,
			Phones:      phones,
			GroupStatus: groupStatus,
			Err:         err,
		}
	}
}

// nonEmpty returns only the non-empty strings from the provided list.
func nonEmpty(phones ...string) []string {
	var out []string
	for _, p := range phones {
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func (a App) doConfirmPayment(event EventType, docID string) tea.Cmd {
	// For merch, delegate entirely to the richer doConfirmMerchCmd which
	// already has access to the App receiver and handles both pre-order and
	// full-payment cases.
	if event == EventMerch {
		return a.doConfirmMerchCmd(docID)
	}

	svc := a.svc
	cfg := svc.Config()
	return func() (tm tea.Msg) {
		defer func() {
			if r := recover(); r != nil {
				tm = actionDoneMsg{err: fmt.Errorf("unexpected error: %v", r), reloadDetail: true}
			}
		}()
		if event == EventCTF {
			r, err := svc.ConfirmCTF(docID)
			if err != nil {
				return actionDoneMsg{err: err, reloadDetail: true}
			}
			emailErr := email.SendConfirmation(cfg, email.ConfirmationData{
				EventName:        "CTF",
				RecipientName:    r.LeaderName,
				RecipientEmail:   r.LeaderEmail,
				TeamName:         r.TeamName,
				RegistrationType: r.RegistrationType,
				ConfirmedAt:      time.Now().Format("02 Jan 2006, 15:04 MST"),
			})
			if emailErr != nil {
				return actionDoneMsg{
					ok:           fmt.Sprintf("Confirmed! Email to %s failed: %v", r.LeaderEmail, emailErr),
					reloadDetail: true,
				}
			}
			return actionDoneMsg{
				ok:           fmt.Sprintf("✓ Confirmed & email sent to %s", r.LeaderEmail),
				reloadDetail: true,
			}
		}
		return actionDoneMsg{err: fmt.Errorf("payment confirmation not available for this event"), reloadDetail: true}
	}
}

func (a App) doDelete(event EventType, docID string) tea.Cmd {
	svc := a.svc
	return func() (tm tea.Msg) {
		defer func() {
			if r := recover(); r != nil {
				tm = actionDoneMsg{err: fmt.Errorf("unexpected error: %v", r), goList: true}
			}
		}()
		var err error
		switch event {
		case EventCTF:
			err = svc.DeleteCTF(docID)
		case EventSchoolHackathon:
			err = svc.DeleteSchoolHackathon(docID)
		case EventUniversityHackathon:
			err = svc.DeleteUniversityHackathon(docID)
		case EventDesignathon:
			err = svc.DeleteDesignathon(docID)
		case EventMerch:
			err = svc.DeleteMerch(docID)
		}
		if err != nil {
			return actionDoneMsg{err: err, goList: true}
		}
		return actionDoneMsg{ok: "✓ Registration deleted", goList: true}
	}
}

func (a App) doDownload(msg DownloadFileMsg) tea.Cmd {
	svc := a.svc
	return func() (tm tea.Msg) {
		defer func() {
			if r := recover(); r != nil {
				tm = downloadDoneMsg{err: fmt.Errorf("unexpected error: %v", r)}
			}
		}()
		var data []byte
		var err error

		switch msg.Event {
		case EventCTF:
			data, err = svc.DownloadPaymentSlip(msg.FileID)
		case EventSchoolHackathon:
			data, err = svc.DownloadSchoolHackathonLogo(msg.FileID)
		case EventUniversityHackathon:
			data, err = svc.DownloadUniversityHackathonLogo(msg.FileID)
		case EventDesignathon:
			data, err = svc.DownloadTeamLogo(msg.FileID)
		case EventMerch:
			data, err = svc.DownloadMerchPaymentSlip(msg.FileID)
		default:
			return downloadDoneMsg{err: fmt.Errorf("no downloadable file for this event type")}
		}

		if err != nil {
			return downloadDoneMsg{err: err}
		}

		// Determine suffix: receipt for CTF payment slips, logo for others.
		suffix := "receipt"
		if msg.Event != EventCTF {
			suffix = "logo"
		}

		// Sanitise team name for use as a filename.
		safeName := strings.Map(func(r rune) rune {
			if r == '/' || r == '\\' || r == ':' || r == '*' ||
				r == '?' || r == '"' || r == '<' || r == '>' || r == '|' {
				return '_'
			}
			return r
		}, msg.TeamName)
		if safeName == "" {
			safeName = "file"
		}

		ext := detectExt(data)
		filename := safeName + "_" + suffix + ext

		home, _ := os.UserHomeDir()
		savePath := filepath.Join(home, "Downloads", filename)
		if writeErr := os.WriteFile(savePath, data, 0o644); writeErr != nil {
			return downloadDoneMsg{err: writeErr}
		}
		return downloadDoneMsg{savedPath: savePath}
	}
}

// detectExt returns a file extension (e.g. ".pdf", ".png") by inspecting
// the first few magic bytes of data. Returns "" if unrecognised.
func detectExt(data []byte) string {
	if len(data) < 4 {
		return ""
	}
	switch {
	case len(data) >= 4 && string(data[:4]) == "%PDF":
		return ".pdf"
	case len(data) >= 8 && string(data[1:4]) == "PNG":
		return ".png"
	case len(data) >= 3 && data[0] == 0xFF && data[1] == 0xD8 && data[2] == 0xFF:
		return ".jpg"
	case len(data) >= 6 && (string(data[:6]) == "GIF87a" || string(data[:6]) == "GIF89a"):
		return ".gif"
	case len(data) >= 4 && data[0] == 0x89 && data[1] == 0x50 && data[2] == 0x4E && data[3] == 0x47:
		return ".png"
	}
	return ""
}

func clearToastCmd() tea.Cmd {
	return tea.Tick(4*time.Second, func(time.Time) tea.Msg {
		return toastClearMsg{}
	})
}

// doSendResend calls the Resend API from a background goroutine.
func (a App) doSendResend(msg composeSendViaResendMsg) tea.Cmd {
	cfg := a.svc.Config()
	return func() tea.Msg {
		// Split To by commas.
		var recipients []string
		for _, t := range strings.Split(msg.to, ",") {
			t = strings.TrimSpace(t)
			if t != "" {
				recipients = append(recipients, t)
			}
		}
		if len(recipients) == 0 {
			return composeSendDoneMsg{err: fmt.Errorf("no recipients specified")}
		}

		// Determine sender address from first recipient domain or use default.
		from := "CryptX <info@cryptx.lk>"

		err := email.SendCustomEmail(cfg, email.CustomEmailData{
			From:        from,
			To:          recipients,
			Subject:     msg.subject,
			HTML:        msg.body,
			Attachments: msg.attachments,
		})
		return composeSendDoneMsg{err: err}
	}
}

// doSendPop suspends the TUI, hands control to pop, then resumes.
func (a App) doSendPop(msg composeSendViaPopMsg) tea.Cmd {
	// Build the pop command upfront (before suspending the TUI).
	popCmd, tmpFiles, err := buildPopCmd(msg.to, msg.subject, msg.body, msg.attach)
	if err != nil {
		// Can't build command — return an error without suspending.
		return func() tea.Msg {
			return composeSendDoneMsg{err: err}
		}
	}

	return tea.ExecProcess(popCmd, func(execErr error) tea.Msg {
		// Clean up temp attachment files after pop exits.
		for _, p := range tmpFiles {
			os.Remove(p)
		}
		return composeSendDoneMsg{err: execErr}
	})
}


// ConfirmCTFRegistration is the exported helper to confirm and email a CTF registration
// directly (used outside TUI context if needed).
func ConfirmCTFRegistration(svc *aw.Services, r *models.CTFRegistration) error {
	cfg := svc.Config()
	_, err := svc.ConfirmCTF(r.ID)
	if err != nil {
		return err
	}
	return email.SendConfirmation(cfg, email.ConfirmationData{
		EventName:        "CTF",
		RecipientName:    r.LeaderName,
		RecipientEmail:   r.LeaderEmail,
		TeamName:         r.TeamName,
		RegistrationType: r.RegistrationType,
	})
}

// ── Merch action helpers ───────────────────────────────────────────────────────


func (a App) doConfirmMerchCmd(docID string) tea.Cmd {
	svc := a.svc
	cfg := svc.Config()
	return func() (tm tea.Msg) {
		defer func() {
			if r := recover(); r != nil {
				tm = actionDoneMsg{err: fmt.Errorf("unexpected error: %v", r), reloadDetail: true}
			}
		}()
		// Fetch to determine current status.
		o, err := svc.GetMerch(docID)
		if err != nil {
			return actionDoneMsg{err: err, reloadDetail: true}
		}
		switch o.PaymentStatus {
		case models.MerchStatusPendingPreOrder:
			updated, err := svc.ConfirmMerchPreOrder(docID)
			if err != nil {
				return actionDoneMsg{err: err, reloadDetail: true}
			}
			emailErr := email.SendMerchPreOrderConfirm(cfg, email.MerchEmailData{
				RecipientName:  updated.FullName,
				RecipientEmail: updated.Email,
				ProductName:    updated.ProductName,
				Size:           updated.Size,
				Quantity:       updated.Quantity,
				TotalPrice:     updated.TotalPrice,
				PaymentOption:  updated.PaymentOption,
				DeliveryMethod: updated.DeliveryMethod,
				DocID:          updated.ID,
			})
			if emailErr != nil {
				return actionDoneMsg{
					ok:           fmt.Sprintf("Pre-order confirmed! Email to %s failed: %v", updated.Email, emailErr),
					reloadDetail: true,
				}
			}
			return actionDoneMsg{
				ok:           fmt.Sprintf("✓ Pre-order confirmed & email sent to %s", updated.Email),
				reloadDetail: true,
			}
		case models.MerchStatusPendingFullPayment:
			updated, err := svc.ConfirmMerchFullPayment(docID)
			if err != nil {
				return actionDoneMsg{err: err, reloadDetail: true}
			}
			emailErr := email.SendMerchFullPaymentConfirm(cfg, email.MerchEmailData{
				RecipientName:  updated.FullName,
				RecipientEmail: updated.Email,
				ProductName:    updated.ProductName,
				Size:           updated.Size,
				Quantity:       updated.Quantity,
				TotalPrice:     updated.TotalPrice,
				PaymentOption:  updated.PaymentOption,
				DeliveryMethod: updated.DeliveryMethod,
				DocID:          updated.ID,
			})
			if emailErr != nil {
				return actionDoneMsg{
					ok:           fmt.Sprintf("Full payment confirmed! Email to %s failed: %v", updated.Email, emailErr),
					reloadDetail: true,
				}
			}
			return actionDoneMsg{
				ok:           fmt.Sprintf("✓ Full payment confirmed & email sent to %s", updated.Email),
				reloadDetail: true,
			}
		default:
			return actionDoneMsg{
				err:          fmt.Errorf("order is in state %q — nothing to confirm", o.PaymentStatus),
				reloadDetail: true,
			}
		}
	}
}

func (a App) doRejectMerch(docID string) tea.Cmd {
	svc := a.svc
	return func() (tm tea.Msg) {
		defer func() {
			if r := recover(); r != nil {
				tm = actionDoneMsg{err: fmt.Errorf("unexpected error: %v", r), reloadDetail: true}
			}
		}()
		o, err := svc.GetMerch(docID)
		if err != nil {
			return actionDoneMsg{err: err, reloadDetail: true}
		}
		switch o.PaymentStatus {
		case models.MerchStatusPendingPreOrder:
			if _, err := svc.RejectMerchPreOrder(docID); err != nil {
				return actionDoneMsg{err: err, reloadDetail: true}
			}
			return actionDoneMsg{ok: "✓ Pre-order payment rejected", reloadDetail: true}
		case models.MerchStatusPendingFullPayment:
			if _, err := svc.RejectMerchFullPayment(docID); err != nil {
				return actionDoneMsg{err: err, reloadDetail: true}
			}
			return actionDoneMsg{ok: "✓ Full payment rejected", reloadDetail: true}
		default:
			return actionDoneMsg{
				err:          fmt.Errorf("order is in state %q — nothing to reject", o.PaymentStatus),
				reloadDetail: true,
			}
		}
	}
}

func (a App) doDispatchMerch(docID string) tea.Cmd {
	svc := a.svc
	cfg := svc.Config()
	return func() (tm tea.Msg) {
		defer func() {
			if r := recover(); r != nil {
				tm = actionDoneMsg{err: fmt.Errorf("unexpected error: %v", r), reloadDetail: true}
			}
		}()
		o, err := svc.GetMerch(docID)
		if err != nil {
			return actionDoneMsg{err: err, reloadDetail: true}
		}
		if !o.CanBeDispatched() {
			return actionDoneMsg{
				err:          fmt.Errorf("order cannot be dispatched: current status is %q", o.PaymentStatus),
				reloadDetail: true,
			}
		}
		if _, err := svc.DispatchMerch(docID); err != nil {
			return actionDoneMsg{err: err, reloadDetail: true}
		}
		// Send the dispatch email.
		emailData := email.MerchEmailData{
			RecipientName:  o.FullName,
			RecipientEmail: o.Email,
			ProductName:    o.ProductName,
			Size:           o.Size,
			Quantity:       o.Quantity,
			TotalPrice:     o.TotalPrice,
			DeliveryMethod: o.DeliveryMethod,
			DocID:          o.ID,
			// Collection details come from env vars (set in config).
			EventDate: cfg.MerchEventDate,
			EventTime: cfg.MerchEventTime,
			Venue:     cfg.MerchEventVenue,
		}
		emailErr := email.SendMerchDispatch(cfg, emailData)
		if emailErr != nil {
			return actionDoneMsg{
				ok:           fmt.Sprintf("Dispatched! Email to %s failed: %v", o.Email, emailErr),
				reloadDetail: true,
			}
		}
		return actionDoneMsg{
			ok:           fmt.Sprintf("✓ Order dispatched & email sent to %s", o.Email),
			reloadDetail: true,
		}
	}
}


// ── Add-to-group ──────────────────────────────────────────────────────────────

type addToGroupDoneMsg struct {
	ok  string
	err error
}

func (a App) doAddToGroup(msg AddToGroupMsg) tea.Cmd {
	wahaClient := a.waha
	cfg := a.svc.Config()
	return func() (tm tea.Msg) {
		defer func() {
			if r := recover(); r != nil {
				tm = addToGroupDoneMsg{err: fmt.Errorf("unexpected error: %v", r)}
			}
		}()
		if wahaClient == nil || !wahaClient.IsEnabled() {
			return addToGroupDoneMsg{err: fmt.Errorf("WAHA is not configured")}
		}
		var groupID string
		switch msg.Event {
		case EventCTF:
			groupID = cfg.WAHACTFGroupID
		case EventSchoolHackathon:
			groupID = cfg.WAHASchoolHackGroupID
		case EventUniversityHackathon:
			groupID = cfg.WAHAUniHackGroupID
		case EventDesignathon:
			groupID = cfg.WAHADesignathonGroupID
		}
		if groupID == "" {
			return addToGroupDoneMsg{err: fmt.Errorf("no WhatsApp group configured for this event")}
		}
		if err := wahaClient.AddParticipants(groupID, msg.Phones); err != nil {
			return addToGroupDoneMsg{err: err}
		}
		return addToGroupDoneMsg{ok: fmt.Sprintf("✓ Added %d number(s) to group", len(msg.Phones))}
	}
}

// ── Group analyser ────────────────────────────────────────────────────────────

func (a App) doAnalyse() tea.Cmd {
	svc := a.svc
	wahaClient := a.waha
	cfg := svc.Config()
	return func() (tm tea.Msg) {
		defer func() {
			if r := recover(); r != nil {
				tm = analyserDoneMsg{err: fmt.Errorf("unexpected error: %v", r)}
			}
		}()

		type eventSpec struct {
			event   EventType
			label   string
			groupID string
		}
		specs := []eventSpec{
			{EventCTF, "CTF Registrations", cfg.WAHACTFGroupID},
			{EventSchoolHackathon, "School Hackathon", cfg.WAHASchoolHackGroupID},
			{EventUniversityHackathon, "University Hackathon", cfg.WAHAUniHackGroupID},
			{EventDesignathon, "Designathon", cfg.WAHADesignathonGroupID},
		}

		var evReports []EventGroupReport

		for _, spec := range specs {
			ev := EventGroupReport{Label: spec.label, GroupID: spec.groupID}

			if spec.groupID == "" {
				ev.Err = "no group ID configured"
				evReports = append(evReports, ev)
				continue
			}
			if wahaClient == nil || !wahaClient.IsEnabled() {
				ev.Err = "WAHA not configured"
				evReports = append(evReports, ev)
				continue
			}

			// One API call per group to fetch the full participant set.
			participantSet, err := wahaClient.GetParticipantSet(spec.groupID)
			if err != nil {
				ev.Err = err.Error()
				evReports = append(evReports, ev)
				continue
			}

			// Paginate through all registrations.
			page := 0
			fetched := 0
			for {
				var newTeams []TeamGroupResult
				var total int
				var pageErr error

				switch spec.event {
				case EventCTF:
					regs, t, e := svc.ListCTF(page, "", "")
					total, pageErr = t, e
					for _, r := range regs {
						tr := TeamGroupResult{DocID: r.ID, TeamName: r.DisplayName()}
						for _, pair := range []struct{ name, phone string }{
							{r.LeaderName, r.LeaderWhatsapp},
							{r.Member2Name, r.Member2Whatsapp},
							{r.Member3Name, r.Member3Whatsapp},
							{r.Member4Name, r.Member4Whatsapp},
						} {
							if pair.phone == "" {
								continue
							}
							tr.Members = append(tr.Members, MemberGroupResult{
								Name:    pair.name,
								Phone:   pair.phone,
								InGroup: participantSet[waha.NormalizePhone(pair.phone)],
							})
						}
						newTeams = append(newTeams, tr)
					}

				case EventSchoolHackathon:
					regs, t, e := svc.ListSchoolHackathon(page, "")
					total, pageErr = t, e
					for _, r := range regs {
						tr := TeamGroupResult{DocID: r.ID, TeamName: r.DisplayName()}
						for _, pair := range []struct{ name, phone string }{
							{r.LeaderFullName, r.LeaderContactNumber},
							{r.Member2FullName, r.Member2ContactNumber},
							{r.Member3FullName, r.Member3ContactNumber},
							{r.Member4FullName, r.Member4ContactNumber},
						} {
							if pair.phone == "" {
								continue
							}
							tr.Members = append(tr.Members, MemberGroupResult{
								Name:    pair.name,
								Phone:   pair.phone,
								InGroup: participantSet[waha.NormalizePhone(pair.phone)],
							})
						}
						newTeams = append(newTeams, tr)
					}

				case EventUniversityHackathon:
					regs, t, e := svc.ListUniversityHackathon(page, "")
					total, pageErr = t, e
					for _, r := range regs {
						tr := TeamGroupResult{DocID: r.ID, TeamName: r.DisplayName()}
						for _, pair := range []struct{ name, phone string }{
							{r.LeaderName, r.LeaderWhatsapp},
							{r.Member2Name, r.Member2Whatsapp},
							{r.Member3Name, r.Member3Whatsapp},
							{r.Member4Name, r.Member4Whatsapp},
						} {
							if pair.phone == "" {
								continue
							}
							tr.Members = append(tr.Members, MemberGroupResult{
								Name:    pair.name,
								Phone:   pair.phone,
								InGroup: participantSet[waha.NormalizePhone(pair.phone)],
							})
						}
						newTeams = append(newTeams, tr)
					}

				case EventDesignathon:
					regs, t, e := svc.ListDesignathon(page, "")
					total, pageErr = t, e
					for _, r := range regs {
						tr := TeamGroupResult{DocID: r.ID, TeamName: r.DisplayName()}
						for _, pair := range []struct{ name, phone string }{
							{r.Member1FullName, r.Member1Phone},
							{r.Member2FullName, r.Member2Phone},
							{r.Member3FullName, r.Member3Phone},
						} {
							if pair.phone == "" {
								continue
							}
							tr.Members = append(tr.Members, MemberGroupResult{
								Name:    pair.name,
								Phone:   pair.phone,
								InGroup: participantSet[waha.NormalizePhone(pair.phone)],
							})
						}
						newTeams = append(newTeams, tr)
					}
				}

				if pageErr != nil {
					ev.Err = pageErr.Error()
					break
				}

				ev.Teams = append(ev.Teams, newTeams...)
				fetched += len(newTeams)
				if fetched >= total || len(newTeams) == 0 {
					break
				}
				page++
			}

			evReports = append(evReports, ev)
		}

		return analyserDoneMsg{report: BuildGroupReport(evReports)}
	}
}

// appwriteErrVerbose returns an error string with Appwrite HTTP status code and
// raw response body when the error originates from the Appwrite SDK.
func appwriteErrVerbose(err error) string {
	var ae *awclient.AppwriteError
	if errors.As(err, &ae) {
		return fmt.Sprintf("%s | HTTP %d | response: %s",
			ae.GetMessage(), ae.GetStatusCode(), ae.GetResponse())
	}
	return err.Error()
}

