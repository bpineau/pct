package cli

import (
	"fmt"
	"slices"
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
	name     string
	alias    string // optional short or long form, "" if none
	args     string // placeholder list shown in usage messages
	summary  string
	arity    int // exact argument count, or variadic
	examples []example
	run      func(args []string) (float64, error)
}

// An example is one sample invocation shown by help, annotated with its
// result and the feature it demonstrates.
type example struct {
	cmd  string // the command line, without the leading "pct"
	note string // result and explanation, shown as a "#" comment
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
		examples: []example{
			{`add 2 100`, "102 — 100 raised by 2%"},
			{`add -2 100`, "98 — a negative percentage subtracts"},
		},
		run: func(args []string) (float64, error) {
			ns, err := parseNumbers(args, "percentage", "number")
			if err != nil {
				return 0, err
			}
			return percent.Add(ns[0], ns[1]), nil
		},
	},
	{
		name:    "eval",
		alias:   "ev",
		args:    "<expression>",
		summary: "evaluate an expression with percentage terms",
		arity:   variadic,
		examples: []example{
			{`ev "20 + 10%"`, "22 — a percentage added applies to what precedes it"},
			{`ev "(10 + 5) * 5%"`, "0.75 — multiplied, a percentage is a plain fraction"},
			{`ev "1000 - 20% - 20%"`, "640 — consecutive percentages compound"},
			{`ev "5^3"`, "125 — ^ raises to a power"},
		},
		run: func(args []string) (float64, error) {
			return expr.Evaluate(strings.Join(args, " "))
		},
	},
	{
		name:    "to",
		args:    "<from> <to>",
		summary: "percentage to add to go from one number to the other",
		arity:   2,
		examples: []example{
			{`to 20 20.8`, "4 — adding 4% takes 20 to 20.8"},
		},
		run: func(args []string) (float64, error) {
			ns, err := parseNumbers(args, "starting number", "target number")
			if err != nil {
				return 0, err
			}
			return percent.Change(ns[0], ns[1])
		},
	},
	{
		name:    "whatof",
		args:    "<part> <whole>",
		summary: "percentage the first number is of the second",
		arity:   2,
		examples: []example{
			{`whatof 10 200`, "5 — 10 is 5% of 200"},
		},
		run: func(args []string) (float64, error) {
			ns, err := parseNumbers(args, "part", "whole")
			if err != nil {
				return 0, err
			}
			return percent.Share(ns[0], ns[1])
		},
	},
	{
		name:    "compound",
		alias:   "comp",
		args:    "<percent> <number> <periods>",
		summary: "compound a percentage over a number of periods",
		arity:   3,
		examples: []example{
			{`comp 7 1000 5`, "1402.55 — 7% per period over 5 periods"},
			{`comp 7 1000 5 --precision 4`, "1402.5517 — more decimal places"},
		},
		run: func(args []string) (float64, error) {
			ns, err := parseNumbers(args[:2], "percentage", "number")
			if err != nil {
				return 0, err
			}
			periods, err := parseNonNegativeInt("number of periods", args[2])
			if err != nil {
				return 0, err
			}
			return percent.Compound(ns[0], ns[1], periods), nil
		},
	},
}

// lookup finds a command by name or alias.
func lookup(name string) *command {
	i := slices.IndexFunc(commands, func(c command) bool {
		return c.name == name || c.alias == name
	})
	if i < 0 {
		return nil
	}
	return &commands[i]
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

// parseNumbers parses one positional argument per role.
func parseNumbers(args []string, roles ...string) ([]float64, error) {
	ns := make([]float64, len(args))
	for i, s := range args {
		n, err := parseNumber(roles[i], s)
		if err != nil {
			return nil, err
		}
		ns[i] = n
	}
	return ns, nil
}

// parseNonNegativeInt parses a count-like argument, naming the argument in
// the error message.
func parseNonNegativeInt(role, s string) (int, error) {
	n, err := strconv.Atoi(s)
	if err != nil || n < 0 {
		return 0, fmt.Errorf("invalid %s %q: want a non-negative integer", role, s)
	}
	return n, nil
}
