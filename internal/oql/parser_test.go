package oql

import (
	"testing"
)

func TestParseSimpleSelect(t *testing.T) {
	stmt, err := Parse("SELECT @class, @shallowSize FROM java.util.HashMap")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(stmt.Columns) != 2 {
		t.Fatalf("columns = %d, want 2", len(stmt.Columns))
	}
	if stmt.From.ClassName != "java.util.HashMap" {
		t.Errorf("from = %q, want %q", stmt.From.ClassName, "java.util.HashMap")
	}
	if stmt.From.Instanceof {
		t.Error("instanceof should be false")
	}
}

func TestParseInstanceof(t *testing.T) {
	stmt, err := Parse("SELECT * FROM INSTANCEOF java.util.AbstractMap")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if !stmt.From.Instanceof {
		t.Error("instanceof should be true")
	}
	if stmt.From.ClassName != "java.util.AbstractMap" {
		t.Errorf("class = %q, want %q", stmt.From.ClassName, "java.util.AbstractMap")
	}
}

func TestParseWhereClause(t *testing.T) {
	stmt, err := Parse("SELECT * FROM java.lang.String WHERE @shallowSize > 1024")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if stmt.Where == nil {
		t.Fatal("WHERE should not be nil")
	}
	be, ok := stmt.Where.(BinaryExpr)
	if !ok {
		t.Fatalf("WHERE should be BinaryExpr, got %T", stmt.Where)
	}
	if be.Op != TokGt {
		t.Errorf("op = %d, want TokGt", be.Op)
	}
}

func TestParseGroupByOrderByLimit(t *testing.T) {
	stmt, err := Parse("SELECT @class, count(*) FROM instanceof java.lang.Object GROUP BY @class ORDER BY count DESC LIMIT 10")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(stmt.GroupBy) != 1 {
		t.Errorf("groupBy = %d, want 1", len(stmt.GroupBy))
	}
	if len(stmt.OrderBy) != 1 {
		t.Errorf("orderBy = %d, want 1", len(stmt.OrderBy))
	}
	if !stmt.OrderBy[0].Desc {
		t.Error("orderBy should be DESC")
	}
	if stmt.Limit != 10 {
		t.Errorf("limit = %d, want 10", stmt.Limit)
	}
}

func TestLexer(t *testing.T) {
	tokens, err := NewLexer("SELECT *, count(*) FROM 'test'").Tokenize()
	if err != nil {
		t.Fatalf("Tokenize: %v", err)
	}
	// SELECT, *, ,, count, (, *, ), FROM, 'test', EOF
	expected := []TokenType{TokSelect, TokStar, TokComma, TokIdent, TokLParen, TokStar, TokRParen, TokFrom, TokString, TokEOF}
	if len(tokens) != len(expected) {
		t.Fatalf("token count = %d, want %d", len(tokens), len(expected))
	}
	for i, tok := range tokens {
		if tok.Type != expected[i] {
			t.Errorf("token[%d] = %d (%q), want %d", i, tok.Type, tok.Value, expected[i])
		}
	}
}
