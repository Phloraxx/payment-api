# OMP Supervised Agent Routine

Use this workflow for meaningful PayGate implementation tasks on the Oracle VM.

## Default model roles

- Supervisor/coordinator: `openai-codex/gpt-5.6-luna:xhigh`
- Parallel scouts: `openai-codex/gpt-5.6-luna:medium`
- Deep planning / unusually hard reasoning: `openai-codex/gpt-5.6-luna:max`
- Keep `task.maxConcurrency` capped at 4 unless a task clearly needs less.

## Mandatory workflow

1. Inspect the current real system first: live records, logs, device output, or exact user-provided evidence.
2. Fetch the current remote branch and create a fresh isolated git worktree from the exact intended baseline.
3. Run a baseline test gate before mutation.
4. For non-trivial bugs, spawn 2-3 independent medium scouts in parallel with distinct roles: trace, regression design, adversarial review.
5. Stop reconnaissance once the diagnosis is settled. Do not let agents keep wandering through unrelated code.
6. Give one xhigh implementation agent the exact evidence, agreed constraints, and narrow scope.
7. Require exact real-world regression fixtures whenever possible, not simplified synthetic approximations.
8. If a real fixture disproves the scout hypothesis, treat the failure as new evidence and revise the root cause before proceeding.
9. Run agent-owned tests, but do not trust the agent's self-reported success as the final gate.
10. The ChatGPT supervisor reruns focused tests, race tests, the relevant package suite, vet/static checks, formatting, and `git diff --check` independently.
11. Give the final diff to a fresh reviewer. Feed back only concrete blocking issues or high-value regression cases.
12. Kill cancelled/stale OMP child processes explicitly; cancelling a wrapper may leave a child process alive.
13. Inspect the final diff and git status. No unrelated files, secrets, generated junk, or prompt files may remain.
14. Only after all gates are green: commit, push, open PR, wait for CI, merge, then deploy using the repository's normal production path.
15. After deployment, verify the real runtime and, if historical rows are already wrong, perform a narrowly targeted backup-first data repair rather than relying on the new parser to rewrite history.

## Safety boundaries

- Never work directly in the production checkout when an isolated worktree can be used.
- Never expose `.env`, OAuth tokens, API secrets, private keys, or credential-store contents to an agent.
- Do not use broad autonomous `yolo` access on production hosts; confine writable agents to isolated worktrees.
- Agents must not push, merge, deploy, alter live databases, or send external messages unless the supervisor explicitly performs/authorizes that phase.
- Preserve payment verification and matching invariants; a parser convenience must not weaken incoming-payment evidence checks.
- Prefer a general parser fix over app/package-specific hacks when the evidence shows a shared notification shape.

## Evidence standard

A task is not considered complete because an agent says it is complete. Completion requires reproducible supervisor-owned verification against the exact final diff and, where possible, the real input that originally failed.
