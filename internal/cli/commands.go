package cli

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/bpineau/pct/internal/expr"
	"github.com/bpineau/pct/internal/percent"
)

// variadic marks a command that takes any positive number of arguments.
const variadic = -1

// A command is one pct subcommand: a name, the shape of its arguments and
// a function from those arguments to a number.
type command struct {
	name    string
	alias   string // optional short or long form, "" if none
	args    string // placeholder list shown in usage messages
	summary string
	arity   int // exact argument count, or variadic
	run     func(args []string) (float64, error)
}

// accepts reports whether the command can be called with n arguments.
func (c *command) accepts(n int) bool {
	if c.arity == variadic {
		return n > 0
	}
	return n == c.arity
}

// commands lists every subcommand in the order shown by help.
var commands = []command{
	{
		name:    "add",
		args:    "<percent> <number>",
		summary: "add a percentage to a number",
		arity:   2,
		run: func(args []string) (float64, error) {
			p, err := parseNumber("percentage", args[0])
			if err != nil {
				return 0, err
			}
			n, err := parseNumber("number", args[1])
			if err != nil {
				return 0, err
			}
			return percent.Add(p, n), nil
		},
	},
	{
		name:    "eval",
		alias:   "ev",
		args:    "<expression>",
		summary: "evaluate an expression with percentage terms",
		arity:   variadic,
		run: func(args []string) (float64, error) {
			return expr.Evaluate(strings.Join(args, " "))
		},
	},
	{
		name:    "to",
		args:    "<from> <to>",
		summary: "percentage to add to go from one number to the other",
		arity:   2,
		run: func(args []string) (float64, error) {
			from, err := parseNumber("starting number", args[0])
			if err != nil {
				return 0, err
			}
			to, err := parseNumber("target number", args[1])
			if err != nil {
				return 0, err
			}
			return percent.Change(from, to)
		},
	},
	{
		name:    "whatof",
		args:    "<part> <whole>",
		summary: "percentage the first number is of the second",
		arity:   2,
		run: func(args []string) (float64, error) {
			part, err := parseNumber("part", args[0])
			if err != nil {
				return 0, err
			}
			whole, err := parseNumber("whole", args[1])
			if err != nil {
				return 0, err
			}
			return percent.Share(part, whole)
		},
	},
	{
		name:    "compound",
		alias:   "comp",
		args:    "<percent> <number> <periods>",
		summary: "compound a percentage over a number of periods",
		arity:   3,
		run: func(args []string) (float64, error) {
			p, err := parseNumber("percentage", args[0])
			if err != nil {
				return 0, err
			}
			n, err := parseNumber("number", args[1])
			if err != nil {
				return 0, err
			}
			periods, err := strconv.Atoi(args[2])
			if err != nil || periods < 0 {
				return 0, fmt.Errorf("invalid number of periods %q: want a non-negative integer", args[2])
			}
			return percent.Compound(p, n, periods), nil
		},
	},
}

// lookup finds a command by name or alias.
func lookup(name string) *command {
	for i := range commands {
		if commands[i].name == name || commands[i].alias == name {
			return &commands[i]
		}
	}
	return nil
}

// parseNumber parses a positional argument as a float, naming the argument
// in the error message.
func parseNumber(role, s string) (float64, error) {
	n, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid %s %q", role, s)
	}
	return n, nil
}
