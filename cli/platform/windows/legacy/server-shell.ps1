$ErrorActionPreference = 'Continue'
$stateDir = Join-Path $PSScriptRoot '.myference-state'
$stateFile = Join-Path $stateDir 'server.json'
$headlessStateFile = Join-Path $stateDir 'headless-shell.json'
$errorFile = Join-Path $stateDir 'headless-error.txt'
$startTask = 'Myference Headless Start'
$stopTask = 'Myference Headless Stop'

Clear-Host
Write-Host 'MYFERENCE HEADLESS SERVER MODE' -ForegroundColor Cyan
Write-Host 'Starting Ollama without the Windows Explorer shell...'
if (Test-Path -LiteralPath $errorFile) { Remove-Item -LiteralPath $errorFile -Force }
Start-ScheduledTask -TaskName $startTask

$ready = $false
for ($i = 0; $i -lt 60; $i++) {
    if (Test-Path -LiteralPath $stateFile) { $ready = $true; break }
    if (Test-Path -LiteralPath $errorFile) { break }
    Start-Sleep -Seconds 1
}

if ($ready) {
    $state = Get-Content -LiteralPath $stateFile -Raw | ConvertFrom-Json
    $dashboard = Join-Path $PSScriptRoot 'myference-dashboard.exe'
    & $dashboard -view -metrics "http://127.0.0.1:$($state.port)/_myference/metrics"
} else {
    Write-Host 'Myference failed to start.' -ForegroundColor Red
    if (Test-Path -LiteralPath $errorFile) { Get-Content -LiteralPath $errorFile }
    Write-Host ''
    Write-Host 'Press Q to restore the normal shell and sign out.'
    while ($true) {
        $key = [Console]::ReadKey($true)
        if ($key.Key -eq [ConsoleKey]::Q) { break }
    }
}
Write-Host 'Stopping server and restoring Windows...'
Start-ScheduledTask -TaskName $stopTask
for ($i = 0; $i -lt 60; $i++) {
    $serverStopped = -not (Test-Path -LiteralPath $stateFile)
    $shellRestored = -not (Test-Path -LiteralPath $headlessStateFile)
    if ($serverStopped -and $shellRestored) { break }
    Start-Sleep -Seconds 1
}
shutdown.exe /l
