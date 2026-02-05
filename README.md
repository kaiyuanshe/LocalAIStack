# LocalAIStack

最简使用步骤：下载代码 → 编译 → 使用 `./build/las`。

## 1) 下载代码

```bash
git clone <repo-url> LocalAIStack
cd LocalAIStack
```

## 2) 编译

```bash
make build
```

编译产物：`./build/las`（CLI）与 `./build/las-server`（服务端）。

## 3) 使用 `./build/las`

```bash
./build/las --help
```

常用示例：

```bash
./build/las version
./build/las help
./build/las <command> --help
```
