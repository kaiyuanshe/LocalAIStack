. (Join-Path $PSScriptRoot "..\..\scripts_common.ps1")

function Find-OpenClawCommand {
  $command = Get-Command openclaw -ErrorAction SilentlyContinue
  if ($command) {
    return $command.Source
  }

  $candidates = @(
    (Join-Path (Get-LocalBinDir) "openclaw.cmd"),
    (Join-Path (Get-LocalBinDir) "openclaw.exe"),
    (Join-Path (Get-LocalBinDir) "openclaw"),
    (Join-Path (Get-HomeDir) "AppData\Roaming\npm\openclaw.cmd"),
    (Join-Path (Get-HomeDir) "AppData\Roaming\npm\openclaw")
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

if (Find-OpenClawCommand) {
  Write-Output "OpenClaw already installed: $(Find-OpenClawCommand)"
  exit 0
}

$installUrl = if ($env:OPENCLAW_INSTALL_URL) { $env:OPENCLAW_INSTALL_URL } else { "https://openclaw.ai/install.ps1" }
$script = Invoke-RestMethod -Uri $installUrl

$tempScript = Join-Path ([System.IO.Path]::GetTempPath()) ("openclaw-install-" + [guid]::NewGuid().ToString("N") + ".ps1")
try {
  Set-Content -LiteralPath $tempScript -Value $script -Encoding UTF8
  & powershell.exe -NoProfile -ExecutionPolicy Bypass -File $tempScript
  if ($LASTEXITCODE -ne 0) {
    throw "OpenClaw installer exited with code $LASTEXITCODE."
  }
} finally {
  if (Test-Path -LiteralPath $tempScript) {
    Remove-Item -LiteralPath $tempScript -Force
  }
}

$installed = Find-OpenClawCommand
if (-not $installed) {
  throw "OpenClaw install script finished, but openclaw was not found in PATH or common install locations."
}

Write-Output "OpenClaw installed at: $installed"
