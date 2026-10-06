# scripts/windows-smoke.ps1 - the controller's server lifecycle on real
# Windows (I-2, I-7): a CLI command auto-starts the server from inside a
# PowerShell job and the server outlives the job; a hard kill leaves a stale
# socket the next start recovers from; `jumpgate stop` ends it.
$ErrorActionPreference = 'Stop'
$root = Join-Path ($env:RUNNER_TEMP ?? $env:TEMP) 'jgsmoke'
Remove-Item -Recurse -Force $root -ErrorAction SilentlyContinue
New-Item -ItemType Directory -Force $root | Out-Null
$env:USERPROFILE = $root
$env:HOME = $root
$exe = Join-Path $root 'jg.exe'
go build -o $exe ./cmd/jumpgate
if ($LASTEXITCODE) { throw 'build failed' }
$run = Join-Path $root '.jumpgate\run'

function Get-Server {
  $deadline = (Get-Date).AddSeconds(15)
  while ((Get-Date) -lt $deadline) {
    $f = Join-Path $run 'server.json'
    if (Test-Path $f) {
      $info = Get-Content $f -Raw | ConvertFrom-Json
      try {
        $h = Invoke-WebRequest -UseBasicParsing -Headers @{ Authorization = "Bearer $($info.token)" } "http://$($info.httpAddr)/api/health"
        if ($h.StatusCode -eq 200) { return $info }
      } catch { }
    }
    Start-Sleep -Milliseconds 300
  }
  Get-Content (Join-Path $run 'server.log') -ErrorAction SilentlyContinue
  throw 'no healthy server'
}

# 1. Auto-start from inside a job, then end the job.
$job = Start-Job -ScriptBlock { param($e) & $e status nosuch-target 2>&1 | Out-Null } -ArgumentList $exe
Wait-Job $job | Out-Null
Remove-Job $job
$info = Get-Server
Write-Host "server pid $($info.pid) outlived the job that started it"

# 2. Kill it hard; the stale socket must not block the next start.
Stop-Process -Id $info.pid -Force
Start-Sleep -Seconds 1
& $exe status nosuch-target 2>&1 | Out-Null
$info2 = Get-Server
if ($info2.pid -eq $info.pid) { throw 'the same pid answered after a kill' }

# 3. Stop it over the API.
& $exe stop
if ($LASTEXITCODE -ne 0) { throw "stop exited $LASTEXITCODE" }
Start-Sleep -Seconds 2
if (Get-Process -Id $info2.pid -ErrorAction SilentlyContinue) { throw 'the server is still running after stop' }
Write-Host 'windows-smoke: OK'
