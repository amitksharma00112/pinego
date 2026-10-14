package ir

import "testing"

func TestValidateRejectsInvalidJumpTarget(t *testing.T) {
	program := &Program{
		GlobalBlocks: []Block{
			{
				ID: 0,
				Instructions: []Instruction{
					{Op: OpJump, Target: 99},
				},
			},
		},
	}

	if err := program.Validate(); err == nil {
		t.Fatal("expected error for invalid jump target")
	}
}
func TestValidateRejectsInvalidBranchTarget(t *testing.T) {
	program := &Program{
		GlobalBlocks: []Block{
			{
				ID: 0,
				Instructions: []Instruction{
					{Op: OpBranch, Targets: []int{1, 99}},
				},
			},
			{ID: 1},
		},
	}

	if err := program.Validate(); err == nil {
		t.Fatal("expected error for invalid branch target")
	}
}
func TestValidateRejectsDuplicateBlockIDs(t *testing.T) {
	program := &Program{
		GlobalBlocks: []Block{
			{ID: 0},
			{ID: 0},
		},
	}

	if err := program.Validate(); err == nil {
		t.Fatal("expected error for duplicate block IDs")
	}
}
func TestValidateRejectsInstructionAfterTerminator(t *testing.T) {
	program := &Program{
		GlobalBlocks: []Block{
			{
				ID: 0,
				Instructions: []Instruction{
					{Op: OpReturn},
					{Op: OpConst, Type: "number", Value: 1},
				},
			},
		},
	}

	if err := program.Validate(); err == nil {
		t.Fatal("expected error for instruction after terminator")
	}
}

func TestValidateRejectsBranchWithWrongTargetCount(t *testing.T) {
	program := &Program{
		GlobalBlocks: []Block{
			{
				ID: 0,
				Instructions: []Instruction{
					{Op: OpBranch, Targets: []int{1}},
				},
			},
			{ID: 1},
		},
	}

	if err := program.Validate(); err == nil {
		t.Fatal("expected error for branch with wrong target count")
	}
}
func TestValidateRejectsJumpWithBranchTargets(t *testing.T) {
	program := &Program{
		GlobalBlocks: []Block{
			{
				ID: 0,
				Instructions: []Instruction{
					{Op: OpJump, Target: 1, Targets: []int{1}},
				},
			},
			{ID: 1},
		},
	}

	if err := program.Validate(); err == nil {
		t.Fatal("expected error for jump with unexpected branch targets")
	}
}
func TestValidateRejectsReturnWithTargets(t *testing.T) {
	program := &Program{
		GlobalBlocks: []Block{
			{
				ID: 0,
				Instructions: []Instruction{
					{Op: OpReturn, Target: 1},
				},
			},
		},
	}

	if err := program.Validate(); err == nil {
		t.Fatal("expected error for return with unexpected target")
	}
}
func TestValidateRejectsNegativeBlockID(t *testing.T) {
	program := &Program{
		GlobalBlocks: []Block{
			{ID: -1},
		},
	}

	if err := program.Validate(); err == nil {
		t.Fatal("expected error for negative block ID")
	}
}
func TestValidateRejectsNonContiguousBlockIDs(t *testing.T) {
	program := &Program{
		GlobalBlocks: []Block{
			{ID: 0},
			{ID: 2},
		},
	}

	if err := program.Validate(); err == nil {
		t.Fatal("expected error for non-contiguous block IDs")
	}
}
func TestValidateRejectsFunctionWithoutBlocks(t *testing.T) {
	program := &Program{
		Functions: []Function{
			{Name: "main"},
		},
	}

	if err := program.Validate(); err == nil {
		t.Fatal("expected error for function without blocks")
	}
}
func TestValidateRejectsBranchWithoutCondition(t *testing.T) {
	program := &Program{
		GlobalBlocks: []Block{
			{
				ID: 0,
				Instructions: []Instruction{
					{
						Op:      OpBranch,
						Targets: []int{1, 2},
					},
				},
			},
			{ID: 1},
			{ID: 2},
		},
	}

	if err := program.Validate(); err == nil {
		t.Fatal("expected error for branch without condition")
	}
}
func TestValidateRejectsEmptyOperationInGlobalBlock(t *testing.T) {
	program := &Program{
		GlobalBlocks: []Block{
			{
				ID: 0,
				Instructions: []Instruction{
					{Op: ""},
				},
			},
		},
	}

	if err := program.Validate(); err == nil {
		t.Fatal("expected error for empty operation in global block")
	}
}
func TestValidateTrapTerminator(t *testing.T) {
	program := &Program{
		GlobalBlocks: []Block{
			{
				ID: 0,
				Instructions: []Instruction{
					{Op: OpTrap, Name: "for loop step cannot be zero"},
				},
			},
		},
	}

	if err := program.Validate(); err != nil {
		t.Fatalf("expected valid trap instruction, got: %v", err)
	}
}
