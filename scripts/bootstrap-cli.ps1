[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)][string]$BaseUrl,
    [Parameter(Mandatory = $true)][string]$Version,
    [string]$InstallDir = (Join-Path $HOME ".local\bin"),
    [string]$OS = "windows",
    [string]$Arch = ""
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

if ($Version -notmatch '^[0-9]+\.[0-9]+\.[0-9]+([.-][0-9A-Za-z.-]+)?$') {
    throw "Invalid version: $Version"
}
if ($BaseUrl -notmatch '^https://' -and $BaseUrl -notmatch '^http://(127\.0\.0\.1|localhost):[0-9]+/') {
    throw "BaseUrl must use HTTPS (localhost HTTP is allowed for tests)."
}
if ($OS -ne "windows") {
    throw "bootstrap-cli.ps1 supports Windows only."
}
if (-not $Arch) {
    $runtimeArch = [System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture.ToString().ToLowerInvariant()
    $Arch = switch ($runtimeArch) {
        "x64" { "amd64" }
        "arm64" { "arm64" }
        default { throw "Unsupported architecture: $runtimeArch" }
    }
}
$Asset = switch ($Arch) {
    "amd64" { "orchestrator_windows_amd64.exe" }
    "arm64" { "orchestrator_windows_arm64.exe" }
    default { throw "Unsupported architecture: $Arch" }
}

$ReleaseUrl = "$($BaseUrl.TrimEnd('/'))/v$Version"
$TempDir = Join-Path ([System.IO.Path]::GetTempPath()) ("orchestrator-bootstrap-" + [guid]::NewGuid().ToString("N"))
$Checksums = Join-Path $TempDir "checksums.txt"
$Download = Join-Path $TempDir $Asset
$Staged = $null

try {
    New-Item -ItemType Directory -Path $TempDir | Out-Null
    Invoke-WebRequest -Uri "$ReleaseUrl/checksums.txt" -OutFile $Checksums
    Invoke-WebRequest -Uri "$ReleaseUrl/$Asset" -OutFile $Download

    $Expected = @(
        Get-Content -LiteralPath $Checksums | ForEach-Object {
            if ($_ -match '^([0-9A-Fa-f]{64})\s+\*?(.+)$' -and $Matches[2] -eq $Asset) {
                $Matches[1].ToLowerInvariant()
            }
        }
    )
    if ($Expected.Count -ne 1) {
        throw "Checksum manifest must contain exactly one SHA-256 for $Asset."
    }
    $Actual = (Get-FileHash -LiteralPath $Download -Algorithm SHA256).Hash.ToLowerInvariant()
    if ($Actual -ne $Expected[0]) {
        throw "SHA-256 mismatch for $Asset."
    }

    & $Download doctor --json | Out-Null
    New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null
    $Target = Join-Path $InstallDir "orchestrator.exe"
    $Staged = Join-Path $InstallDir (".orchestrator.tmp-" + [guid]::NewGuid().ToString("N") + ".exe")
    Copy-Item -LiteralPath $Download -Destination $Staged
    if (Test-Path -LiteralPath $Target) {
        $Backup = Join-Path $InstallDir (".orchestrator.backup-" + [guid]::NewGuid().ToString("N") + ".exe")
        [System.IO.File]::Replace($Staged, $Target, $Backup, $true)
        Remove-Item -LiteralPath $Backup -Force
    } else {
        [System.IO.File]::Move($Staged, $Target)
    }
    $Staged = $null
    & $Target doctor --json
    Write-Output "Installed verified orchestrator $Version at $Target"
} finally {
    if ($Staged -and (Test-Path -LiteralPath $Staged)) {
        Remove-Item -LiteralPath $Staged -Force
    }
    if (Test-Path -LiteralPath $TempDir) {
        Remove-Item -LiteralPath $TempDir -Recurse -Force
    }
}
