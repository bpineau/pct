// Package expr evaluates arithmetic expressions in which percentages are
// first-class terms.
//
// The grammar is conventional infix arithmetic:
//
//	expression = term { ("+" | "-") term } .
//	term       = factor { ("*" | "/") factor } .
//	factor     = [ "+" | "-" ] ( number [ "%" ] | "(" expression ")" ) .
//
// A percentage literal such as "20%" has no value of its own; it borrows one
// from its context:
//
//   - Added or subtracted, it is relative to the preceding operand:
//     "20 + 10%" increases 20 by ten percent and yields 22, while
//     "80 - 25%" takes a quarter off 80 and yields 60. The preceding
//     operand is the previous plain term — in "100 + 10 + 50%" the
//     percentage applies to 10 — or the running result when percentages
//     follow one another, so "1000 - 20% - 20%" compounds to 640.
//
//   - Multiplied, divided, parenthesized or standing alone, it is simply
//     its fraction of one: "150 * 10%" is 15 and "50%" is 0.5.
package expr

import (
	"errors"
	"fmt"
)

// ErrDivisionByZero is returned when an expression divides by zero.
var ErrDivisionByZero = errors.New("division by zero")

// A SyntaxError describes an invalid expression and where it became invalid.
type SyntaxError struct {
	Pos int    // byte offset into the input
	Msg string // human-readable description
}

func (e *SyntaxError) Error() string {
	return fmt.Sprintf("position %d: %s", e.Pos+1, e.Msg)
}

// Evaluate computes the value of an arithmetic expression that may contain
// percentage terms. It returns a *SyntaxError for malformed input and
// ErrDivisionByZero for divisions by zero.
func Evaluate(input string) (float64, error) {
	p := parser{scan: scanner{input: input}}
	if err := p.advance(); err != nil {
		return 0, err
	}
	v, err := p.expression()
	if err != nil {
		return 0, err
	}
	if p.tok.kind != tokenEOF {
		return 0, p.unexpected()
	}
	return v.scalar(), nil
}

// A value is the result of a parsed subexpression. A percentage literal
// keeps the number as written (20 for "20%") and a marker, because its
// meaning depends on the operator that consumes it; every other consumer
// collapses it with scalar.
type value struct {
	num     float64
	percent bool
}

// scalar returns the context-free reading of v, where a percentage is its
// fraction of one: 20% becomes 0.2.
func (v value) scalar() float64 {
	if v.percent {
		return v.num / 100
	}
	return v.num
}

// A parser evaluates the token stream by recursive descent, one production
// per method. It keeps a single token of lookahead in tok.
type parser struct {
	scan scanner
	tok  token
}

// advance moves the lookahead to the next token.
func (p *parser) advance() error {
	tok, err := p.scan.next()
	if err != nil {
		return err
	}
	p.tok = tok
	return nil
}

// expression parses and evaluates a chain of added and subtracted terms.
//
// Percentage terms are relative, so the chain is folded with two
// accumulators: acc, the running result, and base, the value the next
// percentage applies to. A plain term contributes its own value and becomes
// the base; a percentage term adjusts acc by a share of the base and makes
// the new result the base, which is what lets consecutive percentages
// compound.
func (p *parser) expression() (value, error) {
	first, err := p.term()
	if err != nil {
		return value{}, err
	}
	acc := first.scalar()
	base := acc

	for p.tok.kind == tokenPlus || p.tok.kind == tokenMinus {
		sign := 1.0
		if p.tok.kind == tokenMinus {
			sign = -1
		}
		if err := p.advance(); err != nil {
			return value{}, err
		}
		next, err := p.term()
		if err != nil {
			return value{}, err
		}
		if next.percent {
			acc += sign * base * next.num / 100
			base = acc
		} else {
			acc += sign * next.num
			base = next.num
		}
	}
	return value{num: acc}, nil
}

// term parses and evaluates a chain of multiplied and divided factors.
// Multiplication and division read percentages as plain fractions, so the
// result of any such chain is a plain value.
func (p *parser) term() (value, error) {
	left, err := p.factor()
	if err != nil {
		return value{}, err
	}
	for p.tok.kind == tokenStar || p.tok.kind == tokenSlash {
		divide := p.tok.kind == tokenSlash
		if err := p.advance(); err != nil {
			return value{}, err
		}
		right, err := p.factor()
		if err != nil {
			return value{}, err
		}
		if divide {
			if right.scalar() == 0 {
				return value{}, ErrDivisionByZero
			}
			left = value{num: left.scalar() / right.scalar()}
		} else {
			left = value{num: left.scalar() * right.scalar()}
		}
	}
	return left, nil
}

// factor parses an optionally signed number, percentage or parenthesized
// expression. Parentheses seal their content into a plain value, so
// "100 + (10%)" adds a tenth, not ten percent of 100.
func (p *parser) factor() (value, error) {
	switch p.tok.kind {
	case tokenPlus, tokenMinus:
		negate := p.tok.kind == tokenMinus
		if err := p.advance(); err != nil {
			return value{}, err
		}
		v, err := p.factor()
		if err != nil {
			return value{}, err
		}
		if negate {
			v.num = -v.num
		}
		return v, nil

	case tokenNumber:
		v := value{num: p.tok.num}
		if err := p.advance(); err != nil {
			return value{}, err
		}
		if p.tok.kind == tokenPercent {
			v.percent = true
			if err := p.advance(); err != nil {
				return value{}, err
			}
		}
		return v, nil

	case tokenLeftParen:
		if err := p.advance(); err != nil {
			return value{}, err
		}
		v, err := p.expression()
		if err != nil {
			return value{}, err
		}
		if p.tok.kind != tokenRightParen {
			return value{}, p.unexpected()
		}
		if err := p.advance(); err != nil {
			return value{}, err
		}
		return v, nil

	default:
		return value{}, p.unexpected()
	}
}

// unexpected reports the lookahead token as out of place.
func (p *parser) unexpected() error {
	if p.tok.kind == tokenEOF {
		return &SyntaxError{Pos: p.tok.pos, Msg: "unexpected end of expression"}
	}
	return &SyntaxError{Pos: p.tok.pos, Msg: fmt.Sprintf("unexpected %q", p.tok.text)}
}
