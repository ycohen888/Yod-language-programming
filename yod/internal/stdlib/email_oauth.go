package stdlib

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/smtp"
	"os"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"

	"yod/internal/object"
)

const gmailOAuthScope = "https://mail.google.com/"

type emailOAuthTokenFile struct {
	AccessToken  string    `json:"access_token"`
	TokenType    string    `json:"token_type"`
	RefreshToken string    `json:"refresh_token"`
	Expiry       time.Time `json:"expiry"`
}

func emailGmailOAuthConnect(args ...object.Object) object.Object {
	if len(args) != 1 {
		return errObj("אימייל.גימייל_oauth_התחבר מצפה למילון")
	}
	h, ok := args[0].(*object.Hash)
	if !ok {
		return errObj("גימייל_oauth_התחבר מצפה למילון")
	}
	clientID := hashStr(h, "מזהה_לקוח")
	if clientID == "" {
		clientID = hashStr(h, "client_id")
	}
	secret := hashStr(h, "סוד")
	if secret == "" {
		secret = hashStr(h, "client_secret")
	}
	tokenPath := hashStr(h, "קובץ_טוקן")
	if tokenPath == "" {
		tokenPath = "gmail-token.json"
	}
	tokenPath = resolveAppPath(tokenPath)
	if clientID == "" || secret == "" {
		return errObj("חסרים מזהה_לקוח וסוד (מ־Google Cloud Console)")
	}

	cfg := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: secret,
		Endpoint:     google.Endpoint,
		Scopes:       []string{gmailOAuthScope},
		RedirectURL:  "http://127.0.0.1:8765/oauth2callback",
	}

	codeCh := make(chan string, 1)
	errCh := make(chan error, 1)
	mux := http.NewServeMux()
	srv := &http.Server{Addr: "127.0.0.1:8765", Handler: mux}
	mux.HandleFunc("/oauth2callback", func(w http.ResponseWriter, r *http.Request) {
		if e := r.URL.Query().Get("error"); e != "" {
			errCh <- fmt.Errorf("%s", e)
			fmt.Fprint(w, "שגיאה באימות. אפשר לסגור את החלון.")
			return
		}
		code := r.URL.Query().Get("code")
		if code == "" {
			errCh <- fmt.Errorf("חסר קוד אימות")
			fmt.Fprint(w, "חסר קוד. אפשר לסגור.")
			return
		}
		codeCh <- code
		fmt.Fprint(w, "האימות הצליח. אפשר לסגור את החלון ולחזור ליוד.")
	})

	go func() {
		_ = srv.ListenAndServe()
	}()
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
	}()

	authURL := cfg.AuthCodeURL("yod-state", oauth2.AccessTypeOffline, oauth2.ApprovalForce)
	_ = openBrowserURL(authURL)

	var code string
	select {
	case code = <-codeCh:
	case err := <-errCh:
		return errObj("OAuth נכשל: " + err.Error())
	case <-time.After(5 * time.Minute):
		return errObj("OAuth פג זמן — נסו שוב")
	}

	tok, err := cfg.Exchange(context.Background(), code)
	if err != nil {
		return errObj("החלפת קוד OAuth נכשלה: " + err.Error())
	}
	if err := emailSaveOAuthToken(tokenPath, tok); err != nil {
		return errObj("שמירת טוקן נכשלה: " + err.Error())
	}
	return &object.String{Value: tokenPath}
}

func emailSaveOAuthToken(path string, tok *oauth2.Token) error {
	f := emailOAuthTokenFile{
		AccessToken:  tok.AccessToken,
		TokenType:    tok.TokenType,
		RefreshToken: tok.RefreshToken,
		Expiry:       tok.Expiry,
	}
	data, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

func emailLoadOAuthToken(path string) (*oauth2.Token, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var f emailOAuthTokenFile
	if err := json.Unmarshal(data, &f); err != nil {
		return nil, err
	}
	tok := &oauth2.Token{
		AccessToken:  f.AccessToken,
		TokenType:    f.TokenType,
		RefreshToken: f.RefreshToken,
		Expiry:       f.Expiry,
	}
	if tok.AccessToken == "" {
		return nil, fmt.Errorf("קובץ טוקן ריק")
	}
	// רענון אם יש refresh + פג תוקף
	if tok.RefreshToken != "" && tok.Expiry.Before(time.Now().Add(time.Minute)) {
		clientID := os.Getenv("GMAIL_CLIENT_ID")
		secret := os.Getenv("GMAIL_CLIENT_SECRET")
		if clientID != "" && secret != "" {
			cfg := &oauth2.Config{
				ClientID:     clientID,
				ClientSecret: secret,
				Endpoint:     google.Endpoint,
				Scopes:       []string{gmailOAuthScope},
			}
			src := cfg.TokenSource(context.Background(), tok)
			nt, err := src.Token()
			if err == nil {
				_ = emailSaveOAuthToken(path, nt)
				return nt, nil
			}
		}
	}
	return tok, nil
}

// XOAUTH2 for SMTP
type xoauth2Auth struct {
	user, token string
}

func emailXOAuth2Auth(user, accessToken string) smtp.Auth {
	return &xoauth2Auth{user: user, token: accessToken}
}

func (a *xoauth2Auth) Start(server *smtp.ServerInfo) (string, []byte, error) {
	resp := fmt.Sprintf("user=%s\x01auth=Bearer %s\x01\x01", a.user, a.token)
	return "XOAUTH2", []byte(resp), nil
}

func (a *xoauth2Auth) Next(fromServer []byte, more bool) ([]byte, error) {
	if more {
		return nil, fmt.Errorf("שרת SMTP דחה XOAUTH2: %s", string(fromServer))
	}
	return nil, nil
}

// IMAP SASL XOAUTH2
type xoauth2SASL struct {
	user, token string
}

func (a *xoauth2SASL) Start() (mech string, ir []byte, err error) {
	resp := "user=" + a.user + "\x01auth=Bearer " + a.token + "\x01\x01"
	return "XOAUTH2", []byte(resp), nil
}

func (a *xoauth2SASL) Next(challenge []byte) ([]byte, error) {
	return nil, fmt.Errorf("XOAUTH2 נדחה: %s", string(challenge))
}

func openBrowserURL(url string) error {
	return runOpenURL(url)
}
