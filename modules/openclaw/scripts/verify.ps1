. (Join-Path $PSScriptRoot "..\..\scripts_common.ps1")

function Find-OpenClawCommand {
  $command = Get-Command openclaw -ErrorAction SilentlyContinue
  if ($command) {
    return $command.Source
  }

  $candidates = @(
    (Join-Path (Get-LocalBinDir) "openclaw.cmd"),
    (Join-Path (Get-LocalBinDir) "openclaw.exe"),
    (Join-Path (Get-HomeDir) "AppData\Roaming\npm\openclaw.cmd")
  )

  $pipScripts = Get-PipUserSiteScriptsPath
  if ($pipScripts) {
    $candidates += @(
      (Join-Path $pipScripts "openclaw.exe"),
      (Join-Path $pipScripts "openclaw.cmd")
    )
  }

  foreach ($candidate in $candidates) {
    if ($candidate -and (Test-Path -LiteralPath $candidate)) {
      return $candidate
    }
  }

  return $null
}

$openclaw = Find-OpenClawCommand
if (-not $openclaw) {
  throw "OpenClaw CLI is not available in PATH or common Windows install locations."
}

& $openclaw --help | Out-Null
Write-Output "OpenClaw verification succeeded: $openclaw"
