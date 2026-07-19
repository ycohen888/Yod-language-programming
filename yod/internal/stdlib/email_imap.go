package stdlib

import (
	"crypto/tls"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
	"github.com/emersion/go-message/mail"

	"yod/internal/object"
)

type imapBox struct {
	mu     sync.Mutex
	client *imapclient.Client
	closed bool
}

func emailOpenMailbox(args ...object.Object) object.Object {
	if len(args) != 1 {
		return errObj("אימייל.פתח_תיבה מצפה למילון אפשרויות")
	}
	h, ok := args[0].(*object.Hash)
	if !ok {
		return errObj("פתח_תיבה מצפה למילון")
	}
	return emailConnectIMAP(h, false)
}

func emailGmailMailbox(args ...object.Object) object.Object {
	if len(args) != 1 {
		return errObj("אימייל.תיבת_גימייל מצפה למילון")
	}
	h, ok := args[0].(*object.Hash)
	if !ok {
		return errObj("תיבת_גימייל מצפה למילון")
	}
	out := object.NewHash()
	for k, v := range h.Pairs {
		out.Set(k, v)
	}
	out.Set("שרת", &object.String{Value: "imap.gmail.com"})
	out.Set("פורט", &object.Number{Value: 993})
	out.Set("ssl", &object.Boolean{Value: true})
	return emailConnectIMAP(out, true)
}

func emailConnectIMAP(h *object.Hash, gmail bool) object.Object {
	host := hashStr(h, "שרת")
	if host == "" {
		return errObj("חסר שרת IMAP")
	}
	port := hashInt(h, "פורט", 993)
	user := hashStr(h, "משתמש")
	if user == "" {
		user = hashStr(h, "מאת")
	}
	if user == "" {
		return errObj("חסר משתמש לתיבת דואר")
	}
	pass := strings.ReplaceAll(hashStr(h, "סיסמה"), " ", "")
	oauthPath := hashStr(h, "oauth_טוקן")
	if oauthPath == "" {
		oauthPath = hashStr(h, "oauth")
	}
	if pass == "" && oauthPath == "" {
		return errObj("חסרה סיסמה או oauth_טוקן ל־IMAP")
	}

	addr := fmt.Sprintf("%s:%d", host, port)
	var c *imapclient.Client
	var err error
	if hashBool(h, "ssl", true) || port == 993 {
		c, err = imapclient.DialTLS(addr, &imapclient.Options{
			TLSConfig: &tls.Config{ServerName: host},
		})
	} else {
		c, err = imapclient.DialStartTLS(addr, &imapclient.Options{
			TLSConfig: &tls.Config{ServerName: host},
		})
	}
	if err != nil {
		return errObj("חיבור IMAP נכשל: " + err.Error())
	}

	if oauthPath != "" {
		tok, err := emailLoadOAuthToken(resolveAppPath(oauthPath))
		if err != nil {
			_ = c.Close()
			return errObj("טעינת טוקן OAuth נכשלה: " + err.Error())
		}
		if err := c.Authenticate(&xoauth2SASL{user: user, token: tok.AccessToken}); err != nil {
			_ = c.Close()
			msg := "אימות IMAP OAuth נכשל: " + err.Error()
			if gmail {
				msg += " — ודאו scopes של Gmail IMAP"
			}
			return errObj(msg)
		}
	} else {
		if err := c.Login(user, pass).Wait(); err != nil {
			_ = c.Close()
			msg := "התחברות IMAP נכשלה: " + err.Error()
			if gmail {
				msg += " — השתמשו בסיסמת אפליקציה"
			}
			return errObj(msg)
		}
	}

	if _, err := c.Select("INBOX", nil).Wait(); err != nil {
		_ = c.Close()
		return errObj("בחירת INBOX נכשלה: " + err.Error())
	}

	box := &imapBox{client: c}
	return newIMAPHandle(box)
}

func newIMAPHandle(box *imapBox) *object.Module {
	m := &object.Module{Name: "תיבת_דואר", Attrs: map[string]object.Object{}}
	m.Attrs["רשימה"] = &object.Builtin{Fn: func(args ...object.Object) object.Object {
		return imapList(box, args...)
	}}
	m.Attrs["קרא"] = &object.Builtin{Fn: func(args ...object.Object) object.Object {
		return imapRead(box, args...)
	}}
	m.Attrs["סגור"] = &object.Builtin{Fn: func(args ...object.Object) object.Object {
		return imapClose(box, args...)
	}}
	return m
}

func imapList(box *imapBox, args ...object.Object) object.Object {
	limit := 10
	if len(args) == 1 {
		if n, ok := args[0].(*object.Number); ok {
			limit = int(n.Value)
		}
	} else if len(args) > 1 {
		return errObj("רשימה מצפה ל־0 או 1 ארגומנט (מספר הודעות)")
	}
	if limit < 1 {
		limit = 1
	}

	box.mu.Lock()
	defer box.mu.Unlock()
	if box.closed {
		return errObj("התיבה כבר סגורה")
	}

	data, err := box.client.Select("INBOX", nil).Wait()
	if err != nil {
		return errObj("בחירת INBOX נכשלה: " + err.Error())
	}
	total := data.NumMessages
	if total == 0 {
		return &object.Array{Elements: nil}
	}
	from := uint32(1)
	if int(total) > limit {
		from = total - uint32(limit) + 1
	}
	seqSet := imap.SeqSet{}
	seqSet.AddRange(from, total)

	fetchOpts := &imap.FetchOptions{
		Envelope: true,
		UID:      true,
	}
	msgs, err := box.client.Fetch(seqSet, fetchOpts).Collect()
	if err != nil {
		return errObj("שליפת רשימת הודעות נכשלה: " + err.Error())
	}

	els := make([]object.Object, 0, len(msgs))
	for i := len(msgs) - 1; i >= 0; i-- {
		msg := msgs[i]
		h := object.NewHash()
		h.Set("מזהה", &object.Number{Value: float64(msg.SeqNum)})
		if msg.UID != 0 {
			h.Set("uid", &object.Number{Value: float64(msg.UID)})
		}
		if msg.Envelope != nil {
			h.Set("נושא", &object.String{Value: msg.Envelope.Subject})
			h.Set("מאת", &object.String{Value: formatIMAPAddr(msg.Envelope.From)})
			if !msg.Envelope.Date.IsZero() {
				h.Set("תאריך", &object.String{Value: msg.Envelope.Date.Format(time.RFC3339)})
			}
		}
		els = append(els, h)
	}
	return &object.Array{Elements: els}
}

func formatIMAPAddr(addrs []imap.Address) string {
	if len(addrs) == 0 {
		return ""
	}
	a := addrs[0]
	if a.Mailbox == "" {
		return a.Name
	}
	email := a.Mailbox + "@" + a.Host
	if a.Name != "" {
		return a.Name + " <" + email + ">"
	}
	return email
}

func imapRead(box *imapBox, args ...object.Object) object.Object {
	if err := expectArgs("קרא", 1, args); err != nil {
		return err
	}
	n, ok := args[0].(*object.Number)
	if !ok {
		return errObj("קרא מצפה למזהה הודעה (מספר)")
	}
	seq := uint32(n.Value)
	if seq < 1 {
		return errObj("מזהה הודעה לא תקין")
	}

	box.mu.Lock()
	defer box.mu.Unlock()
	if box.closed {
		return errObj("התיבה כבר סגורה")
	}

	seqSet := imap.SeqSet{}
	seqSet.AddNum(seq)
	bodySection := &imap.FetchItemBodySection{}
	fetchOpts := &imap.FetchOptions{
		Envelope: true,
		UID:      true,
		BodySection: []*imap.FetchItemBodySection{bodySection},
	}
	msgs, err := box.client.Fetch(seqSet, fetchOpts).Collect()
	if err != nil || len(msgs) == 0 {
		if err != nil {
			return errObj("קריאת הודעה נכשלה: " + err.Error())
		}
		return errObj("הודעה לא נמצאה")
	}
	msg := msgs[0]
	out := object.NewHash()
	out.Set("מזהה", &object.Number{Value: float64(msg.SeqNum)})
	if msg.Envelope != nil {
		out.Set("נושא", &object.String{Value: msg.Envelope.Subject})
		out.Set("מאת", &object.String{Value: formatIMAPAddr(msg.Envelope.From)})
		if !msg.Envelope.Date.IsZero() {
			out.Set("תאריך", &object.String{Value: msg.Envelope.Date.Format(time.RFC3339)})
		}
	}

	plain, html := "", ""
	for _, buf := range msg.BodySection {
		raw := buf.Bytes
		if len(raw) == 0 {
			continue
		}
		mr, err := mail.CreateReader(strings.NewReader(string(raw)))
		if err != nil {
			out.Set("תוכן", &object.String{Value: string(raw)})
			continue
		}
		for {
			p, err := mr.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				break
			}
			switch h := p.Header.(type) {
			case *mail.InlineHeader:
				ct, _, _ := h.ContentType()
				b, _ := io.ReadAll(p.Body)
				switch {
				case strings.HasPrefix(ct, "text/plain"):
					plain = string(b)
				case strings.HasPrefix(ct, "text/html"):
					html = string(b)
				}
			}
		}
	}
	out.Set("תוכן", &object.String{Value: plain})
	if html != "" {
		out.Set("html", &object.String{Value: html})
	}
	return out
}

func imapClose(box *imapBox, args ...object.Object) object.Object {
	if len(args) != 0 {
		return errObj("סגור מצפה ל־0 ארגומנטים")
	}
	box.mu.Lock()
	defer box.mu.Unlock()
	if box.closed {
		return &object.Null{}
	}
	_ = box.client.Logout().Wait()
	_ = box.client.Close()
	box.closed = true
	return &object.Null{}
}
