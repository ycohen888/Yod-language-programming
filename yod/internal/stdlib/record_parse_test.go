package stdlib

import "testing"

func TestParseDShowAudioDevices(t *testing.T) {
	sample := `
[dshow @ 000001a] DirectShow video devices (some may be both video and audio devices)
[dshow @ 000001a]  "Integrated Camera" (video)
[dshow @ 000001a] DirectShow audio devices
[dshow @ 000001a]  "Microphone (Realtek Audio)" (audio)
[dshow @ 000001a]  "Stereo Mix (Realtek Audio)" (audio)
`
	got := parseDShowAudioDevices(sample)
	if len(got) != 2 {
		t.Fatalf("expected 2 devices, got %d: %v", len(got), got)
	}
	if got[0] != "Microphone (Realtek Audio)" {
		t.Fatalf("first device: %q", got[0])
	}
	if got[1] != "Stereo Mix (Realtek Audio)" {
		t.Fatalf("second device: %q", got[1])
	}
}
