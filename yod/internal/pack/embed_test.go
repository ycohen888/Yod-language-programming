package pack

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
)

func TestSetPESubsystem(t *testing.T) {
	enginePath, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	engine, err := engineBytes(enginePath)
	if err != nil {
		t.Fatal(err)
	}
	if err := setPESubsystem(engine, subsystemGUI); err != nil {
		t.Fatal(err)
	}
	peOff := binary.LittleEndian.Uint32(engine[0x3C:0x40])
	optOff := int(peOff) + 4 + 20
	got := binary.LittleEndian.Uint16(engine[optOff+68 : optOff+70])
	if got != subsystemGUI {
		t.Fatalf("subsystem=%d, want %d", got, subsystemGUI)
	}
}

func TestEXERoundTrip(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "שלום.יוד")
	code := "הדפס: \"שלום מה־EXE\"\n"
	if err := os.WriteFile(src, []byte(code), 0644); err != nil {
		t.Fatal(err)
	}

	fakeEngine := filepath.Join(dir, "engine.exe")
	// מינימום PE מדומה לא יעבוד עם setPESubsystem — בודקים רק overlay
	engineBody := make([]byte, 0x100)
	copy(engineBody[0:], "MZ")
	binary.LittleEndian.PutUint32(engineBody[0x3C:], 0x40)
	copy(engineBody[0x40:], "PE\x00\x00")
	binary.LittleEndian.PutUint16(engineBody[0x40+4+20:], 0x20b) // PE32+
	binary.LittleEndian.PutUint16(engineBody[0x40+4+20+68:], subsystemConsole)
	if err := os.WriteFile(fakeEngine, engineBody, 0644); err != nil {
		t.Fatal(err)
	}

	out := filepath.Join(dir, "app.exe")
	engine, err := engineBytes(fakeEngine)
	if err != nil {
		t.Fatal(err)
	}
	if err := setPESubsystem(engine, subsystemGUI); err != nil {
		t.Fatal(err)
	}
	script := []byte(code)
	f, err := os.Create(out)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.Write(engine); err != nil {
		t.Fatal(err)
	}
	if _, err := f.Write(script); err != nil {
		t.Fatal(err)
	}
	meta := make([]byte, embedMetaLen)
	binary.LittleEndian.PutUint64(meta[0:8], uint64(len(script)))
	copy(meta[8:], embedMagic)
	if _, err := f.Write(meta); err != nil {
		t.Fatal(err)
	}
	f.Close()

	got, ok, err := ReadEmbedded(out)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("לא זוהתה הטמעה")
	}
	if got != code {
		t.Fatalf("קוד שונה: %q", got)
	}

	engine2, err := engineBytes(out)
	if err != nil {
		t.Fatal(err)
	}
	peOff := binary.LittleEndian.Uint32(engine2[0x3C:0x40])
	optOff := int(peOff) + 4 + 20
	sub := binary.LittleEndian.Uint16(engine2[optOff+68 : optOff+70])
	if sub != subsystemGUI {
		t.Fatalf("subsystem אחרי אריזה=%d", sub)
	}
}
