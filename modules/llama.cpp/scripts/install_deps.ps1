. (Join-Path $PSScriptRoot "..\..\scripts_common.ps1")
Write-UnsupportedWindowsScript -ModuleName "llama.cpp" -ScriptName "install_deps" -Reason "Dependency bootstrap is still Linux-only."
