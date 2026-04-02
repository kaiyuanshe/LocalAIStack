. (Join-Path $PSScriptRoot "..\..\scripts_common.ps1")
Write-UnsupportedWindowsScript -ModuleName "llama.cpp" -ScriptName "cleanup_soft" -Reason "llama.cpp installation and cleanup are still Linux-only in this repository."
