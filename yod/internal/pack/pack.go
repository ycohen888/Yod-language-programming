package pack

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// App יוצר תיקיית הפצה עם המנוע, המניפסט, קובץ התוכנית ומפעילי .bat.
// אם outDir ריק — נוצרת תיקייה dist_exe ליד/בתוך הפרויקט.
func App(srcPath string, outDir string) (string, error) {
	srcPath = filepath.Clean(srcPath)
	data, err := os.ReadFile(srcPath)
	if err != nil {
		return "", fmt.Errorf("לא הצלחתי לקרוא את %s: %v", srcPath, err)
	}

	base := strings.TrimSuffix(filepath.Base(srcPath), filepath.Ext(srcPath))
	if base == "" {
		base = "תוכנית"
	}
	if outDir == "" {
		outDir = filepath.Join(filepath.Dir(srcPath), "dist_exe")
		// אם המקור בתוך פרויקט עם התחל.יוד בתיקיית האב — עדיף דרך CLI (DefaultDistDir)
	}
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return "", fmt.Errorf("לא הצלחתי ליצור תיקייה %s: %v", outDir, err)
	}

	progName := filepath.Base(srcPath)
	progDest := filepath.Join(outDir, progName)
	if err := os.WriteFile(progDest, rewriteSiblingIncludes(data), 0644); err != nil {
		return "", err
	}

	exePath, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("לא הצלחתי למצוא את yod.exe: %v", err)
	}
	if resolved, err := filepath.EvalSymlinks(exePath); err == nil {
		exePath = resolved
	}
	if err := copyFile(exePath, filepath.Join(outDir, "yod.exe")); err != nil {
		return "", fmt.Errorf("העתקת yod.exe נכשלה: %v", err)
	}

	manifestCandidates := []string{
		exePath + ".manifest",
		filepath.Join(filepath.Dir(exePath), "yod.exe.manifest"),
		"yod.exe.manifest",
	}
	for _, m := range manifestCandidates {
		if _, err := os.Stat(m); err == nil {
			_ = copyFile(m, filepath.Join(outDir, "yod.exe.manifest"))
			break
		}
	}

	bat := fmt.Sprintf("@echo off\r\nchcp 65001 >nul\r\n\"%%~dp0yod.exe\" הרץ \"%%~dp0%s\"\r\nif errorlevel 1 pause\r\n", progName)
	if err := os.WriteFile(filepath.Join(outDir, "הרץ.bat"), []byte(bat), 0644); err != nil {
		return "", err
	}
	batVM := fmt.Sprintf("@echo off\r\nchcp 65001 >nul\r\n\"%%~dp0yod.exe\" מכונה \"%%~dp0%s\"\r\nif errorlevel 1 (\r\necho נפילה למפרש...\r\n\"%%~dp0yod.exe\" הרץ \"%%~dp0%s\"\r\n)\r\nif errorlevel 1 pause\r\n", progName, progName)
	_ = os.WriteFile(filepath.Join(outDir, "הרץ-מכונה.bat"), []byte(batVM), 0644)

	manifestDest := filepath.Join(outDir, "yod.exe.manifest")
	if _, err := os.Stat(manifestDest); err != nil {
		_ = os.WriteFile(manifestDest, []byte(DefaultManifest), 0644)
	}

	// העתקת איקון הפרויקט (פרטי או יוד) להפצה — לחלון ולפס משימות
	_ = copyProjectIcon(srcPath, outDir)

	readme := "תוכנית יוד — הפצה\r\n" +
		"==================\r\n\r\n" +
		"להרצה: לחצו פעמיים על הרץ.bat (מפרש מלא)\r\n" +
		"או הרץ-מכונה.bat (bytecode — מהיר יותר כשנתמך)\r\n\r\n" +
		"מהטרמינל:\r\n" +
		"  yod.exe הרץ " + progName + "\r\n" +
		"  yod.exe מכונה " + progName + "\r\n\r\n" +
		"נוצר עם: yod ארוז\r\n"
	_ = os.WriteFile(filepath.Join(outDir, "קרא-אותי.txt"), []byte(readme), 0644)

	return outDir, nil
}

// DefaultManifest — מניפסט בסיסי ל־Common Controls 6 ו־DPI.
const DefaultManifest = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<assembly xmlns="urn:schemas-microsoft-com:asm.v1" manifestVersion="1.0">
  <assemblyIdentity version="1.0.0.0" processorArchitecture="*" name="Yod" type="win32"/>
  <dependency>
    <dependentAssembly>
      <assemblyIdentity type="win32" name="Microsoft.Windows.Common-Controls" version="6.0.0.0" processorArchitecture="*" publicKeyToken="6595b64144ccf1df" language="*"/>
    </dependentAssembly>
  </dependency>
  <application xmlns="urn:schemas-microsoft-com:asm.v3">
    <windowsSettings>
      <dpiAware xmlns="http://schemas.microsoft.com/SMI/2005/WindowsSettings">true</dpiAware>
    </windowsSettings>
  </application>
</assembly>
`

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
}

// copyProjectIcon מעתיק איקון פרטי (או של יוד) לתיקיית ההפצה — לחלון ולפס משימות.
func copyProjectIcon(srcPath, destDir string) error {
	icoPath, icoTemp, err := resolvePackIcon(srcPath)
	if err != nil || icoPath == "" {
		return err
	}
	if icoTemp {
		defer os.Remove(icoPath)
	}
	_ = copyFile(icoPath, filepath.Join(destDir, "app.ico"))
	_ = copyFile(icoPath, filepath.Join(destDir, "יוד.ico"))
	return copyFile(icoPath, filepath.Join(destDir, "yod.ico"))
}
