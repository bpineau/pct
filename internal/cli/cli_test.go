package cli

import (
	"bytes"
	"errors"
	"os"
	"slices"
	"strings"
	"testing"
)

// run invokes Run with empty standard input and returns the exit code and
// both output streams.
func run(args []string) (code int, stdout, stderr string) {
	var out, errw bytes.Buffer
	code = Run(args, strings.NewReader(""), &out, &errw)
	return code, out.String(), errw.String()
}

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
		{"ev power", []string{"ev", "5^3"}, "125\n"},
		{
			"eval long expression",
			[]string{"eval", "(42 *270.42 - 20%) / 12 -20% + 150 * 10% + 5 + 8.1%"},
			"626.15\n",
		},
		{"of", []string{"of", "20", "2000"}, "400\n"},
		{"base", []string{"base", "20", "120"}, "100\n"},
		{"base undoes a decrease", []string{"base", "-20", "80"}, "100\n"},
		{"to", []string{"to", "20", "20.8"}, "4\n"},
		{"whatof", []string{"whatof", "10", "200"}, "5\n"},
		{"comp", []string{"comp", "7", "1000", "5"}, "1402.55\n"},

		// Functions and constants.
		{"ev sqrt", []string{"ev", "sqrt(2)"}, "1.41\n"},
		{"ev constants", []string{"ev", "2 * pi"}, "6.28\n"},
		{"ev rounding helpers", []string{"ev", "ceil(min(2.3, 4))"}, "3\n"},

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
			code, stdout, stderr := run(tt.args)
			if code != 0 {
				t.Fatalf("Run(%q) = %d, want 0; stderr: %s", tt.args, code, stderr)
			}
			if stdout != tt.want {
				t.Errorf("Run(%q) printed %q, want %q", tt.args, stdout, tt.want)
			}
		})
	}
}

func TestRunStdin(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		input      string
		wantOut    string
		wantCode   int
		wantStderr string // substring of stderr, "" for none
	}{
		{
			name:    "one expression per line",
			input:   "20 + 10%\n1000 - 20% - 20%\n",
			wantOut: "22\n640\n",
		},
		{
			name:    "ans names the previous result",
			input:   "100 + 100\nans - 50%\nans + 10\n",
			wantOut: "200\n100\n110\n",
		},
		{
			name:    "blank lines are skipped",
			input:   "\n  \n1 + 1\n\n",
			wantOut: "2\n",
		},
		{
			name:       "an error does not end the session",
			input:      "20 +\n1 + 1\n",
			wantOut:    "2\n",
			wantCode:   1,
			wantStderr: "unexpected end of expression",
		},
		{
			name:    "precision option applies",
			args:    []string{"--precision", "4"},
			input:   "sqrt(2)\n",
			wantOut: "1.4142\n",
		},
		{
			name:    "empty input",
			input:   "",
			wantOut: "",
		},
		{
			name:    "comments are ignored",
			input:   "# a full-line comment\n100 + 10% # a trailing comment\n   # indented comment\n",
			wantOut: "110\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := Run(tt.args, strings.NewReader(tt.input), &stdout, &stderr)
			if code != tt.wantCode {
				t.Fatalf("Run(%q) = %d, want %d; stderr: %s", tt.args, code, tt.wantCode, stderr.String())
			}
			if got := stdout.String(); got != tt.wantOut {
				t.Errorf("Run(%q) printed %q, want %q", tt.args, got, tt.wantOut)
			}
			if got := stderr.String(); !strings.Contains(got, tt.wantStderr) {
				t.Errorf("Run(%q) stderr = %q, want it to contain %q", tt.args, got, tt.wantStderr)
			}
		})
	}
}

func TestInteractiveSession(t *testing.T) {
	var stdout, stderr bytes.Buffer
	in := strings.NewReader("1 + 1\nnonsense\nquit\n3 + 3\n")
	code := session(in, &stdout, &stderr, 2, true)
	if code != 0 {
		t.Fatalf("session() = %d, want 0 (interactive errors are not fatal)", code)
	}
	if got, want := stdout.String(), "2\n"; got != want {
		t.Errorf("session() printed %q, want %q (quit must end the session)", got, want)
	}
	errOut := stderr.String()
	for _, want := range []string{"> ", `unknown name "nonsense"`} {
		if !strings.Contains(errOut, want) {
			t.Errorf("session() stderr = %q, want it to contain %q", errOut, want)
		}
	}
}

func TestRunErrors(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantCode   int
		wantStderr string // substring of stderr
	}{
		{"unknown command", []string{"frobnicate", "1"}, 2, `unknown command "frobnicate"`},
		{"missing arguments", []string{"add", "2"}, 2, "usage: pct add <percent> <number>"},
		{"extra arguments", []string{"to", "1", "2", "3"}, 2, "usage: pct to <from> <to>"},
		{"eval without expression", []string{"eval"}, 2, "usage: pct eval <expression>"},
		{"missing precision value", []string{"--precision"}, 2, `option --precision requires a value`},
		{"bad precision", []string{"--precision", "x", "add", "2", "100"}, 2, `invalid precision "x"`},
		{"negative precision", []string{"--precision=-1", "add", "2", "100"}, 2, `invalid precision "-1"`},
		{"unknown option", []string{"--frob", "add", "2", "100"}, 2, `unknown option "--frob"`},
		{"bad number", []string{"add", "two", "100"}, 1, `invalid percentage "two"`},
		{"of bad number", []string{"of", "x", "100"}, 1, `invalid percentage "x"`},
		{"base bad number", []string{"base", "x", "100"}, 1, `invalid percentage "x"`},
		{"to bad number", []string{"to", "x", "100"}, 1, `invalid starting number "x"`},
		{"whatof bad number", []string{"whatof", "x", "100"}, 1, `invalid part "x"`},
		{"comp bad number", []string{"comp", "x", "100", "2"}, 1, `invalid percentage "x"`},
		{"bad expression", []string{"ev", "20 ++"}, 1, "unexpected end of expression"},
		{"division by zero", []string{"ev", "1/0"}, 1, "division by zero"},
		{"change from zero", []string{"to", "0", "5"}, 1, "reference value is zero"},
		{"base of a total loss", []string{"base", "-100", "50"}, 1, "collapses every number to zero"},
		{"share of zero", []string{"whatof", "5", "0"}, 1, "reference value is zero"},
		{"fractional periods", []string{"comp", "7", "1000", "1.5"}, 1, `invalid number of periods "1.5"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, _, stderr := run(tt.args)
			if code != tt.wantCode {
				t.Fatalf("Run(%q) = %d, want %d; stderr: %s", tt.args, code, tt.wantCode, stderr)
			}
			if !strings.Contains(stderr, tt.wantStderr) {
				t.Errorf("Run(%q) stderr = %q, want it to contain %q", tt.args, stderr, tt.wantStderr)
			}
		})
	}
}

func TestRunHelp(t *testing.T) {
	for _, args := range [][]string{{"-h"}, {"--help"}, {"help"}} {
		t.Run(args[0], func(t *testing.T) {
			code, out, stderr := run(args)
			if code != 0 {
				t.Fatalf("Run(%q) = %d, want 0; stderr: %s", args, code, stderr)
			}
			for _, want := range []string{
				"Usage:", "add", "eval", "to", "whatof", "compound", "--precision",
				"of 20 2000",
				"base 20 120",
				"standard input",
				`"ans"`,            // the previous-result variable
				"starts a comment", // "#" comments
				"quit",             // how to leave the interactive session
				"sqrt",
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

// failingReader always fails, standing in for an unreadable stdin.
type failingReader struct{}

func (failingReader) Read([]byte) (int, error) { return 0, errors.New("stdin unreadable") }

func TestSessionReadError(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := session(failingReader{}, &stdout, &stderr, 2, false); code != 1 {
		t.Fatalf("session() = %d, want 1 on a read error", code)
	}
	if got := stderr.String(); !strings.Contains(got, "stdin unreadable") {
		t.Errorf("session() stderr = %q, want the read error reported", got)
	}
}

func TestIsTerminal(t *testing.T) {
	if isTerminal(strings.NewReader("")) {
		t.Error("isTerminal(strings.Reader) = true, want false")
	}
	f, err := os.CreateTemp(t.TempDir(), "pct")
	if err != nil {
		t.Fatal(err)
	}
	if isTerminal(f) {
		t.Error("isTerminal(regular file) = true, want false")
	}
	if err := f.Close(); err != nil {
		t.Error(err)
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
		{24.4, 0, "24"}, // precision 0 rounds and prints no decimal point
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
