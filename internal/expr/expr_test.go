package expr

import (
	"errors"
	"math"
	"strings"
	"testing"
)

func TestEvaluate(t *testing.T) {
	tests := []struct {
		input string
		want  float64
	}{
		// Plain arithmetic.
		{"42", 42},
		{"2 + 3 * 4", 14},
		{"(2 + 3) * 4", 20},
		{"10 / 4", 2.5},
		{"-5 + 10", 5},
		{"--5", 5},
		{"2 * -3", -6},
		{"1 - 2 - 3", -4},
		{"12 / 3 / 2", 2},

		// Percentages added or subtracted are relative to the
		// preceding operand.
		{"20 + 10%", 22},
		{"20 +10%", 22},
		{"80 - 25%", 60},
		{"100 + -10%", 90},
		{"100 - 20 + 5%", 81},      // 5% of 20, the preceding operand
		{"100 + 10 + 50%", 115},    // 50% of 10, not of 110
		{"1000 - 20% - 20%", 640},  // successive percentages compound
		{"20 + 10% + 10%", 24.2},   // 22, then 10% of 22
		{"(10 + 50%) * 2 + 10%", 33}, // 30, then 10% of 30

		// Percentages multiplied, divided, parenthesized or standing
		// alone are plain fractions of one.
		{"50%", 0.5},
		{"-50%", -0.5},
		{"150 * 10%", 15},
		{"50% * 200", 100},
		{"10 / 50%", 20},
		{"(20%)", 0.2},
		{"100 + (10%)", 100.1},

		// Examples from the command specification.
		{"(10 + 5) * 5%", 0.75},
		{"(10 + 50%) * 5%", 0.75},
		{"(42 *270.42 - 20%) / 12 -20% + 150 * 10% + 5 + 8.1%", 626.1458},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := Evaluate(tt.input)
			if err != nil {
				t.Fatalf("Evaluate(%q) returned error: %v", tt.input, err)
			}
			if math.Abs(got-tt.want) > 1e-9*max(1, math.Abs(tt.want)) {
				t.Errorf("Evaluate(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestEvaluateSyntaxErrors(t *testing.T) {
	tests := []struct {
		input   string
		wantMsg string // substring of the error message
	}{
		{"", "unexpected end of expression"},
		{"1 +", "unexpected end of expression"},
		{"(1 + 2", `unexpected end of expression`},
		{"1)", `unexpected ")"`},
		{"1 2", `unexpected "2"`},
		{"* 3", `unexpected "*"`},
		{"%", `unexpected "%"`},
		{"(1 + 2)%", `unexpected "%"`},
		{"1..2", `invalid number "1..2"`},
		{"2 # 3", "unexpected character '#'"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			_, err := Evaluate(tt.input)
			if err == nil {
				t.Fatalf("Evaluate(%q) succeeded, want error containing %q", tt.input, tt.wantMsg)
			}
			var syntaxErr *SyntaxError
			if !errors.As(err, &syntaxErr) {
				t.Errorf("Evaluate(%q) error is %T, want *SyntaxError", tt.input, err)
			}
			if !strings.Contains(err.Error(), tt.wantMsg) {
				t.Errorf("Evaluate(%q) error = %q, want it to contain %q", tt.input, err, tt.wantMsg)
			}
		})
	}
}

func TestEvaluateSyntaxErrorPosition(t *testing.T) {
	_, err := Evaluate("12 + )")
	var syntaxErr *SyntaxError
	if !errors.As(err, &syntaxErr) {
		t.Fatalf("Evaluate(\"12 + )\") error = %v, want *SyntaxError", err)
	}
	if syntaxErr.Pos != 5 {
		t.Errorf("SyntaxError.Pos = %d, want 5", syntaxErr.Pos)
	}
}

func TestEvaluateDivisionByZero(t *testing.T) {
	for _, input := range []string{"1 / 0", "1 / (3 - 3)", "5 / 0%"} {
		t.Run(input, func(t *testing.T) {
			if _, err := Evaluate(input); !errors.Is(err, ErrDivisionByZero) {
				t.Errorf("Evaluate(%q) error = %v, want ErrDivisionByZero", input, err)
			}
		})
	}
}

func FuzzEvaluate(f *testing.F) {
	for _, seed := range []string{
		"20 + 10%",
		"(10 + 50%) * 5%",
		"(42 *270.42 - 20%) / 12 -20% + 150 * 10% + 5 + 8.1%",
		"1 / 0",
		"((((1))))",
		"-1--1",
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, input string) {
		// Evaluate must never panic, whatever the input.
		_, _ = Evaluate(input)
	})
}

func BenchmarkEvaluate(b *testing.B) {
	const input = "(42 *270.42 - 20%) / 12 -20% + 150 * 10% + 5 + 8.1%"
	for b.Loop() {
		if _, err := Evaluate(input); err != nil {
			b.Fatal(err)
		}
	}
}
