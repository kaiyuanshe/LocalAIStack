. (Join-Path $PSScriptRoot "..\..\scripts_common.ps1")

& (Join-Path $PSScriptRoot "uninstall.ps1")
Write-Output "Soft cleanup completed (data preserved)."
