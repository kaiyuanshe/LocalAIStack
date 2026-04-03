# obeaver

`obeaver` is a LocalAIStack module that installs the upstream
[microsoft/obeaver](https://github.com/microsoft/obeaver) project from source
using:

```bash
git clone https://github.com/microsoft/obeaver.git
pip install -e .
```

Default managed paths:

- Linux/macOS: `~/.localaistack/tools/obeaver`
- Windows: `%USERPROFILE%\\.localaistack\\tools\\obeaver`

Runtime setup notes:

- Windows: the installer auto-checks `Foundry Local` and runs `winget install Microsoft.FoundryLocal` when needed
- macOS: the installer auto-checks `Foundry Local` and runs `brew install microsoft/foundrylocal/foundrylocal` when needed
- Linux: `Foundry Local` is not supported, so use `obeaver run --engine ort <model_dir>`

Optional environment variables:

- `OBEAVER_HOME`: override the managed install directory
- `OBEAVER_REPO_URL`: override the git clone URL
- `OBEAVER_REPO_REF`: override the git ref used for clone
- `PYTHON_BIN` (POSIX only): override the Python executable used to create the virtual environment
