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
