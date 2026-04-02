. (Join-Path $PSScriptRoot "..\..\scripts_common.ps1")

$candidates = @(
  (Join-Path (Get-LocalBinDir) "openclaw.cmd"),
  (Join-Path (Get-LocalBinDir) "openclaw.exe"),
  (Join-Path (Get-LocalBinDir) "openclaw"),
  (Join-Path (Get-HomeDir) "AppData\Roaming\npm\openclaw.cmd"),
  (Join-Path (Get-HomeDir) "AppData\Roaming\npm\openclaw"),
  (Join-Path (Get-HomeDir) ".config\openclaw")
)

$pipScripts = Get-PipUserSiteScriptsPath
if ($pipScripts) {
  $candidates += @(
    (Join-Path $pipScripts "openclaw.exe"),
    (Join-Path $pipScripts "openclaw.cmd")
  )
}

foreach ($candidate in $candidates) {
  if (Test-Path -LiteralPath $candidate) {
    Remove-Item -LiteralPath $candidate -Recurse -Force
  }
}

Write-Output "OpenClaw uninstall cleanup completed."
