# Pack Windows release with Electron IDE (not yod.exe alone).
# From repo root:
#   powershell -File yod\tools\pack_windows_release.ps1
#   powershell -File yod\tools\pack_windows_release.ps1 -SkipIdeBuild

param(
  [string]$Version = "0.77.0",
  [switch]$SkipIdeBuild
)

$ErrorActionPreference = "Stop"
$Root = Resolve-Path (Join-Path $PSScriptRoot "..\..")
Set-Location $Root

$OutName = "yod-$Version-windows-amd64"
$OutDir = Join-Path $Root $OutName
$ZipPath = Join-Path $Root "$OutName.zip"
$YodExe = Join-Path $Root "yod.exe"
$IdeSrc = Join-Path $Root "yod-ide"

if (-not (Test-Path $YodExe)) {
  Write-Host "Building yod.exe..."
  Push-Location (Join-Path $Root "yod")
  go build -o $YodExe .\cmd\yod
  Pop-Location
}

if (-not $SkipIdeBuild) {
  Write-Host "Building yod-ide..."
  Push-Location $IdeSrc
  npm run build
  Pop-Location
}

if (-not (Test-Path (Join-Path $IdeSrc "dist\index.html"))) {
  throw "Missing yod-ide/dist - run: cd yod-ide; npm run build"
}

$electronDist = Join-Path $IdeSrc "node_modules\electron\dist"
$yodElectron = Join-Path $electronDist ([char]0x05D9 + [char]0x05D5 + [char]0x05D3 + ".exe") # יוד.exe
$plainElectron = Join-Path $electronDist "electron.exe"
if (-not (Test-Path $yodElectron) -and -not (Test-Path $plainElectron)) {
  throw "Missing Electron - run: cd yod-ide; npm install"
}

if (Test-Path $OutDir) { Remove-Item $OutDir -Recurse -Force }
New-Item -ItemType Directory -Path $OutDir | Out-Null

Copy-Item $YodExe (Join-Path $OutDir "yod.exe")
foreach ($ico in @("yod.ico", ([char]0x05D9 + [char]0x05D5 + [char]0x05D3 + ".ico"))) {
  $p = Join-Path $Root $ico
  if (Test-Path $p) { Copy-Item $p (Join-Path $OutDir (Split-Path $p -Leaf)) }
}

$relNotes = Join-Path $Root "docs\RELEASE-v$Version.md"
if (Test-Path $relNotes) {
  Copy-Item $relNotes (Join-Path $OutDir "RELEASE.md")
}

$examplesName = [char]0x05E4 + [char]0x05E8 + [char]0x05D5 + [char]0x05D9 + [char]0x05E7 + [char]0x05D8 + " " + [char]0x05D3 + [char]0x05D5 + [char]0x05D2 + [char]0x05DE + [char]0x05D4
$examples = Join-Path $Root $examplesName
if (Test-Path $examples) {
  Copy-Item $examples (Join-Path $OutDir $examplesName) -Recurse
}

$IdeOut = Join-Path $OutDir "yod-ide"
New-Item -ItemType Directory -Path $IdeOut | Out-Null
Copy-Item (Join-Path $IdeSrc "package.json") (Join-Path $IdeOut "package.json")
Copy-Item (Join-Path $IdeSrc "electron") (Join-Path $IdeOut "electron") -Recurse
Copy-Item (Join-Path $IdeSrc "dist") (Join-Path $IdeOut "dist") -Recurse
foreach ($extra in @("build", "public")) {
  $src = Join-Path $IdeSrc $extra
  if (Test-Path $src) {
    Copy-Item $src (Join-Path $IdeOut $extra) -Recurse
  }
}

$ePkgSrc = Join-Path $IdeSrc "node_modules\electron"
$ePkgOut = Join-Path $IdeOut "node_modules\electron"
New-Item -ItemType Directory -Path (Join-Path $IdeOut "node_modules") -Force | Out-Null
& robocopy $ePkgSrc $ePkgOut /E /NFL /NDL /NJH /NJS /nc /ns /np | Out-Null
if ($LASTEXITCODE -ge 8) { throw "robocopy electron failed: $LASTEXITCODE" }

$outYod = Join-Path $ePkgOut ("dist\" + [char]0x05D9 + [char]0x05D5 + [char]0x05D3 + ".exe")
$outEle = Join-Path $ePkgOut "dist\electron.exe"
if ((Test-Path $outYod) -and (Test-Path $outEle)) {
  Remove-Item $outEle -Force
}

$readmeLines = @(
  "Yod $Version - Windows (amd64)",
  "",
  "How to run",
  "----------",
  "1. Extract the FULL folder (do not run only the exe from inside the zip).",
  "2. Run yod.exe - opens the new Electron editor (folder yod-ide required).",
  "3. Or: yod.exe editor",
  "4. Examples: folder next to yod.exe (Hebrew name).",
  "",
  "If you delete yod-ide, yod.exe falls back to the old Win32 editor.",
  "",
  "Force legacy editor:",
  "  set YOD_LEGACY_EDITOR=1",
  "  yod.exe editor"
)
[IO.File]::WriteAllLines((Join-Path $OutDir "README.txt"), $readmeLines, [Text.UTF8Encoding]::new($true))

if (Test-Path $ZipPath) { Remove-Item $ZipPath -Force }
Add-Type -AssemblyName System.IO.Compression.FileSystem
Write-Host "Creating ZIP..."
[IO.Compression.ZipFile]::CreateFromDirectory($OutDir, $ZipPath, [IO.Compression.CompressionLevel]::Optimal, $false)

$zipMB = [math]::Round((Get-Item $ZipPath).Length / 1MB, 1)
$folderMB = [math]::Round(((Get-ChildItem $OutDir -Recurse -File | Measure-Object Length -Sum).Sum) / 1MB, 1)
Write-Host ("Ready: {0} ({1} MB zip, {2} MB unpacked)" -f $ZipPath, $zipMB, $folderMB)
Write-Host ("Folder: {0}" -f $OutDir)
