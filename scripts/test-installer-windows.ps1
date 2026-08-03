$ErrorActionPreference = 'Stop'
Set-StrictMode -Version Latest

$root = (Resolve-Path (Join-Path $PSScriptRoot '..')).Path
$installer = Join-Path $root 'web\public\install.ps1'
$temporary = Join-Path ([IO.Path]::GetTempPath()) ("myference-windows-installer-test-" + [guid]::NewGuid().ToString('N'))
$tag = 'v9.8.7-test'
$release = Join-Path $temporary "releases\$tag"
$package = Join-Path $temporary 'package'
$asset = "myference-windows-amd64-$tag.zip"
$archive = Join-Path $release $asset
$oldUserPath = [Environment]::GetEnvironmentVariable('Path', 'User')
$oldArchitecture = $env:PROCESSOR_ARCHITECTURE
$oldNativeArchitecture = $env:PROCESSOR_ARCHITEW6432

try {
    New-Item -ItemType Directory -Path $release, $package -Force | Out-Null
    Set-Content -LiteralPath (Join-Path $package 'myference.exe') -Value 'windows-cli'
    Set-Content -LiteralPath (Join-Path $package 'myference-agent-proxy') -Value 'linux-container-proxy'
    Set-Content -LiteralPath (Join-Path $package 'install-windows.ps1') -Value '# service installer'
    Compress-Archive -Path (Join-Path $package '*') -DestinationPath $archive
    $hash = (Get-FileHash -Algorithm SHA256 -LiteralPath $archive).Hash.ToLowerInvariant()
    Set-Content -LiteralPath (Join-Path $release 'SHA256SUMS') -Value "$hash  $asset"

    $env:MYFERENCE_RELEASE_TAG = $tag
    $env:MYFERENCE_DOWNLOAD_BASE = (Join-Path $temporary 'releases')
    $env:MYFERENCE_INSTALL_DIR = (Join-Path $temporary 'installed')
    Remove-Item Env:MYFERENCE_ARCH -ErrorAction SilentlyContinue
    $env:PROCESSOR_ARCHITECTURE = 'x86'
    $env:PROCESSOR_ARCHITEW6432 = 'AMD64'
    & $installer | Out-Null

    if ((Get-Content -LiteralPath (Join-Path $env:MYFERENCE_INSTALL_DIR 'myference.exe') -Raw).Trim() -ne 'windows-cli') { throw 'CLI content was not installed' }
    if ((Get-Content -LiteralPath (Join-Path $env:MYFERENCE_INSTALL_DIR 'myference-agent-proxy') -Raw).Trim() -ne 'linux-container-proxy') { throw 'Linux container proxy content was not installed' }
    if (-not (Test-Path -LiteralPath (Join-Path $env:MYFERENCE_INSTALL_DIR 'install-windows.ps1'))) { throw 'service installer was not installed' }
    if (([Environment]::GetEnvironmentVariable('Path', 'User') -split ';') -notcontains $env:MYFERENCE_INSTALL_DIR) { throw 'install directory was not persisted to user PATH' }

    Add-Content -LiteralPath $archive -Value 'tampered'
    $env:MYFERENCE_INSTALL_DIR = (Join-Path $temporary 'rejected')
    $rejected = $false
    try { & $installer | Out-Null } catch { $rejected = $_.Exception.Message -match 'Checksum verification failed' }
    if (-not $rejected) { throw 'tampered archive was not rejected by checksum verification' }
    if (Test-Path -LiteralPath (Join-Path $env:MYFERENCE_INSTALL_DIR 'myference.exe')) { throw 'tampered CLI was installed' }
	$replacementHash = (Get-FileHash -Algorithm SHA256 -LiteralPath $archive).Hash.ToLowerInvariant()
	Set-Content -LiteralPath (Join-Path $release 'SHA256SUMS') -Value "$replacementHash  $asset"

    $rollback = Join-Path $temporary 'rollback'
    New-Item -ItemType Directory -Path $rollback -Force | Out-Null
    Set-Content -LiteralPath (Join-Path $rollback 'myference.exe') -Value 'old-cli'
    Set-Content -LiteralPath (Join-Path $rollback 'myference-agent-proxy') -Value 'old-proxy'
    Set-Content -LiteralPath (Join-Path $rollback 'install-windows.ps1') -Value 'old-installer'
    $env:MYFERENCE_INSTALL_DIR = $rollback
    $env:MYFERENCE_INSTALL_FAIL_AFTER_BACKUP = '1'
    $failed = $false
    try { & $installer | Out-Null } catch { $failed = $true }
    Remove-Item Env:MYFERENCE_INSTALL_FAIL_AFTER_BACKUP -ErrorAction SilentlyContinue
    if (-not $failed) { throw 'injected replacement failure was not surfaced' }
    if ((Get-Content -LiteralPath (Join-Path $rollback 'myference.exe') -Raw).Trim() -ne 'old-cli' -or (Get-Content -LiteralPath (Join-Path $rollback 'myference-agent-proxy') -Raw).Trim() -ne 'old-proxy' -or (Get-Content -LiteralPath (Join-Path $rollback 'install-windows.ps1') -Raw).Trim() -ne 'old-installer') { throw 'failed update did not preserve the previous installation' }
} finally {
    [Environment]::SetEnvironmentVariable('Path', $oldUserPath, 'User')
    $env:PROCESSOR_ARCHITECTURE = $oldArchitecture
    if ($null -eq $oldNativeArchitecture) { Remove-Item Env:PROCESSOR_ARCHITEW6432 -ErrorAction SilentlyContinue } else { $env:PROCESSOR_ARCHITEW6432 = $oldNativeArchitecture }
    Remove-Item Env:MYFERENCE_RELEASE_TAG, Env:MYFERENCE_DOWNLOAD_BASE, Env:MYFERENCE_INSTALL_DIR, Env:MYFERENCE_ARCH, Env:MYFERENCE_INSTALL_FAIL_AFTER_BACKUP -ErrorAction SilentlyContinue
    Remove-Item -LiteralPath $temporary -Recurse -Force -ErrorAction SilentlyContinue
}

Write-Host 'Windows installer tests passed'
