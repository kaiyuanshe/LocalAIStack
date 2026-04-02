. (Join-Path $PSScriptRoot "..\..\scripts_common.ps1")

& (Join-Path $PSScriptRoot "uninstall.ps1")

$dataDir = Join-Path (Get-LocalShareDir) "openclaw"
Remove-IfExists -Path $dataDir
Write-Output "OpenClaw purged."
