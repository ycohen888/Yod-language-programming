package stdlib

import (
	"encoding/binary"
	"fmt"
)

// parseMO — קורא קובץ GNU MO (little-endian או big-endian).
func parseMO(data []byte) (map[string]*i18nMsg, string, error) {
	if len(data) < 28 {
		return nil, "", fmt.Errorf("קובץ MO קצר מדי")
	}
	magic := binary.LittleEndian.Uint32(data[0:4])
	var order binary.ByteOrder = binary.LittleEndian
	switch magic {
	case 0x950412de:
		order = binary.LittleEndian
	case 0xde120495:
		order = binary.BigEndian
	default:
		return nil, "", fmt.Errorf("קסם MO לא מוכר")
	}
	n := order.Uint32(data[8:12])
	oID := order.Uint32(data[12:16])
	oStr := order.Uint32(data[16:20])
	out := map[string]*i18nMsg{}
	var pluralHeader string

	for i := uint32(0); i < n; i++ {
		idOff := oID + 8*i
		strOff := oStr + 8*i
		if int(idOff)+8 > len(data) || int(strOff)+8 > len(data) {
			break
		}
		idLen := order.Uint32(data[idOff : idOff+4])
		idPos := order.Uint32(data[idOff+4 : idOff+8])
		strLen := order.Uint32(data[strOff : strOff+4])
		strPos := order.Uint32(data[strOff+4 : strOff+8])
		if int(idPos)+int(idLen) > len(data) || int(strPos)+int(strLen) > len(data) {
			continue
		}
		idRaw := string(data[idPos : idPos+idLen])
		strRaw := string(data[strPos : strPos+strLen])

		ctxt := ""
		id := idRaw
		idPlural := ""
		if idx := indexByte(idRaw, 0x04); idx >= 0 {
			ctxt = idRaw[:idx]
			id = idRaw[idx+1:]
		}
		if idx := indexByte(id, 0); idx >= 0 {
			idPlural = id[idx+1:]
			id = id[:idx]
		}
		strs := splitNUL(strRaw)
		msg := &i18nMsg{Context: ctxt, ID: id, IDPlural: idPlural, Strs: strs}
		if id == "" && ctxt == "" {
			if len(strs) > 0 {
				pluralHeader = extractPluralForms(strs[0])
			}
			continue
		}
		out[i18nKey(ctxt, id)] = msg
	}
	return out, pluralHeader, nil
}

func indexByte(s string, b byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == b {
			return i
		}
	}
	return -1
}

func splitNUL(s string) []string {
	if s == "" {
		return []string{""}
	}
	var out []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == 0 {
			out = append(out, s[start:i])
			start = i + 1
		}
	}
	out = append(out, s[start:])
	return out
}
