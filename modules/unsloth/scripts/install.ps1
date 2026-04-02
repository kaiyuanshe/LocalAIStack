. (Join-Path $PSScriptRoot "..\..\scripts_common.ps1")

Invoke-PythonCode -Code @"
import sys
if sys.version_info >= (3, 14):
    raise SystemExit("Unsloth supports Python 3.13 or lower")
"@

Invoke-Python -Arguments @("-m", "pip", "install", "--upgrade", "--user", "unsloth")
