# pct

A terminal calculator that speaks percentages.

`pct` answers the percentage questions that come up at a keyboard all day —
"what is 20 plus 10%?", "what growth takes 20 to 20.8?", "what does 7% a year
do to 1000 over 5 years?" — without reaching for a spreadsheet.

```console
$ pct ev "20 + 10%"
22
$ pct to 20 20.8
4
$ pct comp 7 1000 5
1402.55
```

## Installation

```console
$ go install github.com/bpineau/pct@latest
```

or, from a checkout:

```console
$ make install
```

## Commands

### `eval` (alias `ev`) — evaluate an expression

Evaluates ordinary infix arithmetic (`+`, `-`, `*`, `/`, `^`, parentheses)
in which percentages are first-class terms:

```console
$ pct ev "20 + 10%"          # 20 increased by 10%
22
$ pct ev "(10 + 5) * 5%"     # 5% of 15
0.75
$ pct ev "(10 + 50%) * 5%"   # 5% of (10 increased by 50%)
0.75
```

Quoting the expression is optional as long as the shell leaves its pieces
alone; `pct ev 20 +10%` works as well.

Exponentiation follows the usual conventions: `^` binds tighter than the
other operators (`2 * 5^2` is 50), associates to the right (`2^3^2` is
2⁹ = 512), and a leading sign applies to the whole power (`-2^2` is −4):

```console
$ pct ev "5^3"
125
$ pct ev "9^(1/2)"      # fractional exponents work too
3
```

A few functions and constants are built in: `sqrt`, `abs`, `round`,
`floor` and `ceil` take one argument, `min` and `max` take any number of
them, and `pi` and `e` are predefined:

```console
$ pct ev "sqrt(2) * 100"
141.42
$ pct ev "max(1500, 2000) - 12.5%"
1750
```

### `add` — add a percentage to a number

```console
$ pct add 2 100      # 100 increased by 2%
102
$ pct add -2 100     # 100 decreased by 2%
98
```

`add x y` is shorthand for `eval "y + x%"`, kept for compatibility with an
older version of this tool.

### `to` — percentage change between two numbers

How much percent must be added to the first number to reach the second:

```console
$ pct to 20 20.8     # adding 4% to 20 gives 20.8
4
```

### `whatof` — percentage one number is of another

```console
$ pct whatof 10 200  # 10 is 5% of 200
5
```

### `compound` (alias `comp`) — compound growth

Increases a number by a percentage repeatedly:

```console
$ pct comp 7 1000 5  # 1000 growing 7% over 5 periods
1402.55
```

## Standard input and interactive use

Run without a command, `pct` evaluates expressions from standard input,
one per line — handy in pipes and scripts:

```console
$ printf '42 * 270.42 - 20%%\nans / 12\n' | pct
9086.11
757.18
```

`ans` names the previous result, at full precision rather than as
displayed, so chained steps don't accumulate rounding error. A line that
fails is reported on stderr, the remaining lines still run, and the exit
code records the failure.

On a terminal the same mode is interactive: type an expression per line,
reuse `ans`, and leave with `quit`, `exit` or Ctrl-D:

```console
$ pct
pct — type an expression; "ans" is the previous result, "quit" leaves
> 1402.55 - 20%
1122.04
> ans * 12
13464.48
> quit
```

## How percentages are evaluated

A percentage literal such as `20%` has no value of its own; it borrows one
from its context:

- **Added or subtracted, it is relative to the preceding operand.**
  `20 + 10%` increases 20 by ten percent, giving 22 — not 20.1. The
  preceding operand is the previous term: in `100 + 10 + 50%`, the
  percentage applies to 10, giving 115. When percentages follow one
  another they compound on the running result: `1000 - 20% - 20%` is 640.

- **Multiplied, divided, exponentiated, parenthesized or standing alone,
  it is simply a fraction of one.** `150 * 10%` is 15, `10 / 50%` is 20,
  `5^50%` is √5, and `50%` alone prints 0.5.

A longer expression, decomposed:

```console
$ pct eval "(42 * 270.42 - 20%) / 12 - 20% + 150 * 10% + 5 + 8.1%"
626.15
```

| Step              | Effect                          | Running result |
| ----------------- | ------------------------------- | -------------- |
| `42 * 270.42`     | product                         | 11357.64       |
| `- 20%`           | minus 20% of the result         | 9086.11        |
| `/ 12`            | quotient                        | 757.18         |
| `- 20%`           | minus 20% of the result         | 605.74         |
| `+ 150 * 10%`     | plus 10% of 150                 | 620.74         |
| `+ 5`             | plus 5                          | 625.74         |
| `+ 8.1%`          | plus 8.1% of 5                  | 626.15         |

## Precision

Results print with at most two decimal places, trailing zeros trimmed.
`--precision` (before or after the command) raises or lowers that limit:

```console
$ pct comp 7 1000 5 --precision 4
1402.5517
```

## Exit codes

| Code | Meaning                                          |
| ---- | ------------------------------------------------ |
| 0    | success                                          |
| 1    | the calculation failed (bad number, syntax error, division by zero, …) |
| 2    | usage error (unknown command, option or arity)   |

## Development

Standard library only. The expression evaluator is a small
recursive-descent parser in [`internal/expr`](internal/expr), the
fixed-form arithmetic lives in [`internal/percent`](internal/percent), and
the argument handling in [`internal/cli`](internal/cli).

```console
$ make help
  build      Build the pct binary
  install    Install pct into GOBIN
  test       Run all tests
  fuzz       Fuzz the expression evaluator for a short while
  cover      Run tests and report coverage
  fmt        Reformat all sources with gofmt
  fmt-check  Fail if any source file is not gofmt-formatted
  vet        Run go vet
  lint       Run golangci-lint
  check      Run every quality gate
  clean      Remove build and coverage artifacts
  help       Show this help
```

`make check` runs the full gate: formatting, `go vet`,
[golangci-lint](https://golangci-lint.run) and the test suite.
