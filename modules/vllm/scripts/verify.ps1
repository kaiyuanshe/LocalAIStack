. (Join-Path $PSScriptRoot "..\..\scripts_common.ps1")
Write-UnsupportedWindowsScript -ModuleName "vllm" -ScriptName "verify" -Reason "The vLLM verify flow in this repository is Linux-only."
