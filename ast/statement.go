package ast

import "github.com/amitksharma00112/pinego/token"

// AssignmentStmt represents:
//
// x = 10
// x := 10
// x += 5
type AssignmentStmt struct {
	Target     Expr
	Operator   string
	Value      Expr
	SourceSpan token.Span
}

// WhileStmt represents:
//
// while condition
//
//	statements
type WhileStmt struct {
	Condition  Expr
	Body       *BlockStmt
	SourceSpan token.Span
}

func (*WhileStmt) node() {}
func (*WhileStmt) stmt() {}

func (w *WhileStmt) Span() token.Span {
	return w.SourceSpan
}

// ForStmt represents:
//
// for i = 0 to 10
//
//	statements
//
// and:
//
// for item in collection
//
//	statements
type ForStmt struct {
	Variable   *Identifier
	From       Expr
	To         Expr
	Step       Expr
	Iterable   Expr
	Body       *BlockStmt
	IsInLoop   bool
	SourceSpan token.Span
}

func (*ForStmt) node() {}
func (*ForStmt) stmt() {}

func (f *ForStmt) Span() token.Span {
	return f.SourceSpan
}

func (*AssignmentStmt) node() {}
func (*AssignmentStmt) stmt() {}

func (a *AssignmentStmt) Span() token.Span {
	return a.SourceSpan
}

// ExpressionStmt represents an expression used as a statement.
type ExpressionStmt struct {
	Expression Expr
	SourceSpan token.Span
}

func (*ExpressionStmt) node() {}
func (*ExpressionStmt) stmt() {}

func (e *ExpressionStmt) Span() token.Span {
	return e.SourceSpan
}

// VariableDeclStmt represents:
//
// var x = 10
// varip x = 10
// var float x = 10
// varip int x = 10
// float x = 10
// int x = 10
type VariableDeclStmt struct {
	Keyword    string
	TypeName   string
	Name       *Identifier
	Value      Expr
	SourceSpan token.Span
}

func (*VariableDeclStmt) node() {}
func (*VariableDeclStmt) stmt() {}

func (v *VariableDeclStmt) Span() token.Span {
	return v.SourceSpan
}

// Parameter represents a function parameter.
//
// x
// float x
// int length
type Parameter struct {
	TypeName   string
	Name       *Identifier
	SourceSpan token.Span
}

func (p *Parameter) Span() token.Span {
	return p.SourceSpan
}

// FunctionDeclStmt represents:
//
// add(x, y) =>
//
//	x + y
//
// add(float x, float y) =>
//
//	result = x + y
//	return result
type FunctionDeclStmt struct {
	Name       *Identifier
	Parameters []*Parameter
	Body       *BlockStmt
	SourceSpan token.Span
}

func (*FunctionDeclStmt) node() {}
func (*FunctionDeclStmt) stmt() {}

func (f *FunctionDeclStmt) Span() token.Span {
	return f.SourceSpan
}

// BlockStmt represents an indented block.
type BlockStmt struct {
	Statements []Stmt
	SourceSpan token.Span
}

func (*BlockStmt) node() {}
func (*BlockStmt) stmt() {}

func (b *BlockStmt) Span() token.Span {
	return b.SourceSpan
}

// IfStmt represents:
//
// if condition
//
//	...
//
// else if condition
//
//	...
//
// else
//
//	...
type IfStmt struct {
	Condition  Expr
	Then       *BlockStmt
	ElseIf     *IfStmt
	Else       *BlockStmt
	SourceSpan token.Span
}

func (*IfStmt) node() {}
func (*IfStmt) stmt() {}

func (i *IfStmt) Span() token.Span {
	return i.SourceSpan
}

// ReturnStmt represents:
//
// return
// return expression
type ReturnStmt struct {
	Value      Expr
	SourceSpan token.Span
}

func (*ReturnStmt) node() {}
func (*ReturnStmt) stmt() {}

func (r *ReturnStmt) Span() token.Span {
	return r.SourceSpan
}

// BreakStmt represents break.
type BreakStmt struct {
	SourceSpan token.Span
}

func (*BreakStmt) node() {}
func (*BreakStmt) stmt() {}

func (b *BreakStmt) Span() token.Span {
	return b.SourceSpan
}

// ContinueStmt represents continue.
type ContinueStmt struct {
	SourceSpan token.Span
}

func (*ContinueStmt) node() {}
func (*ContinueStmt) stmt() {}

func (c *ContinueStmt) Span() token.Span {
	return c.SourceSpan
}
