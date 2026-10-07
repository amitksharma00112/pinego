package ast

import "github.com/amitksharma00112/pinego/token"

// Node is the base interface for every AST node.
type Node interface {
	node()
	Span() token.Span
}

// Expr is implemented by every expression node.
type Expr interface {
	Node
	expr()
}

// Stmt is implemented by every statement node.
type Stmt interface {
	Node
	stmt()
}

// Program is the root of the PineGo AST.
type Program struct {
	Statements []Stmt
	SourceSpan token.Span
}

func (*Program) node() {}

func (p *Program) Span() token.Span {
	return p.SourceSpan
}
