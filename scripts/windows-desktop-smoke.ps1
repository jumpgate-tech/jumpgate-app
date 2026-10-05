# scripts/windows-desktop-smoke.ps1 STAGE_DIR — double-click jumpgate-tray.exe
# (no arguments, no console) and check it chose the desktop window, logged to
# app.log, served, and stops on `jumpgate.exe stop` (B-3).
param([Parameter(Mandatory)] [string] $Stage)
$ErrorActionPreference = 'Stop'
$Stage = (Resolve-Path $Stage).Path
# No `??`: Windows PowerShell 5.1, the one a stock Windows 10 has, lacks it.
$base = if ($env:RUNNER_TEMP) { $env:RUNNER_TEMP } else { $env:TEMP }
$root = Join-Path $base 'jgdesk'
Remove-Item -Recurse -Force $root -ErrorAction SilentlyContinue
New-Item -ItemType Directory -Force $root | Out-Null
$env:USERPROFILE = $root
$env:HOME = $root
$run = Join-Path $root '.jumpgate\run'

$p = Start-Process -FilePath (Join-Path $Stage 'jumpgate-tray.exe') -PassThru
$ok = $false
$deadline = (Get-Date).AddSeconds(30)
while ((Get-Date) -lt $deadline -and -not $ok) {
  $f = Join-Path $run 'server.json'
  if (Test-Path $f) {
    $info = Get-Content $f -Raw | ConvertFrom-Json
    try {
      $h = Invoke-WebRequest -UseBasicParsing -Headers @{ Authorization = "Bearer $($info.token)" } "http://$($info.httpAddr)/api/health"
      $ok = $h.StatusCode -eq 200
    } catch { }
  }
  if (-not $ok) { Start-Sleep -Milliseconds 500 }
}
$log = Join-Path $run 'app.log'
if (-not (Test-Path $log)) { throw 'the GUI exe wrote no app.log, so its errors would be invisible' }
Get-Content $log
if (-not (Select-String -Path $log -Pattern 'opening the desktop window' -Quiet)) { throw 'the GUI exe did not choose the desktop window' }
if (-not $ok) { throw 'the server never answered /api/health' }

& (Join-Path $Stage 'jumpgate.exe') stop
if ($LASTEXITCODE -ne 0) { throw "jumpgate.exe stop exited $LASTEXITCODE" }
if (-not $p.WaitForExit(15000)) { throw 'jumpgate-tray.exe kept running after stop' }
Write-Host 'windows-desktop-smoke: OK'
