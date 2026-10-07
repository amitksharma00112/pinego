package ast

import "github.com/amitksharma00112/pinego/token"

// Identifier represents a variable, function, namespace,
// type, enum, or field name.
type Identifier struct {
	Name       string
	SourceSpan token.Span
}

func (*Identifier) node() {}
func (*Identifier) expr() {}

func (i *Identifier) Span() token.Span {
	return i.SourceSpan
}

// NumberLiteral represents a numeric literal.
type NumberLiteral struct {
	Value      float64
	Raw        string
	SourceSpan token.Span
}

func (*NumberLiteral) node() {}
func (*NumberLiteral) expr() {}

func (n *NumberLiteral) Span() token.Span {
	return n.SourceSpan
}

// StringLiteral represents a string literal.
type StringLiteral struct {
	Value      string
	Raw        string
	SourceSpan token.Span
}

func (*StringLiteral) node() {}
func (*StringLiteral) expr() {}

func (s *StringLiteral) Span() token.Span {
	return s.SourceSpan
}

// BoolLiteral represents true / false.
type BoolLiteral struct {
	Value      bool
	SourceSpan token.Span
}

func (*BoolLiteral) node() {}
func (*BoolLiteral) expr() {}

func (b *BoolLiteral) Span() token.Span {
	return b.SourceSpan
}

// NALiteral represents Pine's na value.
type NALiteral struct {
	SourceSpan token.Span
}

func (*NALiteral) node() {}
func (*NALiteral) expr() {}

func (n *NALiteral) Span() token.Span {
	return n.SourceSpan
}

// ColorLiteral represents a color literal such as #FF0000.
type ColorLiteral struct {
	Value      string
	SourceSpan token.Span
}

func (*ColorLiteral) node() {}
func (*ColorLiteral) expr() {}

func (c *ColorLiteral) Span() token.Span {
	return c.SourceSpan
}

// UnaryExpr represents:
// -x
// +x
// not x
type UnaryExpr struct {
	Operator   string
	Operand    Expr
	SourceSpan token.Span
}

func (*UnaryExpr) node() {}
func (*UnaryExpr) expr() {}

func (u *UnaryExpr) Span() token.Span {
	return u.SourceSpan
}

// BinaryExpr represents:
// a + b
// a * b
// a > b
// a and b
type BinaryExpr struct {
	Left       Expr
	Operator   string
	Right      Expr
	SourceSpan token.Span
}

func (*BinaryExpr) node() {}
func (*BinaryExpr) expr() {}

func (b *BinaryExpr) Span() token.Span {
	return b.SourceSpan
}

// ConditionalExpr represents Pine's ternary:
// condition ? a : b
type ConditionalExpr struct {
	Condition  Expr
	Then       Expr
	Else       Expr
	SourceSpan token.Span
}

func (*ConditionalExpr) node() {}
func (*ConditionalExpr) expr() {}

func (c *ConditionalExpr) Span() token.Span {
	return c.SourceSpan
}

// MemberExpr represents:
// ta.sma
// obj.field
type MemberExpr struct {
	Object     Expr
	Member     string
	SourceSpan token.Span
}

func (*MemberExpr) node() {}
func (*MemberExpr) expr() {}

func (m *MemberExpr) Span() token.Span {
	return m.SourceSpan
}

// IndexExpr represents:
// close[1]
// high[offset]
type IndexExpr struct {
	Object     Expr
	Index      Expr
	SourceSpan token.Span
}

func (*IndexExpr) node() {}
func (*IndexExpr) expr() {}

func (i *IndexExpr) Span() token.Span {
	return i.SourceSpan
}

// CallExpr represents:
// foo(a, b)
// ta.sma(close, 20)
type CallExpr struct {
	Callee     Expr
	Arguments  []Expr
	SourceSpan token.Span
}

func (*CallExpr) node() {}
func (*CallExpr) expr() {}

func (c *CallExpr) Span() token.Span {
	return c.SourceSpan
}

// ArrayExpr represents a bracketed collection.
type ArrayExpr struct {
	Elements   []Expr
	SourceSpan token.Span
}

func (*ArrayExpr) node() {}
func (*ArrayExpr) expr() {}

func (a *ArrayExpr) Span() token.Span {
	return a.SourceSpan
}

// TupleExpr represents a tuple expression.
type TupleExpr struct {
	Elements   []Expr
	SourceSpan token.Span
}

func (*TupleExpr) node() {}
func (*TupleExpr) expr() {}

func (t *TupleExpr) Span() token.Span {
	return t.SourceSpan
}
