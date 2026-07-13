//go:build !windows

package stdlib

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

func listOSProcesses() ([]osProcess, error) {
	if runtime.GOOS != "linux" {
		return nil, fmt.Errorf("רשימת תהליכים זמינה ב־Windows ו־Linux")
	}
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil, err
	}
	var out []osProcess
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		pid, err := strconv.ParseUint(e.Name(), 10, 32)
		if err != nil {
			continue
		}
		name := readProcComm(e.Name())
		if name == "" {
			name = e.Name()
		}
		ws := readProcRSS(e.Name())
		out = append(out, osProcess{
			PID:        uint32(pid),
			Name:       name,
			WorkingSet: ws,
		})
	}
	return out, nil
}

func readProcComm(pid string) string {
	b, err := os.ReadFile(filepath.Join("/proc", pid, "comm"))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

func readProcRSS(pid string) uint64 {
	f, err := os.Open(filepath.Join("/proc", pid, "status"))
	if err != nil {
		return 0
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		if strings.HasPrefix(line, "VmRSS:") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				n, _ := strconv.ParseUint(fields[1], 10, 64)
				return n * 1024 // kB → bytes
			}
		}
	}
	return 0
}

func killOSProcess(pid uint32) error {
	p, err := os.FindProcess(int(pid))
	if err != nil {
		return err
	}
	return p.Kill()
}

var (
	cpuMu      sync.Mutex
	prevIdle   uint64
	prevTotal  uint64
	cpuPrimed  bool
)

func systemCPUPercent() (float64, bool) {
	if runtime.GOOS != "linux" {
		return 0, false
	}
	idle, total, ok := linuxCPUTimes()
	if !ok {
		return 0, false
	}
	cpuMu.Lock()
	defer cpuMu.Unlock()
	if !cpuPrimed {
		prevIdle, prevTotal = idle, total
		cpuPrimed = true
		time.Sleep(80 * time.Millisecond)
		idle, total, ok = linuxCPUTimes()
		if !ok {
			return 0, true
		}
	}
	dIdle := idle - prevIdle
	dTotal := total - prevTotal
	prevIdle, prevTotal = idle, total
	if dTotal == 0 {
		return 0, true
	}
	busy := dTotal - dIdle
	pct := 100.0 * float64(busy) / float64(dTotal)
	if pct < 0 {
		pct = 0
	}
	if pct > 100 {
		pct = 100
	}
	return pct, true
}

func linuxCPUTimes() (idle, total uint64, ok bool) {
	b, err := os.ReadFile("/proc/stat")
	if err != nil {
		return 0, 0, false
	}
	line := strings.SplitN(string(b), "\n", 2)[0]
	fields := strings.Fields(line)
	if len(fields) < 5 || fields[0] != "cpu" {
		return 0, 0, false
	}
	var sum uint64
	for i := 1; i < len(fields); i++ {
		n, err := strconv.ParseUint(fields[i], 10, 64)
		if err != nil {
			continue
		}
		sum += n
		if i == 4 {
			idle = n
		}
	}
	return idle, sum, true
}

func systemUptimeSeconds() (uint64, bool) {
	if runtime.GOOS != "linux" {
		return 0, false
	}
	b, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return 0, false
	}
	fields := strings.Fields(string(b))
	if len(fields) < 1 {
		return 0, false
	}
	f, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return 0, false
	}
	return uint64(f), true
}

func listOSDrives() ([]osDrive, error) {
	if runtime.GOOS != "linux" {
		return nil, fmt.Errorf("כוננים נתמכים ב־Windows ו־Linux")
	}
	// פשוט: שורש /
	var st struct {
		Bsize  int64
		Blocks uint64
		Bfree  uint64
		Bavail uint64
	}
	_ = st
	// בלי syscall unix כאן — מחזירים ריק אם אין golang.org/x/sys/unix נוח
	// ננסה df דרך /proc/mounts רק לשורש עם Statfs
	total, free, ok := linuxRootSpace()
	if !ok {
		return []osDrive{}, nil
	}
	return []osDrive{{Letter: "/", Total: total, Free: free}}, nil
}

func linuxRootSpace() (total, free uint64, ok bool) {
	// fallback: קריאה מ־/proc/meminfo לא מתאימה לדיסק
	return 0, 0, false
}

func systemMemoryLoadPercent() (uint32, bool) {
	total, avail, ok := systemMemoryBytes()
	if !ok || total == 0 {
		return 0, false
	}
	used := total - avail
	return uint32((used * 100) / total), true
}
