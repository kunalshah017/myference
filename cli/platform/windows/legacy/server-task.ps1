$ErrorActionPreference = 'Stop'
$cli = Join-Path $PSScriptRoot 'myference.ps1'
try {
    & $cli start -NoFirewall -NoDashboard -Focus -HeadlessSession
    if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
    $stateFile = Join-Path $PSScriptRoot '.myference-state\server.json'
    while (Test-Path -LiteralPath $stateFile) {
        $state = Get-Content -LiteralPath $stateFile -Raw | ConvertFrom-Json
        if (-not (Get-Process -Id $state.pid -ErrorAction SilentlyContinue)) { break }
        Start-Sleep -Seconds 5
    }
} catch {
    $_ | Out-String | Set-Content (Join-Path $PSScriptRoot '.myference-state\headless-error.txt')
    exit 1
}
