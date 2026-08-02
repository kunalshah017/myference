[CmdletBinding()]
param(
    [Parameter(Position = 0)]
    [ValidateSet('doctor', 'status', 'start', 'stop', 'dashboard', 'focus', 'restore', 'test', 'models', 'lan-check', 'headless', 'headless-install', 'headless-stop', 'headless-restore', 'headless-status', 'help')]
    [string]$Command = 'help',
    [int]$Port = 0,
    [string]$Model,
    [switch]$Apply,
    [switch]$Focus,
    [switch]$Exclusive,
    [switch]$NoFirewall,
    [switch]$NoDashboard,
    [switch]$HeadlessSession
)

$ErrorActionPreference = 'Stop'
$stateDir = Join-Path $PSScriptRoot '.myference-state'
$serverStateFile = Join-Path $stateDir 'server.json'
$focusStateFile = Join-Path $stateDir 'focus.json'
$lidStateFile = Join-Path $stateDir 'headless-lid.json'
$configFile = Join-Path $PSScriptRoot 'myference.config.json'
$firewallRule = 'Myference - Ollama LAN'
$headlessStateFile = Join-Path $stateDir 'headless-shell.json'
$headlessPolicyPath = 'HKCU:\Software\Microsoft\Windows\CurrentVersion\Policies\System'
$headlessStartTask = 'Myference Headless Start'
$headlessStopTask = 'Myference Headless Stop'
$dashboardExe = Join-Path $PSScriptRoot 'myference-dashboard.exe'

$defaultConfig = [ordered]@{
    port = 11434
    backendPort = 11435
    allowedRemoteAddress = 'LocalSubnet'
    keepAlive = '-1'
    preloadModel = $null
    maxLoadedModels = 1
    numParallel = 1
    contextLength = 4096
    flashAttention = $true
    kvCacheType = 'q8_0'
    performancePowerPlan = $true
    processPriority = 'High'
    requireACPower = $true
    stopServices = @('AdobeUpdateService', 'DiagTrack', 'DoSvc', 'Everything', 'MapsBroker', 'PhoneSvc', 'PCManager Service Store', 'Spooler', 'SysMain', 'UsoSvc', 'WSearch', 'WSLService', 'VMAuthdService', 'VMnetDHCP', 'VMUSBArbService', 'VMware NAT Service')
    stopProcesses = @('OneDrive', 'ms-teams', 'Teams', 'Discord', 'Spotify', 'steam', 'EpicGamesLauncher', 'Dropbox', 'Creative Cloud', 'NVIDIA Broadcast')
}

function Get-Config {
    $config = [ordered]@{}
    foreach ($key in $defaultConfig.Keys) { $config[$key] = $defaultConfig[$key] }
    if (Test-Path -LiteralPath $configFile) {
        $loaded = Get-Content -LiteralPath $configFile -Raw | ConvertFrom-Json
        foreach ($property in $loaded.PSObject.Properties) { $config[$property.Name] = $property.Value }
    }
    if ($Port -gt 0) { $config.port = $Port }
    if ([int]$config.port -lt 1 -or [int]$config.port -gt 65535) { throw 'Port must be between 1 and 65535.' }
    if ([int]$config.backendPort -lt 1 -or [int]$config.backendPort -gt 65535) { throw 'backendPort must be between 1 and 65535.' }
    if ([int]$config.backendPort -eq [int]$config.port) { throw 'backendPort must differ from the public port.' }
    if ([int]$config.contextLength -lt 512) { throw 'contextLength must be at least 512.' }
    if ($config.kvCacheType -notin @('f16', 'q8_0', 'q4_0')) { throw 'kvCacheType must be f16, q8_0, or q4_0.' }
    if ($config.processPriority -notin @('Normal', 'AboveNormal', 'High')) { throw 'processPriority must be Normal, AboveNormal, or High.' }
    return $config
}

function Test-Administrator {
    return ([Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
}

function Require-Administrator([string]$Action) {
    if (-not (Test-Administrator)) { throw "$Action requires Administrator rights. Open PowerShell as Administrator and retry." }
}

function Ensure-StateDirectory {
    if (-not (Test-Path -LiteralPath $stateDir)) { New-Item -ItemType Directory -Path $stateDir | Out-Null }
}

function Get-OllamaCommand {
    $command = Get-Command ollama.exe -ErrorAction SilentlyContinue
    if ($command) { return $command.Source }
    $fallbacks = @(
        (Join-Path $env:LOCALAPPDATA 'Programs\Ollama\ollama.exe'),
        (Join-Path $env:ProgramFiles 'Ollama\ollama.exe')
    )
    foreach ($path in $fallbacks) { if (Test-Path -LiteralPath $path) { return $path } }
    throw 'Ollama was not found. Install it from https://ollama.com/download/windows and retry.'
}

function Get-LanAddresses {
    @(Get-NetIPAddress -AddressFamily IPv4 -AddressState Preferred -ErrorAction SilentlyContinue |
        Where-Object { $_.IPAddress -notlike '127.*' -and $_.IPAddress -notlike '169.254.*' -and $_.InterfaceAlias -notmatch 'Loopback|vEthernet|WSL|Docker|VMware|VirtualBox|Hyper-V' } |
        Sort-Object InterfaceMetric |
        Select-Object -ExpandProperty IPAddress -Unique)
}

function Write-Endpoints([int]$ListenPort) {
    $addresses = Get-LanAddresses
    if ($addresses.Count -eq 0) {
        Write-Warning 'No LAN IPv4 address was detected.'
        return
    }
    Write-Host 'Available to other devices at:' -ForegroundColor Green
    foreach ($address in $addresses) {
        Write-Host "  Ollama API: http://${address}:$ListenPort"
        Write-Host "  OpenAI API: http://${address}:$ListenPort/v1"
    }
}

function Get-ServerState {
    if (Test-Path -LiteralPath $serverStateFile) {
        try { return Get-Content -LiteralPath $serverStateFile -Raw | ConvertFrom-Json } catch { return $null }
    }
    return $null
}

function Test-Port([int]$ListenPort) {
    try {
        Invoke-RestMethod -Uri "http://127.0.0.1:$ListenPort/api/tags" -TimeoutSec 3 | Out-Null
        return $true
    } catch { return $false }
}

function Test-LanListener([int]$ListenPort) {
    $listeners = @(Get-NetTCPConnection -State Listen -LocalPort $ListenPort -ErrorAction SilentlyContinue)
    return @($listeners | Where-Object { $_.LocalAddress -in @('0.0.0.0', '::') }).Count -gt 0
}

function Test-OnACPower {
    $batteries = @(Get-CimInstance -ClassName Win32_Battery -ErrorAction SilentlyContinue)
    if ($batteries.Count -eq 0) { return $true }
    return @($batteries | Where-Object { $_.BatteryStatus -eq 1 }).Count -eq 0
}

function Start-KeepAwakeWorker {
    $hostExe = (Get-Process -Id $PID).Path
    $script = Join-Path $PSScriptRoot 'keepawake.ps1'
    $quotedScript = [char]34 + $script + [char]34
    return Start-Process -FilePath $hostExe -ArgumentList @('-NoProfile', '-ExecutionPolicy', 'Bypass', '-File', $quotedScript) -WindowStyle Hidden -PassThru
}


function Start-MyferenceServer($config) {
    $ollama = Get-OllamaCommand
    $listenPort = [int]$config.port
    $backendPort = [int]$config.backendPort

    if (-not (Test-Path -LiteralPath $dashboardExe)) {
        throw "Dashboard gateway is missing: $dashboardExe. Run go build -o myference-dashboard.exe dashboard.go"
    }
    if ($config.requireACPower -and -not (Test-OnACPower)) {
        throw 'This laptop is running on battery. Connect AC power or set requireACPower to false in myference.config.json.'
    }
    foreach ($candidate in @($listenPort, $backendPort)) {
        if (Get-NetTCPConnection -State Listen -LocalPort $candidate -ErrorAction SilentlyContinue) {
            throw "Port $candidate is already in use. Quit the existing Ollama/server process before starting Myference."
        }
    }

    if (-not $NoFirewall) {
        Require-Administrator 'Creating the LAN firewall rule'
        Get-NetFirewallRule -DisplayName $firewallRule -ErrorAction SilentlyContinue | Remove-NetFirewallRule
        New-NetFirewallRule -DisplayName $firewallRule -Direction Inbound -Action Allow -Protocol TCP -LocalPort $listenPort -Profile Private -RemoteAddress $config.allowedRemoteAddress | Out-Null
    }

    Ensure-StateDirectory
    $previousPowerScheme = Get-ActivePowerScheme
    if ($config.performancePowerPlan) {
        powercfg /setactive SCHEME_MIN | Out-Null
    }

    $environmentNames = @('OLLAMA_HOST', 'OLLAMA_KEEP_ALIVE', 'OLLAMA_MAX_LOADED_MODELS', 'OLLAMA_NUM_PARALLEL', 'OLLAMA_CONTEXT_LENGTH', 'OLLAMA_FLASH_ATTENTION', 'OLLAMA_KV_CACHE_TYPE')
    $oldEnvironment = @{}
    foreach ($name in $environmentNames) { $oldEnvironment[$name] = [Environment]::GetEnvironmentVariable($name, 'Process') }
    $env:OLLAMA_HOST = "127.0.0.1:$backendPort"
    $env:OLLAMA_KEEP_ALIVE = [string]$config.keepAlive
    $env:OLLAMA_MAX_LOADED_MODELS = [string]$config.maxLoadedModels
    $env:OLLAMA_NUM_PARALLEL = [string]$config.numParallel
    $env:OLLAMA_CONTEXT_LENGTH = [string]$config.contextLength
    $env:OLLAMA_FLASH_ATTENTION = if ($config.flashAttention) { '1' } else { '0' }
    $env:OLLAMA_KV_CACHE_TYPE = [string]$config.kvCacheType
    try {
        $process = Start-Process -FilePath $ollama -ArgumentList 'serve' -WindowStyle Hidden -PassThru
    } catch {
        if ($config.performancePowerPlan -and $previousPowerScheme) { powercfg /setactive $previousPowerScheme | Out-Null }
        if (-not $NoFirewall) { Remove-NetFirewallRule -DisplayName $firewallRule -ErrorAction SilentlyContinue }
        throw
    } finally {
        foreach ($name in $environmentNames) { [Environment]::SetEnvironmentVariable($name, $oldEnvironment[$name], 'Process') }
    }

    $backendReady = $false
    for ($i = 0; $i -lt 30; $i++) {
        Start-Sleep -Milliseconds 500
        if (Test-Port $backendPort) { $backendReady = $true; break }
        if ($process.HasExited) { break }
    }
    if (-not $backendReady) {
        if (-not $process.HasExited) { Stop-Process -Id $process.Id -Force -ErrorAction SilentlyContinue }
        if (-not $NoFirewall) { Remove-NetFirewallRule -DisplayName $firewallRule -ErrorAction SilentlyContinue }
        if ($config.performancePowerPlan -and $previousPowerScheme) { powercfg /setactive $previousPowerScheme | Out-Null }
        throw 'The private Ollama backend did not become ready.'
    }

    $gateway = Start-Process -FilePath $dashboardExe -ArgumentList @('-serve', '-listen', "0.0.0.0:$listenPort", '-backend', "http://127.0.0.1:$backendPort") -WindowStyle Hidden -PassThru
    $gatewayReady = $false
    for ($i = 0; $i -lt 20; $i++) {
        Start-Sleep -Milliseconds 250
        if ((Test-Port $listenPort) -and (Test-LanListener $listenPort)) { $gatewayReady = $true; break }
        if ($gateway.HasExited) { break }
    }
    if (-not $gatewayReady) {
        if (-not $gateway.HasExited) { Stop-Process -Id $gateway.Id -Force -ErrorAction SilentlyContinue }
        if (-not $process.HasExited) { Stop-Process -Id $process.Id -Force -ErrorAction SilentlyContinue }
        if (-not $NoFirewall) { Remove-NetFirewallRule -DisplayName $firewallRule -ErrorAction SilentlyContinue }
        if ($config.performancePowerPlan -and $previousPowerScheme) { powercfg /setactive $previousPowerScheme | Out-Null }
        throw 'The Myference LAN gateway did not become ready.'
    }

    $preloadSeconds = $null
    if (-not [string]::IsNullOrWhiteSpace([string]$config.preloadModel)) {
        $preloadName = [string]$config.preloadModel
        Write-Host "Preloading $preloadName into memory; first startup may take a while..." -ForegroundColor Cyan
        $preloadWatch = [Diagnostics.Stopwatch]::StartNew()
        try {
            $preloadKeepAlive = if ([string]$config.keepAlive -match '^-?\d+$') { [int]$config.keepAlive } else { [string]$config.keepAlive }
            $preloadBody = @{
                model = $preloadName
                stream = $false
                keep_alive = $preloadKeepAlive
                options = @{ num_ctx = [int]$config.contextLength }
            } | ConvertTo-Json -Depth 4
            Invoke-RestMethod -Method Post -Uri "http://127.0.0.1:$backendPort/api/generate" -ContentType 'application/json' -Body $preloadBody -TimeoutSec 900 | Out-Null
            $preloadWatch.Stop()
            $preloadSeconds = [math]::Round($preloadWatch.Elapsed.TotalSeconds, 1)
            Write-Host "$preloadName is warm and ready ($preloadSeconds seconds)." -ForegroundColor Green
        } catch {
            $preloadWatch.Stop()
            if (-not $gateway.HasExited) { Stop-Process -Id $gateway.Id -Force -ErrorAction SilentlyContinue }
            if (-not $process.HasExited) { Stop-Process -Id $process.Id -Force -ErrorAction SilentlyContinue }
            if (-not $NoFirewall) { Remove-NetFirewallRule -DisplayName $firewallRule -ErrorAction SilentlyContinue }
            if ($config.performancePowerPlan -and $previousPowerScheme) { powercfg /setactive $previousPowerScheme | Out-Null }
            throw "Could not preload '$preloadName'. Confirm it is installed with 'ollama list'. $($_.Exception.Message)"
        }
    }

    $lidRestore = $null
    if ($HeadlessSession) {
        $lidScheme = Get-ActivePowerScheme
        $previousAcLidAction = Get-LidActionIndex $lidScheme 'AC'
        $previousDcLidAction = Get-LidActionIndex $lidScheme 'DC'
        if ($null -eq $previousAcLidAction -or $null -eq $previousDcLidAction) {
            if (-not $gateway.HasExited) { Stop-Process -Id $gateway.Id -Force -ErrorAction SilentlyContinue }
            if (-not $process.HasExited) { Stop-Process -Id $process.Id -Force -ErrorAction SilentlyContinue }
            if (-not $NoFirewall) { Remove-NetFirewallRule -DisplayName $firewallRule -ErrorAction SilentlyContinue }
            if ($config.performancePowerPlan -and $previousPowerScheme) { powercfg /setactive $previousPowerScheme | Out-Null }
            throw 'Could not read the current lid-close actions; refusing to enter an untracked headless state.'
        }
        try {
            $lidRestore = [ordered]@{ scheme = $lidScheme; ac = $previousAcLidAction; dc = $previousDcLidAction }
            $lidRestore | ConvertTo-Json | Set-Content -LiteralPath $lidStateFile -Encoding UTF8
            Set-LidActionIndex $lidScheme 0 0
            Write-Host 'Headless lid mode active: closing the lid does nothing on AC or battery.' -ForegroundColor Green
        } catch {
            try { Set-LidActionIndex $lidScheme $previousAcLidAction $previousDcLidAction } catch {}
            if (Test-Path -LiteralPath $lidStateFile) { Remove-Item -LiteralPath $lidStateFile -Force }
            if (-not $gateway.HasExited) { Stop-Process -Id $gateway.Id -Force -ErrorAction SilentlyContinue }
            if (-not $process.HasExited) { Stop-Process -Id $process.Id -Force -ErrorAction SilentlyContinue }
            if (-not $NoFirewall) { Remove-NetFirewallRule -DisplayName $firewallRule -ErrorAction SilentlyContinue }
            if ($config.performancePowerPlan -and $previousPowerScheme) { powercfg /setactive $previousPowerScheme | Out-Null }
            throw
        }
    }

    try { $process.PriorityClass = [System.Diagnostics.ProcessPriorityClass]::$($config.processPriority) } catch {
        Write-Warning "Could not set Ollama process priority: $($_.Exception.Message)"
    }
    $awake = Start-KeepAwakeWorker

    [ordered]@{
        pid = $process.Id
        gatewayPid = $gateway.Id
        keepAwakePid = $awake.Id
        port = $listenPort
        backendPort = $backendPort
        previousPowerScheme = $previousPowerScheme
        performancePowerPlan = [bool]$config.performancePowerPlan
        contextLength = [int]$config.contextLength
        flashAttention = [bool]$config.flashAttention
        kvCacheType = [string]$config.kvCacheType
        preloadedModel = [string]$config.preloadModel
        preloadSeconds = $preloadSeconds
        headlessLidManaged = [bool]$HeadlessSession
        lidRestore = $lidRestore
        startedAt = (Get-Date).ToString('o')
        firewallManaged = (-not $NoFirewall)
    } | ConvertTo-Json | Set-Content -LiteralPath $serverStateFile -Encoding UTF8

    if ($Focus -or $Exclusive) { Enable-FocusMode $config $true $Exclusive.IsPresent }

    Write-Host "Myference started (Gateway PID $($gateway.Id), Ollama PID $($process.Id))." -ForegroundColor Green
    if ($Exclusive) { Write-Host 'Exclusive mode is active. Press Q in the dashboard to restore the desktop.' -ForegroundColor Yellow }
    Write-Endpoints $listenPort
    Test-LanAvailability $config

    if (-not $NoDashboard) {
        try {
            & $dashboardExe -view -metrics "http://127.0.0.1:$listenPort/_myference/metrics"
        } finally {
            Stop-MyferenceServer
        }
    }
}
function Stop-MyferenceServer {
    $state = Get-ServerState
    try {
        if ($state -and $state.gatewayPid) {
            $gateway = Get-Process -Id $state.gatewayPid -ErrorAction SilentlyContinue
            if ($gateway -and $gateway.ProcessName -like 'myference-dashboard*') {
                Stop-Process -Id $state.gatewayPid -Force
                Write-Host "Stopped managed gateway process $($state.gatewayPid)."
            }
        }
        if ($state -and $state.pid) {
            $process = Get-Process -Id $state.pid -ErrorAction SilentlyContinue
            if ($process -and $process.ProcessName -like 'ollama*') {
                Stop-Process -Id $state.pid -Force
                Write-Host "Stopped managed Ollama process $($state.pid)."
            }
        }
        if ($state -and $state.keepAwakePid) {
            Stop-Process -Id $state.keepAwakePid -Force -ErrorAction SilentlyContinue
            Write-Host 'Released the temporary keep-awake request.'
        }
        if (Get-NetFirewallRule -DisplayName $firewallRule -ErrorAction SilentlyContinue) {
            if (Test-Administrator) {
                Remove-NetFirewallRule -DisplayName $firewallRule
                Write-Host 'Removed the Myference firewall rule.'
            } else {
                Write-Warning 'Run stop as Administrator to remove the Myference firewall rule.'
            }
        }
    } finally {
        $lidRecovery = $null
        if (Test-Path -LiteralPath $lidStateFile) {
            try { $lidRecovery = Get-Content -LiteralPath $lidStateFile -Raw | ConvertFrom-Json } catch {}
        } elseif ($state -and $state.headlessLidManaged -and $state.lidRestore) {
            $lidRecovery = $state.lidRestore
        }
        if ($lidRecovery) {
            try {
                Set-LidActionIndex ([string]$lidRecovery.scheme) ([int]$lidRecovery.ac) ([int]$lidRecovery.dc)
                if (Test-Path -LiteralPath $lidStateFile) { Remove-Item -LiteralPath $lidStateFile -Force }
                Write-Host 'Restored the previous AC and battery lid-close actions.'
            } catch {
                Write-Warning "Could not restore the previous lid-close actions; recovery state was preserved: $($_.Exception.Message)"
            }
        }
        Restore-FocusMode
        if ($state -and $state.performancePowerPlan -and $state.previousPowerScheme) {
            powercfg /setactive $state.previousPowerScheme | Out-Null
            Write-Host 'Restored the previous power plan.'
        }
        if (Test-Path -LiteralPath $serverStateFile) { Remove-Item -LiteralPath $serverStateFile }
    }
    Write-Host 'Normal Windows workspace restored.' -ForegroundColor Green
}

function Show-Dashboard($config) {
    $state = Get-ServerState
    $listenPort = if ($state -and $state.port) { [int]$state.port } else { [int]$config.port }
    if (-not (Test-Port $listenPort)) { throw 'Myference is not running.' }
    & $dashboardExe -view -metrics "http://127.0.0.1:$listenPort/_myference/metrics"
}
function Get-ActivePowerScheme {
    $output = powercfg /getactivescheme
    if ($output -match '([0-9a-fA-F-]{36})') { return $Matches[1] }
    return $null
}

function Get-LidActionIndex([string]$Scheme, [ValidateSet('AC', 'DC')][string]$PowerSource) {
    $output = powercfg /qh $Scheme SUB_BUTTONS LIDACTION | Out-String
    $label = if ($PowerSource -eq 'AC') { 'Current AC Power Setting Index' } else { 'Current DC Power Setting Index' }
    if ($output -match ([regex]::Escape($label) + ':\s+0x([0-9a-fA-F]+)')) {
        return [Convert]::ToInt32($Matches[1], 16)
    }
    return $null
}

function Set-LidActionIndex([string]$Scheme, [int]$AcIndex, [int]$DcIndex) {
    powercfg /setacvalueindex $Scheme SUB_BUTTONS LIDACTION $AcIndex | Out-Null
    if ($LASTEXITCODE -ne 0) { throw "Could not set the AC lid action on power scheme $Scheme." }
    powercfg /setdcvalueindex $Scheme SUB_BUTTONS LIDACTION $DcIndex | Out-Null
    if ($LASTEXITCODE -ne 0) { throw "Could not set the battery lid action on power scheme $Scheme." }
    powercfg /setactive $Scheme | Out-Null
    if ($LASTEXITCODE -ne 0) { throw "Could not activate power scheme $Scheme after updating lid actions." }
}

function Enable-FocusMode($config, [bool]$DoApply, [bool]$CloseShell = $false) {
    if ($DoApply -and (Test-Path -LiteralPath $focusStateFile)) {
        Write-Host 'Focus mode is already active; preserving the original restore point.' -ForegroundColor Yellow
        return
    }

    $processNames = @($config.stopProcesses)
    if ($CloseShell) { $processNames += 'explorer' }
    $found = @()
    foreach ($name in $processNames) {
        $found += @(Get-Process -Name $name -ErrorAction SilentlyContinue)
    }
    $found = @($found | Sort-Object Id -Unique)

    $runningServices = @()
    foreach ($name in @($config.stopServices)) {
        $service = Get-Service -Name $name -ErrorAction SilentlyContinue
        if ($service -and $service.Status -eq 'Running') { $runningServices += $service }
    }

    if (-not $DoApply) {
        Write-Host 'Focus preview (no changes made):' -ForegroundColor Cyan
        if ($found.Count -eq 0 -and $runningServices.Count -eq 0) {
            Write-Host '  No configured optional applications or services are running.'
        }
        foreach ($process in $found) { Write-Host "  Would close $($process.ProcessName) (PID $($process.Id))" }
        foreach ($service in $runningServices) { Write-Host "  Would stop service $($service.Name)" }
        if ($CloseShell) { Write-Host '  Would close the Windows desktop shell (Explorer).' }
        Write-Host '  Would switch to the High Performance power plan.'
        Write-Host 'Run: .\myference.ps1 focus -Apply'
        return
    }

    Ensure-StateDirectory
    $apps = @()
    foreach ($process in $found) {
        $path = $null
        try { $path = $process.Path } catch {}
        $apps += [ordered]@{ name = $process.ProcessName; path = $path }
    }

    # Persist the complete restore point before changing the user session.
    $previousScheme = Get-ActivePowerScheme
    [ordered]@{
        previousPowerScheme = $previousScheme
        apps = $apps
        services = @($runningServices | Select-Object -ExpandProperty Name)
        targetProcesses = @($processNames | Select-Object -Unique)
        targetServices = @($config.stopServices)
        explorerWasClosed = $CloseShell
        appliedAt = (Get-Date).ToString('o')
    } | ConvertTo-Json -Depth 4 | Set-Content -LiteralPath $focusStateFile -Encoding UTF8

    foreach ($process in $found) {
        try {
            [void]$process.CloseMainWindow()
            if (-not $process.WaitForExit(1500)) {
                Stop-Process -Id $process.Id -Force -ErrorAction SilentlyContinue
                $process.WaitForExit(1500)
            }
        } catch {
            Stop-Process -Id $process.Id -Force -ErrorAction SilentlyContinue
        }
        if (Get-Process -Id $process.Id -ErrorAction SilentlyContinue) {
            Write-Warning "Could not close $($process.ProcessName) (PID $($process.Id))."
        } else {
            Write-Host "Closed $($process.ProcessName) (PID $($process.Id))."
        }
    }

    foreach ($service in $runningServices) {
        Require-Administrator "Stopping service $($service.Name)"
        Stop-Service -Name $service.Name -Force
        Write-Host "Stopped service $($service.Name)."
    }

    powercfg /setactive SCHEME_MIN | Out-Null
    Write-Host 'Server focus mode applied. The stop command will restore it.' -ForegroundColor Green
}

function Restore-FocusMode {
    if (-not (Test-Path -LiteralPath $focusStateFile)) { return }

    $state = Get-Content -LiteralPath $focusStateFile -Raw | ConvertFrom-Json
    if ($state.previousPowerScheme) {
        powercfg /setactive $state.previousPowerScheme | Out-Null
        Write-Host 'Restored the previous power plan.'
    }

    foreach ($serviceName in @($state.services)) {
        try {
            Start-Service -Name $serviceName
            Write-Host "Restarted service $serviceName."
        } catch {
            Write-Warning "Could not restart service $($serviceName): $($_.Exception.Message)"
        }
    }

    if ($state.explorerWasClosed -and -not (Get-Process -Name explorer -ErrorAction SilentlyContinue)) {
        Start-Process explorer.exe
        Write-Host 'Restored the Windows desktop shell.'
    }

    foreach ($app in @($state.apps)) {
        if ($app.name -eq 'explorer') { continue }
        if ($app.path -and (Test-Path -LiteralPath $app.path) -and -not (Get-Process -Name $app.name -ErrorAction SilentlyContinue)) {
            try {
                Start-Process -FilePath $app.path
                Write-Host "Reopened $($app.name)."
            } catch {
                Write-Warning "Could not reopen $($app.name): $($_.Exception.Message)"
            }
        }
    }
    Remove-Item -LiteralPath $focusStateFile
}
function Show-Doctor($config) {
    Write-Host "Administrator: $(if (Test-Administrator) { 'yes' } else { 'no (needed for managed firewall setup)' })"
    try { Write-Host "Ollama:       $(Get-OllamaCommand)" } catch { Write-Host "Ollama:       NOT FOUND" -ForegroundColor Red }
    Write-Host "Port:         $($config.port)"
    Write-Host "LAN access:   $($config.allowedRemoteAddress) on Private networks"
    $profiles = Get-NetConnectionProfile -ErrorAction SilentlyContinue
    foreach ($profile in $profiles) {
        Write-Host "Network:      $($profile.Name) [$($profile.NetworkCategory)]"
        if ($profile.NetworkCategory -ne 'Private') { Write-Warning "'$($profile.Name)' is not Private; the firewall rule will not accept traffic on it." }
    }
    $responding = Test-Port ([int]$config.port)
    if ($responding -and -not (Test-LanListener ([int]$config.port))) {
        Write-Warning 'The current Ollama process is bound to localhost only. Quit it before using Myference start.'
    }
    Write-Endpoints ([int]$config.port)
    try { & (Get-OllamaCommand) list } catch {}
}

function Show-Status($config) {
    $state = Get-ServerState
    $responding = Test-Port ([int]$config.port)
    $lanListening = $responding -and (Test-LanListener ([int]$config.port))
    Write-Host "Gateway responding: $responding"
    Write-Host "LAN listener:       $lanListening"
    if ($state) {
        Write-Host "Gateway process:    PID $($state.gatewayPid)"
        Write-Host "Ollama backend:     PID $($state.pid), loopback port $($state.backendPort)"
        Write-Host "Started:            $($state.startedAt)"
        Write-Host 'Live dashboard:     .\myference.ps1 dashboard'
    }
    Write-Host "Focus mode:         $(if (Test-Path -LiteralPath $focusStateFile) { 'active' } else { 'inactive' })"
    if ($lanListening) { Write-Endpoints ([int]$config.port) }
}
function Test-LanAvailability($config) {
    $listenPort = [int]$config.port
    if (-not (Test-Port $listenPort)) {
        throw "Ollama is not responding. Start it with: .\myference.ps1 start"
    }
    if (-not (Test-LanListener $listenPort)) {
        throw "Ollama responds locally but is not bound to the LAN. Listener must be 0.0.0.0:$listenPort."
    }

    $privateProfiles = @(Get-NetConnectionProfile -ErrorAction SilentlyContinue |
        Where-Object { $_.NetworkCategory -eq 'Private' -and $_.IPv4Connectivity -ne 'Disconnected' })
    if ($privateProfiles.Count -eq 0) {
        throw 'No connected Private network profile exists. Mark the trusted home Wi-Fi as Private.'
    }

    $rule = Get-NetFirewallRule -DisplayName $firewallRule -ErrorAction SilentlyContinue |
        Where-Object { $_.Enabled -eq 'True' -and $_.Direction -eq 'Inbound' -and $_.Action -eq 'Allow' }
    if (-not $rule) {
        Write-Warning 'The Myference firewall rule is absent. Remote access may be blocked unless another inbound rule allows this port.'
    }

    $verified = @()
    foreach ($address in Get-LanAddresses) {
        try {
            $version = Invoke-RestMethod -Uri "http://$($address):$listenPort/api/version" -TimeoutSec 5
            $verified += [ordered]@{ address = $address; version = $version.version }
        } catch {
            Write-Warning "LAN self-test failed for $($address): $($_.Exception.Message)"
        }
    }
    if ($verified.Count -eq 0) {
        throw 'Ollama is listening, but no LAN-address API self-test succeeded.'
    }

    Write-Host 'LAN availability verified:' -ForegroundColor Green
    foreach ($item in $verified) {
        Write-Host "  http://$($item.address):$listenPort (Ollama $($item.version))"
        Write-Host "  Test from another device: curl http://$($item.address):$listenPort/api/version"
    }
    if ($rule) {
        Write-Host '  Firewall: enabled for the managed Private-network rule.'
    }
}
function Test-Myference($config) {
    $base = "http://127.0.0.1:$($config.port)"
    $tags = Invoke-RestMethod -Uri "$base/api/tags" -TimeoutSec 5
    Write-Host "API is healthy; $(@($tags.models).Count) model(s) installed." -ForegroundColor Green
    if ($Model) {
        $body = @{ model = $Model; prompt = 'Reply with only: myference works'; stream = $false } | ConvertTo-Json
        $reply = Invoke-RestMethod -Method Post -Uri "$base/api/generate" -ContentType 'application/json' -Body $body -TimeoutSec 300
        Write-Host "Model response: $($reply.response.Trim())"
    } else {
        Write-Host 'Pass -Model <installed-name> to test generation.'
    }
}

function Get-HeadlessShellCommand {
    $powerShellExe = Join-Path $env:SystemRoot 'System32\WindowsPowerShell\v1.0\powershell.exe'
    $shellScript = Join-Path $PSScriptRoot 'server-shell.ps1'
    $quote = [char]34
    return "$quote$powerShellExe$quote -NoProfile -ExecutionPolicy Bypass -File $quote$shellScript$quote"
}

function Restore-HeadlessMode {
    if (-not (Test-Path -LiteralPath $headlessStateFile)) {
        Write-Host 'No recorded headless-shell change exists.'
        return
    }

    $state = Get-Content -LiteralPath $headlessStateFile -Raw | ConvertFrom-Json
    $current = (Get-ItemProperty -Path $headlessPolicyPath -Name Shell -ErrorAction SilentlyContinue).Shell
    if ($current -eq $state.installedShell) {
        if ($state.hadOriginalShell) {
            Set-ItemProperty -Path $headlessPolicyPath -Name Shell -Value $state.originalShell -Type String
        } else {
            Remove-ItemProperty -Path $headlessPolicyPath -Name Shell -ErrorAction SilentlyContinue
        }
        Write-Host 'Restored the original Windows login shell policy.' -ForegroundColor Green
    } else {
        Write-Warning 'The shell policy changed after Myference installed it; leaving the newer value untouched.'
    }

    if (Test-Administrator) {
        Unregister-ScheduledTask -TaskName $headlessStartTask -Confirm:$false -ErrorAction SilentlyContinue
        Unregister-ScheduledTask -TaskName $headlessStopTask -Confirm:$false -ErrorAction SilentlyContinue
    } else {
        Write-Warning 'Run headless-restore as Administrator later to remove the two scheduled tasks.'
    }
    Remove-Item -LiteralPath $headlessStateFile -Force
}

function Install-HeadlessMode($config) {
    Require-Administrator 'Installing headless login mode'
    if (Test-Path -LiteralPath $serverStateFile) {
        throw 'Stop the current Myference server before installing headless mode.'
    }
    if (Test-Path -LiteralPath $headlessStateFile) {
        throw 'Headless mode is already installed. Run headless-status or headless-restore.'
    }

    Ensure-StateDirectory
    $existing = (Get-ItemProperty -Path $headlessPolicyPath -Name Shell -ErrorAction SilentlyContinue).Shell
    $hadExisting = $null -ne $existing
    $shellCommand = Get-HeadlessShellCommand
    [ordered]@{
        hadOriginalShell = $hadExisting
        originalShell = $existing
        installedShell = $shellCommand
        installedAt = (Get-Date).ToString('o')
    } | ConvertTo-Json | Set-Content -LiteralPath $headlessStateFile -Encoding UTF8

    try {
        New-Item -Path $headlessPolicyPath -Force | Out-Null
        Set-ItemProperty -Path $headlessPolicyPath -Name Shell -Value $shellCommand -Type String

        Get-NetFirewallRule -DisplayName $firewallRule -ErrorAction SilentlyContinue | Remove-NetFirewallRule
        New-NetFirewallRule -DisplayName $firewallRule -Direction Inbound -Action Allow -Protocol TCP -LocalPort ([int]$config.port) -Profile Private -RemoteAddress $config.allowedRemoteAddress | Out-Null

        $identity = [Security.Principal.WindowsIdentity]::GetCurrent().Name
        $principal = New-ScheduledTaskPrincipal -UserId $identity -LogonType Interactive -RunLevel Highest
        $settings = New-ScheduledTaskSettingsSet -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries -ExecutionTimeLimit ([TimeSpan]::Zero)
        $quote = [char]34

        $worker = Join-Path $PSScriptRoot 'server-task.ps1'
        $startArguments = '-NoProfile -ExecutionPolicy Bypass -File ' + $quote + $worker + $quote
        $startAction = New-ScheduledTaskAction -Execute 'powershell.exe' -Argument $startArguments
        Register-ScheduledTask -TaskName $headlessStartTask -Action $startAction -Principal $principal -Settings $settings -Force | Out-Null

        $cli = Join-Path $PSScriptRoot 'myference.ps1'
        $stopArguments = '-NoProfile -ExecutionPolicy Bypass -File ' + $quote + $cli + $quote + ' headless-stop'
        $stopAction = New-ScheduledTaskAction -Execute 'powershell.exe' -Argument $stopArguments
        Register-ScheduledTask -TaskName $headlessStopTask -Action $stopAction -Principal $principal -Settings $settings -Force | Out-Null
    } catch {
        try { Restore-HeadlessMode } catch {}
        throw
    }

    Write-Host 'Headless login mode is prepared; the current desktop has not been changed.' -ForegroundColor Green
    Write-Host 'Save your work, sign out manually, then sign in again.'
    Write-Host 'At the Myference console, press Q to stop, restore Explorer, and sign out.'
    Write-Warning 'Emergency recovery: Ctrl+Shift+Esc, Run new task, then run myference.ps1 headless-restore as Administrator.'
}

function Enter-HeadlessMode($config) {
    Require-Administrator 'Entering headless mode'

    if (Test-Path -LiteralPath $headlessStateFile) {
        $saved = Get-Content -LiteralPath $headlessStateFile -Raw | ConvertFrom-Json
        $current = (Get-ItemProperty -Path $headlessPolicyPath -Name Shell -ErrorAction SilentlyContinue).Shell
        $alreadyNormal = ((-not $saved.hadOriginalShell -and $null -eq $current) -or
            ($saved.hadOriginalShell -and $current -eq $saved.originalShell))
        if ($alreadyNormal) {
            Write-Host 'Cleaning state left by the previous completed headless session.'
            Restore-HeadlessMode
        }
    }

    Install-HeadlessMode $config

    Write-Host ''
    Write-Host 'Headless mode is ready.' -ForegroundColor Green
    Write-Host 'Press Enter to sign out immediately, or Esc to cancel and roll back.'
    Write-Host 'Automatic sign-out in 10 seconds.'

    $signOutNow = $false
    for ($remaining = 10; $remaining -ge 1; $remaining--) {
        Write-Host ([char]13 + "Signing out in $remaining second(s)...  ") -NoNewline
        $until = (Get-Date).AddSeconds(1)
        while ((Get-Date) -lt $until) {
            try {
                if ([Console]::KeyAvailable) {
                    $key = [Console]::ReadKey($true)
                    if ($key.Key -eq [ConsoleKey]::Enter) {
                        $signOutNow = $true
                        break
                    }
                    if ($key.Key -eq [ConsoleKey]::Escape) {
                        Write-Host ''
                        Write-Host 'Cancelled. Rolling back headless login configuration...'
                        Restore-HeadlessMode
                        Write-Host 'Headless mode was not entered.' -ForegroundColor Yellow
                        return
                    }
                }
            } catch {}
            Start-Sleep -Milliseconds 100
        }
        if ($signOutNow) { break }
    }

    Write-Host ''
    Write-Host 'Signing out now. Sign back in to start the headless inference server.' -ForegroundColor Cyan
    shutdown.exe /l
}
function Stop-HeadlessMode {
    try { Stop-MyferenceServer } finally { Restore-HeadlessMode }
}

function Show-HeadlessStatus {
    $installed = Test-Path -LiteralPath $headlessStateFile
    $current = (Get-ItemProperty -Path $headlessPolicyPath -Name Shell -ErrorAction SilentlyContinue).Shell
    Write-Host "Headless mode installed: $installed"
    Write-Host "Current custom shell:     $(if ($current) { $current } else { '<not configured>' })"
    foreach ($taskName in @($headlessStartTask, $headlessStopTask)) {
        $task = Get-ScheduledTask -TaskName $taskName -ErrorAction SilentlyContinue
        Write-Host "Task '$taskName': $(if ($task) { $task.State } else { 'not installed' })"
    }
}
function Show-Help {
    Write-Host @'
Myference - share a Windows Ollama server over your private LAN

Usage:
  .\myference.ps1 doctor
  .\myference.ps1 status
  .\myference.ps1 start [-Port 11434] [-Focus | -Exclusive] [-NoFirewall]
  .\myference.ps1 dashboard
  .\myference.ps1 stop
  .\myference.ps1 focus [-Apply]
  .\myference.ps1 restore
  .\myference.ps1 models
  .\myference.ps1 test [-Model model-name]
  .\myference.ps1 lan-check
  .\myference.ps1 headless
  .\myference.ps1 headless-install
  .\myference.ps1 headless-status
  .\myference.ps1 headless-restore
'@
}

$config = Get-Config
switch ($Command) {
    'doctor'  { Show-Doctor $config }
    'status'  { Show-Status $config }
    'start'   { Start-MyferenceServer $config }
    'stop'    { Stop-MyferenceServer }
    'dashboard' { Show-Dashboard $config }
    'focus'   { Enable-FocusMode $config $Apply.IsPresent }
    'restore' { Restore-FocusMode }
    'test'    { Test-Myference $config }
    'models'           { & (Get-OllamaCommand) list }
    'lan-check'        { Test-LanAvailability $config }
    'headless'         { Enter-HeadlessMode $config }
    'headless-install' { Install-HeadlessMode $config }
    'headless-stop'    { Stop-HeadlessMode }
    'headless-restore' {
        if ((Test-Path -LiteralPath $serverStateFile) -or (Test-Path -LiteralPath $lidStateFile)) { Stop-MyferenceServer }
        Restore-HeadlessMode
    }
    'headless-status'  { Show-HeadlessStatus }
    default            { Show-Help }
}
