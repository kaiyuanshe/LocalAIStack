. (Join-Path $PSScriptRoot "..\..\scripts_common.ps1")

& (Join-Path $PSScriptRoot "install.ps1")

$openclaw = Get-Command openclaw -ErrorAction SilentlyContinue
if (-not $openclaw) {
  $verifyScript = Join-Path $PSScriptRoot "verify.ps1"
  & $verifyScript | Out-Null
  $openclaw = Get-Command openclaw -ErrorAction SilentlyContinue
}

if (-not $openclaw) {
  throw "OpenClaw CLI is not available in PATH after update."
}

try {
  & $openclaw.Source onboard --install-daemon
} catch {
}

& (Join-Path $PSScriptRoot "configure_local.ps1")
Write-Output "OpenClaw updated successfully."
