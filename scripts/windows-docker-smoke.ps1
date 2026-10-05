# Drives a real jumpgate.exe through the local Docker features (plan Task 15).
#
#   -Expect windows-containers  The engine runs Windows containers: GitHub's
#                               runners, or a Windows Server VM with Moby.
#                               /api/docker and the containers view must both
#                               carry the Linux-containers hint.
#   -Expect linux-containers    Docker Desktop in Linux mode, on a real
#                               Windows desktop. /api/docker must say running,
#                               with no hint. Provisioning itself is covered by
#                               TestLocalDockerLive, run on the same machine.
#
#   -Exe C:\path\jumpgate.exe   Use this binary instead of building one, for a
#                               machine without Go (the QEMU VM).
param(
  [Parameter(Mandatory)][ValidateSet('windows-containers', 'linux-containers')][string]$Expect,
  [string]$Exe
)
$ErrorActionPreference = 'Stop'
$tmp = if ($env:RUNNER_TEMP) { $env:RUNNER_TEMP } else { $env:TEMP }
$work = Join-Path $tmp ("jg-docker-smoke-" + [guid]::NewGuid().ToString('N').Substring(0, 8))
$home_ = Join-Path $work 'home'
New-Item -ItemType Directory -Force -Path $home_ | Out-Null
if (-not $Exe) {
  $Exe = Join-Path $work 'jumpgate.exe'
  go build -o $Exe ./cmd/jumpgate
  if ($LASTEXITCODE -ne 0) { throw 'go build failed' }
}
$env:USERPROFILE = $home_
$env:HOME = $home_
$port = 18799
$srv = Start-Process -FilePath $Exe -ArgumentList 'serve', '--bind', "127.0.0.1:$port" -PassThru -NoNewWindow `
  -RedirectStandardError (Join-Path $work 'serve.err') -RedirectStandardOutput (Join-Path $work 'serve.out')
try {
  $infoFile = Join-Path $home_ '.jumpgate\run\server.json'
  $info = $null
  for ($i = 0; $i -lt 60 -and -not $info; $i++) {
    Start-Sleep -Seconds 1
    if (Test-Path $infoFile) { $info = Get-Content $infoFile -Raw | ConvertFrom-Json }
  }
  if (-not $info) { throw "server.json never appeared: $(Get-Content (Join-Path $work 'serve.err') -Raw)" }
  $h = @{ Authorization = "Bearer $($info.token)" }
  $base = "http://127.0.0.1:$port"

  $docker = Invoke-RestMethod "$base/api/docker" -Headers $h
  Invoke-RestMethod "$base/api/targets" -Method Post -Headers $h -ContentType 'application/json' `
    -Body '{"id":"me","mode":"local"}' | Out-Null
  $list = Invoke-RestMethod "$base/api/targets/me/containers" -Headers $h

  if ($Expect -eq 'windows-containers') {
    if (-not $docker.windowsContainers -or $docker.running -or $docker.hint -notmatch 'Linux containers') {
      throw "GET /api/docker: $($docker | ConvertTo-Json -Compress)"
    }
    if ($list.docker.hint -notmatch 'Linux containers') {
      throw "containers view: $($list.docker | ConvertTo-Json -Compress)"
    }
    Write-Output "windows-container mode reaches the user: $($docker.hint)"
  } else {
    if (-not $docker.running -or $docker.hint) { throw "GET /api/docker: $($docker | ConvertTo-Json -Compress)" }
    if ($list.docker.hint) { throw "containers view has a hint: $($list.docker.hint)" }
    Write-Output 'Linux containers: ready'
  }
} finally {
  Stop-Process -Id $srv.Id -Force -ErrorAction SilentlyContinue
}
