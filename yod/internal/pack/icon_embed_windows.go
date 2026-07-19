//go:build windows

package pack

import (
	"encoding/binary"
	"fmt"
	"os"
	"syscall"
	"unsafe"
)

var (
	modKernel32            = syscall.NewLazyDLL("kernel32.dll")
	procBeginUpdateResource = modKernel32.NewProc("BeginUpdateResourceW")
	procUpdateResource      = modKernel32.NewProc("UpdateResourceW")
	procEndUpdateResource   = modKernel32.NewProc("EndUpdateResourceW")
)

const (
	rtIcon      = 3
	rtGroupIcon = 14
	langNeutral = 0
)

// applyIconToEXE מטמיע קובץ .ico במשאבי ה־PE (לפני הוספת overlay).
func applyIconToEXE(exePath, icoPath string) error {
	if icoPath == "" {
		return nil
	}
	ico, err := os.ReadFile(icoPath)
	if err != nil {
		return err
	}
	if len(ico) < 6 {
		return fmt.Errorf("קובץ איקון קצר מדי")
	}
	if binary.LittleEndian.Uint16(ico[2:4]) != 1 {
		return fmt.Errorf("הקובץ אינו ICO")
	}
	count := int(binary.LittleEndian.Uint16(ico[4:6]))
	if count <= 0 || 6+count*16 > len(ico) {
		return fmt.Errorf("כותרת ICO לא תקינה")
	}

	exePtr, err := syscall.UTF16PtrFromString(exePath)
	if err != nil {
		return err
	}
	h, _, callErr := procBeginUpdateResource.Call(uintptr(unsafe.Pointer(exePtr)), 0)
	if h == 0 {
		return fmt.Errorf("BeginUpdateResource: %v", callErr)
	}
	abort := true
	defer func() {
		procEndUpdateResource.Call(h, uintptr(boolToUintptr(abort)))
	}()

	// GRPICONDIR + GRPICONDIRENTRY×n (14 בתים לכל תמונה)
	group := make([]byte, 6+count*14)
	binary.LittleEndian.PutUint16(group[0:2], 0)
	binary.LittleEndian.PutUint16(group[2:4], 1)
	binary.LittleEndian.PutUint16(group[4:6], uint16(count))

	for i := 0; i < count; i++ {
		entry := ico[6+i*16 : 6+(i+1)*16]
		size := binary.LittleEndian.Uint32(entry[8:12])
		offset := binary.LittleEndian.Uint32(entry[12:16])
		if int(offset)+int(size) > len(ico) {
			return fmt.Errorf("תמונת איקון #%d חורגת מהקובץ", i+1)
		}
		img := ico[offset : offset+size]
		id := uintptr(i + 1)
		if err := updateResource(h, rtIcon, id, img); err != nil {
			return fmt.Errorf("RT_ICON %d: %w", i+1, err)
		}
		ge := group[6+i*14 : 6+(i+1)*14]
		copy(ge[0:12], entry[0:12]) // width..bytesInRes
		binary.LittleEndian.PutUint16(ge[12:14], uint16(i+1))
	}

	// rsrc של יוד משתמש בדרך כלל ב־ID 1 ל־GROUP_ICON
	if err := updateResource(h, rtGroupIcon, 1, group); err != nil {
		return fmt.Errorf("RT_GROUP_ICON: %w", err)
	}

	abort = false
	return nil
}

func updateResource(h uintptr, resType, resID uintptr, data []byte) error {
	var pData uintptr
	if len(data) > 0 {
		pData = uintptr(unsafe.Pointer(&data[0]))
	}
	r, _, err := procUpdateResource.Call(
		h,
		resType,
		resID,
		langNeutral,
		pData,
		uintptr(len(data)),
	)
	if r == 0 {
		return err
	}
	return nil
}

func boolToUintptr(b bool) uintptr {
	if b {
		return 1
	}
	return 0
}
