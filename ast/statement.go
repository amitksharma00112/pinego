package ast

import "github.com/amitksharma00112/pinego/token"

// AssignmentStmt represents:
// x = 10
// x := 10
// x += 5
type AssignmentStmt struct {
	Target     Expr
	Operator   string
	Value      Expr
	SourceSpan token.Span
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
// var x = 10
// varip x = 10
type VariableDeclStmt struct {
	Keyword    string
	Name       *Identifier
	Value      Expr
	SourceSpan token.Span
}

func (*VariableDeclStmt) node() {}
func (*VariableDeclStmt) stmt() {}

func (v *VariableDeclStmt) Span() token.Span {
	return v.SourceSpan
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
// else
//
//	...
type IfStmt struct {
	Condition  Expr
	Then       *BlockStmt
	Else       *BlockStmt
	SourceSpan token.Span
}

func (*IfStmt) node() {}
func (*IfStmt) stmt() {}

func (i *IfStmt) Span() token.Span {
	return i.SourceSpan
}

// ReturnStmt represents:
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
