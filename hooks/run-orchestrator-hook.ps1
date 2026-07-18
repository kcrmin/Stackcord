param(
    [Parameter(Mandatory = $true)]
    [ValidateSet("session-start", "post-compact")]
    [string]$Event
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$Cli = $null
if ($env:ORCHESTRATOR_CLI -and (Test-Path -LiteralPath $env:ORCHESTRATOR_CLI -PathType Leaf)) {
    $Cli = $env:ORCHESTRATOR_CLI
}
if (-not $Cli -and $env:PLUGIN_ROOT) {
    foreach ($Relative in @("cli\orchestrator.exe", "bin\orchestrator.exe")) {
        $Candidate = Join-Path $env:PLUGIN_ROOT $Relative
        if (Test-Path -LiteralPath $Candidate -PathType Leaf) {
            $Cli = $Candidate
            break
        }
    }
}
if (-not $Cli) {
    $Command = Get-Command orchestrator.exe -ErrorAction SilentlyContinue
    if ($Command) {
        $Cli = $Command.Source
    }
}
if (-not $Cli) {
    exit 0
}

& $Cli hook $Event
exit $LASTEXITCODE
