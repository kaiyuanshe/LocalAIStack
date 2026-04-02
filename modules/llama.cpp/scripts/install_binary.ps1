. (Join-Path $PSScriptRoot "..\..\scripts_common.ps1")
Write-UnsupportedWindowsScript -ModuleName "llama.cpp" -ScriptName "install_binary" -Reason "A native Windows installer has not been implemented yet."
