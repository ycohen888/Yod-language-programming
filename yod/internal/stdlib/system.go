package stdlib

import (
	"net"
	"os"
	"os/user"
	"runtime"
	"strings"

	"yod/internal/object"
)

func NewSystemModule() *object.Module {
	m := &object.Module{Name: "מערכת", Attrs: map[string]object.Object{}}
	m.Attrs["ארגומנטים"] = &object.Builtin{Fn: sysArgs}
	m.Attrs["סביבה"] = &object.Builtin{Fn: sysEnv}
	m.Attrs["צא"] = &object.Builtin{Fn: sysExit}
	m.Attrs["תיקייה"] = &object.Builtin{Fn: sysCwd}
	m.Attrs["שנה_תיקייה"] = &object.Builtin{Fn: sysChdir}
	// מידע מערכת הפעלה / חומרה / רשת
	m.Attrs["שם_מחשב"] = &object.Builtin{Fn: sysHostname}
	m.Attrs["מערכת_הפעלה"] = &object.Builtin{Fn: sysOSName}
	m.Attrs["גרסת_הפעלה"] = &object.Builtin{Fn: sysOSVersion}
	m.Attrs["ארכיטקטורה"] = &object.Builtin{Fn: sysArch}
	m.Attrs["ליבות"] = &object.Builtin{Fn: sysCores}
	m.Attrs["מעבד"] = &object.Builtin{Fn: sysCPU}
	m.Attrs["זיכרון"] = &object.Builtin{Fn: sysMemory}
	m.Attrs["איפי"] = &object.Builtin{Fn: sysIPs}
	m.Attrs["משתמש"] = &object.Builtin{Fn: sysUsername}
	m.Attrs["בית"] = &object.Builtin{Fn: sysHome}
	m.Attrs["מידע"] = &object.Builtin{Fn: sysInfo}
	m.Attrs["הפעל"] = &object.Builtin{Fn: sysRun}
	m.Attrs["הפעל_ברקע"] = &object.Builtin{Fn: sysRunBackground}
	m.Attrs["תהליכים"] = &object.Builtin{Fn: sysProcesses}
	m.Attrs["סיים_תהליך"] = &object.Builtin{Fn: sysKillProcess}
	m.Attrs["שימוש_מעבד"] = &object.Builtin{Fn: sysCPUUsage}
	m.Attrs["זמן_פעיל"] = &object.Builtin{Fn: sysUptime}
	m.Attrs["כוננים"] = &object.Builtin{Fn: sysDrives}
	return m
}

func sysArgs(args ...object.Object) object.Object {
	if len(args) != 0 {
		return errObj("מערכת.ארגומנטים מצפה ל־0 ארגומנטים")
	}
	arr := &object.Array{Elements: make([]object.Object, len(object.ProgramArgs))}
	for i, a := range object.ProgramArgs {
		arr.Elements[i] = &object.String{Value: a}
	}
	return arr
}

func sysEnv(args ...object.Object) object.Object {
	if len(args) != 1 {
		return errObj("מערכת.סביבה מצפה לשם משתנה")
	}
	name, ok := asString(args[0])
	if !ok {
		return errObj("מערכת.סביבה מצפה למחרוזת")
	}
	return &object.String{Value: os.Getenv(name)}
}

func sysExit(args ...object.Object) object.Object {
	code := 0
	if len(args) == 1 {
		if n, ok := args[0].(*object.Number); ok {
			code = int(n.Value)
		} else {
			return errObj("מערכת.צא מצפה לקוד מספרי")
		}
	} else if len(args) > 1 {
		return errObj("מערכת.צא מצפה ל־0 או 1 ארגומנטים")
	}
	os.Exit(code)
	return object.Nil
}

func sysCwd(args ...object.Object) object.Object {
	if len(args) != 0 {
		return errObj("מערכת.תיקייה מצפה ל־0 ארגומנטים")
	}
	dir, err := os.Getwd()
	if err != nil {
		return errObj("לא הצלחתי לקרוא תיקייה נוכחית: " + err.Error())
	}
	return &object.String{Value: dir}
}

func sysChdir(args ...object.Object) object.Object {
	if err := expectArgs("מערכת.שנה_תיקייה", 1, args); err != nil {
		return err
	}
	path, ok := asString(args[0])
	if !ok {
		return errObj("מערכת.שנה_תיקייה מצפה לנתיב מחרוזת")
	}
	if e := os.Chdir(path); e != nil {
		return errObj("לא הצלחתי לשנות תיקייה: " + e.Error())
	}
	return object.Nil
}

func sysHostname(args ...object.Object) object.Object {
	if len(args) != 0 {
		return errObj("מערכת.שם_מחשב מצפה ל־0 ארגומנטים")
	}
	h, err := os.Hostname()
	if err != nil {
		return errObj("לא הצלחתי לקרוא שם מחשב: " + err.Error())
	}
	return &object.String{Value: h}
}

func sysOSName(args ...object.Object) object.Object {
	if len(args) != 0 {
		return errObj("מערכת.מערכת_הפעלה מצפה ל־0 ארגומנטים")
	}
	return &object.String{Value: hebrewOSName(runtime.GOOS)}
}

func hebrewOSName(goos string) string {
	switch goos {
	case "windows":
		return "Windows"
	case "linux":
		return "Linux"
	case "darwin":
		return "macOS"
	case "freebsd":
		return "FreeBSD"
	default:
		return goos
	}
}

func sysOSVersion(args ...object.Object) object.Object {
	if len(args) != 0 {
		return errObj("מערכת.גרסת_הפעלה מצפה ל־0 ארגומנטים")
	}
	return &object.String{Value: osVersionString()}
}

func sysArch(args ...object.Object) object.Object {
	if len(args) != 0 {
		return errObj("מערכת.ארכיטקטורה מצפה ל־0 ארגומנטים")
	}
	return &object.String{Value: runtime.GOARCH}
}

func sysCores(args ...object.Object) object.Object {
	if len(args) != 0 {
		return errObj("מערכת.ליבות מצפה ל־0 ארגומנטים")
	}
	return &object.Number{Value: float64(runtime.NumCPU())}
}

func sysCPU(args ...object.Object) object.Object {
	if len(args) != 0 {
		return errObj("מערכת.מעבד מצפה ל־0 ארגומנטים")
	}
	name := cpuModelName()
	return &object.Hash{Pairs: map[string]object.Object{
		"ליבות":       &object.Number{Value: float64(runtime.NumCPU())},
		"ארכיטקטורה": &object.String{Value: runtime.GOARCH},
		"שם":         &object.String{Value: name},
	}}
}

func sysMemory(args ...object.Object) object.Object {
	if len(args) != 0 {
		return errObj("מערכת.זיכרון מצפה ל־0 ארגומנטים")
	}
	total, avail, ok := systemMemoryBytes()
	if !ok {
		return errObj("לא הצלחתי לקרוא מידע זיכרון")
	}
	const mb = 1024 * 1024
	used := total - avail
	if used < 0 {
		used = 0
	}
	pairs := map[string]object.Object{
		"סהכ_בתים":    &object.Number{Value: float64(total)},
		"פנוי_בתים":   &object.Number{Value: float64(avail)},
		"בשימוש_בתים": &object.Number{Value: float64(used)},
		"סהכ_מגה":     &object.Number{Value: float64(total / mb)},
		"פנוי_מגה":    &object.Number{Value: float64(avail / mb)},
		"בשימוש_מגה":  &object.Number{Value: float64(used / mb)},
	}
	if load, ok := systemMemoryLoadPercent(); ok {
		pairs["אחוז"] = &object.Number{Value: float64(load)}
	} else if total > 0 {
		pairs["אחוז"] = &object.Number{Value: float64((used * 100) / total)}
	}
	return &object.Hash{Pairs: pairs}
}

func sysIPs(args ...object.Object) object.Object {
	if len(args) != 0 {
		return errObj("מערכת.איפי מצפה ל־0 ארגומנטים")
	}
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return errObj("לא הצלחתי לקרוא כתובות IP: " + err.Error())
	}
	var ips []object.Object
	seen := map[string]bool{}
	for _, a := range addrs {
		var ip net.IP
		switch v := a.(type) {
		case *net.IPNet:
			ip = v.IP
		case *net.IPAddr:
			ip = v.IP
		}
		if ip == nil || ip.IsLoopback() || ip.IsLinkLocalUnicast() {
			continue
		}
		// העדפת IPv4 קריא; גם IPv6 נכלל
		s := ip.String()
		if s == "" || seen[s] {
			continue
		}
		seen[s] = true
		ips = append(ips, &object.String{Value: s})
	}
	return &object.Array{Elements: ips}
}

func sysUsername(args ...object.Object) object.Object {
	if len(args) != 0 {
		return errObj("מערכת.משתמש מצפה ל־0 ארגומנטים")
	}
	if u, err := user.Current(); err == nil && u != nil {
		name := u.Username
		if i := strings.LastIndex(name, `\`); i >= 0 {
			name = name[i+1:]
		}
		if i := strings.LastIndex(name, "/"); i >= 0 {
			name = name[i+1:]
		}
		return &object.String{Value: name}
	}
	if v := os.Getenv("USERNAME"); v != "" {
		return &object.String{Value: v}
	}
	if v := os.Getenv("USER"); v != "" {
		return &object.String{Value: v}
	}
	return &object.String{Value: ""}
}

func sysHome(args ...object.Object) object.Object {
	if len(args) != 0 {
		return errObj("מערכת.בית מצפה ל־0 ארגומנטים")
	}
	h, err := os.UserHomeDir()
	if err != nil {
		return errObj("לא הצלחתי לקרוא תיקיית בית: " + err.Error())
	}
	return &object.String{Value: h}
}

func sysInfo(args ...object.Object) object.Object {
	if len(args) != 0 {
		return errObj("מערכת.מידע מצפה ל־0 ארגומנטים")
	}
	host, _ := os.Hostname()
	ips := sysIPs()
	mem := sysMemory()
	cpu := sysCPU()
	userObj := sysUsername()
	homeObj := sysHome()
	pairs := map[string]object.Object{
		"שם_מחשב":      &object.String{Value: host},
		"מערכת_הפעלה":  &object.String{Value: hebrewOSName(runtime.GOOS)},
		"גרסת_הפעלה":   &object.String{Value: osVersionString()},
		"ארכיטקטורה":   &object.String{Value: runtime.GOARCH},
		"ליבות":        &object.Number{Value: float64(runtime.NumCPU())},
		"מעבד":         cpu,
		"זיכרון":       mem,
		"איפי":         ips,
		"משתמש":        userObj,
		"בית":          homeObj,
		"תיקייה":       sysCwd(),
		"שימוש_מעבד":   sysCPUUsage(),
		"זמן_פעיל":     sysUptime(),
		"כוננים":       sysDrives(),
	}
	return &object.Hash{Pairs: pairs}
}
