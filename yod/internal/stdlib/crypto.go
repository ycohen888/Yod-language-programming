package stdlib

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/md5"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"io"
	"strconv"
	"strings"

	"golang.org/x/crypto/pbkdf2"

	"yod/internal/object"
)

func NewCryptoModule() *object.Module {
	m := &object.Module{Name: "הצפנה", Attrs: map[string]object.Object{}}
	m.Attrs["בסיס64"] = &object.Builtin{Fn: cryptoBase64}
	m.Attrs["מ_בסיס64"] = &object.Builtin{Fn: cryptoFromBase64}
	m.Attrs["SHA256"] = &object.Builtin{Fn: cryptoSHA256}
	m.Attrs["MD5"] = &object.Builtin{Fn: cryptoMD5}
	m.Attrs["HMAC_SHA256"] = &object.Builtin{Fn: cryptoHMACSHA256}
	m.Attrs["נגזר_מפתח"] = &object.Builtin{Fn: cryptoDeriveKey}
	m.Attrs["הצפן"] = &object.Builtin{Fn: cryptoEncrypt}
	m.Attrs["פענח"] = &object.Builtin{Fn: cryptoDecrypt}
	m.Attrs["הצפן_בסיסמה"] = &object.Builtin{Fn: cryptoEncryptPassword}
	m.Attrs["פענח_בסיסמה"] = &object.Builtin{Fn: cryptoDecryptPassword}
	return m
}

func cryptoText(args []object.Object, i int) (string, bool) {
	if i >= len(args) {
		return "", false
	}
	if s, ok := args[i].(*object.String); ok {
		return s.Value, true
	}
	return args[i].Inspect(), true
}

func cryptoBase64(args ...object.Object) object.Object {
	if len(args) != 1 {
		return errObj("הצפנה.בסיס64 מצפה למחרוזת אחת")
	}
	text, ok := cryptoText(args, 0)
	if !ok {
		return errObj("הצפנה.בסיס64 מצפה למחרוזת")
	}
	return &object.String{Value: base64.StdEncoding.EncodeToString([]byte(text))}
}

func cryptoFromBase64(args ...object.Object) object.Object {
	if len(args) != 1 {
		return errObj("הצפנה.מ_בסיס64 מצפה למחרוזת אחת")
	}
	text, ok := cryptoText(args, 0)
	if !ok {
		return errObj("הצפנה.מ_בסיס64 מצפה למחרוזת")
	}
	data, err := base64.StdEncoding.DecodeString(text)
	if err != nil {
		return errObj("הצפנה.מ_בסיס64: מחרוזת לא תקינה: " + err.Error())
	}
	return &object.String{Value: string(data)}
}

func cryptoSHA256(args ...object.Object) object.Object {
	if len(args) != 1 {
		return errObj("הצפנה.SHA256 מצפה למחרוזת אחת")
	}
	text, ok := cryptoText(args, 0)
	if !ok {
		return errObj("הצפנה.SHA256 מצפה למחרוזת")
	}
	sum := sha256.Sum256([]byte(text))
	return &object.String{Value: hex.EncodeToString(sum[:])}
}

func cryptoMD5(args ...object.Object) object.Object {
	if len(args) != 1 {
		return errObj("הצפנה.MD5 מצפה למחרוזת אחת")
	}
	text, ok := cryptoText(args, 0)
	if !ok {
		return errObj("הצפנה.MD5 מצפה למחרוזת")
	}
	sum := md5.Sum([]byte(text))
	return &object.String{Value: hex.EncodeToString(sum[:])}
}

func cryptoHMACSHA256(args ...object.Object) object.Object {
	if len(args) != 2 {
		return errObj("הצפנה.HMAC_SHA256 מצפה למפתח ולטקסט")
	}
	key, ok1 := cryptoText(args, 0)
	text, ok2 := cryptoText(args, 1)
	if !ok1 || !ok2 {
		return errObj("הצפנה.HMAC_SHA256 מצפה למחרוזות")
	}
	mac := hmac.New(sha256.New, []byte(key))
	_, _ = mac.Write([]byte(text))
	return &object.String{Value: hex.EncodeToString(mac.Sum(nil))}
}

const (
	cryptoSaltLen   = 16
	cryptoKeyLen    = 32
	cryptoDefaultIt = 120000
	cryptoBlobPref  = "יוד1:"
)

// נגזר_מפתח(סיסמה [, מלח_hex [, סבבים]]) → {מפתח, מלח, סבבים}
func cryptoDeriveKey(args ...object.Object) object.Object {
	if len(args) < 1 || len(args) > 3 {
		return errObj("הצפנה.נגזר_מפתח מצפה לסיסמה [, מלח [, סבבים]]")
	}
	pass, ok := cryptoText(args, 0)
	if !ok || pass == "" {
		return errObj("הצפנה.נגזר_מפתח מצפה לסיסמה")
	}
	iters := cryptoDefaultIt
	var salt []byte
	if len(args) >= 2 {
		if s, ok := args[1].(*object.String); ok && s.Value != "" {
			decoded, err := hex.DecodeString(s.Value)
			if err != nil {
				return errObj("הצפנה.נגזר_מפתח: מלח חייב להיות hex")
			}
			salt = decoded
		}
	}
	if len(args) >= 3 {
		if n, ok := args[2].(*object.Number); ok && n.Value >= 1000 {
			iters = int(n.Value)
		}
	}
	if len(salt) == 0 {
		salt = make([]byte, cryptoSaltLen)
		if _, err := io.ReadFull(rand.Reader, salt); err != nil {
			return errObj("הצפנה.נגזר_מפתח: יצירת מלח נכשלה")
		}
	}
	key := pbkdf2.Key([]byte(pass), salt, iters, cryptoKeyLen, sha256.New)
	return &object.Hash{Pairs: map[string]object.Object{
		"מפתח":  &object.String{Value: hex.EncodeToString(key)},
		"מלח":   &object.String{Value: hex.EncodeToString(salt)},
		"סבבים": &object.Number{Value: float64(iters)},
	}}
}

func cryptoParseKey(hexKey string) ([]byte, object.Object) {
	key, err := hex.DecodeString(hexKey)
	if err != nil || len(key) != cryptoKeyLen {
		return nil, errObj("מפתח הצפנה חייב להיות 64 תווי hex (32 בתים)")
	}
	return key, nil
}

// הצפן(טקסט, מפתח_hex) → base64(nonce||ciphertext)
func cryptoEncrypt(args ...object.Object) object.Object {
	if len(args) != 2 {
		return errObj("הצפנה.הצפן מצפה לטקסט ולמפתח")
	}
	text, ok := cryptoText(args, 0)
	if !ok {
		return errObj("הצפנה.הצפן מצפה למחרוזת")
	}
	keyHex, ok := cryptoText(args, 1)
	if !ok {
		return errObj("הצפנה.הצפן מצפה למפתח hex")
	}
	key, errV := cryptoParseKey(keyHex)
	if errV != nil {
		return errV
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return errObj("הצפנה.הצפן נכשל: " + err.Error())
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return errObj("הצפנה.הצפן נכשל: " + err.Error())
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return errObj("הצפנה.הצפן: יצירת nonce נכשלה")
	}
	sealed := gcm.Seal(nonce, nonce, []byte(text), nil)
	return &object.String{Value: base64.StdEncoding.EncodeToString(sealed)}
}

func cryptoDecrypt(args ...object.Object) object.Object {
	if len(args) != 2 {
		return errObj("הצפנה.פענח מצפה למוצפן ולמפתח")
	}
	blob, ok := cryptoText(args, 0)
	if !ok {
		return errObj("הצפנה.פענח מצפה למחרוזת")
	}
	keyHex, ok := cryptoText(args, 1)
	if !ok {
		return errObj("הצפנה.פענח מצפה למפתח hex")
	}
	key, errV := cryptoParseKey(keyHex)
	if errV != nil {
		return errV
	}
	raw, err := base64.StdEncoding.DecodeString(blob)
	if err != nil {
		return errObj("הצפנה.פענח: מחרוזת base64 לא תקינה")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return errObj("הצפנה.פענח נכשל: " + err.Error())
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return errObj("הצפנה.פענח נכשל: " + err.Error())
	}
	ns := gcm.NonceSize()
	if len(raw) < ns {
		return errObj("הצפנה.פענח: נתונים קצרים מדי")
	}
	plain, err := gcm.Open(nil, raw[:ns], raw[ns:], nil)
	if err != nil {
		return errObj("הצפנה.פענח: מפתח שגוי או נתונים פגומים")
	}
	return &object.String{Value: string(plain)}
}

// הצפן_בסיסמה(טקסט, סיסמה) → "יוד1:" + base64(salt||iters||nonce||cipher)
func cryptoEncryptPassword(args ...object.Object) object.Object {
	if len(args) != 2 {
		return errObj("הצפנה.הצפן_בסיסמה מצפה לטקסט ולסיסמה")
	}
	text, ok1 := cryptoText(args, 0)
	pass, ok2 := cryptoText(args, 1)
	if !ok1 || !ok2 || pass == "" {
		return errObj("הצפנה.הצפן_בסיסמה מצפה למחרוזות")
	}
	derived := cryptoDeriveKey(&object.String{Value: pass})
	h, ok := derived.(*object.Hash)
	if !ok {
		return derived
	}
	keyHex := h.Pairs["מפתח"].(*object.String).Value
	saltHex := h.Pairs["מלח"].(*object.String).Value
	iters := int(h.Pairs["סבבים"].(*object.Number).Value)
	enc := cryptoEncrypt(&object.String{Value: text}, &object.String{Value: keyHex})
	encS, ok := enc.(*object.String)
	if !ok {
		return enc
	}
	payload := saltHex + ":" + strconv.Itoa(iters) + ":" + encS.Value
	return &object.String{Value: cryptoBlobPref + base64.StdEncoding.EncodeToString([]byte(payload))}
}

func cryptoDecryptPassword(args ...object.Object) object.Object {
	if len(args) != 2 {
		return errObj("הצפנה.פענח_בסיסמה מצפה למוצפן ולסיסמה")
	}
	blob, ok1 := cryptoText(args, 0)
	pass, ok2 := cryptoText(args, 1)
	if !ok1 || !ok2 || pass == "" {
		return errObj("הצפנה.פענח_בסיסמה מצפה למחרוזות")
	}
	if !strings.HasPrefix(blob, cryptoBlobPref) {
		return errObj("הצפנה.פענח_בסיסמה: פורמט לא מוכר")
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(blob, cryptoBlobPref))
	if err != nil {
		return errObj("הצפנה.פענח_בסיסמה: פענוח base64 נכשל")
	}
	parts := strings.SplitN(string(raw), ":", 3)
	if len(parts) != 3 {
		return errObj("הצפנה.פענח_בסיסמה: מבנה פגום")
	}
	iters, err := strconv.Atoi(parts[1])
	if err != nil || iters < 1000 {
		return errObj("הצפנה.פענח_בסיסמה: מספר סבבים לא תקין")
	}
	derived := cryptoDeriveKey(
		&object.String{Value: pass},
		&object.String{Value: parts[0]},
		&object.Number{Value: float64(iters)},
	)
	h, ok := derived.(*object.Hash)
	if !ok {
		return derived
	}
	keyHex := h.Pairs["מפתח"].(*object.String).Value
	return cryptoDecrypt(&object.String{Value: parts[2]}, &object.String{Value: keyHex})
}
