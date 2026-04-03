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

$wrapperPath = Join-Path (Get-LocalBinDir) "obeaver.cmd"
Remove-IfExists -Path (Get-EBeaverHome)
Remove-IfExists -Path $wrapperPath

Write-Output "Removed oBeaver virtual environment, launcher, and cloned repository."
