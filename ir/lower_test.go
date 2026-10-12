package ir

import (
	"testing"

	"github.com/amitksharma00112/pinego/ast"
)

func TestLowerBinaryExpression(t *testing.T) {
	expr := &ast.BinaryExpr{
		Left:     &ast.NumberLiteral{Value: 10},
		Operator: "+",
		Right:    &ast.NumberLiteral{Value: 20},
	}

	program := &ast.Program{
		Statements: []ast.Stmt{
			&ast.ExpressionStmt{Expression: expr},
		},
	}

	got, err := Lower(program)
	if err != nil {
		t.Fatalf("Lower() error: %v", err)
	}

	if len(got.Globals) != 3 {
		t.Fatalf("expected 3 instructions, got %d", len(got.Globals))
	}

	binary := got.Globals[2]
	if binary.Op != OpBinary {
		t.Fatalf("expected OpBinary, got %q", binary.Op)
	}
	if binary.Operator != "+" {
		t.Fatalf("expected + operator, got %q", binary.Operator)
	}
	if len(binary.Args) != 2 {
		t.Fatalf("expected 2 arguments, got %d", len(binary.Args))
	}
}

func TestLowerVariableDeclaration(t *testing.T) {
	program := &ast.Program{
		Statements: []ast.Stmt{
			&ast.VariableDeclStmt{
				Name:  &ast.Identifier{Name: "x"},
				Value: &ast.NumberLiteral{Value: 42},
			},
		},
	}

	got, err := Lower(program)
	if err != nil {
		t.Fatalf("Lower() error: %v", err)
	}

	if len(got.Globals) != 2 {
		t.Fatalf("expected 2 instructions, got %d", len(got.Globals))
	}

	store := got.Globals[1]
	if store.Op != OpStore {
		t.Fatalf("expected OpStore, got %q", store.Op)
	}
	if store.Name != "x" {
		t.Fatalf("expected variable x, got %q", store.Name)
	}
	if len(store.Args) != 1 {
		t.Fatalf("expected 1 value argument, got %d", len(store.Args))
	}
}

func TestLowerIdentifier(t *testing.T) {
	program := &ast.Program{
		Statements: []ast.Stmt{
			&ast.ExpressionStmt{
				Expression: &ast.Identifier{Name: "close"},
			},
		},
	}

	got, err := Lower(program)
	if err != nil {
		t.Fatalf("Lower() error: %v", err)
	}

	if len(got.Globals) != 1 {
		t.Fatalf("expected 1 instruction, got %d", len(got.Globals))
	}

	inst := got.Globals[0]
	if inst.Op != OpLoad || inst.Name != "close" {
		t.Fatalf("unexpected instruction: %+v", inst)
	}
}
