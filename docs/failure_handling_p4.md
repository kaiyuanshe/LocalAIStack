# Failure Handling (P4) Runbook

## 1. Scope

P4 adds operational failure handling for install/config/run workflows:

1. structured failure event recording (`~/.localaistack/failures/*.jsonl`)
2. failure classification and remediation advice
3. CLI inspection commands for recent failures

---

## 2. Recorded Flows

Failure events are recorded for:

1. `las module install ...`
2. `las module config-plan ...`
3. `las model run ...`

Each event includes phase, module/model context, provider/model hints, error text, and classification.

---

## 3. Debug Signal

Set:

```bash
export LOCALAISTACK_FAILURE_DEBUG=1
```

When enabled, failed commands print:

1. failure phase
2. classified category
3. retryable hint
4. failure log path
5. remediation suggestion

---

## 4. CLI Commands

List recent failures:

```bash
./build/las failure list --limit 20
```

Filter by phase/category:

```bash
./build/las failure list --phase smart_run --category timeout --limit 10
```

JSON output:

```bash
./build/las failure list --output json --limit 5
```

Inspect one event with advice:

```bash
./build/las failure show <event-id>
```

---

## 5. Validation Checklist

1. trigger one failing install/run command
2. run `./build/las failure list --limit 5` and confirm event appears
3. run `./build/las failure show <event-id>` and confirm `advice` is present
4. enable debug env var and rerun failure to verify inline diagnostics
