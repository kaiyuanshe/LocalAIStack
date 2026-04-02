. (Join-Path $PSScriptRoot "..\..\scripts_common.ps1")

Invoke-PythonCode -Code @"
import importlib.metadata as md
import importlib.util
spec = importlib.util.find_spec("unsloth")
if spec is None:
    raise SystemExit("unsloth is not installed")
print(md.version("unsloth"))
"@
