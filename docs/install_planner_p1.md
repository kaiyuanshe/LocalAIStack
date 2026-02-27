# Install Planner (P1) Design and Runbook

## 1. Scope

This document describes the P1 install planner behavior used by `./build/las module install <module>`.

P1 focuses on:

1. structured LLM planning input
2. validated plan output
3. safe fallback to static install path
4. operator-visible debug and strict modes

It does not cover config planner, run planner, or failure handling orchestration.

---

## 2. Model Roles

LocalAIStack uses two model roles:

1. translation model: `i18n.translation.model` (default `tencent/Hunyuan-MT-7B`)
2. assistant model: `llm.model` (default `deepseek-ai/DeepSeek-V3.2`)

Install planning uses the assistant model only.

---

## 3. Planner Flow

For `module install`:

1. load and validate `INSTALL.yaml`
2. select initial mode from static decision logic
3. build planner input from module spec:
4. current mode
5. available modes
6. current mode step list + category summary
7. mode catalog for all install modes
8. preconditions summary
9. call LLM provider with strict JSON schema prompt
10. parse and validate response
11. apply selected mode/steps only if valid
12. otherwise fallback to static steps

Service-related steps are always preserved when applicable.

---

## 4. Step Categories

P1 classifies steps to improve model planning quality:

1. `dependency`
2. `download`
3. `binary_install`
4. `source_build`
5. `configure`
6. `service`
7. `verify`

Classification is derived from step id/intent/command/tool and used only as planner context.

---

## 5. Output Contract

Planner expects JSON:

```json
{
  "mode": "native",
  "steps": ["deps", "download", "verify"],
  "reason": "short reason",
  "risk_level": "low",
  "fallback_hint": "optional"
}
```

Compatibility notes:

1. `selected_steps` is accepted as alias of `steps`
2. `risk_level` is normalized to `low|medium|high`
3. unknown mode or unknown step IDs will reject the plan and trigger fallback

---

## 6. Safety and Fallback

If any of the following fails:

1. provider call
2. JSON extraction/parsing
3. mode validation
4. step ID validation

then LocalAIStack falls back to static install mode/steps by default.

This keeps installation availability high while still enabling LLM planning.

---

## 7. Debug and Strict Modes

P1 exposes two environment switches:

1. `LOCALAISTACK_INSTALL_PLANNER_DEBUG=1`
2. `LOCALAISTACK_INSTALL_PLANNER_STRICT=1`

Debug mode prints planner source and final plan:

1. `source=llm` means LLM plan was accepted
2. `source=static` means fallback path was used
3. fallback reason is printed when available

Strict mode converts fallback into hard failure. This is useful for testing whether LLM planning is truly active.

Example:

```bash
export LOCALAISTACK_INSTALL_PLANNER_DEBUG=1
export LOCALAISTACK_INSTALL_PLANNER_STRICT=1
./build/las module install p2-smoke
```

---

## 8. Proving DeepSeek-V3.2 Participation

To prove the assistant model participates in install planning:

1. ensure `llm.model=deepseek-ai/DeepSeek-V3.2`
2. enable debug mode and observe `source=llm`
3. optionally enable strict mode and verify invalid key causes hard failure
4. compare with fallback provider (for example `eino`) to confirm behavior difference
5. verify provider-side request logs by model and timestamp

If output shows `source=static`, DeepSeek did not affect planning for that run.

---

## 9. Known Limits (P1)

1. fallback path can mask planner issues unless debug/strict is enabled
2. planner decisions are currently printed only in debug mode
3. no persistent planner trace store yet

These are planned follow-ups outside P1.
