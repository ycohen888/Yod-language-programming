package stdlib

import (
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net"
	"net/smtp"
	"net/textproto"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"yod/internal/object"
)

func NewEmailModule() *object.Module {
	m := &object.Module{Name: "אימייל", Attrs: map[string]object.Object{}}
	m.Attrs["שלח"] = &object.Builtin{Fn: emailSend}
	m.Attrs["גימייל"] = &object.Builtin{Fn: emailGmail}
	registerEmailAsync(m)
	m.Attrs["פתח_תיבה"] = &object.Builtin{Fn: emailOpenMailbox}
	m.Attrs["תיבת_גימייל"] = &object.Builtin{Fn: emailGmailMailbox}
	m.Attrs["גימייל_oauth_התחבר"] = &object.Builtin{Fn: emailGmailOAuthConnect}
	return m
}

// emailGmail — שליחה דרך SMTP של Gmail (סיסמת אפליקציה, לא סיסמת החשבון הרגילה).
func emailGmail(args ...object.Object) object.Object {
	if len(args) != 1 {
		return errObj("אימייל.גימייל מצפה למילון אפשרויות אחד")
	}
	h, ok := args[0].(*object.Hash)
	if !ok {
		return errObj("אימייל.גימייל מצפה למילון (משתמש, סיסמה, אל, נושא, …)")
	}
	prepared := emailApplyGmailDefaults(h)
	if err, ok := prepared.(*object.Error); ok {
		return err
	}
	return emailSendFromHash(prepared.(*object.Hash))
}

func emailSend(args ...object.Object) object.Object {
	if len(args) != 1 {
		return errObj("אימייל.שלח מצפה למילון אפשרויות אחד")
	}
	h, ok := args[0].(*object.Hash)
	if !ok {
		return errObj("אימייל.שלח מצפה למילון (שרת, מאת, אל, נושא, …)")
	}
	// קיצור: ספק: "גימייל" ממלא שרת/פורט של Gmail
	if sp := strings.TrimSpace(hashStr(h, "ספק")); sp != "" {
		if emailIsGmailProvider(sp) {
			prepared := emailApplyGmailDefaults(h)
			if err, ok := prepared.(*object.Error); ok {
				return err
			}
			h = prepared.(*object.Hash)
		}
	}
	return emailSendFromHash(h)
}

func emailIsGmailProvider(s string) bool {
	s = strings.ToLower(s)
	return s == "גימייל" || s == "gmail" || s == "google"
}

// emailApplyGmailDefaults ממלא smtp.gmail.com ומאת מהמשתמש; מחזיר Hash או Error.
func emailApplyGmailDefaults(src *object.Hash) object.Object {
	user := hashStr(src, "משתמש")
	if user == "" {
		user = hashStr(src, "מאת")
	}
	if user == "" {
		return errObj("לגימייל חובה משתמש (כתובת Gmail) או מאת")
	}
	pass := hashStr(src, "סיסמה")
	oauthPath := hashStr(src, "oauth_טוקן")
	if oauthPath == "" {
		oauthPath = hashStr(src, "oauth")
	}
	if pass == "" && oauthPath == "" {
		return errObj("לגימייל חובה סיסמת אפליקציה או oauth_טוקן")
	}
	// סיסמאות אפליקציה של Google לעיתים עם רווחים להעתקה
	pass = strings.ReplaceAll(pass, " ", "")

	toList, errMsg := emailRecipients(src, "אל")
	if errMsg != "" {
		return errObj(errMsg)
	}
	if len(toList) == 0 {
		return errObj("חסר נמען (מפתח \"אל\")")
	}

	out := object.NewHash()
	out.Set("שרת", &object.String{Value: "smtp.gmail.com"})
	port := hashInt(src, "פורט", 587)
	useSSL := hashBool(src, "ssl", false)
	if port == 465 {
		useSSL = true
	}
	out.Set("פורט", &object.Number{Value: float64(port)})
	if useSSL {
		out.Set("ssl", &object.Boolean{Value: true})
	}
	out.Set("משתמש", &object.String{Value: user})
	out.Set("סיסמה", &object.String{Value: pass})
	from := hashStr(src, "מאת")
	if from == "" {
		from = user
	}
	out.Set("מאת", &object.String{Value: from})
	if len(toList) == 1 {
		out.Set("אל", &object.String{Value: toList[0]})
	} else {
		els := make([]object.Object, len(toList))
		for i, t := range toList {
			els[i] = &object.String{Value: t}
		}
		out.Set("אל", &object.Array{Elements: els})
	}
	if s := hashStr(src, "נושא"); s != "" {
		out.Set("נושא", &object.String{Value: s})
	}
	if s := hashStr(src, "תוכן"); s != "" {
		out.Set("תוכן", &object.String{Value: s})
	}
	if s := hashStr(src, "תוכן_HTML"); s != "" {
		out.Set("תוכן_HTML", &object.String{Value: s})
	} else if s := hashStr(src, "html"); s != "" {
		out.Set("תוכן_HTML", &object.String{Value: s})
	}
	if v, ok := src.Pairs["קבצים"]; ok {
		out.Set("קבצים", v)
	}
	if s := hashStr(src, "oauth_טוקן"); s != "" {
		out.Set("oauth_טוקן", &object.String{Value: s})
	} else if s := hashStr(src, "oauth"); s != "" {
		out.Set("oauth_טוקן", &object.String{Value: s})
	}
	return out
}

func emailSendFromHash(h *object.Hash) object.Object {
	host := hashStr(h, "שרת")
	if host == "" {
		return errObj("חסר שרת SMTP (מפתח \"שרת\")")
	}
	from := hashStr(h, "מאת")
	if from == "" {
		return errObj("חסר שולח (מפתח \"מאת\")")
	}
	toList, errMsg := emailRecipients(h, "אל")
	if errMsg != "" {
		return errObj(errMsg)
	}
	if len(toList) == 0 {
		return errObj("חסר נמען (מפתח \"אל\")")
	}
	subject := hashStr(h, "נושא")
	body := hashStr(h, "תוכן")
	htmlBody := hashStr(h, "תוכן_HTML")
	if htmlBody == "" {
		htmlBody = hashStr(h, "html")
	}
	user := hashStr(h, "משתמש")
	pass := strings.ReplaceAll(hashStr(h, "סיסמה"), " ", "")
	port := hashInt(h, "פורט", 587)
	useSSL := hashBool(h, "ssl", false)
	if port == 465 {
		useSSL = true
	}

	oauthTokenPath := hashStr(h, "oauth_טוקן")
	if oauthTokenPath == "" {
		oauthTokenPath = hashStr(h, "oauth")
	}

	attachments, errMsg := emailAttachmentPaths(h)
	if errMsg != "" {
		return errObj(errMsg)
	}

	msg, err := buildMIMEMessage(from, toList, subject, body, htmlBody, attachments)
	if err != nil {
		return errObj("בניית הודעה נכשלה: " + err.Error())
	}

	addr := net.JoinHostPort(host, strconv.Itoa(port))
	var auth smtp.Auth
	if oauthTokenPath != "" {
		tok, err := emailLoadOAuthToken(resolveAppPath(oauthTokenPath))
		if err != nil {
			return errObj("טעינת טוקן OAuth נכשלה: " + err.Error())
		}
		if user == "" {
			user = from
		}
		auth = emailXOAuth2Auth(user, tok.AccessToken)
	} else if user != "" {
		auth = smtp.PlainAuth("", user, pass, host)
	}

	var sendErr error
	if useSSL {
		sendErr = sendMailTLS(addr, host, auth, from, toList, msg)
	} else {
		sendErr = sendMailStartTLS(addr, host, auth, from, toList, msg)
	}
	if sendErr != nil {
		msg := "שליחת אימייל נכשלה: " + sendErr.Error()
		if strings.Contains(strings.ToLower(host), "gmail") {
			msg += " — ל־Gmail נדרשת סיסמת אפליקציה (חשבון Google ← אבטחה ← אימות דו־שלבי ← סיסמאות אפליקציות)"
		}
		return errObj(msg)
	}
	return &object.Null{}
}

func hashStr(h *object.Hash, key string) string {
	v, ok := h.Pairs[key]
	if !ok {
		return ""
	}
	if s, ok := asString(v); ok {
		return s
	}
	return v.Inspect()
}

func hashInt(h *object.Hash, key string, def int) int {
	v, ok := h.Pairs[key]
	if !ok {
		return def
	}
	if n, ok := v.(*object.Number); ok {
		return int(n.Value)
	}
	return def
}

func hashBool(h *object.Hash, key string, def bool) bool {
	v, ok := h.Pairs[key]
	if !ok {
		return def
	}
	if b, ok := v.(*object.Boolean); ok {
		return b.Value
	}
	return def
}

func emailRecipients(h *object.Hash, key string) ([]string, string) {
	v, ok := h.Pairs[key]
	if !ok {
		return nil, ""
	}
	if s, ok := asString(v); ok {
		parts := strings.Split(s, ",")
		out := make([]string, 0, len(parts))
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p != "" {
				out = append(out, p)
			}
		}
		return out, ""
	}
	arr, ok := v.(*object.Array)
	if !ok {
		return nil, "\"אל\" חייב להיות מחרוזת או רשימת כתובות"
	}
	out := make([]string, 0, len(arr.Elements))
	for _, el := range arr.Elements {
		s, ok := asString(el)
		if !ok {
			return nil, "כל נמען חייב להיות מחרוזת"
		}
		s = strings.TrimSpace(s)
		if s != "" {
			out = append(out, s)
		}
	}
	return out, ""
}

func emailAttachmentPaths(h *object.Hash) ([]string, string) {
	v, ok := h.Pairs["קבצים"]
	if !ok {
		return nil, ""
	}
	arr, ok := v.(*object.Array)
	if !ok {
		return nil, "\"קבצים\" חייב להיות רשימת נתיבים"
	}
	out := make([]string, 0, len(arr.Elements))
	for _, el := range arr.Elements {
		s, ok := asString(el)
		if !ok {
			return nil, "כל קובץ מצורף חייב להיות מחרוזת נתיב"
		}
		out = append(out, resolveAppPath(s))
	}
	return out, ""
}

func encodeRFC2047(s string) string {
	return mime.QEncoding.Encode("utf-8", s)
}

func buildMIMEMessage(from string, to []string, subject, body, htmlBody string, attachments []string) ([]byte, error) {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)

	headers := textproto.MIMEHeader{}
	headers.Set("From", from)
	headers.Set("To", strings.Join(to, ", "))
	headers.Set("Subject", encodeRFC2047(subject))
	headers.Set("MIME-Version", "1.0")
	headers.Set("Content-Type", "multipart/mixed; boundary="+w.Boundary())

	var head bytes.Buffer
	for k, vals := range headers {
		for _, v := range vals {
			fmt.Fprintf(&head, "%s: %s\r\n", k, v)
		}
	}
	head.WriteString("\r\n")

	if htmlBody != "" && body != "" {
		var altBuf bytes.Buffer
		alt := multipart.NewWriter(&altBuf)
		plainPart, err := alt.CreatePart(textproto.MIMEHeader{
			"Content-Type":              {"text/plain; charset=UTF-8"},
			"Content-Transfer-Encoding": {"8bit"},
		})
		if err != nil {
			return nil, err
		}
		if _, err := io.WriteString(plainPart, body); err != nil {
			return nil, err
		}
		htmlPart, err := alt.CreatePart(textproto.MIMEHeader{
			"Content-Type":              {"text/html; charset=UTF-8"},
			"Content-Transfer-Encoding": {"8bit"},
		})
		if err != nil {
			return nil, err
		}
		if _, err := io.WriteString(htmlPart, htmlBody); err != nil {
			return nil, err
		}
		if err := alt.Close(); err != nil {
			return nil, err
		}
		wrap, err := w.CreatePart(textproto.MIMEHeader{
			"Content-Type": {"multipart/alternative; boundary=" + alt.Boundary()},
		})
		if err != nil {
			return nil, err
		}
		if _, err := wrap.Write(altBuf.Bytes()); err != nil {
			return nil, err
		}
	} else if htmlBody != "" {
		bodyPart, err := w.CreatePart(textproto.MIMEHeader{
			"Content-Type":              {"text/html; charset=UTF-8"},
			"Content-Transfer-Encoding": {"8bit"},
		})
		if err != nil {
			return nil, err
		}
		if _, err := io.WriteString(bodyPart, htmlBody); err != nil {
			return nil, err
		}
	} else {
		bodyPart, err := w.CreatePart(textproto.MIMEHeader{
			"Content-Type":              {"text/plain; charset=UTF-8"},
			"Content-Transfer-Encoding": {"8bit"},
		})
		if err != nil {
			return nil, err
		}
		if _, err := io.WriteString(bodyPart, body); err != nil {
			return nil, err
		}
	}

	for _, path := range attachments {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("קריאת קובץ מצורף %s: %w", path, err)
		}
		name := filepath.Base(path)
		ctype := mime.TypeByExtension(filepath.Ext(name))
		if ctype == "" {
			ctype = "application/octet-stream"
		}
		part, err := w.CreatePart(textproto.MIMEHeader{
			"Content-Type":              {ctype + "; name=\"" + name + "\""},
			"Content-Transfer-Encoding": {"base64"},
			"Content-Disposition":       {"attachment; filename=\"" + name + "\""},
		})
		if err != nil {
			return nil, err
		}
		enc := base64.NewEncoder(base64.StdEncoding, part)
		if _, err := enc.Write(data); err != nil {
			_ = enc.Close()
			return nil, err
		}
		_ = enc.Close()
	}
	if err := w.Close(); err != nil {
		return nil, err
	}

	return append(head.Bytes(), buf.Bytes()...), nil
}

func sendMailStartTLS(addr, host string, auth smtp.Auth, from string, to []string, msg []byte) error {
	c, err := smtp.Dial(addr)
	if err != nil {
		return err
	}
	defer c.Close()

	if ok, _ := c.Extension("STARTTLS"); ok {
		cfg := &tls.Config{ServerName: host}
		if err := c.StartTLS(cfg); err != nil {
			return err
		}
	}
	if auth != nil {
		if ok, _ := c.Extension("AUTH"); ok {
			if err := c.Auth(auth); err != nil {
				return err
			}
		}
	}
	if err := c.Mail(from); err != nil {
		return err
	}
	for _, rcpt := range to {
		if err := c.Rcpt(rcpt); err != nil {
			return err
		}
	}
	wc, err := c.Data()
	if err != nil {
		return err
	}
	if _, err := wc.Write(msg); err != nil {
		_ = wc.Close()
		return err
	}
	if err := wc.Close(); err != nil {
		return err
	}
	return c.Quit()
}

func sendMailTLS(addr, host string, auth smtp.Auth, from string, to []string, msg []byte) error {
	tlsCfg := &tls.Config{ServerName: host}
	conn, err := tls.Dial("tcp", addr, tlsCfg)
	if err != nil {
		return err
	}
	c, err := smtp.NewClient(conn, host)
	if err != nil {
		_ = conn.Close()
		return err
	}
	defer c.Close()

	if auth != nil {
		if ok, _ := c.Extension("AUTH"); ok {
			if err := c.Auth(auth); err != nil {
				return err
			}
		}
	}
	if err := c.Mail(from); err != nil {
		return err
	}
	for _, rcpt := range to {
		if err := c.Rcpt(rcpt); err != nil {
			return err
		}
	}
	wc, err := c.Data()
	if err != nil {
		return err
	}
	if _, err := wc.Write(msg); err != nil {
		_ = wc.Close()
		return err
	}
	if err := wc.Close(); err != nil {
		return err
	}
	return c.Quit()
}

// buildMIMEMessageForTest — לבדיקות בלי שליחה
func buildMIMEMessageForTest(from string, to []string, subject, body string, attachments []string) ([]byte, error) {
	return buildMIMEMessage(from, to, subject, body, "", attachments)
}
