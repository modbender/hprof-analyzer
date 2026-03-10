package oql

import (
	"fmt"
	"strconv"
	"strings"
)

// Parser parses OQL tokens into an AST.
type Parser struct {
	tokens []Token
	pos    int
}

// Parse parses an OQL query string into a SelectStmt.
func Parse(query string) (*SelectStmt, error) {
	lexer := NewLexer(query)
	tokens, err := lexer.Tokenize()
	if err != nil {
		return nil, err
	}
	p := &Parser{tokens: tokens}
	return p.parseSelect()
}

func (p *Parser) peek() Token {
	if p.pos >= len(p.tokens) {
		return Token{Type: TokEOF}
	}
	return p.tokens[p.pos]
}

func (p *Parser) advance() Token {
	tok := p.peek()
	p.pos++
	return tok
}

func (p *Parser) expect(tt TokenType) (Token, error) {
	tok := p.advance()
	if tok.Type != tt {
		return tok, fmt.Errorf("expected %d, got %q at position %d", tt, tok.Value, tok.Pos)
	}
	return tok, nil
}

func (p *Parser) parseSelect() (*SelectStmt, error) {
	if _, err := p.expect(TokSelect); err != nil {
		return nil, fmt.Errorf("expected SELECT: %w", err)
	}

	stmt := &SelectStmt{}

	// Parse columns
	cols, err := p.parseColumns()
	if err != nil {
		return nil, err
	}
	stmt.Columns = cols

	// Parse FROM
	if _, err := p.expect(TokFrom); err != nil {
		return nil, fmt.Errorf("expected FROM: %w", err)
	}
	from, err := p.parseFrom()
	if err != nil {
		return nil, err
	}
	stmt.From = from

	// Optional WHERE
	if p.peek().Type == TokWhere {
		p.advance()
		where, err := p.parseExpr()
		if err != nil {
			return nil, fmt.Errorf("parsing WHERE: %w", err)
		}
		stmt.Where = where
	}

	// Optional GROUP BY
	if p.peekKeyword("GROUP") {
		p.advance() // consume GROUP
		if !p.peekKeyword("BY") {
			return nil, fmt.Errorf("expected BY after GROUP")
		}
		p.advance() // consume BY (it's lexed as TokIdent since "BY" isn't in keywords map alone)
		gb, err := p.parseGroupBy()
		if err != nil {
			return nil, err
		}
		stmt.GroupBy = gb
	}

	// Optional ORDER BY
	if p.peekKeyword("ORDER") {
		p.advance()
		if !p.peekKeyword("BY") {
			return nil, fmt.Errorf("expected BY after ORDER")
		}
		p.advance()
		ob, err := p.parseOrderBy()
		if err != nil {
			return nil, err
		}
		stmt.OrderBy = ob
	}

	// Optional LIMIT
	if p.peek().Type == TokLimit {
		p.advance()
		tok, err := p.expect(TokNumber)
		if err != nil {
			return nil, fmt.Errorf("expected number after LIMIT: %w", err)
		}
		n, err := strconv.Atoi(tok.Value)
		if err != nil {
			return nil, fmt.Errorf("invalid LIMIT value: %w", err)
		}
		stmt.Limit = n
	}

	return stmt, nil
}

func (p *Parser) peekKeyword(kw string) bool {
	tok := p.peek()
	return (tok.Type == TokIdent || tok.Type == TokGroupBy || tok.Type == TokOrderBy) &&
		strings.EqualFold(tok.Value, kw)
}

func (p *Parser) parseColumns() ([]Column, error) {
	var cols []Column
	for {
		col, err := p.parseColumn()
		if err != nil {
			return nil, err
		}
		cols = append(cols, col)
		if p.peek().Type != TokComma {
			break
		}
		p.advance() // consume comma
	}
	return cols, nil
}

func (p *Parser) parseColumn() (Column, error) {
	expr, err := p.parseExpr()
	if err != nil {
		return Column{}, err
	}
	col := Column{Expr: expr}

	// Optional alias
	if p.peek().Type == TokAs {
		p.advance()
		tok, err := p.expect(TokIdent)
		if err != nil {
			return col, fmt.Errorf("expected alias: %w", err)
		}
		col.Alias = tok.Value
	}

	return col, nil
}

func (p *Parser) parseFrom() (FromClause, error) {
	from := FromClause{}

	if p.peek().Type == TokInstanceof {
		from.Instanceof = true
		p.advance()
	}

	tok := p.advance()
	if tok.Type != TokIdent && tok.Type != TokString {
		return from, fmt.Errorf("expected class name, got %q", tok.Value)
	}
	from.ClassName = tok.Value

	// Optional alias
	if p.peek().Type == TokIdent && !p.isKeyword(p.peek()) {
		from.Alias = p.advance().Value
	}

	return from, nil
}

func (p *Parser) isKeyword(tok Token) bool {
	upper := strings.ToUpper(tok.Value)
	_, ok := keywords[upper]
	return ok || upper == "BY"
}

func (p *Parser) parseGroupBy() ([]string, error) {
	var fields []string
	for {
		field := ""
		if p.peek().Type == TokAt {
			p.advance()
			field = "@"
		}
		tok := p.advance()
		if tok.Type != TokIdent {
			return nil, fmt.Errorf("expected field name in GROUP BY, got %q", tok.Value)
		}
		field += tok.Value
		fields = append(fields, field)
		if p.peek().Type != TokComma {
			break
		}
		p.advance()
	}
	return fields, nil
}

func (p *Parser) parseOrderBy() ([]OrderByClause, error) {
	var clauses []OrderByClause
	for {
		field := ""
		if p.peek().Type == TokAt {
			p.advance()
			field = "@"
		}
		tok := p.advance()
		if tok.Type != TokIdent {
			return nil, fmt.Errorf("expected field name in ORDER BY, got %q", tok.Value)
		}
		ob := OrderByClause{Field: field + tok.Value}
		if p.peek().Type == TokDesc {
			ob.Desc = true
			p.advance()
		} else if p.peek().Type == TokAsc {
			p.advance()
		}
		clauses = append(clauses, ob)
		if p.peek().Type != TokComma {
			break
		}
		p.advance()
	}
	return clauses, nil
}

func (p *Parser) parseExpr() (Expr, error) {
	return p.parseOr()
}

func (p *Parser) parseOr() (Expr, error) {
	left, err := p.parseAnd()
	if err != nil {
		return nil, err
	}
	for p.peek().Type == TokOr {
		p.advance()
		right, err := p.parseAnd()
		if err != nil {
			return nil, err
		}
		left = LogicalExpr{Left: left, Op: TokOr, Right: right}
	}
	return left, nil
}

func (p *Parser) parseAnd() (Expr, error) {
	left, err := p.parseComparison()
	if err != nil {
		return nil, err
	}
	for p.peek().Type == TokAnd {
		p.advance()
		right, err := p.parseComparison()
		if err != nil {
			return nil, err
		}
		left = LogicalExpr{Left: left, Op: TokAnd, Right: right}
	}
	return left, nil
}

func (p *Parser) parseComparison() (Expr, error) {
	left, err := p.parsePrimary()
	if err != nil {
		return nil, err
	}

	switch p.peek().Type {
	case TokEq, TokNeq, TokLt, TokGt, TokLte, TokGte, TokLike:
		op := p.advance()
		right, err := p.parsePrimary()
		if err != nil {
			return nil, err
		}
		return BinaryExpr{Left: left, Op: op.Type, Right: right}, nil
	case TokIs:
		p.advance()
		negate := false
		if p.peek().Type == TokNot {
			negate = true
			p.advance()
		}
		if p.peek().Type != TokNull {
			return nil, fmt.Errorf("expected NULL after IS")
		}
		p.advance()
		return IsNullExpr{Expr: left, Negate: negate}, nil
	}

	return left, nil
}

func (p *Parser) parsePrimary() (Expr, error) {
	tok := p.peek()

	switch tok.Type {
	case TokStar:
		p.advance()
		return StarExpr{}, nil

	case TokNumber:
		p.advance()
		return NumberLit{Value: tok.Value}, nil

	case TokString:
		p.advance()
		return StringLit{Value: tok.Value}, nil

	case TokNull:
		p.advance()
		return NullLit{}, nil

	case TokNot:
		p.advance()
		expr, err := p.parsePrimary()
		if err != nil {
			return nil, err
		}
		return NotExpr{Expr: expr}, nil

	case TokLParen:
		p.advance()
		expr, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		if _, err := p.expect(TokRParen); err != nil {
			return nil, fmt.Errorf("expected ')': %w", err)
		}
		return expr, nil

	case TokAt:
		p.advance()
		name, err := p.expect(TokIdent)
		if err != nil {
			return nil, fmt.Errorf("expected property name after @: %w", err)
		}
		return FieldAccess{Field: name.Value, IsBuiltin: true}, nil

	case TokIdent:
		p.advance()
		name := tok.Value

		// Check for function call
		if p.peek().Type == TokLParen {
			p.advance()
			var args []Expr
			if p.peek().Type != TokRParen {
				for {
					arg, err := p.parseExpr()
					if err != nil {
						return nil, err
					}
					args = append(args, arg)
					if p.peek().Type != TokComma {
						break
					}
					p.advance()
				}
			}
			if _, err := p.expect(TokRParen); err != nil {
				return nil, fmt.Errorf("expected ')': %w", err)
			}
			return FuncCall{Name: name, Args: args}, nil
		}

		// Check for dot access: ident.field
		if p.peek().Type == TokDot {
			p.advance()
			if p.peek().Type == TokAt {
				p.advance()
				field, err := p.expect(TokIdent)
				if err != nil {
					return nil, err
				}
				return FieldAccess{Object: name, Field: field.Value, IsBuiltin: true}, nil
			}
			field, err := p.expect(TokIdent)
			if err != nil {
				return nil, err
			}
			return FieldAccess{Object: name, Field: field.Value}, nil
		}

		return FieldAccess{Field: name}, nil

	default:
		// If we hit FROM, WHERE, etc., return nil to signal end of expression
		if p.isKeyword(tok) || tok.Type == TokEOF || tok.Type == TokComma || tok.Type == TokRParen {
			return nil, fmt.Errorf("unexpected token %q at position %d", tok.Value, tok.Pos)
		}
		return nil, fmt.Errorf("unexpected token %q at position %d", tok.Value, tok.Pos)
	}
}
