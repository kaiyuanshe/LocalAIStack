# Ollama 模块安装技术方案（面向 `./build/las module install ollama`）

## 目标

当用户执行 `./build/las module install ollama` 时：

- LocalAIStack 能够解析模块元数据、依赖关系与 InstallSpec。
- 自动完成 Ollama 的安装、配置、验证与状态记录。
- 提供可回滚、可卸载、可清理的完整生命周期。

该方案以 InstallSpec v0.1.2 为唯一执行来源，并遵循模块化目录结构与脚本约束。`modules/ollama/manifest.yaml` 与 `modules/ollama/INSTALL.yaml` 已提供，需要保证 CLI 与执行器能够正确读取与执行。方案细节如下。

---

## 现状与缺口

- CLI 的 `module install` 仅打印文本，未接入模块解析与安装引擎。
- `internal/module` 已实现模块注册表与依赖解析，但 `LoadRegistryFromDir` 会读取所有 `.yaml` 文件（包含 `INSTALL.yaml`），这会导致 YAML 结构解析错误，需要限制只读取 `manifest.yaml`。

---

## 方案概览

### 1. 模块规范化（对齐 InstallSpec 与现有文件）

#### 1.1 `modules/ollama/manifest.yaml`

当前清单文件已定义模块元数据、硬件需求与依赖，重点校验字段包括：

```yaml
name: ollama
category: runtime
version: 0.1.0
description: Local LLM inference runtime powered by Ollama
license: MIT
hardware:
  cpu:
    cores_min: 2
  memory:
    ram_min: 4GB
  gpu:
    vram_min: 6GB
    multi_gpu: false

dependencies:
  system:
    - curl
    - ca-certificates

runtime:
  modes: [native]
  preferred: native

interfaces:
  provides:
    - local_llm_inference
```

- `dependencies.system` 要与 InstallSpec 的依赖模型一致。
- `runtime.modes` 需满足校验器对 `runtime.modes` 的要求。

#### 1.2 `modules/ollama/INSTALL.yaml`

InstallSpec 文件已定义安装步骤、环境重建与安全约束，关键结构如下：

```yaml
apiVersion: las.installspec/v0.1.2
kind: InstallPlan
id: ollama
category: runtime
supported_platforms:
  - linux/amd64
  - linux/arm64
install_modes:
  - native
rebuild_modes:
  - none
  - soft
  - full
tools_required:
  - bash
  - curl
  - systemctl
preconditions:
  - id: P10
    intent: Must be Linux
    tool: shell
    command: uname -s
    expected:
      equals: Linux
install:
  native:
    - id: S10
      intent: Download and install Ollama
      tool: shell
      command: curl -fsSL https://ollama.com/install.sh | bash
      expected:
        bin: /usr/local/bin/ollama
      idempotent: true
verification:
  script: scripts/verify.sh
rollback:
  script: scripts/rollback.sh
uninstall:
  script: scripts/uninstall.sh
purge:
  script: scripts/purge.sh
```

补充说明：

- 安装脚本可后续替换为受控的 tarball 下载与校验（生产级建议）。
- 由于现有模板固定 `OLLAMA_HOST=127.0.0.1:11434`，`security.network.bind` 必须明确为 `localhost`。

---

### 2. 安装引擎实现（InstallSpec 执行器）

#### 2.1 Registry 加载只读取 `manifest.yaml`

`LoadRegistryFromDir` 应限制为仅加载文件名为 `manifest.yaml` 的 YAML 文件，避免误解析 `INSTALL.yaml` 或其他 YAML 模板。

建议修改逻辑：

- `if filepath.Base(path) != "manifest.yaml" { return nil }`
- 仍支持多版本目录结构（如 `modules/ollama/v0.1.0/manifest.yaml`）

#### 2.2 新增 InstallSpec 解析与执行器

需要新增 `internal/module/installer` 或 `internal/install` 包：

**关键能力：**

1. 解析 `INSTALL.yaml` 并进行结构校验。
2. 解析 install mode（当前为 native）。
3. 执行 `preconditions` 并在失败时终止。
4. 执行 `environment_rebuild`，处理 soft/full cleanup。
5. 按顺序执行 `install` steps：
   - `tool: shell` -> `command` 执行
   - `tool: template` -> 渲染模板写入 destination
6. 在失败时调用 `rollback`。
7. 执行 `verification` 作为最终成功判定。
8. 执行 `state` 更新（安装完成后 state=installed）。

#### 2.3 命令行入口对接

`cmd/las module install` 应：

1. 加载模块注册表（默认 `./modules`）。
2. 解析模块依赖，生成安装顺序（已有 resolver）。
3. 依序执行 InstallSpec。
4. 更新 `internal/control/state` 中模块状态。
5. 提供 `--dry-run` 选项输出 InstallSpec 执行计划。

---

### 3. 运行时与服务管理

#### 3.1 systemd 服务

`templates/ollama.service.tmpl` 已提供基本 unit，需要确认：

- `User` 默认值是否需要创建用户（如 `ollama`）
- 如果需要创建用户，应在 install steps 增加 useradd 或 fallback 为当前用户
- `OLLAMA_HOST` 绑定本地地址，符合安全要求

#### 3.2 端口与可用性验证

`scripts/verify.sh` 已检查 `http://127.0.0.1:11434/api/tags` 与 CLI 可用性，应保留。可以追加重试逻辑或服务启动等待，以提高稳定性。

---

### 4. 安全与可维护性

1. 所有网络下载应在 InstallSpec 中显式声明，且建议未来加入 checksum 或 GPG 验证。
2. InstallSpec 中应记录需要 sudo 权限。
3. 所有数据目录必须在 uninstall 中保留，在 purge 中彻底删除。
4. `cleanup_soft.sh` 与 `cleanup_full.sh` 已存在，应在 `environment_rebuild` 中显式调用。

---

## 实施顺序（建议）

1. 复核 `modules/ollama/manifest.yaml` 与 `modules/ollama/INSTALL.yaml` 与 InstallSpec v0.1.2 对齐。
2. 限制 registry 仅加载 `manifest.yaml`。
3. 实现 InstallSpec 解析与执行器。
4. 将 `cmd/las module install` 接入执行器。
5. 加入 `--dry-run`，输出预期执行步骤。
6. 执行 `./build/las module install ollama` 并验证 `scripts/verify.sh`。

---

## 预期结果

执行 `./build/las module install ollama` 后：

- Ollama 二进制存在于 `/usr/local/bin/ollama`
- systemd 服务处于 enabled+active 状态
- `curl http://127.0.0.1:11434/api/tags` 返回模型列表
- 模块状态更新为 `installed`

完成上述步骤即可满足 InstallSpec v0.1.2 规范，保证 Ollama 模块在 LocalAIStack 中可被正确安装、验证与管理。
