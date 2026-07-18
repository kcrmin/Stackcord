param(
    [string]$Binary,
    [string]$Output,
    [string]$Workspace
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"
$Root = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path
$Temporary = Join-Path ([System.IO.Path]::GetTempPath()) ("orchestrator-dogfood-" + [Guid]::NewGuid().ToString("N"))

if (-not $Binary -or -not $Output -or -not $Workspace) {
    New-Item -ItemType Directory -Path $Temporary -Force | Out-Null
}
if (-not $Binary) {
    $Binary = Join-Path $Temporary "orchestrator.exe"
    Push-Location (Join-Path $Root "cli")
    try {
        & go build -trimpath -o $Binary ./cmd/orchestrator
        if ($LASTEXITCODE -ne 0) { throw "Go build failed" }
    } finally {
        Pop-Location
    }
}
if (-not $Output) { $Output = Join-Path $Temporary "result.json" }
if (-not $Workspace) { $Workspace = Join-Path $Temporary "fixture" }

$Python = Get-Command python3 -ErrorAction SilentlyContinue
if (-not $Python) { $Python = Get-Command python -ErrorAction Stop }
& $Python.Source (Join-Path $PSScriptRoot "run.py") --binary $Binary --output $Output --workspace $Workspace
exit $LASTEXITCODE
