. (Join-Path $PSScriptRoot "..\..\scripts_common.ps1")

& (Join-Path $PSScriptRoot "uninstall.ps1")

$dataDir = Join-Path (Get-HomeDir) ".ollama"
Remove-IfExists -Path $dataDir

Write-Output "Ollama fully removed (including data)."
