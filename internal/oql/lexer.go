package oql

import (
	"fmt"
	"strings"
	"unicode"
)

// TokenType represents the type of a lexer token.
type TokenType int

const (
	// Keywords
	TokSelect     TokenType = iota
	TokFrom
	TokWhere
	TokGroupBy
	TokOrderBy
	TokAsc
	TokDesc
	TokLimit
	TokInstanceof
	TokAnd
	TokOr
	TokNot
	TokLike
	TokIs
	TokNull
	TokAs

	// Literals and identifiers
	TokIdent
	TokString
	TokNumber

	// Operators
	TokStar    // *
	TokComma   // ,
	TokDot     // .
	TokAt      // @
	TokLParen  // (
	TokRParen  // )
	TokEq      // =
	TokNeq     // !=
	TokLt      // <
	TokGt      // >
	TokLte     // <=
	TokGte     // >=

	TokEOF
)

// Token represents a lexer token.
type Token struct {
	Type    TokenType
	Value   string
	Pos     int
}

var keywords = map[string]TokenType{
	"SELECT":     TokSelect,
	"FROM":       TokFrom,
	"WHERE":      TokWhere,
	"GROUP":      TokGroupBy, // "GROUP BY" handled in parser
	"ORDER":      TokOrderBy, // "ORDER BY" handled in parser
	"ASC":        TokAsc,
	"DESC":       TokDesc,
	"LIMIT":      TokLimit,
	"INSTANCEOF": TokInstanceof,
	"AND":        TokAnd,
	"OR":         TokOr,
	"NOT":        TokNot,
	"LIKE":       TokLike,
	"IS":         TokIs,
	"NULL":       TokNull,
	"AS":         TokAs,
}

// Lexer tokenizes an OQL query string.
type Lexer struct {
	input string
	pos   int
}

// NewLexer creates a new lexer for the given input.
func NewLexer(input string) *Lexer {
	return &Lexer{input: input}
}

// Tokenize returns all tokens from the input.
func (l *Lexer) Tokenize() ([]Token, error) {
	var tokens []Token
	for {
		tok, err := l.next()
		if err != nil {
			return nil, err
		}
		tokens = append(tokens, tok)
		if tok.Type == TokEOF {
			break
		}
	}
	return tokens, nil
}

func (l *Lexer) next() (Token, error) {
	l.skipWhitespace()

	if l.pos >= len(l.input) {
		return Token{Type: TokEOF, Pos: l.pos}, nil
	}

	pos := l.pos
	ch := l.input[l.pos]

	switch ch {
	case '*':
		l.pos++
		return Token{Type: TokStar, Value: "*", Pos: pos}, nil
	case ',':
		l.pos++
		return Token{Type: TokComma, Value: ",", Pos: pos}, nil
	case '.':
		l.pos++
		return Token{Type: TokDot, Value: ".", Pos: pos}, nil
	case '@':
		l.pos++
		return Token{Type: TokAt, Value: "@", Pos: pos}, nil
	case '(':
		l.pos++
		return Token{Type: TokLParen, Value: "(", Pos: pos}, nil
	case ')':
		l.pos++
		return Token{Type: TokRParen, Value: ")", Pos: pos}, nil
	case '=':
		l.pos++
		return Token{Type: TokEq, Value: "=", Pos: pos}, nil
	case '!':
		if l.pos+1 < len(l.input) && l.input[l.pos+1] == '=' {
			l.pos += 2
			return Token{Type: TokNeq, Value: "!=", Pos: pos}, nil
		}
		return Token{}, fmt.Errorf("unexpected character '!' at position %d", pos)
	case '<':
		if l.pos+1 < len(l.input) && l.input[l.pos+1] == '=' {
			l.pos += 2
			return Token{Type: TokLte, Value: "<=", Pos: pos}, nil
		}
		l.pos++
		return Token{Type: TokLt, Value: "<", Pos: pos}, nil
	case '>':
		if l.pos+1 < len(l.input) && l.input[l.pos+1] == '=' {
			l.pos += 2
			return Token{Type: TokGte, Value: ">=", Pos: pos}, nil
		}
		l.pos++
		return Token{Type: TokGt, Value: ">", Pos: pos}, nil
	case '\'', '"':
		return l.readString(ch)
	}

	if ch >= '0' && ch <= '9' {
		return l.readNumber()
	}

	if isIdentStart(ch) {
		return l.readIdent()
	}

	return Token{}, fmt.Errorf("unexpected character %q at position %d", ch, pos)
}

func (l *Lexer) skipWhitespace() {
	for l.pos < len(l.input) && unicode.IsSpace(rune(l.input[l.pos])) {
		l.pos++
	}
}

func (l *Lexer) readString(quote byte) (Token, error) {
	pos := l.pos
	l.pos++ // skip opening quote
	var sb strings.Builder
	for l.pos < len(l.input) {
		ch := l.input[l.pos]
		if ch == quote {
			l.pos++
			return Token{Type: TokString, Value: sb.String(), Pos: pos}, nil
		}
		if ch == '\\' && l.pos+1 < len(l.input) {
			l.pos++
			sb.WriteByte(l.input[l.pos])
		} else {
			sb.WriteByte(ch)
		}
		l.pos++
	}
	return Token{}, fmt.Errorf("unterminated string at position %d", pos)
}

func (l *Lexer) readNumber() (Token, error) {
	pos := l.pos
	for l.pos < len(l.input) && (l.input[l.pos] >= '0' && l.input[l.pos] <= '9' || l.input[l.pos] == '.') {
		l.pos++
	}
	return Token{Type: TokNumber, Value: l.input[pos:l.pos], Pos: pos}, nil
}

func (l *Lexer) readIdent() (Token, error) {
	pos := l.pos
	for l.pos < len(l.input) && isIdentPart(l.input[l.pos]) {
		l.pos++
	}
	value := l.input[pos:l.pos]

	if tt, ok := keywords[strings.ToUpper(value)]; ok {
		return Token{Type: tt, Value: value, Pos: pos}, nil
	}
	return Token{Type: TokIdent, Value: value, Pos: pos}, nil
}

func isIdentStart(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || ch == '_' || ch == '$'
}

func isIdentPart(ch byte) bool {
	return isIdentStart(ch) || (ch >= '0' && ch <= '9') || ch == '/' || ch == '.'
}
