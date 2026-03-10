package oql

// SelectStmt represents a parsed OQL SELECT statement.
type SelectStmt struct {
	Columns    []Column
	From       FromClause
	Where      Expr
	GroupBy    []string
	OrderBy    []OrderByClause
	Limit      int
}

// Column represents a selected column or expression.
type Column struct {
	Expr  Expr
	Alias string
}

// FromClause specifies the source of objects.
type FromClause struct {
	ClassName  string
	Instanceof bool // true if "FROM INSTANCEOF className"
	Alias      string
}

// OrderByClause represents an ORDER BY specification.
type OrderByClause struct {
	Field string
	Desc  bool
}

// Expr is the interface for all expression types.
type Expr interface {
	exprNode()
}

// FieldAccess represents obj.field or obj.@property access.
type FieldAccess struct {
	Object   string // empty means the FROM alias/implicit
	Field    string
	IsBuiltin bool // true for @shallowSize, @retainedSize, @class, @objectId
}

func (FieldAccess) exprNode() {}

// FuncCall represents a function call like count(*), sum(x), etc.
type FuncCall struct {
	Name string
	Args []Expr
}

func (FuncCall) exprNode() {}

// StarExpr represents *.
type StarExpr struct{}

func (StarExpr) exprNode() {}

// StringLit represents a string literal.
type StringLit struct {
	Value string
}

func (StringLit) exprNode() {}

// NumberLit represents a numeric literal.
type NumberLit struct {
	Value string
}

func (NumberLit) exprNode() {}

// BinaryExpr represents a binary comparison (a = b, a > b, etc.)
type BinaryExpr struct {
	Left  Expr
	Op    TokenType
	Right Expr
}

func (BinaryExpr) exprNode() {}

// LogicalExpr represents AND/OR logical operators.
type LogicalExpr struct {
	Left  Expr
	Op    TokenType // TokAnd or TokOr
	Right Expr
}

func (LogicalExpr) exprNode() {}

// NotExpr represents NOT expression.
type NotExpr struct {
	Expr Expr
}

func (NotExpr) exprNode() {}

// NullLit represents NULL.
type NullLit struct{}

func (NullLit) exprNode() {}

// IsNullExpr represents "expr IS NULL" or "expr IS NOT NULL".
type IsNullExpr struct {
	Expr   Expr
	Negate bool // true for IS NOT NULL
}

func (IsNullExpr) exprNode() {}
