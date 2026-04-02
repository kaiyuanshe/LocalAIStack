. (Join-Path $PSScriptRoot "..\..\scripts_common.ps1")
Write-UnsupportedWindowsScript -ModuleName "llama.cpp" -ScriptName "install_source" -Reason "Source build automation is still Linux-only."
