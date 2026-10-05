# jumpgate-firstboot.ps1: runs once from autounattend FirstLogonCommands as
# Administrator. It turns a fresh Windows Server 2022 Core install into an SSH
# test box: OpenSSH server (key auth only, PowerShell shell), firewall open on
# TCP 22, no surprise update reboots, and Defender exclusions for the build dirs.
# The public key comes from jumpgate_ed25519.pub on the same answer ISO.
$ErrorActionPreference = 'Continue'
[Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12
Start-Transcript -Path C:\jumpgate-firstboot.transcript.txt -Append

$pub = Get-Content -Raw (Join-Path $PSScriptRoot 'jumpgate_ed25519.pub')

# The evaluation's default 42-day password expiry would lock out autologon tools.
net accounts /maxpwage:unlimited

# No automatic Windows Update installs or reboots in the middle of a test run.
New-Item -Force 'HKLM:\SOFTWARE\Policies\Microsoft\Windows\WindowsUpdate\AU' | Out-Null
Set-ItemProperty 'HKLM:\SOFTWARE\Policies\Microsoft\Windows\WindowsUpdate\AU' NoAutoUpdate 1 -Type DWord

# Defender scanning every compiled test binary makes go test several times slower.
foreach ($p in 'C:\jg', 'C:\Program Files\Go', "$env:LOCALAPPDATA", 'C:\Windows\Temp') {
  Add-MpPreference -ExclusionPath $p -ErrorAction SilentlyContinue
}

# OpenSSH server: the Windows capability first (needs Windows Update reachable),
# then the official Win32-OpenSSH MSI from Microsoft's GitHub releases.
$cap = Get-WindowsCapability -Online -Name 'OpenSSH.Server*' | Select-Object -First 1
if ($cap -and $cap.State -ne 'Installed') {
  for ($i = 0; $i -lt 3 -and -not (Get-Service sshd -ErrorAction SilentlyContinue); $i++) {
    Add-WindowsCapability -Online -Name $cap.Name
    if (-not (Get-Service sshd -ErrorAction SilentlyContinue)) { Start-Sleep 30 }
  }
}
if (-not (Get-Service sshd -ErrorAction SilentlyContinue)) {
  Write-Output 'capability install failed; falling back to the Win32-OpenSSH MSI'
  $rel = Invoke-RestMethod -UseBasicParsing 'https://api.github.com/repos/PowerShell/Win32-OpenSSH/releases/latest'
  $msi = $rel.assets | Where-Object name -like 'OpenSSH-Win64-v*.msi' | Select-Object -First 1
  Invoke-WebRequest -UseBasicParsing $msi.browser_download_url -OutFile C:\Windows\Temp\openssh.msi
  Start-Process msiexec.exe -Wait -ArgumentList '/i', 'C:\Windows\Temp\openssh.msi', '/qn', '/norestart'
}

Set-Service sshd -StartupType Automatic
Start-Service sshd   # first start generates host keys and C:\ProgramData\ssh

New-ItemProperty -Force -Path 'HKLM:\SOFTWARE\OpenSSH' -Name DefaultShell -PropertyType String `
  -Value 'C:\Windows\System32\WindowsPowerShell\v1.0\powershell.exe' | Out-Null

Remove-NetFirewallRule -Name 'jumpgate-sshd' -ErrorAction SilentlyContinue
New-NetFirewallRule -Name 'jumpgate-sshd' -DisplayName 'OpenSSH Server (jumpgate, TCP 22)' `
  -Direction Inbound -Protocol TCP -LocalPort 22 -Action Allow -Profile Any | Out-Null

# Administrators authenticate with this file only; sshd rejects it unless just
# SYSTEM and Administrators can access it.
$keys = 'C:\ProgramData\ssh\administrators_authorized_keys'
New-Item -ItemType Directory -Force C:\ProgramData\ssh | Out-Null
[IO.File]::WriteAllText($keys, $pub.Trim() + "`n", (New-Object Text.ASCIIEncoding))
icacls.exe $keys /inheritance:r /grant '*S-1-5-32-544:F' /grant '*S-1-5-18:F'

# Key auth only.
$cfg = 'C:\ProgramData\ssh\sshd_config'
if (Test-Path $cfg) {
  $c = Get-Content $cfg | Where-Object { $_ -notmatch '^\s*(PasswordAuthentication|KbdInteractiveAuthentication|ChallengeResponseAuthentication)\s' }
  Set-Content -Encoding ascii $cfg (@('PasswordAuthentication no', 'ChallengeResponseAuthentication no') + $c)
}
Restart-Service sshd

New-Item -ItemType Directory -Force C:\jg | Out-Null
Set-Content C:\jumpgate-firstboot.done (Get-Date -Format o)
Stop-Transcript
