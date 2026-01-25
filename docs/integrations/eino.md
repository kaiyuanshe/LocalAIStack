# Eino Integration Plan for LocalAIStack

## 1. Goals and Scope

The goal is to **add Eino-powered LLM calling capability inside LocalAIStack** as a built-in integration, not as a separately installed module with its own InstallSpec or module directory.
Eino should be treated as an internal provider/adapter that LocalAIStack can use to invoke LLMs, similar to how other providers are wired into the runtime and API layers.

The initial integration focuses on:

- embedding Eino as a Go dependency in the LocalAIStack codebase
- adding an internal provider/adapter that maps LocalAIStack’s LLM interface to Eino
- exposing the provider through existing configuration and API/CLI surfaces
- keeping deployment simple (no new `modules/eino/` directory or install scripts)

## 2. Where the Integration Lives

**No `modules/eino/` directory should be created.**  
Instead, Eino should live as a **first-party integration within the Go codebase**, alongside other LLM providers or runtime adapters.

Recommended locations (adjust to actual project structure):

- `internal/` or `pkg/` provider packages to implement an **Eino-backed LLM provider**
- configuration wiring in `configs/` to select `provider = "eino"`
- API/CLI paths that list available providers

## 3. Dependency Management

Add the Eino dependency to the root Go module:

- `go.mod` includes `github.com/cloudwego/eino` (pinned version)
- optional minimal wrapper interfaces to isolate Eino-specific types

This keeps the dependency managed by standard Go tooling, with no separate install step.

## 4. Provider/Adapter Design

Implement an internal provider that adapts LocalAIStack’s LLM abstraction to Eino:

- map LocalAIStack’s request/response models to Eino APIs
- support standard features: model selection, streaming, timeouts, tool calling (if supported)
- expose provider name `eino` in config and discovery endpoints

If LocalAIStack already has a provider registry, Eino should be registered there with a clear capability list.

## 5. Runtime Integration (No Module Lifecycle)

Eino is a library, not a standalone service.
Therefore:

- **no module install/uninstall**
- **no runtime lifecycle hooks**
- **no scripts or manifests**

Any future Eino tooling that requires a service (UI, tracing, DevOps, etc.) should be treated as a separate optional component, not part of the core Eino provider.

## 6. CLI / API Exposure

Expose Eino through existing mechanisms for providers:

- `localaistack provider list` or equivalent should include `eino`
- `localaistack config set provider=eino` should enable it
- API provider listing should surface `eino` as a built-in option

## 7. Suggested Next Steps (Implementation Order)

1. Add Eino dependency to `go.mod`.
2. Create an internal provider adapter (e.g., `internal/provider/eino`).
3. Wire provider registration and configuration.
4. Add basic integration tests to confirm the adapter can route a request through Eino.
