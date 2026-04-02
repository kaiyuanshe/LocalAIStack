. (Join-Path $PSScriptRoot "..\..\scripts_common.ps1")
Write-UnsupportedWindowsScript -ModuleName "vllm" -ScriptName "purge" -Reason "The vLLM purge flow in this repository is Linux-only."
