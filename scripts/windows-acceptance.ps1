[CmdletBinding()]
param(
    [string]$Config,
    [switch]$NonDestructive
)

$ErrorActionPreference = 'Stop'
Set-StrictMode -Version Latest
$root = (Resolve-Path (Join-Path $PSScriptRoot '..')).Path
$shellProperty = Get-ItemProperty -LiteralPath 'HKCU:\Software\Microsoft\Windows NT\CurrentVersion\Winlogon' -Name Shell -ErrorAction SilentlyContinue
$shellValue = $null
if ($null -ne $shellProperty -and $shellProperty.PSObject.Properties.Name -contains 'Shell') { $shellValue = [string]$shellProperty.Shell }
$report = [ordered]@{
    timestamp = (Get-Date).ToUniversalTime().ToString('o')
    windows = [Environment]::OSVersion.VersionString
    architecture = [Runtime.InteropServices.RuntimeInformation]::OSArchitecture.ToString()
    powerScheme = (& powercfg.exe /getactivescheme | Out-String).Trim()
    shell = $shellValue
    myferenceTasks = @(Get-ScheduledTask -TaskName 'Myference*' -ErrorAction SilentlyContinue | Select-Object TaskName,State)
}

go test ./cli/internal/platform/windows ./cli/internal/provider ./cli/cmd/myference -count=1
if ($LASTEXITCODE -ne 0) { throw 'Windows acceptance Go tests failed' }

$temporary = Join-Path ([IO.Path]::GetTempPath()) ('myference-acceptance-' + [guid]::NewGuid().ToString('N') + '.json')
try {
    $report | ConvertTo-Json -Depth 5 | Set-Content -LiteralPath $temporary -Encoding utf8
    Get-Content -LiteralPath $temporary
    if ($NonDestructive) { exit 0 }
    if ([string]::IsNullOrWhiteSpace($Config)) { throw 'Full acceptance requires -Config pointing to a real provider configuration' }
    $cli = Join-Path $root 'myference.exe'
    if (-not (Test-Path -LiteralPath $cli)) { throw 'Build myference.exe in the repository root before full acceptance' }
    & $cli status --json --config $Config
    & $cli backend list --config $Config
    Write-Host 'Automated read-only checks passed. Complete every manual checkbox in docs/windows-acceptance.md.' -ForegroundColor Green
} finally {
    Remove-Item -LiteralPath $temporary -Force -ErrorAction SilentlyContinue
}
