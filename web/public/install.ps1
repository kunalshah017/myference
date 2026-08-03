$ErrorActionPreference = 'Stop'
Set-StrictMode -Version Latest

$repository = 'kunalshah017/myference'
$machine = if ($env:MYFERENCE_ARCH) { $env:MYFERENCE_ARCH } elseif ($env:PROCESSOR_ARCHITEW6432) { $env:PROCESSOR_ARCHITEW6432 } else { $env:PROCESSOR_ARCHITECTURE }
if ($machine -notin @('AMD64', 'amd64', 'x86_64')) { throw "Unsupported Windows architecture: $machine" }

$tag = $env:MYFERENCE_RELEASE_TAG
if ([string]::IsNullOrWhiteSpace($tag)) {
    $release = Invoke-RestMethod -Uri "https://api.github.com/repos/$repository/releases/latest" -Headers @{ 'User-Agent' = 'myference-installer' }
    $tag = [string]$release.tag_name
}
if ($tag -notmatch '^[A-Za-z0-9][A-Za-z0-9._-]*$') { throw "Invalid release tag: $tag" }
$asset = "myference-windows-amd64-$tag.zip"
$base = if ($env:MYFERENCE_DOWNLOAD_BASE) { $env:MYFERENCE_DOWNLOAD_BASE.TrimEnd('/') } else { "https://github.com/$repository/releases/download" }
$temporary = Join-Path ([IO.Path]::GetTempPath()) ("myference-install-" + [guid]::NewGuid().ToString('N'))
$archive = Join-Path $temporary $asset
$checksums = Join-Path $temporary 'SHA256SUMS'
$package = Join-Path $temporary 'package'
$destinationTemps = @()

function Receive-ReleaseFile([string]$name, [string]$destination) {
    if ([IO.Path]::IsPathRooted($base)) {
        Copy-Item -LiteralPath (Join-Path (Join-Path $base $tag) $name) -Destination $destination
    } else {
        Invoke-WebRequest -UseBasicParsing -Uri "$base/$tag/$name" -OutFile $destination
    }
}

try {
    New-Item -ItemType Directory -Path $temporary, $package -Force | Out-Null
    Receive-ReleaseFile $asset $archive
    Receive-ReleaseFile 'SHA256SUMS' $checksums

    $escapedAsset = [regex]::Escape($asset)
    $checksumLine = Get-Content -LiteralPath $checksums | Where-Object { $_ -match "^([0-9a-fA-F]{64})\s+\*?$escapedAsset$" } | Select-Object -First 1
    if (-not $checksumLine) { throw "No checksum published for $asset" }
    $expected = ([regex]::Match($checksumLine, '^([0-9a-fA-F]{64})')).Groups[1].Value.ToLowerInvariant()
    $actual = (Get-FileHash -Algorithm SHA256 -LiteralPath $archive).Hash.ToLowerInvariant()
    if ($actual -ne $expected) { throw "Checksum verification failed for $asset" }

    Expand-Archive -LiteralPath $archive -DestinationPath $package -Force
    foreach ($file in @('myference.exe', 'myference-agent-proxy.exe', 'install-windows.ps1')) {
        if (-not (Test-Path -LiteralPath (Join-Path $package $file))) { throw "Release is missing $file" }
    }

    $installDir = if ($env:MYFERENCE_INSTALL_DIR) { $env:MYFERENCE_INSTALL_DIR } else { Join-Path $env:LOCALAPPDATA 'Programs\Myference' }
    New-Item -ItemType Directory -Path $installDir -Force | Out-Null
    $files = @('myference-agent-proxy.exe', 'install-windows.ps1', 'myference.exe')
    $transaction = [guid]::NewGuid().ToString('N')
    $staged = @{}
    $backups = @{}
    $installed = @()
    foreach ($file in $files) {
        $staged[$file] = Join-Path $installDir ".$file.new-$transaction"
        $destinationTemps += $staged[$file]
        Copy-Item -LiteralPath (Join-Path $package $file) -Destination $staged[$file]
    }
    try {
        foreach ($file in $files) {
            $destination = Join-Path $installDir $file
            if (Test-Path -LiteralPath $destination) {
                $backups[$file] = Join-Path $installDir ".$file.backup-$transaction"
                Move-Item -LiteralPath $destination -Destination $backups[$file]
            }
        }
        if ($env:MYFERENCE_INSTALL_FAIL_AFTER_BACKUP -eq '1') { throw 'Injected installer replacement failure' }
        foreach ($file in $files) {
            Move-Item -LiteralPath $staged[$file] -Destination (Join-Path $installDir $file)
            $installed += $file
        }
    } catch {
        foreach ($file in $installed) { Remove-Item -LiteralPath (Join-Path $installDir $file) -Force -ErrorAction SilentlyContinue }
        foreach ($file in $backups.Keys) { Move-Item -LiteralPath $backups[$file] -Destination (Join-Path $installDir $file) -Force -ErrorAction SilentlyContinue }
        throw
    }
    foreach ($backup in $backups.Values) { Remove-Item -LiteralPath $backup -Force -ErrorAction SilentlyContinue }

    $userPath = [Environment]::GetEnvironmentVariable('Path', 'User')
    $entries = @($userPath -split ';' | Where-Object { -not [string]::IsNullOrWhiteSpace($_) })
    if ($entries -notcontains $installDir) {
        $newPath = (@($entries) + $installDir) -join ';'
        [Environment]::SetEnvironmentVariable('Path', $newPath, 'User')
    }
    if (($env:Path -split ';') -notcontains $installDir) { $env:Path = "$installDir;$env:Path" }

    Write-Host "Installed Myference $tag for Windows AMD64 in $installDir" -ForegroundColor Green
    Write-Host 'Run: myference host'
} finally {
    foreach ($path in $destinationTemps) { Remove-Item -LiteralPath $path -Force -ErrorAction SilentlyContinue }
    Remove-Item -LiteralPath $temporary -Recurse -Force -ErrorAction SilentlyContinue
}
