package stdlib

import (
	"regexp"
	"strings"
)

// parseDShowAudioDevices מחלץ שמות התקני אודיו מפלט ffmpeg -list_devices (dshow).
func parseDShowAudioDevices(stderr string) []string {
	// דוגמה: [dshow @ 000001]  "Microphone (Realtek)" (audio)
	re := regexp.MustCompile(`(?m)^\s*\[dshow[^\]]*\]\s+"([^"]+)"\s+\(audio\)`)
	matches := re.FindAllStringSubmatch(stderr, -1)
	seen := map[string]bool{}
	var out []string
	for _, m := range matches {
		if len(m) < 2 {
			continue
		}
		name := strings.TrimSpace(m[1])
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		out = append(out, name)
	}
	// פורמט חלופי בלי המילה audio בשורה נפרדת אחרי "DirectShow audio devices"
	if len(out) == 0 {
		lines := strings.Split(stderr, "\n")
		inAudio := false
		reAlt := regexp.MustCompile(`"([^"]+)"`)
		for _, line := range lines {
			low := strings.ToLower(line)
			if strings.Contains(low, "directshow audio") {
				inAudio = true
				continue
			}
			if inAudio && strings.Contains(low, "directshow video") {
				break
			}
			if !inAudio {
				continue
			}
			m := reAlt.FindStringSubmatch(line)
			if m == nil {
				continue
			}
			name := strings.TrimSpace(m[1])
			if name == "" || seen[name] {
				continue
			}
			seen[name] = true
			out = append(out, name)
		}
	}
	return out
}
