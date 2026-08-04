$ErrorActionPreference = 'Stop'
Set-StrictMode -Version Latest

$root = (Resolve-Path (Join-Path $PSScriptRoot '..')).Path
$installer = Join-Path $root 'packaging\windows\install.ps1'
$temporary = Join-Path ([IO.Path]::GetTempPath()) ("myference-service-installer-test-" + [guid]::NewGuid().ToString('N'))
$executable = Join-Path $temporary 'myference.exe'
$config = Join-Path $temporary 'config.json'

function Invoke-ServiceInstaller([switch]$Headless) {
    $capture = [pscustomobject]@{ Principal = $null }

    & {
        function New-ScheduledTaskAction { param($Execute, $Argument) [pscustomobject]@{ Execute = $Execute; Argument = $Argument } }
        function New-ScheduledTaskTrigger { param([switch]$AtLogOn, $User) [pscustomobject]@{ AtLogOn = $AtLogOn; User = $User } }
        function New-ScheduledTaskPrincipal {
            param($UserId, $LogonType, $RunLevel)
            $capture.Principal = [pscustomobject]@{ UserId = $UserId; LogonType = $LogonType; RunLevel = $RunLevel }
            $capture.Principal
        }
        function New-ScheduledTaskSettingsSet {
            param([switch]$AllowStartIfOnBatteries, [switch]$DontStopIfGoingOnBatteries, $ExecutionTimeLimit, $RestartCount, $RestartInterval)
            [pscustomobject]@{}
        }
        function Register-ScheduledTask { param($TaskName, $Action, $Trigger, $Principal, $Settings, [switch]$Force) }

        if ($Headless) {
            & $installer -Executable $executable -Config $config -Headless
        } else {
            & $installer -Executable $executable -Config $config
        }
    }

    return $capture.Principal
}

try {
    New-Item -ItemType Directory -Path $temporary | Out-Null
    New-Item -ItemType File -Path $executable, $config | Out-Null

    $normal = Invoke-ServiceInstaller
    if ($normal.RunLevel -ne 'Limited') {
        throw "Normal provider task run level was '$($normal.RunLevel)', expected 'Limited'"
    }

    $headless = Invoke-ServiceInstaller -Headless
    if ($headless.RunLevel -ne 'Highest') {
        throw "Headless provider task run level was '$($headless.RunLevel)', expected 'Highest'"
    }
} finally {
    Remove-Item -LiteralPath $temporary -Recurse -Force -ErrorAction SilentlyContinue
}

Write-Host 'Windows service installer tests passed'
