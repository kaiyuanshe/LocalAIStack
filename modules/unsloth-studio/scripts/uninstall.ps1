. (Join-Path $PSScriptRoot "..\..\scripts_common.ps1")

$studioHome = if ($env:UNSLOTH_STUDIO_HOME) { $env:UNSLOTH_STUDIO_HOME } else { Join-Path (Get-HomeDir) ".unsloth\studio" }
$appDir = if ($env:LOCALAPPDATA) { Join-Path $env:LOCALAPPDATA "Unsloth Studio" } else { $null }
$desktopDir = [Environment]::GetFolderPath("Desktop")
$startMenuDir = if ($env:APPDATA) { Join-Path $env:APPDATA "Microsoft\Windows\Start Menu\Programs" } else { $null }

Remove-IfExists -Path $studioHome
if ($appDir) {
  Remove-IfExists -Path $appDir
}
if ($desktopDir) {
  Remove-IfExists -Path (Join-Path $desktopDir "Unsloth Studio.lnk")
}
if ($startMenuDir) {
  Remove-IfExists -Path (Join-Path $startMenuDir "Unsloth Studio.lnk")
}
