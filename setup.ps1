# setup.ps1 — one-off setup for building si on Windows (PowerShell).
# Checks for Go (prints install instructions if missing), then builds .\bin\si.exe
$ErrorActionPreference = 'Stop'
$RequiredGo = '1.26'
Set-Location -Path $PSScriptRoot

function Say($m) { Write-Host $m -ForegroundColor Cyan }
function Fail($m) { Write-Host $m -ForegroundColor Red }

if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
  Fail "Go is not installed (or not on your PATH)."
  Write-Host ""
  Write-Host "Install Go $RequiredGo or newer, then re-run .\setup.ps1 :"
  Write-Host "  - winget:     winget install GoLang.Go"
  Write-Host "  - Chocolatey: choco install golang"
  Write-Host "  - or the Windows installer from https://go.dev/dl/"
  exit 1
}

$have = (go env GOVERSION) -replace '^go', ''
Say "Found Go $have"
if ([version]($have -replace '^(\d+\.\d+).*', '$1') -lt [version]$RequiredGo) {
  Fail "Warning: go.mod needs Go $RequiredGo+; you have $have. Upgrade if the build fails."
}

Say "Building .\bin\si.exe ..."
New-Item -ItemType Directory -Force -Path bin | Out-Null
go build -o bin\si.exe .\cmd\si

Say "Done. Try:"
Write-Host "  .\bin\si.exe --help"
Write-Host "  .\bin\si.exe new talk ; .\bin\si.exe play talk --open"
