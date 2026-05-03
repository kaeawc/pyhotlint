# pyhotlint — a Krit-shaped linter for Python inference servers and ML hot paths

## What you're building

Python is the lingua franca of ML serving, but the failure modes that hurt inference servers — GIL-held work inside async, blocking I/O on the event loop, tensor device mismatches, tokenizer/model version skew, CUDA stream misuse, accidental copies between host and device — are exactly the things ruff and mypy don't catch. pyhotlint fills that gap with cheap, structural rules and an opt-in oracle that consults torch/transformers metadata when a rule needs it.

Architecture mirrors Krit (Go-first, tree-sitter Python, single-pass AST, capability-gated rules, autofix tiers, multi-frontend). Read `~/kaeawc/krit/CLAUDE.md` and study `internal/typeinfer/` and `internal/oracle/` — those map directly to pyhotlint's source-level type inference and PyType/torch oracle.

## Rule taxonomy

**Async correctness**
- `sync-io-in-async-fn` — `requests.get`, `time.sleep`, file open without aio inside `async def`.
- `cpu-bound-loop-in-event-loop` — tight `for` over large iterable inside async.
- `lock-held-across-await` — `threading.Lock`/`asyncio.Lock` held across `await` without comment.

**Tensor / device hygiene**
- `device-mismatch-binop` — `tensor_a + tensor_b` where one is on CPU and the other on CUDA (requires oracle).
- `host-device-copy-in-loop` — `.cpu()` / `.cuda()` inside a hot loop.
- `untracked-grad-in-eval` — `model.eval()` without `torch.no_grad()` or `torch.inference_mode()` on the surrounding block.

**Versioning / drift**
- `tokenizer-model-id-mismatch` — `AutoTokenizer.from_pretrained("X")` paired with a model from `"Y"` family.
- `transformers-pinned-but-config-newer` — config field requires a transformers version above what's pinned.

**Server hygiene**
- `unbounded-batching` — request handler accumulates without flush.
- `metric-not-labeled` — Prometheus counter without service label, breaks dashboards.
- `pickle-load-from-untrusted-path` — security.

## Architecture

- **Go**, tree-sitter Python.
- **Source typeinfer** — same idea as Krit's `internal/typeinfer/`: cheap AST-level type tracking that handles most local code without an oracle.
- **PyOracle** — opt-in subprocess that attaches to a project's venv to answer "what's the device of `model.embed.weight`?" or "does this class subclass `nn.Module`?". Gated by `NeedsPyOracle` capability. The oracle just needs a Python interpreter path; it's agnostic to how the venv was produced.
- **Environment detection** — first-class support for **uv** (read `uv.lock`, prefer `uv sync` to materialize a missing venv), with fallbacks for poetry, pdm, and bare `requirements.txt` + `.venv/`. Mirrors how Krit reads Gradle output rather than running its own build: pyhotlint reads whichever lockfile is authoritative.
- **Project model** — `pyproject.toml` (PEP 621 + `[tool.uv]`/`[tool.poetry]`), lockfile, and `python-version` to know torch/transformers/python versions. Used by version-drift rules.
- **Outputs**: SARIF, JSON, LSP, PR comment, MCP server.
- **Autofix tiers** as in Krit.

## MVP

1. Repo skeleton + tree-sitter Python ingestion.
2. Five rules: `sync-io-in-async-fn`, `lock-held-across-await`, `untracked-grad-in-eval`, `tokenizer-model-id-mismatch`, `pickle-load-from-untrusted-path`.
3. Project model (pyproject + requirements).
4. PyOracle stub (returns "unknown" for everything) + the architecture to plug a real one in.
5. CI corpus: vLLM, TGI, a couple of HF inference example repos. Track precision/recall manually.

## Stretch

- **Real PyOracle** — actually load the project venv in a subprocess, resolve symbols, expose a JSON-RPC interface (mirrors Krit's KAA daemon).
- **Hot-path heatmap** — combine static rules with optional pyspy/profile traces to weight findings by actual hotness.
- **Type-stub authoring** — autogenerate stubs from observed call patterns, fed back into rules.
- **CUDA-graph awareness** — flag patterns that break captured graphs.

## Why this is the right shape

Inference servers are where small Python mistakes turn into seven-figure GPU bills. Most teams have no static tooling for this layer; they discover problems via load tests or production incidents. Krit's "cheap by default, oracle on demand" pattern is the right architecture because the expensive checks must stay opt-in to keep CI fast.

## Non-goals

- Replacing mypy or pyright.
- Performance profiling (different problem).
- Model-quality opinions.
