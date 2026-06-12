// Package cli implements the pct command-line interface.
package cli

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/bpineau/pct/internal/expr"
)

// defaultPrecision is the number of decimal places shown when --precision
// is not given.
const defaultPrecision = 2

// Exit codes returned by Run, following the usual Unix conventions.
const (
	exitOK    = 0
	exitError = 1
	exitUsage = 2
)

// options holds the global flags shared by every command.
type options struct {
	precision int
	help      bool
}

// Run executes pct with the given arguments (excluding the program name),
// writing the result to stdout and diagnostics to stderr. Without a command
// it evaluates expressions from stdin instead, interactively when stdin is
// a terminal. It returns the process exit code: 0 on success, 1 when a
// command or an input line fails, 2 on usage errors.
func Run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	opts, rest, err := splitArgs(args)
	if err != nil {
		fmt.Fprintf(stderr, "pct: %v\n", err)
		return exitUsage
	}
	if opts.help || (len(rest) > 0 && rest[0] == "help") {
		printUsage(stdout)
		return exitOK
	}
	if len(rest) == 0 {
		return session(stdin, stdout, stderr, opts.precision, isTerminal(stdin))
	}

	cmd := lookup(rest[0])
	if cmd == nil {
		fmt.Fprintf(stderr, "pct: unknown command %q (run \"pct help\")\n", rest[0])
		return exitUsage
	}
	cmdArgs := rest[1:]
	if !cmd.accepts(len(cmdArgs)) {
		fmt.Fprintf(stderr, "pct: usage: pct %s %s\n", cmd.name, cmd.args)
		return exitUsage
	}

	result, err := cmd.run(cmdArgs)
	if err != nil {
		fmt.Fprintf(stderr, "pct: %s: %v\n", cmd.name, err)
		return exitError
	}
	fmt.Fprintln(stdout, formatNumber(result, opts.precision))
	return exitOK
}

// session evaluates expressions read from in, one per line, printing each
// result to out. The previous result is available to the next line as
// "ans". Interactively it greets and prompts on errw, ends at "quit",
// "exit" or end of input, and a bad line never ends it; on piped input a
// bad line is reported, the remaining lines still run, and the exit code
// records the failure.
func session(in io.Reader, out, errw io.Writer, precision int, interactive bool) int {
	if interactive {
		fmt.Fprintln(errw, `pct — type an expression; "ans" is the previous result, "quit" leaves`)
	}
	vars := make(map[string]float64)
	code := exitOK
	scanner := bufio.NewScanner(in)
	for {
		if interactive {
			fmt.Fprint(errw, "> ")
		}
		if !scanner.Scan() {
			break
		}
		line := strings.TrimSpace(scanner.Text())
		switch {
		case line == "":
			continue
		case interactive && (line == "quit" || line == "exit"):
			return exitOK
		}
		result, err := expr.EvaluateWith(line, vars)
		if err != nil {
			fmt.Fprintf(errw, "pct: %v\n", err)
			if !interactive {
				code = exitError
			}
			continue
		}
		vars["ans"] = result
		fmt.Fprintln(out, formatNumber(result, precision))
	}
	if err := scanner.Err(); err != nil {
		fmt.Fprintf(errw, "pct: %v\n", err)
		return exitError
	}
	return code
}

// isTerminal reports whether r reads from an interactive terminal.
func isTerminal(r io.Reader) bool {
	f, ok := r.(*os.File)
	if !ok {
		return false
	}
	info, err := f.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}

// splitArgs separates global options from positional arguments. Only exact
// option names are recognized, so positional arguments that begin with a
// dash — like the negative percentage in "pct add -2 100" — pass through
// untouched, and options may appear before or after the command.
func splitArgs(args []string) (options, []string, error) {
	opts := options{precision: defaultPrecision}
	var rest []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--":
			rest = append(rest, args[i+1:]...)
			return opts, rest, nil
		case arg == "-h" || arg == "--help":
			opts.help = true
		case arg == "--precision" || strings.HasPrefix(arg, "--precision="):
			value, inline := strings.CutPrefix(arg, "--precision=")
			if !inline {
				i++
				if i == len(args) {
					return opts, nil, fmt.Errorf("option --precision requires a value")
				}
				value = args[i]
			}
			p, err := parseNonNegativeInt("precision", value)
			if err != nil {
				return opts, nil, err
			}
			opts.precision = p
		case strings.HasPrefix(arg, "--"):
			return opts, nil, fmt.Errorf("unknown option %q (run \"pct help\")", arg)
		default:
			rest = append(rest, arg)
		}
	}
	return opts, rest, nil
}

// formatNumber renders n with at most prec decimal places, trimming
// trailing zeros so that whole results print bare: 102 rather than 102.00.
func formatNumber(n float64, prec int) string {
	s := strconv.FormatFloat(n, 'f', prec, 64)
	if strings.Contains(s, ".") {
		s = strings.TrimRight(s, "0")
		s = strings.TrimSuffix(s, ".")
	}
	if s == "-0" {
		s = "0"
	}
	return s
}

// printUsage writes the global help screen, with the command table built
// from commands so it can never drift out of date.
func printUsage(w io.Writer) {
	fmt.Fprint(w, `pct — a terminal calculator that speaks percentages

Usage:

  pct [options] <command> <arguments>
  pct [options]                          evaluate expressions from standard input

Commands:

`)
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	for _, cmd := range commands {
		name := cmd.name
		if cmd.alias != "" {
			name += ", " + cmd.alias
		}
		fmt.Fprintf(tw, "  %s %s\t%s\n", name, cmd.args, cmd.summary)
	}
	tw.Flush()

	fmt.Fprint(w, "\nExamples:\n\n")
	tw = tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	for _, cmd := range commands {
		for _, ex := range cmd.examples {
			fmt.Fprintf(tw, "  pct %s\t# %s\n", ex.cmd, ex.note)
		}
	}
	tw.Flush()

	fmt.Fprintf(w, `
Options:

  --precision <digits>  decimal places shown (default %d)
  -h, --help            show this help
`, defaultPrecision)
}
