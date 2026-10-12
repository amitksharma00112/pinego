package ir

import "fmt"

// ValueID identifies a value produced by an IR instruction.
type ValueID int

// Op represents an IR operation.
type Op string

const (
	OpConst  Op = "const"
	OpLoad   Op = "load"
	OpStore  Op = "store"
	OpUnary  Op = "unary"
	OpBinary Op = "binary"
	OpCall   Op = "call"
	OpIndex  Op = "index"
	OpMember Op = "member"
	OpPhi    Op = "phi"
	OpBranch Op = "branch"
	OpJump   Op = "jump"
	OpReturn Op = "return"
)

// Instruction represents one IR operation.
type Instruction struct {
	Result   ValueID
	Op       Op
	Type     string
	Args     []ValueID
	Name     string
	Operator string
	Value    any
	Target   int
	Targets  []int
}

// Block groups a sequence of IR instructions.
type Block struct {
	ID           int
	Instructions []Instruction
}

// Function represents an IR function.
type Function struct {
	Name   string
	Params []string
	Blocks []Block
}

// Program is the root of the intermediate representation.
type Program struct {
	Functions    []Function
	Globals      []Instruction
	GlobalBlocks []Block
}

// Validate checks basic IR invariants.
func (p *Program) Validate() error {
	if p == nil {
		return fmt.Errorf("cannot validate nil IR program")
	}

	for _, fn := range p.Functions {
		if fn.Name == "" {
			return fmt.Errorf("IR function has empty name")
		}

		for _, block := range fn.Blocks {
			for _, inst := range block.Instructions {
				if inst.Op == "" {
					return fmt.Errorf(
						"function %q block %d has instruction with empty operation",
						fn.Name,
						block.ID,
					)
				}
			}
		}
	}

	for _, inst := range p.Globals {
		if inst.Op == "" {
			return fmt.Errorf("global IR instruction has empty operation")
		}
	}

	return nil
}
