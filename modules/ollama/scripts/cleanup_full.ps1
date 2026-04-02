. (Join-Path $PSScriptRoot "..\..\scripts_common.ps1")

& (Join-Path $PSScriptRoot "purge.ps1")
Write-Output "Full cleanup completed (destructive)."
