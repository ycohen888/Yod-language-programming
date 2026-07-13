//go:build windows

package stdlib

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

type memoryStatusEx struct {
	Length               uint32
	MemoryLoad           uint32
	TotalPhys            uint64
	AvailPhys            uint64
	TotalPageFile        uint64
	AvailPageFile        uint64
	TotalVirtual         uint64
	AvailVirtual         uint64
	AvailExtendedVirtual uint64
}

var (
	modkernel32              = windows.NewLazySystemDLL("kernel32.dll")
	procGlobalMemoryStatusEx = modkernel32.NewProc("GlobalMemoryStatusEx")
)

func systemMemoryBytes() (total, available uint64, ok bool) {
	var ms memoryStatusEx
	ms.Length = uint32(unsafe.Sizeof(ms))
	r1, _, _ := procGlobalMemoryStatusEx.Call(uintptr(unsafe.Pointer(&ms)))
	if r1 == 0 {
		return 0, 0, false
	}
	return ms.TotalPhys, ms.AvailPhys, true
}

func osVersionString() string {
	info := windows.RtlGetVersion()
	s := fmt.Sprintf("%d.%d.%d", info.MajorVersion, info.MinorVersion, info.BuildNumber)
	if prod := windowsProductName(); prod != "" {
		return prod + " (" + s + ")"
	}
	return "Windows " + s
}

func windowsProductName() string {
	k, err := registry.OpenKey(registry.LOCAL_MACHINE,
		`SOFTWARE\Microsoft\Windows NT\CurrentVersion`, registry.QUERY_VALUE)
	if err != nil {
		return ""
	}
	defer k.Close()
	name, _, err := k.GetStringValue("ProductName")
	if err != nil {
		return ""
	}
	return name
}

func cpuModelName() string {
	if v := os.Getenv("PROCESSOR_IDENTIFIER"); v != "" {
		// מזהה ארוך — מעדיפים את שם המעבד מהרישום
	}
	k, err := registry.OpenKey(registry.LOCAL_MACHINE,
		`HARDWARE\DESCRIPTION\System\CentralProcessor\0`, registry.QUERY_VALUE)
	if err == nil {
		defer k.Close()
		if name, _, err := k.GetStringValue("ProcessorNameString"); err == nil && name != "" {
			return strings.TrimSpace(name)
		}
	}
	if v := os.Getenv("PROCESSOR_IDENTIFIER"); v != "" {
		return strings.TrimSpace(v)
	}
	if v := os.Getenv("PROCESSOR_ARCHITECTURE"); v != "" {
		return v
	}
	return runtime.GOARCH
}
