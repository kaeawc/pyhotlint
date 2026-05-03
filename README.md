# pyhotlint — a Krit-shaped linter for Python inference servers and ML hot paths

Python is the lingua franca of ML serving, but the failure modes that hurt inference servers — GIL-held work inside async, blocking I/O on the event loop, tensor device mismatches, tokenizer/model version skew, CUDA stream misuse, accidental copies between host and device — are exactly the things ruff and mypy don't catch. pyhotlint fills that gap with cheap, structural rules and an opt-in oracle that consults torch/transformers metadata when a rule needs it.

Architecture mirrors [Krit](https://github.com/kaeawc/krit) (Go-first, tree-sitter Python, single-pass AST, capability-gated rules, autofix tiers, multi-frontend).

## Quick start

```bash
go install github.com/kaeawc/pyhotlint/cmd/pyhotlint@latest

pyhotlint path/to/module.py                       # one file
pyhotlint src/                                    # walk a tree (skips .venv, __pycache__, etc.)
pyhotlint 'src/**/handlers/*.py'                  # globs
pyhotlint --format sarif src/ > findings.sarif    # SARIF for CI / GitHub Code Scanning
pyhotlint --oracle src/                           # spawn the PyOracle subprocess for richer rules
pyhotlint --config pyhotlint.yml src/             # explicit config (auto-discovered otherwise)
```

Exit codes: `0` clean, `1` findings or parse error, `2` usage / config error.

## Rules

Eleven rules across four categories. Severities are defaults; override per-rule in `pyhotlint.yml`.

### Async correctness

| Rule | Severity | What it catches |
|------|----------|-----------------|
| `sync-io-in-async-fn` | warning | `time.sleep`, `requests.get`, blocking `open()`, etc. inside `async def` |
| `cpu-bound-loop-in-event-loop` | warning | `for` loop in `async def` with no `await` in its body |
| `lock-held-across-await` | warning | `threading.Lock` / `asyncio.Lock` held while awaiting; inline-comment escape |

### Tensor / device hygiene

| Rule | Severity | What it catches |
|------|----------|-----------------|
| `device-mismatch-binop` | error | `tensor_a + tensor_b` where one is on CPU and the other on CUDA |
| `host-device-copy-in-loop` | warning | `.cpu()` / `.cuda()` / `.to(device)` inside a `for` / `while` loop |
| `untracked-grad-in-eval` | warning | `model.eval()` without `torch.no_grad()` / `torch.inference_mode()` in scope |

### Versioning / drift

| Rule | Severity | What it catches |
|------|----------|-----------------|
| `tokenizer-model-id-mismatch` | error | `AutoTokenizer.from_pretrained("X")` paired with a model from a different family |
| `transformers-pinned-but-config-newer` | error | `from_pretrained` kwarg requires a transformers version newer than the project pins |

### Server hygiene

| Rule | Severity | What it catches |
|------|----------|-----------------|
| `unbounded-batching` | warning | `self.<name>.append(...)` inside `async def` with no flush/clear/measure |
| `metric-not-labeled` | warning | Prometheus `Counter`/`Gauge`/`Histogram`/... created without `labelnames=` |
| `pickle-load-from-untrusted-path` | error | `pickle.load` / `pickle.loads` (deserialization RCE) |

## Configuration

`pyhotlint.yml` (or `.pyhotlint.yml`) at the project root, auto-discovered by walking up from cwd:

```yaml
rules:
  pickle-load-from-untrusted-path:
    enabled: false             # disable this rule for the project
  sync-io-in-async-fn:
    severity: error            # bump from default warning
  metric-not-labeled:
    severity: info
```

Unknown rule IDs surface as stderr warnings rather than silently disabling rules — typos will not quietly weaken the lint set.

## Suppression

Pragmas tolerate whitespace; rule IDs use the same names as the table above.

```python
time.sleep(1)              # pyhotlint: ignore
time.sleep(1)              # pyhotlint: ignore[sync-io-in-async-fn]
do_thing()                 # pyhotlint: ignore[rule-a, rule-b]

# pyhotlint: ignore-file
# pyhotlint: ignore-file[pickle-load-from-untrusted-path]
```

Suppression is applied as a post-dispatch filter — every rule still runs, findings whose `(rule, start-line)` matches a pragma are dropped.

## Architecture

- **Go**, tree-sitter Python, single-pass AST dispatcher.
- **`internal/scanner/`** — parser pool, `ParsedFile` ownership.
- **`internal/rules/v2/`** — `Rule` registry, `Capabilities` bitset (`NeedsProject`, `NeedsTypeInfer`, `NeedsOracle`), `Context` carrying source bytes plus optional project/oracle, dispatcher walking the AST once and routing nodes by type.
- **`internal/rules/<taxonomy>/`** — rule implementations grouped by category (`async`, `tensor`, `versioning`, `server`).
- **`internal/typeinfer/`** — source-level type tracker. Currently exposes `DeviceTracker` for `.cpu()` / `.cuda()` / `.to(<literal>)` transitions; consumed by `device-mismatch-binop`.
- **`internal/project/`** — reads `pyproject.toml` (PEP 621) and `uv.lock`. uv-resolved versions take precedence; Poetry / requirements.txt are tracked-not-yet-supported.
- **`internal/oracle/`** — opt-in `Oracle` interface plus `Subprocess` (Python interpreter speaking newline-delimited JSON), `Stub` (default), and `Fake` (tests). Discovers `<project>/.venv/bin/python`, then `python3` / `python` on PATH.
- **`internal/config/`** — YAML config loader with auto-discovery and validation.
- **`internal/walker/`** — CLI path expansion: files, directories (recursive, with skiplist for venv / cache / build dirs), and globs.
- **`internal/output/`** — JSON and SARIF v2.1.0 formatters.
- **`internal/suppress/`** — pragma parser and per-(rule, line) filter.
- **`tests/fixtures/<rule-id>/{positive,negative}/`** — fixture-driven rule tests.
- **`cmd/pyhotlint/`** — CLI entry point.

## Roadmap

Implemented:

- All 11 rules from the README's original taxonomy.
- Directory recursion + glob expansion.
- YAML config with per-rule enable/severity overrides.
- `pyproject.toml` + `uv.lock` reader.
- JSON and SARIF v2.1.0 output.
- Suppression pragmas (line and file scope).
- PyOracle subprocess infrastructure (newline-delimited JSON).
- Source-level device tracker.

Next:

- Real `device_of` symbol resolution in the oracle helper (today returns Unknown).
- LSP server (`cmd/pyhotlint-lsp`).
- MCP server (`cmd/pyhotlint-mcp`).
- Autofix application (the `FixLevel` enum is in the schema; no fixes are written yet).
- CI corpus tracking — vLLM, TGI, HF inference repos — manual precision/recall.
- Cross-file analysis (multi-file dead-code, import drift).
- Stretch: hot-path heatmap from pyspy/profile traces; CUDA-graph awareness; type-stub authoring.

## Why this is the right shape

Inference servers are where small Python mistakes turn into seven-figure GPU bills. Most teams have no static tooling for this layer; they discover problems via load tests or production incidents. Krit's "cheap by default, oracle on demand" pattern is the right architecture because the expensive checks must stay opt-in to keep CI fast.

## Non-goals

- Replacing mypy or pyright.
- Performance profiling (different problem).
- Model-quality opinions.
