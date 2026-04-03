. (Join-Path $PSScriptRoot "..\..\scripts_common.ps1")

function Get-EBeaverHome {
  if ($env:OBEAVER_HOME) {
    return $env:OBEAVER_HOME
  }
  $preferred = Join-Path (Get-LocalAIStackDir) "tools\obeaver"
  $legacy = Join-Path (Get-LocalAIStackDir) "tools\ebeaver"
  if (Test-Path -LiteralPath $preferred) {
    return $preferred
  }
  if (Test-Path -LiteralPath $legacy) {
    return $legacy
  }
  return $preferred
}

function Get-EBeaverWrapperPath {
  return Join-Path (Get-LocalBinDir) "obeaver.cmd"
}

function Find-OBeaverCommand {
  $candidates = @(
    (Join-Path (Get-EBeaverHome) ".venv\Scripts\obeaver.exe"),
    (Join-Path (Get-EBeaverHome) ".venv\Scripts\obeaver"),
    (Get-EBeaverWrapperPath)
  )

  foreach ($candidate in $candidates) {
    if ($candidate -and (Test-Path -LiteralPath $candidate)) {
      return $candidate
    }
  }

  $command = Get-Command obeaver -ErrorAction SilentlyContinue
  if ($command) {
    return $command.Source
  }

  return $null
}

$obeaver = Find-OBeaverCommand
if (-not $obeaver) {
  throw "oBeaver CLI is not available in PATH, ~/.local/bin, or the managed virtual environment."
}

& $obeaver --help | Out-Null
if ($LASTEXITCODE -ne 0) {
  throw "Failed to run '$obeaver --help'."
}

& $obeaver version | Out-Null
if ($LASTEXITCODE -ne 0) {
  throw "Failed to run '$obeaver version'."
}

Write-Output "oBeaver verification succeeded: $obeaver"
