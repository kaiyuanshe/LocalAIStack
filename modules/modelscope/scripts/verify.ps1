. (Join-Path $PSScriptRoot "..\..\scripts_common.ps1")

Invoke-PythonCode -Code @"
import importlib.util
spec = importlib.util.find_spec("modelscope")
if spec is None:
    raise SystemExit("modelscope is not installed")
import modelscope
print(modelscope.__version__)
"@

Invoke-PythonCode -Code @"
import importlib.metadata as md
eps = md.entry_points()
group = eps.select(group="console_scripts") if hasattr(eps, "select") else eps.get("console_scripts", [])
names = {ep.name for ep in group}
if "modelscope" not in names:
    raise SystemExit(f"Expected modelscope console script, found: {sorted(names)}")
"@
