package ir

import "testing"

func TestValidateValidProgram(t *testing.T) {
	p := &Program{
		Functions: []Function{
			{
				Name: "main",
				Blocks: []Block{
					{
						ID: 0,
						Instructions: []Instruction{
							{Result: 1, Op: OpConst, Value: 10},
						},
					},
				},
			},
		},
	}

	if err := p.Validate(); err != nil {
		t.Fatalf("expected valid IR, got error: %v", err)
	}
}

func TestValidateEmptyFunctionName(t *testing.T) {
	p := &Program{
		Functions: []Function{{Name: ""}},
	}

	if err := p.Validate(); err == nil {
		t.Fatal("expected error for empty function name")
	}
}

func TestValidateEmptyOperation(t *testing.T) {
	p := &Program{
		Globals: []Instruction{{}},
	}

	if err := p.Validate(); err == nil {
		t.Fatal("expected error for empty operation")
	}
}

func TestValidateNilProgram(t *testing.T) {
	var p *Program

	if err := p.Validate(); err == nil {
		t.Fatal("expected error for nil program")
	}
}
func TestValidatePhiIncomingBlocks(t *testing.T) {
	program := &Program{
		GlobalBlocks: []Block{
			{
				ID: 0,
				Instructions: []Instruction{
					{Op: OpBranch, Args: []ValueID{1}, Targets: []int{1, 2}},
				},
			},
			{
				ID: 1,
				Instructions: []Instruction{
					{Op: OpJump, Target: 3},
				},
			},
			{
				ID: 2,
				Instructions: []Instruction{
					{Op: OpJump, Target: 3},
				},
			},
			{
				ID: 3,
				Instructions: []Instruction{
					{
						Op: OpPhi, Result: 4,
						Args:           []ValueID{2, 3},
						IncomingBlocks: []int{1, 2},
					},
				},
			},
		},
	}

	if err := program.Validate(); err != nil {
		t.Fatalf("expected valid Phi incoming blocks: %v", err)
	}
}
func TestValidateRejectsPhiIncomingBlockMismatch(t *testing.T) {
	program := &Program{
		GlobalBlocks: []Block{
			{
				ID: 0,
				Instructions: []Instruction{
					{
						Op:             OpPhi,
						Result:         1,
						Args:           []ValueID{10, 20},
						IncomingBlocks: []int{0},
					},
				},
			},
		},
	}

	if err := program.Validate(); err == nil {
		t.Fatal("expected validation error for mismatched Phi inputs")
	}
}
func TestValidateRejectsNonPredecessorPhiBlock(t *testing.T) {
	program := &Program{
		GlobalBlocks: []Block{
			{ID: 0, Instructions: []Instruction{
				{Op: OpBranch, Args: []ValueID{1}, Targets: []int{1, 2}},
			}},
			{ID: 1, Instructions: []Instruction{
				{Op: OpJump, Target: 3},
			}},
			{ID: 2, Instructions: []Instruction{
				{Op: OpJump, Target: 3},
			}},
			{ID: 3, Instructions: []Instruction{
				{
					Op: OpPhi, Result: 4,
					Args:           []ValueID{10, 20},
					IncomingBlocks: []int{0, 2},
				},
			}},
		},
	}

	if err := program.Validate(); err == nil {
		t.Fatal("expected error because block 0 is not a predecessor of block 3")
	}
}
