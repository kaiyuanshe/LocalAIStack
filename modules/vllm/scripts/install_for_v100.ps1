. (Join-Path $PSScriptRoot "..\..\scripts_common.ps1")
Write-UnsupportedWindowsScript -ModuleName "vllm" -ScriptName "install_for_v100" -Reason "The vLLM V100 installer is Linux-only."
