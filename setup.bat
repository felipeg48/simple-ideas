@echo off
REM setup.bat - one-off setup for building si on Windows (cmd.exe).
REM Checks for Go (prints install instructions if missing), then builds bin\si.exe
setlocal
cd /d "%~dp0"

where go >nul 2>nul
if errorlevel 1 (
  echo Go is not installed ^(or not on your PATH^).
  echo.
  echo Install Go 1.26 or newer, then re-run setup.bat :
  echo   - winget:     winget install GoLang.Go
  echo   - Chocolatey: choco install golang
  echo   - or the Windows installer from https://go.dev/dl/
  exit /b 1
)

for /f "tokens=*" %%v in ('go env GOVERSION') do set GOVER=%%v
echo Found Go %GOVER%

echo Building bin\si.exe ...
if not exist bin mkdir bin
go build -o bin\si.exe .\cmd\si
if errorlevel 1 (
  echo Build failed. Ensure your Go is version 1.26 or newer.
  exit /b 1
)

echo Done. Try:
echo   bin\si.exe --help
echo   bin\si.exe new talk ^&^& bin\si.exe play talk --open
endlocal
