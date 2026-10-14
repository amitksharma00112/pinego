package ir

import (
	"strings"
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
func TestLowerForLoop(t *testing.T) {
	program := &ast.Program{
		Statements: []ast.Stmt{
			&ast.ForStmt{
				Variable: &ast.Identifier{Name: "i"},
				From:     &ast.NumberLiteral{Value: 0},
				To:       &ast.NumberLiteral{Value: 3},
				Body: &ast.BlockStmt{
					Statements: []ast.Stmt{
						&ast.ExpressionStmt{
							Expression: &ast.Identifier{Name: "i"},
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

	if len(got.GlobalBlocks) < 4 {
		t.Fatalf("expected loop CFG blocks, got %d", len(got.GlobalBlocks))
	}

	if err := got.Validate(); err != nil {
		t.Fatalf("invalid lowered IR: %v", err)
	}
}
func TestLowerForLoopNegativeStep(t *testing.T) {
	program := &ast.Program{
		Statements: []ast.Stmt{
			&ast.ForStmt{
				Variable: &ast.Identifier{Name: "i"},
				From:     &ast.NumberLiteral{Value: 3},
				To:       &ast.NumberLiteral{Value: 0},
				Step:     &ast.NumberLiteral{Value: -1},
				Body:     &ast.BlockStmt{},
			},
		},
	}

	got, err := Lower(program)
	if err != nil {
		t.Fatalf("Lower() error: %v", err)
	}

	var hasDescendingCondition bool
	for _, block := range got.GlobalBlocks {
		for _, inst := range block.Instructions {
			if inst.Op == OpBinary && inst.Operator == ">=" {
				hasDescendingCondition = true
			}
		}
	}

	if !hasDescendingCondition {
		t.Fatal("expected descending loop condition using >=")
	}
}

func TestLowerForLoopRejectsZeroStep(t *testing.T) {
	cfg := newCFGBuilder(&lowerer{nextID: 1})

	err := cfg.lowerFor(&ast.ForStmt{
		Variable: &ast.Identifier{Name: "i"},
		From:     &ast.NumberLiteral{Value: 0},
		To: &ast.CallExpr{
			Callee: &ast.Identifier{Name: "getTo"},
		},
		Step: &ast.NumberLiteral{Value: 0},
		Body: &ast.BlockStmt{},
	})

	if err == nil || !strings.Contains(err.Error(), "step cannot be zero") {
		t.Fatalf("expected zero-step error, got: %v", err)
	}

	for _, block := range cfg.blocks {
		for _, inst := range block.Instructions {
			if inst.Op == OpCall {
				t.Fatal("loop limit must not be evaluated for a zero-step loop")
			}
		}
	}
}

func TestLowerForInLoopRejectsUnsupportedForm(t *testing.T) {
	program := &ast.Program{
		Statements: []ast.Stmt{
			&ast.ForStmt{
				Variable: &ast.Identifier{Name: "item"},
				Iterable: &ast.Identifier{Name: "items"},
				IsInLoop: true,
				Body:     &ast.BlockStmt{},
			},
		},
	}

	got, err := Lower(program)
	if err == nil {
		t.Fatal("expected unsupported iterable loop error")
	}
	if !strings.Contains(err.Error(), "iterable") {
		t.Fatalf("expected iterable-loop error, got: %v", err)
	}
	_ = got
}
func TestLowerBreakInForLoop(t *testing.T) {
	program := &ast.Program{
		Statements: []ast.Stmt{
			&ast.ForStmt{
				Variable: &ast.Identifier{Name: "i"},
				From:     &ast.NumberLiteral{Value: 0},
				To:       &ast.NumberLiteral{Value: 3},
				Body: &ast.BlockStmt{
					Statements: []ast.Stmt{
						&ast.BreakStmt{},
					},
				},
			},
		},
	}

	if _, err := Lower(program); err != nil {
		t.Fatalf("Lower() error: %v", err)
	}
}
func TestLowerContinueInForLoop(t *testing.T) {
	program := &ast.Program{
		Statements: []ast.Stmt{
			&ast.ForStmt{
				Variable: &ast.Identifier{Name: "i"},
				From:     &ast.NumberLiteral{Value: 0},
				To:       &ast.NumberLiteral{Value: 3},
				Body: &ast.BlockStmt{
					Statements: []ast.Stmt{
						&ast.ContinueStmt{},
					},
				},
			},
		},
	}

	if _, err := Lower(program); err != nil {
		t.Fatalf("Lower() error: %v", err)
	}
}
func TestLowerBreakOutsideLoop(t *testing.T) {
	program := &ast.Program{
		Statements: []ast.Stmt{
			&ast.BreakStmt{},
		},
	}

	if _, err := Lower(program); err == nil || !strings.Contains(err.Error(), "break outside loop") {
		t.Fatalf("expected 'break outside loop' error, got: %v", err)
	}
}

func TestLowerContinueOutsideLoop(t *testing.T) {
	program := &ast.Program{
		Statements: []ast.Stmt{
			&ast.ContinueStmt{},
		},
	}

	if _, err := Lower(program); err == nil || !strings.Contains(err.Error(), "continue outside loop") {
		t.Fatalf("expected 'continue outside loop' error, got: %v", err)
	}
}
func TestLowerBreakInWhileLoop(t *testing.T) {
	program := &ast.Program{
		Statements: []ast.Stmt{
			&ast.WhileStmt{
				Condition: &ast.BoolLiteral{Value: true},
				Body: &ast.BlockStmt{
					Statements: []ast.Stmt{
						&ast.BreakStmt{},
					},
				},
			},
		},
	}

	if _, err := Lower(program); err != nil {
		t.Fatalf("Lower() error: %v", err)
	}
}

func TestLowerContinueInWhileLoop(t *testing.T) {
	program := &ast.Program{
		Statements: []ast.Stmt{
			&ast.WhileStmt{
				Condition: &ast.BoolLiteral{Value: true},
				Body: &ast.BlockStmt{
					Statements: []ast.Stmt{
						&ast.ContinueStmt{},
					},
				},
			},
		},
	}

	if _, err := Lower(program); err != nil {
		t.Fatalf("Lower() error: %v", err)
	}
}
func TestLowerNestedForLoopBreak(t *testing.T) {
	program := &ast.Program{
		Statements: []ast.Stmt{
			&ast.ForStmt{
				Variable: &ast.Identifier{Name: "i"},
				From:     &ast.NumberLiteral{Value: 0},
				To:       &ast.NumberLiteral{Value: 3},
				Body: &ast.BlockStmt{
					Statements: []ast.Stmt{
						&ast.ForStmt{
							Variable: &ast.Identifier{Name: "j"},
							From:     &ast.NumberLiteral{Value: 0},
							To:       &ast.NumberLiteral{Value: 2},
							Body: &ast.BlockStmt{
								Statements: []ast.Stmt{
									&ast.BreakStmt{},
								},
							},
						},
					},
				},
			},
		},
	}

	if _, err := Lower(program); err != nil {
		t.Fatalf("Lower() error: %v", err)
	}
}

func TestLowerNestedForLoopContinue(t *testing.T) {
	program := &ast.Program{
		Statements: []ast.Stmt{
			&ast.ForStmt{
				Variable: &ast.Identifier{Name: "i"},
				From:     &ast.NumberLiteral{Value: 0},
				To:       &ast.NumberLiteral{Value: 3},
				Body: &ast.BlockStmt{
					Statements: []ast.Stmt{
						&ast.ForStmt{
							Variable: &ast.Identifier{Name: "j"},
							From:     &ast.NumberLiteral{Value: 0},
							To:       &ast.NumberLiteral{Value: 2},
							Body: &ast.BlockStmt{
								Statements: []ast.Stmt{
									&ast.ContinueStmt{},
								},
							},
						},
					},
				},
			},
		},
	}

	result, err := Lower(program)
	if err != nil {
		t.Fatalf("Lower() error: %v", err)
	}
	if err := result.Validate(); err != nil {
		t.Fatalf("invalid IR: %v", err)
	}

	// Find the inner loop's increment block.
	innerIncrementID := -1
	for _, block := range result.GlobalBlocks {
		loadJ, add, storeJ := false, false, false

		for _, inst := range block.Instructions {
			switch {
			case inst.Op == OpLoad && inst.Name == "j":
				loadJ = true
			case inst.Op == OpBinary && inst.Operator == "+":
				add = true
			case inst.Op == OpStore && inst.Name == "j":
				storeJ = true
			}
		}

		if loadJ && add && storeJ {
			innerIncrementID = block.ID
			break
		}
	}

	if innerIncrementID == -1 {
		t.Fatal("inner loop increment block not found")
	}

	// Verify that continue jumps directly to that block.
	continueTargetsIncrement := false
	for _, block := range result.GlobalBlocks {
		for _, inst := range block.Instructions {
			if inst.Op == OpJump && inst.Target == innerIncrementID {
				continueTargetsIncrement = true
			}
		}
	}

	if !continueTargetsIncrement {
		t.Fatalf(
			"inner continue should jump to increment block %d",
			innerIncrementID,
		)
	}
}

func TestLowerForLoopNegativeVariableStep(t *testing.T) {
	program := &ast.Program{
		Statements: []ast.Stmt{
			&ast.VariableDeclStmt{
				Name:  &ast.Identifier{Name: "step"},
				Value: &ast.NumberLiteral{Value: -1},
			},
			&ast.ForStmt{
				Variable: &ast.Identifier{Name: "i"},
				From:     &ast.NumberLiteral{Value: 3},
				To:       &ast.NumberLiteral{Value: 0},
				Step:     &ast.Identifier{Name: "step"},
				Body:     &ast.BlockStmt{},
			},
		},
	}

	got, err := Lower(program)
	if err != nil {
		t.Fatalf("Lower() error: %v", err)
	}

	var hasAscending, hasDescending bool
	for _, block := range got.GlobalBlocks {
		for _, inst := range block.Instructions {
			if inst.Op == OpBinary {
				switch inst.Operator {
				case "<=":
					hasAscending = true
				case ">=":
					hasDescending = true
				}
			}
		}
	}

	if !hasAscending || !hasDescending {
		t.Fatalf(
			"expected runtime conditions for both step directions; ascending=%v descending=%v",
			hasAscending, hasDescending,
		)
	}
}
func TestLowerForLoopLiteralStepProducesValidIR(t *testing.T) {
	program := &ast.Program{
		Statements: []ast.Stmt{
			&ast.ForStmt{
				Variable: &ast.Identifier{Name: "i"},
				From:     &ast.NumberLiteral{Value: 0},
				To:       &ast.NumberLiteral{Value: 3},
				Step:     &ast.NumberLiteral{Value: 1},
				Body:     &ast.BlockStmt{},
			},
		},
	}

	got, err := Lower(program)
	if err != nil {
		t.Fatalf("Lower() error: %v", err)
	}

	if err := got.Validate(); err != nil {
		t.Fatalf("invalid lowered IR: %v", err)
	}
}
func TestLowerForLoopEvaluatesToAndStepBeforeLoop(t *testing.T) {
	program := &ast.Program{
		Statements: []ast.Stmt{
			&ast.ForStmt{
				Variable: &ast.Identifier{Name: "i"},
				From:     &ast.NumberLiteral{Value: 0},
				To: &ast.CallExpr{
					Callee: &ast.Identifier{Name: "getTo"},
				},
				Step: &ast.CallExpr{
					Callee: &ast.Identifier{Name: "getStep"},
				},
				Body: &ast.BlockStmt{},
			},
		},
	}

	got, err := Lower(program)
	if err != nil {
		t.Fatalf("Lower() error: %v", err)
	}

	var guardBlock *Block
	var stepCallBlock *Block
	var toCallBlock *Block

	for i := range got.GlobalBlocks {
		block := &got.GlobalBlocks[i]

		for _, inst := range block.Instructions {
			if inst.Op == OpLoad {
				switch inst.Name {
				case "getStep":
					stepCallBlock = block
				case "getTo":
					toCallBlock = block
				}
			}

			if inst.Op == OpBinary && inst.Operator == "==" {
				guardBlock = block
			}
		}
	}

	if stepCallBlock == nil {
		t.Fatal("expected getStep() to be evaluated")
	}
	if toCallBlock == nil {
		t.Fatal("expected getTo() to be evaluated")
	}
	if guardBlock == nil {
		t.Fatal("expected zero-step guard")
	}

	if stepCallBlock.ID != guardBlock.ID {
		t.Fatal("getStep() must be evaluated before the zero-step guard")
	}

	if len(guardBlock.Instructions) == 0 {
		t.Fatal("zero-step guard block is empty")
	}

	last := guardBlock.Instructions[len(guardBlock.Instructions)-1]
	if last.Op != OpBranch || len(last.Targets) != 2 {
		t.Fatal("zero-step guard must end with a two-target branch")
	}

	if toCallBlock.ID != last.Targets[1] {
		t.Fatal("getTo() must be evaluated only on the non-zero-step path")
	}

}
func TestLowerForLoopGuardsDynamicZeroStep(t *testing.T) {
	program := &ast.Program{
		Statements: []ast.Stmt{
			&ast.ForStmt{
				Variable: &ast.Identifier{Name: "i"},
				From:     &ast.NumberLiteral{Value: 0},
				To: &ast.CallExpr{
					Callee: &ast.Identifier{Name: "getTo"},
				},
				Step: &ast.Identifier{Name: "step"},
				Body: &ast.BlockStmt{},
			},
		},
	}

	got, err := Lower(program)
	if err != nil {
		t.Fatalf("unexpected lowering error: %v", err)
	}

	blocks := make(map[int]Block)
	for _, block := range got.GlobalBlocks {
		blocks[block.ID] = block
	}
	var guardBlock *Block
	for i := range got.GlobalBlocks {
		block := &got.GlobalBlocks[i]
		for _, inst := range block.Instructions {
			if inst.Op == OpBinary && inst.Operator == "==" {
				guardBlock = block
				break
			}
		}
		if guardBlock != nil {
			break
		}
	}

	if guardBlock == nil {
		t.Fatal("expected zero-step guard")
	}

	var branch *Instruction
	for i := range guardBlock.Instructions {
		if guardBlock.Instructions[i].Op == OpBranch {
			branch = &guardBlock.Instructions[i]
		}
	}
	if branch == nil || len(branch.Targets) != 2 {
		t.Fatal("zero-step guard must end with a two-target branch")
	}

	nonZeroBlock, ok := blocks[branch.Targets[1]]
	if !ok {
		t.Fatalf("missing non-zero-step block %d", branch.Targets[1])
	}

	toCallFound := false
	for _, inst := range nonZeroBlock.Instructions {
		if inst.Op == OpLoad && inst.Name == "getTo" {
			toCallFound = true
		}
	}
	if !toCallFound {
		t.Fatal("expected getTo to be evaluated only after the zero-step guard")
	}
	guardFound := false
	for _, block := range got.GlobalBlocks {
		for _, inst := range block.Instructions {
			if inst.Op != OpBinary || inst.Operator != "==" {
				continue
			}

			guardFound = true
			if len(block.Instructions) == 0 {
				t.Fatal("zero-step guard block is empty")
			}

			last := block.Instructions[len(block.Instructions)-1]
			if last.Op != OpBranch || len(last.Targets) != 2 {
				t.Fatal("zero-step guard must end with a two-target branch")
			}

			trapBlock, ok := blocks[last.Targets[0]]
			if !ok {
				t.Fatalf("missing trap block %d", last.Targets[0])
			}

			if len(trapBlock.Instructions) != 1 ||
				trapBlock.Instructions[0].Op != OpTrap {
				t.Fatal("zero-step true branch must target an OpTrap block")
			}
		}
	}

	if !guardFound {
		t.Fatal("expected runtime guard to detect a zero loop step")
	}
}
