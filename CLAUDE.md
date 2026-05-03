# pyhotlint

Go-first static analyzer for Python inference servers and ML hot paths. Parses
Python source with tree-sitter, runs rules through a v2-style registry and
single-pass AST dispatcher, and emits JSON findings. Architecture mirrors Krit
(`~/kaeawc/krit`).

See `README.md` for the rule taxonomy and motivation.

## Key Rules

- Keep analyzer and rule work in Go.
- After implementation changes, run `go build ./cmd/pyhotlint/ && go vet ./...`.
- Run `go test ./... -count=1` for full validation; use focused package tests while iterating.
- Use tree-sitter AST nodes for structural analysis and regex only for line-oriented checks.
- New rules use the v2 pipeline: implement a local rule struct, expose dispatch
  metadata through `v2.Register`, and declare the capabilities the dispatcher
  must provide.
- Rules that need project context must declare the matching capability:
  `NeedsProject` (pyproject + lockfile), `NeedsTypeInfer` (source-level type
  inference), `NeedsOracle` (PyOracle subprocess).
- Add positive and negative fixtures under `tests/fixtures/`.

## Rule Implementation Guardrails

Mirror Krit's discipline:

- Prefer tree-sitter AST traversal over `strings.Contains` or raw regexes.
- Stop body walks at real scope boundaries: nested `def`/`async def`, `class`,
  `lambda`. Do not flag a `time.sleep` that is inside a sync nested function
  declared inside an `async def`.
- Require receiver/owner proof for common method names. `time.sleep` is
  blocking; `mytimer.sleep` may not be.
- Walk all relevant operands and siblings; do not stop at the first child.
- For each rule, add positive fixtures (must fire), negative fixtures (must
  NOT fire), and a unit test that pins both.

## Project Structure

- `cmd/pyhotlint/` — CLI entry point.
- `internal/scanner/` — tree-sitter Python parsing helpers.
- `internal/rules/v2/` — the rule registry, dispatcher, and shared context.
- `internal/rules/<category>/` — rule implementations grouped by taxonomy
  (`async`, `tensor`, `versioning`, `server`, `security`).
- `internal/typeinfer/` — source-level type inference (stub for MVP).
- `internal/oracle/` — PyOracle subprocess (stub for MVP).
- `internal/project/` — pyproject + lockfile reader.
- `internal/output/` — JSON / SARIF / LSP formatters (JSON only at MVP).
- `tests/fixtures/<rule-id>/{positive,negative}/` — rule fixtures.

## Build & Validate

```bash
go build ./cmd/pyhotlint/   # Build binary
go vet ./...                # Vet
go test ./... -count=1      # All tests
go test ./internal/rules/async/ -v
```

## Adding a Rule

1. Create the rule struct in `internal/rules/<category>/<rule_id>.go`.
2. Implement the rule's `Check` method using `*v2.Context`.
3. Register it with `v2.Register(&v2.Rule{...})` in an `init()`.
4. Declare `NodeTypes`, `Needs`, `Fix`, `Confidence` as needed.
5. Create positive and negative fixtures under
   `tests/fixtures/<rule-id>/{positive,negative}/`.
6. Add a focused test under `internal/rules/<category>/`.
