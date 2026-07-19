param(
    [Parameter(Mandatory = $true)]
    [ValidateSet("session-start", "post-compact")]
    [string]$Event
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$Cli = $null
if ($env:STACKCORD_CLI -and (Test-Path -LiteralPath $env:STACKCORD_CLI -PathType Leaf)) {
    $Cli = $env:STACKCORD_CLI
}
if (-not $Cli -and $env:PLUGIN_ROOT) {
    foreach ($Relative in @("cli\stackcord.exe", "bin\stackcord.exe")) {
        $Candidate = Join-Path $env:PLUGIN_ROOT $Relative
        if (Test-Path -LiteralPath $Candidate -PathType Leaf) {
            $Cli = $Candidate
            break
        }
    }
}
if (-not $Cli) {
    $Command = Get-Command stackcord.exe -ErrorAction SilentlyContinue
    if ($Command) {
        $Cli = $Command.Source
    }
}
if (-not $Cli) {
    exit 0
}

& $Cli hook $Event
exit $LASTEXITCODE
