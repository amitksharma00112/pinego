package ir

import (
	"fmt"

	"github.com/amitksharma00112/pinego/ast"
)

// Lower converts a validated AST into basic IR.
func Lower(program *ast.Program) (*Program, error) {
	if program == nil {
		return nil, fmt.Errorf("cannot lower nil AST program")
	}

	result := &Program{}
	l := &lowerer{nextID: 1}
	var globalCFG *cfgBuilder
	for _, stmt := range program.Statements {
		switch s := stmt.(type) {
		case *ast.FunctionDeclStmt:
			if s.Name == nil || s.Body == nil {
				return nil, fmt.Errorf("invalid function declaration")
			}

			fn := Function{Name: s.Name.Name}
			for _, param := range s.Parameters {
				if param == nil || param.Name == nil {
					return nil, fmt.Errorf(
						"function %q has invalid parameter",
						fn.Name,
					)
				}
				fn.Params = append(fn.Params, param.Name.Name)
			}

			cfg := newCFGBuilder(l)

			if err := cfg.lowerStmt(s.Body); err != nil {
				return nil, fmt.Errorf("function %q: %w", fn.Name, err)
			}

			fn.Blocks = cfg.blocks
			result.Functions = append(result.Functions, fn)

		default:
			if globalCFG == nil {
				switch stmt.(type) {
				case *ast.IfStmt, *ast.WhileStmt:
					// CFG required
				default:
					if err := l.lowerStmt(stmt, &result.Globals); err != nil {
						return nil, fmt.Errorf("global statement: %w", err)
					}
					continue
				}

				globalCFG = &cfgBuilder{
					lowerer: l,
					blocks: []Block{{
						ID:           0,
						Instructions: result.Globals,
					}},
					current: 0,
				}
				result.Globals = nil
			}

			if globalCFG.terminated() {
				globalCFG.newBlock()
			}

			if err := globalCFG.lowerStmt(stmt); err != nil {
				return nil, fmt.Errorf("global statement: %w", err)
			}
		}
	}
	if globalCFG != nil {
		result.GlobalBlocks = globalCFG.blocks
	}
	if err := result.Validate(); err != nil {
		return nil, err
	}
	return result, nil
}

type lowerer struct {
	nextID ValueID
}

type cfgBuilder struct {
	lowerer *lowerer
	blocks  []Block
	current int
}

func newCFGBuilder(l *lowerer) *cfgBuilder {
	return &cfgBuilder{
		lowerer: l,
		blocks:  []Block{{ID: 0}},
		current: 0,
	}
}

func (c *cfgBuilder) newBlock() int {
	id := len(c.blocks)
	c.blocks = append(c.blocks, Block{ID: id})
	return id
}

func (c *cfgBuilder) emit(inst Instruction) {
	c.blocks[c.current].Instructions = append(
		c.blocks[c.current].Instructions, inst,
	)
}

func (c *cfgBuilder) terminated() bool {
	instructions := c.blocks[c.current].Instructions
	if len(instructions) == 0 {
		return false
	}

	switch instructions[len(instructions)-1].Op {
	case OpBranch, OpJump, OpReturn:
		return true
	default:
		return false
	}
}

func (c *cfgBuilder) jump(target int) {
	c.emit(Instruction{
		Op:     OpJump,
		Target: target,
	})
}

func (c *cfgBuilder) lowerIf(s *ast.IfStmt) error {
	if s.Condition == nil || s.Then == nil {
		return fmt.Errorf("if statement requires condition and then block")
	}

	condition, err := c.lowerer.lowerExpr(
		s.Condition,
		&c.blocks[c.current].Instructions,
	)
	if err != nil {
		return err
	}

	thenID := c.newBlock()
	elseID := c.newBlock()
	mergeID := c.newBlock()

	c.emit(Instruction{
		Op:      OpBranch,
		Args:    []ValueID{condition},
		Targets: []int{thenID, elseID},
	})

	c.current = thenID
	if err := c.lowerStmt(s.Then); err != nil {
		return err
	}
	if !c.terminated() {
		c.jump(mergeID)
	}

	c.current = elseID
	switch {
	case s.ElseIf != nil:
		if err := c.lowerIf(s.ElseIf); err != nil {
			return err
		}
	case s.Else != nil:
		if err := c.lowerStmt(s.Else); err != nil {
			return err
		}
	}
	if !c.terminated() {
		c.jump(mergeID)
	}

	c.current = mergeID
	return nil
}

func (c *cfgBuilder) lowerWhile(s *ast.WhileStmt) error {
	if s.Condition == nil || s.Body == nil {
		return fmt.Errorf("while statement requires condition and body")
	}

	conditionID := c.newBlock()
	bodyID := c.newBlock()
	exitID := c.newBlock()

	c.jump(conditionID)

	c.current = conditionID
	condition, err := c.lowerer.lowerExpr(
		s.Condition,
		&c.blocks[c.current].Instructions,
	)
	if err != nil {
		return err
	}

	c.emit(Instruction{
		Op:      OpBranch,
		Args:    []ValueID{condition},
		Targets: []int{bodyID, exitID},
	})

	c.current = bodyID
	if err := c.lowerStmt(s.Body); err != nil {
		return err
	}
	if !c.terminated() {
		c.jump(conditionID)
	}

	c.current = exitID
	return nil
}
func (c *cfgBuilder) lowerStmt(stmt ast.Stmt) error {
	if stmt == nil {
		return fmt.Errorf("cannot lower nil statement")
	}

	if c.terminated() {
		return fmt.Errorf("statement follows terminator in block %d", c.current)
	}

	switch s := stmt.(type) {
	case *ast.IfStmt:
		return c.lowerIf(s)
	case *ast.WhileStmt:
		return c.lowerWhile(s)
	case *ast.BlockStmt:
		for _, child := range s.Statements {
			if err := c.lowerStmt(child); err != nil {
				return err
			}
		}
		return nil

	default:
		return c.lowerer.lowerStmt(
			stmt,
			&c.blocks[c.current].Instructions,
		)
	}
}
func (l *lowerer) emit(out *[]Instruction, inst Instruction) ValueID {
	inst.Result = l.nextID
	l.nextID++
	*out = append(*out, inst)
	return inst.Result
}

func (l *lowerer) lowerStmt(
	stmt ast.Stmt,
	out *[]Instruction,
) error {
	switch s := stmt.(type) {
	case *ast.VariableDeclStmt:
		if s.Name == nil {
			return fmt.Errorf("variable declaration has no name")
		}

		value, err := l.lowerExpr(s.Value, out)
		if err != nil {
			return err
		}

		l.emit(out, Instruction{
			Op:   OpStore,
			Type: s.TypeName,
			Name: s.Name.Name,
			Args: []ValueID{value},
		})
		return nil

	case *ast.AssignmentStmt:
		target, ok := s.Target.(*ast.Identifier)
		if !ok {
			return fmt.Errorf("unsupported assignment target %T", s.Target)
		}
		if s.Operator != "=" {
			return fmt.Errorf(
				"unsupported assignment operator %q", s.Operator,
			)
		}

		value, err := l.lowerExpr(s.Value, out)
		if err != nil {
			return err
		}

		l.emit(out, Instruction{
			Op:   OpStore,
			Name: target.Name,
			Args: []ValueID{value},
		})
		return nil
	case *ast.ReturnStmt:
		args := []ValueID{}

		if s.Value != nil {
			value, err := l.lowerExpr(s.Value, out)
			if err != nil {
				return err
			}
			args = append(args, value)
		}

		l.emit(out, Instruction{
			Op:   OpReturn,
			Args: args,
		})
		return nil
	case *ast.ExpressionStmt:
		_, err := l.lowerExpr(s.Expression, out)
		return err

	case *ast.BlockStmt:
		for _, child := range s.Statements {
			if err := l.lowerStmt(child, out); err != nil {
				return err
			}
		}
		return nil

	default:
		return fmt.Errorf("unsupported statement: %T", stmt)
	}
}

func (l *lowerer) lowerExpr(
	expr ast.Expr,
	out *[]Instruction,
) (ValueID, error) {
	switch e := expr.(type) {
	case *ast.NumberLiteral:
		return l.emit(out, Instruction{
			Op: OpConst, Type: "number", Value: e.Value,
		}), nil

	case *ast.StringLiteral:
		return l.emit(out, Instruction{
			Op: OpConst, Type: "string", Value: e.Value,
		}), nil

	case *ast.BoolLiteral:
		return l.emit(out, Instruction{
			Op: OpConst, Type: "bool", Value: e.Value,
		}), nil

	case *ast.NALiteral:
		return l.emit(out, Instruction{
			Op: OpConst, Type: "na", Value: nil,
		}), nil

	case *ast.ColorLiteral:
		return l.emit(out, Instruction{
			Op: OpConst, Type: "color", Value: e.Value,
		}), nil

	case *ast.Identifier:
		return l.emit(out, Instruction{
			Op: OpLoad, Name: e.Name,
		}), nil

	case *ast.UnaryExpr:
		value, err := l.lowerExpr(e.Operand, out)
		if err != nil {
			return 0, err
		}
		return l.emit(out, Instruction{
			Op: OpUnary, Operator: e.Operator,
			Args: []ValueID{value},
		}), nil

	case *ast.BinaryExpr:
		left, err := l.lowerExpr(e.Left, out)
		if err != nil {
			return 0, err
		}
		right, err := l.lowerExpr(e.Right, out)
		if err != nil {
			return 0, err
		}
		return l.emit(out, Instruction{
			Op: OpBinary, Operator: e.Operator,
			Args: []ValueID{left, right},
		}), nil

	case *ast.CallExpr:
		callee, err := l.lowerExpr(e.Callee, out)
		if err != nil {
			return 0, err
		}

		args := []ValueID{callee}
		for _, arg := range e.Arguments {
			value, err := l.lowerExpr(arg, out)
			if err != nil {
				return 0, err
			}
			args = append(args, value)
		}

		return l.emit(out, Instruction{
			Op: OpCall, Args: args,
		}), nil

	case *ast.MemberExpr:
		object, err := l.lowerExpr(e.Object, out)
		if err != nil {
			return 0, err
		}
		return l.emit(out, Instruction{
			Op: OpMember, Name: e.Member,
			Args: []ValueID{object},
		}), nil

	case *ast.IndexExpr:
		object, err := l.lowerExpr(e.Object, out)
		if err != nil {
			return 0, err
		}
		index, err := l.lowerExpr(e.Index, out)
		if err != nil {
			return 0, err
		}
		return l.emit(out, Instruction{
			Op: OpIndex, Args: []ValueID{object, index},
		}), nil

	default:
		return 0, fmt.Errorf("unsupported expression: %T", expr)
	}
}
