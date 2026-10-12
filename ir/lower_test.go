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
func TestLowerIfElseCreatesCFG(t *testing.T) {
	program := &ast.Program{
		Statements: []ast.Stmt{
			&ast.IfStmt{
				Condition: &ast.BoolLiteral{Value: true},
				Then: &ast.BlockStmt{
					Statements: []ast.Stmt{
						&ast.ExpressionStmt{
							Expression: &ast.NumberLiteral{Value: 1},
						},
					},
				},
				Else: &ast.BlockStmt{
					Statements: []ast.Stmt{
						&ast.ExpressionStmt{
							Expression: &ast.NumberLiteral{Value: 2},
						},
					},
				},
			},
		},
	}

	got, err := Lower(program)
	if err != nil {
		t.Fatalf("Lower() error: %v", err)
	}

	if len(got.GlobalBlocks) != 4 {
		t.Fatalf("expected 4 CFG blocks, got %d", len(got.GlobalBlocks))
	}

	branch := got.GlobalBlocks[0].Instructions
	if branch[len(branch)-1].Op != OpBranch {
		t.Fatalf("expected OpBranch, got %q", branch[len(branch)-1].Op)
	}

	if len(branch[len(branch)-1].Targets) != 2 {
		t.Fatal("expected branch to have then and else targets")
	}
	thenID := branch[len(branch)-1].Targets[0]
	elseID := branch[len(branch)-1].Targets[1]

	if thenID == elseID {
		t.Fatal("then and else must target different blocks")
	}

	if got.GlobalBlocks[thenID].Instructions[0].Op != OpConst {
		t.Fatal("then block should contain its expression")
	}

	if got.GlobalBlocks[elseID].Instructions[0].Op != OpConst {
		t.Fatal("else block should contain its expression")
	}
}
func TestLowerReturnStatement(t *testing.T) {
	program := &ast.Program{
		Statements: []ast.Stmt{
			&ast.FunctionDeclStmt{
				Name: &ast.Identifier{Name: "test"},
				Body: &ast.BlockStmt{
					Statements: []ast.Stmt{
						&ast.ReturnStmt{
							Value: &ast.NumberLiteral{Value: 42},
						},
					},
				},
			},
		},
	}

	got, err := Lower(program)
	if err != nil {
		t.Fatalf("Lower() error: %v", err)
	}

	blocks := got.Functions[0].Blocks
	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}

	instructions := blocks[0].Instructions
	last := instructions[len(instructions)-1]

	if last.Op != OpReturn {
		t.Fatalf("expected OpReturn, got %q", last.Op)
	}

	if len(last.Args) != 1 {
		t.Fatalf("expected 1 return value, got %d", len(last.Args))
	}

}
func TestLowerRejectsStatementAfterReturn(t *testing.T) {
	program := &ast.Program{
		Statements: []ast.Stmt{
			&ast.FunctionDeclStmt{
				Name: &ast.Identifier{Name: "test"},
				Body: &ast.BlockStmt{
					Statements: []ast.Stmt{
						&ast.ReturnStmt{
							Value: &ast.NumberLiteral{Value: 42},
						},
						&ast.ExpressionStmt{
							Expression: &ast.NumberLiteral{Value: 10},
						},
					},
				},
			},
		},
	}

	_, err := Lower(program)
	if err == nil {
		t.Fatal("expected error for statement after return")
	}
}
func TestLowerWhileLoop(t *testing.T) {
	program := &ast.Program{
		Statements: []ast.Stmt{
			&ast.WhileStmt{
				Condition: &ast.BoolLiteral{Value: true},
				Body: &ast.BlockStmt{
					Statements: []ast.Stmt{
						&ast.ExpressionStmt{
							Expression: &ast.NumberLiteral{Value: 1},
						},
					},
				},
			},
		},
	}

	got, err := Lower(program)
	if err != nil {
		t.Fatalf("Lower() error: %v", err)
	}

	blocks := got.GlobalBlocks

	if len(blocks) != 4 {
		t.Fatalf("expected 4 CFG blocks, got %d", len(blocks))
	}

	if blocks[0].Instructions[len(blocks[0].Instructions)-1].Op != OpJump {
		t.Fatal("entry block should jump to condition block")
	}

	if blocks[1].Instructions[len(blocks[1].Instructions)-1].Op != OpBranch {
		t.Fatal("condition block should branch to body or exit")
	}

	if blocks[2].Instructions[len(blocks[2].Instructions)-1].Op != OpJump {
		t.Fatal("loop body should jump back to condition block")
	}
}
