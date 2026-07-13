package stdlib

import (
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"

	"yod/internal/object"
)

func NewCryptoModule() *object.Module {
	m := &object.Module{Name: "הצפנה", Attrs: map[string]object.Object{}}
	m.Attrs["בסיס64"] = &object.Builtin{Fn: cryptoBase64}
	m.Attrs["מ_בסיס64"] = &object.Builtin{Fn: cryptoFromBase64}
	m.Attrs["SHA256"] = &object.Builtin{Fn: cryptoSHA256}
	m.Attrs["MD5"] = &object.Builtin{Fn: cryptoMD5}
	m.Attrs["HMAC_SHA256"] = &object.Builtin{Fn: cryptoHMACSHA256}
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
