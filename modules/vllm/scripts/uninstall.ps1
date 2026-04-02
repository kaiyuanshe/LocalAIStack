. (Join-Path $PSScriptRoot "..\..\scripts_common.ps1")
Write-UnsupportedWindowsScript -ModuleName "vllm" -ScriptName "uninstall" -Reason "The vLLM uninstall flow in this repository is Linux-only."
