param(
    [Parameter(Mandatory = $true)]
    [ValidateSet("session-start", "post-compact")]
    [string]$Event
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$Root = $env:PLUGIN_ROOT
if (-not $Root) { $Root = $env:CLAUDE_PLUGIN_ROOT }
$Cli = $null
if ($env:STACKCORD_CLI -and (Test-Path -LiteralPath $env:STACKCORD_CLI -PathType Leaf)) {
    $Cli = $env:STACKCORD_CLI
}
if (-not $Cli -and $Root) {
    foreach ($Relative in @("cli\stackcord.exe", "bin\stackcord.exe")) {
        $Candidate = Join-Path $Root $Relative
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
