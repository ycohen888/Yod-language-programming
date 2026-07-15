# Pack Windows release with standalone Yod IDE (electron-builder).
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
$Unpacked = Join-Path $IdeSrc "release\win-unpacked"
$IdeExeName = "Yod IDE.exe"

if (-not (Test-Path $YodExe)) {
  Write-Host "Building yod.exe..."
  Push-Location (Join-Path $Root "yod")
  go build -o $YodExe .\cmd\yod
  Pop-Location
}

if (-not $SkipIdeBuild) {
  Write-Host "Building packaged Yod IDE (electron-builder)..."
  Push-Location $IdeSrc
  npm run dist
  Pop-Location
}

$IdeExe = Join-Path $Unpacked $IdeExeName
if (-not (Test-Path $IdeExe)) {
  throw "Missing packaged IDE at $IdeExe - run: cd yod-ide; npm run dist"
}

if (Test-Path $OutDir) { Remove-Item $OutDir -Recurse -Force }
New-Item -ItemType Directory -Path $OutDir | Out-Null

Copy-Item $YodExe (Join-Path $OutDir "yod.exe")
foreach ($ico in @("yod.ico", ([char]0x05D9 + [char]0x05D5 + [char]0x05D3 + ".ico"))) {
  $p = Join-Path $Root $ico
  if (Test-Path $p) { Copy-Item $p (Join-Path $OutDir (Split-Path $p -Leaf)) }
}
$ideIco = Join-Path $IdeSrc "build\icon.ico"
if (Test-Path $ideIco) {
  Copy-Item $ideIco (Join-Path $OutDir "yod.ico") -Force
}

$relNotes = Join-Path $Root "docs\RELEASE-v$Version.md"
if (Test-Path $relNotes) {
  Copy-Item $relNotes (Join-Path $OutDir "RELEASE.md")
}

$examplesName = (
  [char]0x05E4 + [char]0x05E8 + [char]0x05D5 + [char]0x05D9 + [char]0x05D9 + [char]0x05E7 + [char]0x05D8 +
  " " +
  [char]0x05D3 + [char]0x05D5 + [char]0x05D2 + [char]0x05DE + [char]0x05D4
) # פרוייקט דוגמה
$examples = Join-Path $Root $examplesName
if (-not (Test-Path $examples)) {
  Write-Warning "Examples folder not found: $examplesName"
} else {
  Write-Host "Copying examples..."
  Copy-Item $examples (Join-Path $OutDir $examplesName) -Recurse
}

$guideName = (
  [char]0x05DE + [char]0x05D3 + [char]0x05E8 + [char]0x05D9 + [char]0x05DA +
  " " +
  [char]0x05E9 + [char]0x05E4 + [char]0x05EA +
  " " +
  [char]0x05D9 + [char]0x05D5 + [char]0x05D3
) # מדריך שפת יוד
$guideSrc = Join-Path $Root $guideName
if (-not (Test-Path $guideSrc)) {
  Write-Warning "Guide folder not found: $guideName"
} else {
  Write-Host "Copying guide..."
  $guideOut = Join-Path $OutDir $guideName
  & robocopy $guideSrc $guideOut /E /NFL /NDL /NJH /NJS /nc /ns /np /XF gen_sections.py | Out-Null
  if ($LASTEXITCODE -ge 8) { throw "robocopy guide failed: $LASTEXITCODE" }
}

# אפליקציית Electron ארוזה ליד yod.exe (בלי node_modules)
Write-Host "Copying win-unpacked -> release folder..."
& robocopy $Unpacked $OutDir /E /NFL /NDL /NJH /NJS /nc /ns /np | Out-Null
if ($LASTEXITCODE -ge 8) { throw "robocopy IDE failed: $LASTEXITCODE" }

$readmeLines = @(
  "Yod $Version - Windows (amd64)",
  "",
  "How to run",
  "----------",
  "1. Extract the FULL folder (do not run only one file from inside the zip).",
  "2. Run yod.exe - opens Yod IDE (Electron app bundled beside yod.exe).",
  "3. Or double-click '" + $IdeExeName + "'.",
  "4. Or: yod.exe editor / yod.exe עורך",
  "5. Examples: Hebrew-named folder next to yod.exe.",
  "6. Guide: folder '" + $guideName + "' - open from Help menu (window inside the IDE).",
  "",
  "No Node.js required to use the editor.",
  "There is no Win32 legacy editor."
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
