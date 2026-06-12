package expr

import (
	"fmt"
	"strconv"
	"unicode/utf8"
)

// tokenKind identifies the lexical class of a token.
type tokenKind int

const (
	tokenEOF tokenKind = iota
	tokenNumber
	tokenPercent
	tokenPlus
	tokenMinus
	tokenStar
	tokenSlash
	tokenLeftParen
	tokenRightParen
)

// A token is one lexical element of an expression. The text field holds the
// raw lexeme so error messages can quote the input verbatim.
type token struct {
	kind tokenKind
	text string
	num  float64 // value of a tokenNumber
	pos  int     // byte offset into the input
}

// A scanner turns an expression string into a stream of tokens.
type scanner struct {
	input string
	pos   int
}

// next returns the next token, skipping any leading whitespace.
func (s *scanner) next() (token, error) {
	for s.pos < len(s.input) && isSpace(s.input[s.pos]) {
		s.pos++
	}
	if s.pos == len(s.input) {
		return token{kind: tokenEOF, pos: s.pos}, nil
	}

	start := s.pos
	c := s.input[s.pos]
	if isDigit(c) || c == '.' {
		return s.scanNumber()
	}

	var kind tokenKind
	switch c {
	case '%':
		kind = tokenPercent
	case '+':
		kind = tokenPlus
	case '-':
		kind = tokenMinus
	case '*':
		kind = tokenStar
	case '/':
		kind = tokenSlash
	case '(':
		kind = tokenLeftParen
	case ')':
		kind = tokenRightParen
	default:
		r, _ := utf8.DecodeRuneInString(s.input[s.pos:])
		return token{}, &SyntaxError{Pos: start, Msg: fmt.Sprintf("unexpected character %q", r)}
	}
	s.pos++
	return token{kind: kind, text: string(c), pos: start}, nil
}

// scanNumber consumes a run of digits and dots and parses it as a float.
func (s *scanner) scanNumber() (token, error) {
	start := s.pos
	for s.pos < len(s.input) && (isDigit(s.input[s.pos]) || s.input[s.pos] == '.') {
		s.pos++
	}
	text := s.input[start:s.pos]
	n, err := strconv.ParseFloat(text, 64)
	if err != nil {
		return token{}, &SyntaxError{Pos: start, Msg: fmt.Sprintf("invalid number %q", text)}
	}
	return token{kind: tokenNumber, text: text, num: n, pos: start}, nil
}

func isSpace(c byte) bool { return c == ' ' || c == '\t' || c == '\n' || c == '\r' }

func isDigit(c byte) bool { return '0' <= c && c <= '9' }
