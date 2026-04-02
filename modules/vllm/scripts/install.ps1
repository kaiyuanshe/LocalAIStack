. (Join-Path $PSScriptRoot "..\..\scripts_common.ps1")
Write-UnsupportedWindowsScript -ModuleName "vllm" -ScriptName "install" -Reason "The vLLM installer in this repository is Linux-only."
