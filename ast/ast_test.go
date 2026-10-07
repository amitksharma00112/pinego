package ast

import (
	"testing"

	"github.com/amitksharma00112/pinego/token"
)

func TestProgramAST(t *testing.T) {
	program := &Program{
		Statements: []Stmt{
			&ExpressionStmt{
				Expression: &Identifier{
					Name: "close",
				},
			},
		},
	}

	if len(program.Statements) != 1 {
		t.Fatalf(
			"got %d statements, want 1",
			len(program.Statements),
		)
	}

	if program.Statements[0].Span() != (token.Span{}) {
		t.Fatalf(
			"unexpected span: %+v",
			program.Statements[0].Span(),
		)
	}
}

func TestExpressionNodes(t *testing.T) {
	var _ Expr = (*Identifier)(nil)
	var _ Expr = (*NumberLiteral)(nil)
	var _ Expr = (*StringLiteral)(nil)
	var _ Expr = (*BoolLiteral)(nil)
	var _ Expr = (*NALiteral)(nil)
	var _ Expr = (*ColorLiteral)(nil)
	var _ Expr = (*UnaryExpr)(nil)
	var _ Expr = (*BinaryExpr)(nil)
	var _ Expr = (*ConditionalExpr)(nil)
	var _ Expr = (*MemberExpr)(nil)
	var _ Expr = (*IndexExpr)(nil)
	var _ Expr = (*CallExpr)(nil)
	var _ Expr = (*ArrayExpr)(nil)
	var _ Expr = (*TupleExpr)(nil)
}

func TestStatementNodes(t *testing.T) {
	var _ Stmt = (*AssignmentStmt)(nil)
	var _ Stmt = (*ExpressionStmt)(nil)
	var _ Stmt = (*VariableDeclStmt)(nil)
	var _ Stmt = (*BlockStmt)(nil)
	var _ Stmt = (*IfStmt)(nil)
	var _ Stmt = (*ReturnStmt)(nil)
	var _ Stmt = (*BreakStmt)(nil)
	var _ Stmt = (*ContinueStmt)(nil)
	var _ Stmt = (*FunctionDeclStmt)(nil)
}

func TestParameterNode(t *testing.T) {
	param := &Parameter{
		TypeName: "float",
		Name: &Identifier{
			Name: "price",
		},
	}

	if param.TypeName != "float" {
		t.Fatalf("got %q, want float", param.TypeName)
	}

	if param.Name.Name != "price" {
		t.Fatalf("got %q, want price", param.Name.Name)
	}
}

func TestFunctionDeclNode(t *testing.T) {
	fn := &FunctionDeclStmt{
		Name: &Identifier{
			Name: "add",
		},
		Parameters: []*Parameter{
			{
				TypeName: "float",
				Name: &Identifier{
					Name: "x",
				},
			},
		},
	}

	if fn.Name.Name != "add" {
		t.Fatalf("got %q, want add", fn.Name.Name)
	}

	if len(fn.Parameters) != 1 {
		t.Fatalf("got %d parameters, want 1", len(fn.Parameters))
	}
}
