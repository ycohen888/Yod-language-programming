//go:build !windows

package stdlib

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
)

func systemMemoryBytes() (total, available uint64, ok bool) {
	switch runtime.GOOS {
	case "linux":
		return linuxMemFromProc()
	case "darwin":
		return darwinMemFromSysctl()
	default:
		return 0, 0, false
	}
}

func linuxMemFromProc() (total, available uint64, ok bool) {
	f, err := os.Open("/proc/meminfo")
	if err != nil {
		return 0, 0, false
	}
	defer f.Close()
	var memTotal, memAvail uint64
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		n, err := strconv.ParseUint(fields[1], 10, 64)
		if err != nil {
			continue
		}
		// ערכים ב־kB
		switch fields[0] {
		case "MemTotal:":
			memTotal = n * 1024
		case "MemAvailable:":
			memAvail = n * 1024
		case "MemFree:":
			if memAvail == 0 {
				memAvail = n * 1024
			}
		}
	}
	if memTotal == 0 {
		return 0, 0, false
	}
	return memTotal, memAvail, true
}

func darwinMemFromSysctl() (total, available uint64, ok bool) {
	out, err := exec.Command("sysctl", "-n", "hw.memsize").Output()
	if err != nil {
		return 0, 0, false
	}
	total, err = strconv.ParseUint(strings.TrimSpace(string(out)), 10, 64)
	if err != nil || total == 0 {
		return 0, 0, false
	}
	pageSize := uint64(4096)
	if psOut, err := exec.Command("sysctl", "-n", "hw.pagesize").Output(); err == nil {
		if ps, err := strconv.ParseUint(strings.TrimSpace(string(psOut)), 10, 64); err == nil && ps > 0 {
			pageSize = ps
		}
	}
	freePages := uint64(0)
	if fpOut, err := exec.Command("sysctl", "-n", "vm.page_free_count").Output(); err == nil {
		freePages, _ = strconv.ParseUint(strings.TrimSpace(string(fpOut)), 10, 64)
	}
	return total, freePages * pageSize, true
}

func osVersionString() string {
	switch runtime.GOOS {
	case "linux":
		if s := linuxPrettyName(); s != "" {
			return s
		}
		if b, err := os.ReadFile("/proc/version"); err == nil {
			line := strings.TrimSpace(string(b))
			if i := strings.Index(line, "("); i > 0 {
				return strings.TrimSpace(line[:i])
			}
			return line
		}
	case "darwin":
		if out, err := exec.Command("sw_vers", "-productVersion").Output(); err == nil {
			v := strings.TrimSpace(string(out))
			if v != "" {
				return "macOS " + v
			}
		}
	}
	return hebrewOSName(runtime.GOOS)
}

func linuxPrettyName() string {
	f, err := os.Open("/etc/os-release")
	if err != nil {
		return ""
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		if strings.HasPrefix(line, "PRETTY_NAME=") {
			v := strings.TrimPrefix(line, "PRETTY_NAME=")
			return strings.Trim(v, `"`)
		}
	}
	return ""
}

func cpuModelName() string {
	switch runtime.GOOS {
	case "linux":
		if name := linuxCPUModel(); name != "" {
			return name
		}
	case "darwin":
		if out, err := exec.Command("sysctl", "-n", "machdep.cpu.brand_string").Output(); err == nil {
			if s := strings.TrimSpace(string(out)); s != "" {
				return s
			}
		}
	}
	return fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH)
}

func linuxCPUModel() string {
	f, err := os.Open("/proc/cpuinfo")
	if err != nil {
		return ""
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		if strings.HasPrefix(line, "model name") || strings.HasPrefix(line, "Hardware") || strings.HasPrefix(line, "cpu model") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				return strings.TrimSpace(parts[1])
			}
		}
	}
	return ""
}
