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
	OpTrap   Op = "trap"
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
	validateBlocks := func(scope string, blocks []Block) error {
		blockIDs := make(map[int]bool)

		for _, block := range blocks {
			if block.ID < 0 {
				return fmt.Errorf(
					"%s has negative block ID %d",
					scope, block.ID,
				)
			}
			if blockIDs[block.ID] {
				return fmt.Errorf("%s has duplicate block ID %d", scope, block.ID)
			}
			blockIDs[block.ID] = true
		}
		for id := 0; id < len(blocks); id++ {
			if !blockIDs[id] {
				return fmt.Errorf("%s has non-contiguous block IDs: missing ID %d", scope, id)
			}
		}

		for _, block := range blocks {
			for i, inst := range block.Instructions {
				switch inst.Op {

				case OpBranch, OpJump, OpReturn, OpTrap:
					if i != len(block.Instructions)-1 {
						return fmt.Errorf(
							"%s block %d has instruction after terminator %q",
							scope, block.ID, inst.Op,
						)
					}
				}
				if inst.Op == "" {
					return fmt.Errorf(
						"%s block %d has instruction with empty operation",
						scope, block.ID,
					)
				}
				switch inst.Op {
				case OpJump:
					if len(inst.Targets) != 0 {
						return fmt.Errorf(
							"%s block %d jump must not have branch targets",
							scope, block.ID,
						)
					}
					if !blockIDs[inst.Target] {
						return fmt.Errorf(
							"%s block %d jumps to missing block %d",
							scope, block.ID, inst.Target,
						)
					}
				case OpReturn:
					if inst.Target != 0 || len(inst.Targets) != 0 {
						return fmt.Errorf(
							"%s block %d return must not have jump targets",
							scope, block.ID,
						)
					}
				case OpTrap:
					if inst.Target != 0 || len(inst.Targets) != 0 || len(inst.Args) != 0 {
						return fmt.Errorf(
							"%s block %d trap must not have arguments or jump targets",
							scope, block.ID,
						)
					}
				case OpBranch:
					if len(inst.Args) != 1 {
						return fmt.Errorf(
							"%s block %d branch requires exactly one condition",
							scope, block.ID,
						)
					}
					if len(inst.Targets) != 2 {
						return fmt.Errorf(
							"%s block %d branch requires exactly 2 targets, got %d",
							scope, block.ID, len(inst.Targets),
						)
					}
					for _, target := range inst.Targets {
						if !blockIDs[target] {
							return fmt.Errorf(
								"%s block %d branches to missing block %d",
								scope, block.ID, target,
							)
						}
					}
				}
			}
		}

		return nil
	}
	for _, fn := range p.Functions {
		if fn.Name == "" {
			return fmt.Errorf("IR function has empty name")
		}
		if len(fn.Blocks) == 0 {
			return fmt.Errorf("function %q has no basic blocks", fn.Name)
		}

	}

	for _, inst := range p.Globals {
		if inst.Op == "" {
			return fmt.Errorf("global IR instruction has empty operation")
		}
	}
	for _, fn := range p.Functions {
		if err := validateBlocks(
			fmt.Sprintf("function %q", fn.Name), fn.Blocks,
		); err != nil {
			return err
		}
	}

	if err := validateBlocks("global", p.GlobalBlocks); err != nil {
		return err
	}
	return nil
}
