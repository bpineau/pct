# AGENTS.md - onboarding for coding agents

Read this first. `README.md` is the complete user manual and every console
block in it is a literal, current output: if you change behaviour, re-run the
examples and update it in the same change.

## What pct is, and why it is shaped this way

`pct` is a single-binary terminal calculator that speaks percentages. Module
`github.com/bpineau/pct`, Go 1.26, **no dependencies at all** (`go.mod` has no
require block), under 2000 lines with tests.

The shape follows from one idea: a percentage literal has no value of its own,
it borrows one from its context. `20 + 10%` is 22, `150 * 10%` is 15. That
rule needs a real parser, so the code is three flat layers, each usable alone:

- `internal/percent` - the fixed-form arithmetic, pure functions on float64.
- `internal/expr` - a recursive-descent scanner/parser/evaluator for the
  expression language; no I/O, and its errors carry a position.
- `internal/cli` - argument handling, output formatting, the stdin and
  interactive session, and the command table.

`main.go` is wiring only: it calls `cli.Run(args, stdin, stdout, stderr)` and
exits with what that returns. Every stream is injected, so the whole CLI is
testable in-process; the suite sits at 100 % statement coverage today.

## Working rules

- Solo repo (`git@github.com:bpineau/pct.git`), one branch: commit and push to
  **master**. Do not leave finished work uncommitted.
- `make check` must pass before every commit. CI
  (`.github/workflows/ci.yml`) runs the same gate on pushes to master and on
  pull requests, with golangci-lint pinned to v2.12.1 and the Go version read
  from `go.mod`.
- Stdlib only. Adding a dependency is a design decision, not a convenience.
- English everywhere: code, comments, godoc, README, commit messages.
- Write plain hyphens only, never an em-dash or en-dash. (The existing
  user-facing strings - README, help screen, a few doc comments - still carry
  typographic dashes from the original style; leave them alone unless a sweep
  is asked for.)
- Commit messages follow the existing form: `internal/<pkg>: lowercase
  imperative summary`, or a plain sentence for repo-wide changes.

## Build, test, verify

```sh
make help      # the target list, generated from the Makefile itself
make build     # go build -o pct .   (the binary is gitignored)
make test      # go test ./...
make check     # fmt-check + vet + lint + test: the commit gate
make cover     # coverage profile + go tool cover -func
make fuzz      # 30 s of FuzzEvaluate on the parser
```

The cheap loop, in order of cost:

```sh
go test ./internal/expr -run TestEvaluate -count=1   # one test, about a second
go test ./internal/cli -run TestRunStdin -v          # one CLI test
make build && ./pct ev "1000 - 20% - 20%"            # drive the real binary
printf '450 * 3\nans - 10%%\n' | ./pct               # the stdin session
./pct help                                           # the generated help screen
```

`make lint` requires `golangci-lint` on PATH and fails if it is missing. The
config is the v2 format in `.golangci.yml`: standard set plus gocritic,
misspell, revive and unconvert, with errcheck excused only for the
`fmt.Fprint*` family and `(*tabwriter.Writer).Flush`.

## The tree

| Path | Owns |
|---|---|
| `main.go` | wiring only; `exit` is a package var so `main_test.go` can intercept the exit code instead of killing the coverage profile |
| `internal/percent/percent.go` | `Add`, `Of`, `Base`, `Change`, `Share`, `Compound`, plus `ErrZeroReference` and `ErrTotalLoss`. Percentages are numbers, not fractions: `Add(2, 100)` is 102 |
| `internal/expr/scan.go` | the scanner: token kinds, byte positions for `SyntaxError`, ASCII-only names |
| `internal/expr/expr.go` | the grammar (written out in the package doc), the percentage semantics, the built-in functions and constants, `Evaluate` and `EvaluateWith` |
| `internal/cli/cli.go` | `Run`, `splitArgs`, `session`, `isTerminal`, `formatNumber`, `printUsage` |
| `internal/cli/commands.go` | the `commands` table (name, alias, arity, examples, `run`) and the argument parsers. A new subcommand is one entry here |

## Invariants and traps

- **Percent semantics are the product.** Added or subtracted, a percentage
  applies to the preceding operand, and consecutive percentages compound on
  the running result (`expression()` carries `acc` and `base` for exactly
  that); multiplied, divided, exponentiated or parenthesized, it is just a
  fraction of one. Parentheses seal a value, so `100 + (10%)` adds 0.1.
  The package doc of `internal/expr` is the normative statement; keep it, the
  README section and the tests in step.
- **`%` binds to a number literal only** (`primary`). `(10)%` is a syntax
  error, deliberately.
- **The help screen is generated from `commands`** and cannot drift; the
  README is not generated, so update it by hand.
- **`splitArgs` recognizes exact `--` option names only**, which is what lets
  `pct add -2 100` pass a negative percentage through. Do not switch to
  `flag`: it would eat `-2`. Options may sit before or after the command.
- **`parseNumbers(args, roles...)` indexes `roles` by argument position**, so
  a caller passing fewer roles than args panics. That is why `compound` passes
  `args[:2]` and parses its third argument separately.
- **Exit codes are contractual** (README table, `cli_test.go`): 0 success, 1
  calculation failure, 2 usage error. Results go to stdout; errors, the
  greeting and the `> ` prompt all go to stderr.
- **In a session a bad line never stops the run.** Piped, it sets the exit
  code to 1 and the remaining lines still run; interactive, it leaves the code
  alone. `ans` holds the previous result at full precision, never the rounded
  display.
- **`formatNumber` trims trailing zeros and maps `-0` to `0`.** Precision is a
  display concern only; never round inside the math.
- **`Evaluate` must never panic on any input** - that is all `FuzzEvaluate`
  asserts. Run `make fuzz` after touching the scanner or the parser.
- Division by zero is caught in `term()` only; `Base` refuses -100 %, `Change`
  and `Share` refuse a zero reference. Everything else is plain float64, so
  `Inf` and `NaN` can reach the output.
