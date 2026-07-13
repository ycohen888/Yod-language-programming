//go:build windows

package stdlib

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"
)

var (
	procGetSystemTimes      = modkernel32.NewProc("GetSystemTimes")
	procGetTickCount64      = modkernel32.NewProc("GetTickCount64")
	procGetDiskFreeSpaceExW = modkernel32.NewProc("GetDiskFreeSpaceExW")
	procGetLogicalDrives    = modkernel32.NewProc("GetLogicalDrives")
)

func listOSProcesses() ([]osProcess, error) {
	cmd := exec.Command("tasklist", "/FO", "CSV", "/NH")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: 0x08000000, // CREATE_NO_WINDOW
	}
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return nil, fmt.Errorf("%s", msg)
	}
	r := csv.NewReader(bytes.NewReader(stdout.Bytes()))
	r.FieldsPerRecord = -1
	rows, err := r.ReadAll()
	if err != nil {
		return nil, err
	}
	out := make([]osProcess, 0, len(rows))
	for _, row := range rows {
		if len(row) < 5 {
			continue
		}
		name := strings.TrimSpace(row[0])
		pid, err := strconv.ParseUint(strings.TrimSpace(row[1]), 10, 32)
		if err != nil {
			continue
		}
		mem := parseTasklistMemory(row[4])
		out = append(out, osProcess{
			PID:        uint32(pid),
			Name:       name,
			WorkingSet: mem,
		})
	}
	return out, nil
}

func parseTasklistMemory(s string) uint64 {
	// דוגמה: "12,345 K" או "12345 K"
	s = strings.TrimSpace(s)
	s = strings.TrimSuffix(s, "K")
	s = strings.TrimSuffix(s, "k")
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, ",", "")
	s = strings.ReplaceAll(s, " ", "")
	n, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return 0
	}
	return n * 1024
}

func killOSProcess(pid uint32) error {
	cmd := exec.Command("taskkill", "/PID", strconv.FormatUint(uint64(pid), 10), "/F")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: 0x08000000, // CREATE_NO_WINDOW
	}
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return fmt.Errorf("%s", msg)
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
