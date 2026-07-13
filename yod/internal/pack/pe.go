package pack

import (
	"encoding/binary"
	"fmt"
)

// מצבי מערכת של Windows PE
const (
	subsystemGUI     = 2 // בלי חלון CMD
	subsystemConsole = 3 // עם קונסול
)

// setPESubsystem משנה את שדה Subsystem בכותרת ה־PE.
func setPESubsystem(exe []byte, subsystem uint16) error {
	if len(exe) < 0x40 {
		return fmt.Errorf("קובץ PE קצר מדי")
	}
	if exe[0] != 'M' || exe[1] != 'Z' {
		return fmt.Errorf("לא קובץ EXE תקין")
	}
	peOff := binary.LittleEndian.Uint32(exe[0x3C:0x40])
	if int(peOff)+24 >= len(exe) {
		return fmt.Errorf("כותרת PE לא תקינה")
	}
	if exe[peOff] != 'P' || exe[peOff+1] != 'E' || exe[peOff+2] != 0 || exe[peOff+3] != 0 {
		return fmt.Errorf("חתימת PE חסרה")
	}
	optOff := int(peOff) + 4 + 20 // אחרי PE\0\0 ו־COFF
	if optOff+70 > len(exe) {
		return fmt.Errorf("כותרת אופציונלית קצרה מדי")
	}
	magic := binary.LittleEndian.Uint16(exe[optOff : optOff+2])
	if magic != 0x10b && magic != 0x20b { // PE32 / PE32+
		return fmt.Errorf("סוג PE לא נתמך: 0x%x", magic)
	}
	// Subsystem באופסט 68 גם ב־PE32 וגם ב־PE32+
	binary.LittleEndian.PutUint16(exe[optOff+68:optOff+70], subsystem)
	return nil
}
