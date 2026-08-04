[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)][string]$Executable,
    [Parameter(Mandatory = $true)][string]$Config,
    [switch]$Remove,
    [switch]$Headless
)

$ErrorActionPreference = 'Stop'
$taskName = if ($Headless) { 'Myference Headless Provider' } else { 'Myference Provider' }

if ($Remove) {
    Unregister-ScheduledTask -TaskName $taskName -Confirm:$false -ErrorAction SilentlyContinue
    exit 0
}

$executablePath = (Resolve-Path -LiteralPath $Executable).Path
$configPath = (Resolve-Path -LiteralPath $Config).Path
$argument = if ($Headless) { 'windows headless run --config "{0}"' -f $configPath } else { 'serve --config "{0}"' -f $configPath }
$action = New-ScheduledTaskAction -Execute $executablePath -Argument $argument
$trigger = New-ScheduledTaskTrigger -AtLogOn -User $env:USERNAME
$runLevel = if ($Headless) { 'Highest' } else { 'Limited' }
$principal = New-ScheduledTaskPrincipal -UserId $env:USERNAME -LogonType Interactive -RunLevel $runLevel
$settings = New-ScheduledTaskSettingsSet -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries -ExecutionTimeLimit ([TimeSpan]::Zero) -RestartCount 5 -RestartInterval (New-TimeSpan -Minutes 1)
Register-ScheduledTask -TaskName $taskName -Action $action -Trigger $trigger -Principal $principal -Settings $settings -Force | Out-Null
