[CmdletBinding()]
param([switch]$Remove, [switch]$Focus)

$ErrorActionPreference = 'Stop'
$taskName = 'Myference Ollama LAN Server'

if (-not ([Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)) {
    throw 'Run this script from PowerShell as Administrator.'
}

if ($Remove) {
    Unregister-ScheduledTask -TaskName $taskName -Confirm:$false -ErrorAction SilentlyContinue
    Write-Host "Removed scheduled task: $taskName"
    exit 0
}

$cli = Join-Path $PSScriptRoot 'myference.ps1'
$arguments = "-NoProfile -ExecutionPolicy Bypass -File `"$cli`" start -NoDashboard"
if ($Focus) { $arguments += ' -Focus' }

$action = New-ScheduledTaskAction -Execute 'powershell.exe' -Argument $arguments
$trigger = New-ScheduledTaskTrigger -AtLogOn -User $env:USERNAME
$principal = New-ScheduledTaskPrincipal -UserId $env:USERNAME -LogonType Interactive -RunLevel Highest
$settings = New-ScheduledTaskSettingsSet -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries -ExecutionTimeLimit ([TimeSpan]::Zero)
Register-ScheduledTask -TaskName $taskName -Action $action -Trigger $trigger -Principal $principal -Settings $settings -Force | Out-Null
Write-Host "Installed scheduled task: $taskName"
