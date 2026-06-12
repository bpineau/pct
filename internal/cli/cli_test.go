package cli

import (
	"bytes"
	"slices"
	"strings"
	"testing"
)

func TestRun(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string // expected stdout
	}{
		// Examples from the command specification.
		{"add", []string{"add", "2", "100"}, "102\n"},
		{"add negative percentage", []string{"add", "-2", "100"}, "98\n"},
		{"ev with split arguments", []string{"ev", "20", "+10%"}, "22\n"},
		{"ev percentage of a sum", []string{"ev", "(10 + 5) * 5%"}, "0.75\n"},
		{"ev nested percentages", []string{"ev", "(10 + 50%) * 5%"}, "0.75\n"},
		{
			"eval long expression",
			[]string{"eval", "(42 *270.42 - 20%) / 12 -20% + 150 * 10% + 5 + 8.1%"},
			"626.15\n",
		},
		{"to", []string{"to", "20", "20.8"}, "4\n"},
		{"whatof", []string{"whatof", "10", "200"}, "5\n"},
		{"comp", []string{"comp", "7", "1000", "5"}, "1402.55\n"},

		// Aliases.
		{"eval alias of ev", []string{"eval", "20", "+10%"}, "22\n"},
		{"compound alias of comp", []string{"compound", "7", "1000", "5"}, "1402.55\n"},

		// Precision control, before or after the command.
		{"precision before", []string{"--precision", "4", "comp", "7", "1000", "5"}, "1402.5517\n"},
		{"precision after", []string{"comp", "7", "1000", "5", "--precision", "4"}, "1402.5517\n"},
		{"precision with equals", []string{"--precision=0", "comp", "7", "1000", "5"}, "1403\n"},
		{"precision zero keeps integers", []string{"--precision=0", "add", "2", "100"}, "102\n"},

		// Negative results survive formatting.
		{"negative result", []string{"to", "100", "80"}, "-20\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := Run(tt.args, &stdout, &stderr)
			if code != 0 {
				t.Fatalf("Run(%q) = %d, want 0; stderr: %s", tt.args, code, stderr.String())
			}
			if got := stdout.String(); got != tt.want {
				t.Errorf("Run(%q) printed %q, want %q", tt.args, got, tt.want)
			}
		})
	}
}

func TestRunErrors(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantCode   int
		wantStderr string // substring of stderr
	}{
		{"no arguments", nil, 2, "Usage:"},
		{"no arguments shows examples", nil, 2, "Examples:"},
		{"unknown command", []string{"frobnicate", "1"}, 2, `unknown command "frobnicate"`},
		{"missing arguments", []string{"add", "2"}, 2, "usage: pct add <percent> <number>"},
		{"extra arguments", []string{"to", "1", "2", "3"}, 2, "usage: pct to <from> <to>"},
		{"eval without expression", []string{"eval"}, 2, "usage: pct eval <expression>"},
		{"missing precision value", []string{"--precision"}, 2, `option --precision requires a value`},
		{"bad precision", []string{"--precision", "x", "add", "2", "100"}, 2, `invalid precision "x"`},
		{"negative precision", []string{"--precision=-1", "add", "2", "100"}, 2, `invalid precision "-1"`},
		{"unknown option", []string{"--frob", "add", "2", "100"}, 2, `unknown option "--frob"`},
		{"bad number", []string{"add", "two", "100"}, 1, `invalid percentage "two"`},
		{"bad expression", []string{"ev", "20 ++"}, 1, "unexpected end of expression"},
		{"division by zero", []string{"ev", "1/0"}, 1, "division by zero"},
		{"change from zero", []string{"to", "0", "5"}, 1, "reference value is zero"},
		{"share of zero", []string{"whatof", "5", "0"}, 1, "reference value is zero"},
		{"fractional periods", []string{"comp", "7", "1000", "1.5"}, 1, `invalid number of periods "1.5"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := Run(tt.args, &stdout, &stderr)
			if code != tt.wantCode {
				t.Fatalf("Run(%q) = %d, want %d; stderr: %s", tt.args, code, tt.wantCode, stderr.String())
			}
			if got := stderr.String(); !strings.Contains(got, tt.wantStderr) {
				t.Errorf("Run(%q) stderr = %q, want it to contain %q", tt.args, got, tt.wantStderr)
			}
		})
	}
}

func TestRunHelp(t *testing.T) {
	for _, args := range [][]string{{"-h"}, {"--help"}, {"help"}} {
		t.Run(args[0], func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			if code := Run(args, &stdout, &stderr); code != 0 {
				t.Fatalf("Run(%q) = %d, want 0; stderr: %s", args, code, stderr.String())
			}
			out := stdout.String()
			for _, want := range []string{
				"Usage:", "add", "eval", "to", "whatof", "compound", "--precision",
				"Examples:",
				`pct ev "20 + 10%"`,
				"# 22",
				"pct add -2 100",
				"# 98",
			} {
				if !strings.Contains(out, want) {
					t.Errorf("Run(%q) help output is missing %q", args, want)
				}
			}
		})
	}
}

func TestSplitArgs(t *testing.T) {
	tests := []struct {
		name          string
		args          []string
		wantPrecision int
		wantRest      []string
	}{
		{"defaults", []string{"add", "2", "100"}, 2, []string{"add", "2", "100"}},
		{
			"dash arguments stay positional",
			[]string{"add", "-2", "100"},
			2, []string{"add", "-2", "100"},
		},
		{
			"option anywhere",
			[]string{"add", "2", "--precision", "4", "100"},
			4, []string{"add", "2", "100"},
		},
		{
			"everything after double dash is positional",
			[]string{"--precision", "4", "--", "--precision"},
			4, []string{"--precision"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts, rest, err := splitArgs(tt.args)
			if err != nil {
				t.Fatalf("splitArgs(%q) returned error: %v", tt.args, err)
			}
			if opts.precision != tt.wantPrecision {
				t.Errorf("splitArgs(%q) precision = %d, want %d", tt.args, opts.precision, tt.wantPrecision)
			}
			if !slices.Equal(rest, tt.wantRest) {
				t.Errorf("splitArgs(%q) rest = %q, want %q", tt.args, rest, tt.wantRest)
			}
		})
	}
}

func TestFormatNumber(t *testing.T) {
	tests := []struct {
		n    float64
		prec int
		want string
	}{
		{102, 2, "102"},
		{102.5, 2, "102.5"},
		{0.75, 2, "0.75"},
		{626.1458, 2, "626.15"},
		{1402.5517307, 4, "1402.5517"},
		{1402.5517307, 0, "1403"},
		{-20, 2, "-20"},
		{-0.0001, 2, "0"}, // rounds to zero, not "-0"
		{0, 2, "0"},
	}
	for _, tt := range tests {
		if got := formatNumber(tt.n, tt.prec); got != tt.want {
			t.Errorf("formatNumber(%v, %d) = %q, want %q", tt.n, tt.prec, got, tt.want)
		}
	}
}
