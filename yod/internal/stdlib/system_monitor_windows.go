//go:build windows

package stdlib

import (
	"fmt"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	procGetSystemTimes      = modkernel32.NewProc("GetSystemTimes")
	procGetTickCount64      = modkernel32.NewProc("GetTickCount64")
	procGetDiskFreeSpaceExW = modkernel32.NewProc("GetDiskFreeSpaceExW")
	procGetLogicalDrives    = modkernel32.NewProc("GetLogicalDrives")

	modpsapi                 = windows.NewLazySystemDLL("psapi.dll")
	procGetProcessMemoryInfo = modpsapi.NewProc("GetProcessMemoryInfo")
)

type processMemoryCounters struct {
	CB                         uint32
	PageFaultCount             uint32
	PeakWorkingSetSize         uintptr
	WorkingSetSize             uintptr
	QuotaPeakPagedPoolUsage    uintptr
	QuotaPagedPoolUsage        uintptr
	QuotaPeakNonPagedPoolUsage uintptr
	QuotaNonPagedPoolUsage     uintptr
	PagefileUsage              uintptr
	PeakPagefileUsage          uintptr
}

func listOSProcesses() ([]osProcess, error) {
	snap, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return nil, fmt.Errorf("CreateToolhelp32Snapshot: %w", err)
	}
	defer windows.CloseHandle(snap)

	var pe windows.ProcessEntry32
	pe.Size = uint32(unsafe.Sizeof(pe))
	if err := windows.Process32First(snap, &pe); err != nil {
		return nil, fmt.Errorf("Process32First: %w", err)
	}

	out := make([]osProcess, 0, 256)
	for {
		pid := pe.ProcessID
		name := windows.UTF16ToString(pe.ExeFile[:])
		if pid != 0 && name != "" {
			out = append(out, osProcess{
				PID:        pid,
				Name:       name,
				WorkingSet: processWorkingSet(pid),
			})
		}
		if err := windows.Process32Next(snap, &pe); err != nil {
			break
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("לא נמצאו תהליכים")
	}
	return out, nil
}

func processWorkingSet(pid uint32) uint64 {
	h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, pid)
	if err != nil {
		return 0
	}
	defer windows.CloseHandle(h)

	var pmc processMemoryCounters
	pmc.CB = uint32(unsafe.Sizeof(pmc))
	r1, _, _ := procGetProcessMemoryInfo.Call(
		uintptr(h),
		uintptr(unsafe.Pointer(&pmc)),
		uintptr(pmc.CB),
	)
	if r1 == 0 {
		return 0
	}
	return uint64(pmc.WorkingSetSize)
}

func killOSProcess(pid uint32) error {
	h, err := windows.OpenProcess(windows.PROCESS_TERMINATE, false, pid)
	if err != nil {
		return fmt.Errorf("%s", err.Error())
	}
	defer windows.CloseHandle(h)
	if err := windows.TerminateProcess(h, 1); err != nil {
		return fmt.Errorf("%s", err.Error())
	}
	return nil
}

type filetime64 struct {
	Lo uint32
	Hi uint32
}

func (ft filetime64) uint64() uint64 {
	return (uint64(ft.Hi) << 32) | uint64(ft.Lo)
}

var (
	cpuMu                          sync.Mutex
	prevIdle, prevKernel, prevUser uint64
	cpuPrimed                      bool
)

func systemCPUPercent() (float64, bool) {
	var idle, kernel, user filetime64
	r1, _, _ := procGetSystemTimes.Call(
		uintptr(unsafe.Pointer(&idle)),
		uintptr(unsafe.Pointer(&kernel)),
		uintptr(unsafe.Pointer(&user)),
	)
	if r1 == 0 {
		return 0, false
	}
	i, k, u := idle.uint64(), kernel.uint64(), user.uint64()

	cpuMu.Lock()
	defer cpuMu.Unlock()
	if !cpuPrimed {
		prevIdle, prevKernel, prevUser = i, k, u
		cpuPrimed = true
		time.Sleep(80 * time.Millisecond)
		r2, _, _ := procGetSystemTimes.Call(
			uintptr(unsafe.Pointer(&idle)),
			uintptr(unsafe.Pointer(&kernel)),
			uintptr(unsafe.Pointer(&user)),
		)
		if r2 == 0 {
			return 0, true
		}
		i, k, u = idle.uint64(), kernel.uint64(), user.uint64()
	}

	dIdle := i - prevIdle
	dKernel := k - prevKernel
	dUser := u - prevUser
	prevIdle, prevKernel, prevUser = i, k, u

	total := dKernel + dUser
	if total == 0 {
		return 0, true
	}
	busy := total - dIdle
	if busy > total {
		busy = total
	}
	pct := 100.0 * float64(busy) / float64(total)
	if pct < 0 {
		pct = 0
	}
	if pct > 100 {
		pct = 100
	}
	return pct, true
}

func systemUptimeSeconds() (uint64, bool) {
	r1, _, _ := procGetTickCount64.Call()
	return uint64(r1) / 1000, true
}

func listOSDrives() ([]osDrive, error) {
	mask, _, _ := procGetLogicalDrives.Call()
	if mask == 0 {
		return nil, fmt.Errorf("אין כוננים")
	}
	var out []osDrive
	for i := 0; i < 26; i++ {
		if mask&(1<<uint(i)) == 0 {
			continue
		}
		letter := string(rune('A'+i)) + `:\`
		total, free, ok := diskSpace(letter)
		if !ok {
			continue
		}
		out = append(out, osDrive{
			Letter: strings.TrimSuffix(letter, `\`),
			Total:  total,
			Free:   free,
		})
	}
	return out, nil
}

func diskSpace(root string) (total, free uint64, ok bool) {
	pathPtr, err := syscall.UTF16PtrFromString(root)
	if err != nil {
		return 0, 0, false
	}
	var freeBytes, totalBytes, totalFreeBytes uint64
	r1, _, _ := procGetDiskFreeSpaceExW.Call(
		uintptr(unsafe.Pointer(pathPtr)),
		uintptr(unsafe.Pointer(&freeBytes)),
		uintptr(unsafe.Pointer(&totalBytes)),
		uintptr(unsafe.Pointer(&totalFreeBytes)),
	)
	if r1 == 0 {
		return 0, 0, false
	}
	return totalBytes, freeBytes, true
}

func systemMemoryLoadPercent() (uint32, bool) {
	var ms memoryStatusEx
	ms.Length = uint32(unsafe.Sizeof(ms))
	r1, _, _ := procGlobalMemoryStatusEx.Call(uintptr(unsafe.Pointer(&ms)))
	if r1 == 0 {
		return 0, false
	}
	return ms.MemoryLoad, true
}
