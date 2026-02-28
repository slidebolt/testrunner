# TASK

## Scope
Harden runtime behavior and test hygiene for this repository.

## Constraints
- No git commits or tags from subprocesses unless explicitly requested.
- Keep changes minimal, testable, and production-safe.
- Prefer deterministic shutdown/startup behavior.

## Required Output
- Small PR-sized patch.
- Repro steps.
- Validation commands and expected results.
- Known risks/limits.

## Priority Tasks
1. Ensure tests never publish to production-like MQTT discovery topics.
2. Add retained-message cleanup in tests that publish MQTT discovery data.
3. Harden harness traps so launcher PID from each run is always cleaned up.
4. Keep plugin-optional test skip behavior deterministic.

## Done Criteria
- No test publishes to `homeassistant/#` by default.
- Repeated local/integration runs do not leave background launcher processes.

## Validation Checklist
- [ ] Build succeeds for this repo.
- [ ] Local targeted tests (if present) pass.
- [ ] No new background orphan processes remain.
- [ ] Logs clearly show failure causes.
