# Start Menu shortcut for Yod IDE + AppUserModelID (taskbar pin reopen).
# ASCII-only: Arguments use "editor" (same as Hebrew עורך in yod CLI).
param(
  [Parameter(Mandatory = $true)][string]$YodExe,
  [Parameter(Mandatory = $true)][string]$WorkDir,
  [Parameter(Mandatory = $true)][string]$IconIco,
  [string]$AppId = "il.yod.ide",
  [string]$ShortcutName = "Yod.lnk",
  [string]$Arguments = "."
)

$ErrorActionPreference = "Stop"

if (-not (Test-Path -LiteralPath $YodExe)) {
  throw "yod.exe not found: $YodExe"
}

$programs = Join-Path $env:APPDATA "Microsoft\Windows\Start Menu\Programs"
New-Item -ItemType Directory -Path $programs -Force | Out-Null
$lnkPath = Join-Path $programs $ShortcutName

$wshell = New-Object -ComObject WScript.Shell
$sc = $wshell.CreateShortcut($lnkPath)
$sc.TargetPath = (Resolve-Path -LiteralPath $YodExe).Path
$sc.Arguments = $Arguments
$sc.WorkingDirectory = (Resolve-Path -LiteralPath $WorkDir).Path
if (Test-Path -LiteralPath $IconIco) {
  $sc.IconLocation = ((Resolve-Path -LiteralPath $IconIco).Path) + ",0"
}
$sc.Description = "Yod IDE"
$sc.Save()
[System.Runtime.InteropServices.Marshal]::ReleaseComObject($sc) | Out-Null
[System.Runtime.InteropServices.Marshal]::ReleaseComObject($wshell) | Out-Null

try {
  $psCode = @'
using System;
using System.Runtime.InteropServices;

public static class ShortcutAumid {
  [DllImport("shell32.dll", CharSet = CharSet.Unicode, PreserveSig = false)]
  static extern void SHGetPropertyStoreFromParsingName(
    string pszPath, IntPtr pbc, int flags,
    [MarshalAs(UnmanagedType.LPStruct)] Guid riid, out IPropertyStore ppv);

  [ComImport, Guid("886D8EEB-8CF2-4446-8D02-CDBA1DBDCF99"), InterfaceType(ComInterfaceType.InterfaceIsIUnknown)]
  interface IPropertyStore {
    void GetCount(out uint cProps);
    void GetAt(uint iProp, out PropertyKey pkey);
    void GetValue(ref PropertyKey key, out PropVariant pv);
    void SetValue(ref PropertyKey key, ref PropVariant pv);
    void Commit();
  }

  [StructLayout(LayoutKind.Sequential, Pack = 4)]
  struct PropertyKey {
    public Guid fmtid;
    public uint pid;
  }

  [StructLayout(LayoutKind.Sequential)]
  struct PropVariant {
    public ushort vt;
    public ushort w1, w2, w3;
    public IntPtr p;
  }

  public static void Set(string path, string appId) {
    var iid = new Guid("886D8EEB-8CF2-4446-8D02-CDBA1DBDCF99");
    IPropertyStore store;
    SHGetPropertyStoreFromParsingName(path, IntPtr.Zero, 0x3, iid, out store);
    var key = new PropertyKey {
      fmtid = new Guid("9F4C2855-9F79-4B39-A8D0-E1D42DE1D5F3"),
      pid = 5
    };
    var pv = new PropVariant { vt = 31 };
    pv.p = Marshal.StringToCoTaskMemUni(appId);
    store.SetValue(ref key, ref pv);
    store.Commit();
    Marshal.FreeCoTaskMem(pv.p);
  }
}
'@
  if (-not ([System.Management.Automation.PSTypeName]'ShortcutAumid').Type) {
    Add-Type -TypeDefinition $psCode -ErrorAction Stop
  }
  [ShortcutAumid]::Set($lnkPath, $AppId)
  Write-Output "OK $lnkPath"
} catch {
  Write-Output "OK_NO_AUMID $lnkPath"
  Write-Output $_.Exception.Message
}
