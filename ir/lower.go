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
				case *ast.IfStmt, *ast.WhileStmt, *ast.ForStmt,
					*ast.BreakStmt, *ast.ContinueStmt:
					// CFG required
				default:
					if containsConditionalExpr(stmt) {
						// CFG required
					} else {
						if err := l.lowerStmt(stmt, &result.Globals); err != nil {
							return nil, fmt.Errorf("global statement: %w", err)
						}
						continue
					}
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
	lowerer        *lowerer
	blocks         []Block
	current        int
	breakTarget    int
	continueTarget int
	loopDepth      int
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
	case OpBranch, OpJump, OpReturn, OpTrap:
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
	oldBreak, oldContinue := c.breakTarget, c.continueTarget
	oldDepth := c.loopDepth

	c.breakTarget = exitID
	c.continueTarget = conditionID
	c.loopDepth++

	defer func() {
		c.breakTarget = oldBreak
		c.continueTarget = oldContinue
		c.loopDepth = oldDepth
	}()
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

func (c *cfgBuilder) lowerFor(s *ast.ForStmt) error {
	if s.IsInLoop || s.Iterable != nil {
		return fmt.Errorf("for-in iterable loops are not supported yet")
	}
	if s.Variable == nil || s.From == nil || s.To == nil || s.Body == nil {
		return fmt.Errorf("for statement requires variable, from, to and body")
	}

	step := s.Step
	if step == nil {
		step = &ast.NumberLiteral{Value: 1}
	}
	if n, ok := step.(*ast.NumberLiteral); ok && n.Value == 0 {
		return fmt.Errorf("for loop step cannot be zero")
	}

	from, err := c.lowerer.lowerExpr(
		s.From, &c.blocks[c.current].Instructions,
	)
	if err != nil {
		return err
	}

	c.emit(Instruction{
		Op:   OpStore,
		Name: s.Variable.Name,
		Args: []ValueID{from},
	})

	stepValue, err := c.lowerer.lowerExpr(
		step, &c.blocks[c.current].Instructions,
	)
	if err != nil {
		return err
	}

	_, literalStep := step.(*ast.NumberLiteral)
	if !literalStep {
		zero := c.lowerer.emit(
			&c.blocks[c.current].Instructions,
			Instruction{Op: OpConst, Type: "number", Value: 0},
		)
		stepIsZero := c.lowerer.emit(
			&c.blocks[c.current].Instructions,
			Instruction{
				Op:       OpBinary,
				Operator: "==",
				Args:     []ValueID{stepValue, zero},
			},
		)

		trapID := c.newBlock()
		continueID := c.newBlock()

		c.emit(Instruction{
			Op:      OpBranch,
			Args:    []ValueID{stepIsZero},
			Targets: []int{trapID, continueID},
		})

		c.current = trapID
		c.emit(Instruction{
			Op:   OpTrap,
			Name: "for loop step cannot be zero",
		})

		c.current = continueID

	}
	toValue, err := c.lowerer.lowerExpr(
		s.To, &c.blocks[c.current].Instructions,
	)
	if err != nil {
		return err
	}
	conditionID := c.newBlock()
	bodyID := c.newBlock()
	incrementID := c.newBlock()
	exitID := c.newBlock()

	var ascendingID, descendingID int
	_, literalStep = step.(*ast.NumberLiteral)
	if !literalStep {
		ascendingID = c.newBlock()
		descendingID = c.newBlock()
	}

	oldBreak, oldContinue := c.breakTarget, c.continueTarget
	oldDepth := c.loopDepth

	c.breakTarget = exitID
	c.continueTarget = incrementID
	c.loopDepth++

	defer func() {
		c.breakTarget = oldBreak
		c.continueTarget = oldContinue
		c.loopDepth = oldDepth
	}()

	c.jump(conditionID)
	c.current = conditionID

	if n, ok := step.(*ast.NumberLiteral); ok {
		operator := "<="
		if n.Value < 0 {
			operator = ">="
		}

		currentValue := c.lowerer.emit(
			&c.blocks[c.current].Instructions,
			Instruction{Op: OpLoad, Name: s.Variable.Name},
		)

		condition := c.lowerer.emit(
			&c.blocks[c.current].Instructions,
			Instruction{
				Op:       OpBinary,
				Operator: operator,
				Args:     []ValueID{currentValue, toValue},
			},
		)

		c.emit(Instruction{
			Op:      OpBranch,
			Args:    []ValueID{condition},
			Targets: []int{bodyID, exitID},
		})
	} else {
		zero := c.lowerer.emit(
			&c.blocks[c.current].Instructions,
			Instruction{Op: OpConst, Type: "number", Value: 0},
		)

		nonNegative := c.lowerer.emit(
			&c.blocks[c.current].Instructions,
			Instruction{
				Op:       OpBinary,
				Operator: ">=",
				Args:     []ValueID{stepValue, zero},
			},
		)

		c.emit(Instruction{
			Op:      OpBranch,
			Args:    []ValueID{nonNegative},
			Targets: []int{ascendingID, descendingID},
		})

		for _, direction := range []struct {
			id       int
			operator string
		}{
			{ascendingID, "<="},
			{descendingID, ">="},
		} {
			c.current = direction.id

			currentValue := c.lowerer.emit(
				&c.blocks[c.current].Instructions,
				Instruction{Op: OpLoad, Name: s.Variable.Name},
			)

			condition := c.lowerer.emit(
				&c.blocks[c.current].Instructions,
				Instruction{
					Op:       OpBinary,
					Operator: direction.operator,
					Args:     []ValueID{currentValue, toValue},
				},
			)

			c.emit(Instruction{
				Op:      OpBranch,
				Args:    []ValueID{condition},
				Targets: []int{bodyID, exitID},
			})
		}
	}

	c.current = bodyID
	if err := c.lowerStmt(s.Body); err != nil {
		return err
	}
	if !c.terminated() {
		c.jump(incrementID)
	}

	c.current = incrementID
	currentValue := c.lowerer.emit(
		&c.blocks[c.current].Instructions,
		Instruction{Op: OpLoad, Name: s.Variable.Name},
	)

	nextValue := c.lowerer.emit(
		&c.blocks[c.current].Instructions,
		Instruction{
			Op:       OpBinary,
			Operator: "+",
			Args:     []ValueID{currentValue, stepValue},
		},
	)

	c.emit(Instruction{
		Op:   OpStore,
		Name: s.Variable.Name,
		Args: []ValueID{nextValue},
	})
	c.jump(conditionID)

	c.current = exitID
	return nil
}
func (c *cfgBuilder) lowerExpr(expr ast.Expr) (ValueID, error) {
	if expr == nil {
		return 0, fmt.Errorf("cannot lower nil expression")
	}

	emit := func(inst Instruction) ValueID {
		return c.lowerer.emit(
			&c.blocks[c.current].Instructions,
			inst,
		)
	}

	switch e := expr.(type) {
	case *ast.ConditionalExpr:
		if e.Condition == nil || e.Then == nil || e.Else == nil {
			return 0, fmt.Errorf(
				"ternary expression requires condition, then, and else",
			)
		}

		condition, err := c.lowerExpr(e.Condition)
		if err != nil {
			return 0, err
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
		thenValue, err := c.lowerExpr(e.Then)
		if err != nil {
			return 0, err
		}
		thenBlockID := c.current
		c.jump(mergeID)

		c.current = elseID
		elseValue, err := c.lowerExpr(e.Else)
		if err != nil {
			return 0, err
		}
		elseBlockID := c.current
		c.jump(mergeID)

		c.current = mergeID
		result := emit(Instruction{
			Op:             OpPhi,
			Args:           []ValueID{thenValue, elseValue},
			IncomingBlocks: []int{thenBlockID, elseBlockID},
		})
		return result, nil

	case *ast.UnaryExpr:
		value, err := c.lowerExpr(e.Operand)
		if err != nil {
			return 0, err
		}
		return emit(Instruction{
			Op:       OpUnary,
			Operator: e.Operator,
			Args:     []ValueID{value},
		}), nil

	case *ast.BinaryExpr:
		left, err := c.lowerExpr(e.Left)
		if err != nil {
			return 0, err
		}
		right, err := c.lowerExpr(e.Right)
		if err != nil {
			return 0, err
		}
		return emit(Instruction{
			Op:       OpBinary,
			Operator: e.Operator,
			Args:     []ValueID{left, right},
		}), nil

	case *ast.CallExpr:
		callee, err := c.lowerExpr(e.Callee)
		if err != nil {
			return 0, err
		}

		args := []ValueID{callee}
		for _, arg := range e.Arguments {
			value, err := c.lowerExpr(arg)
			if err != nil {
				return 0, err
			}
			args = append(args, value)
		}

		return emit(Instruction{
			Op:   OpCall,
			Args: args,
		}), nil

	case *ast.MemberExpr:
		object, err := c.lowerExpr(e.Object)
		if err != nil {
			return 0, err
		}
		return emit(Instruction{
			Op:   OpMember,
			Name: e.Member,
			Args: []ValueID{object},
		}), nil

	case *ast.IndexExpr:
		object, err := c.lowerExpr(e.Object)
		if err != nil {
			return 0, err
		}
		index, err := c.lowerExpr(e.Index)
		if err != nil {
			return 0, err
		}
		return emit(Instruction{
			Op:   OpIndex,
			Args: []ValueID{object, index},
		}), nil

	default:
		return c.lowerer.lowerExpr(
			expr,
			&c.blocks[c.current].Instructions,
		)
	}
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
	case *ast.ForStmt:
		return c.lowerFor(s)
	case *ast.BreakStmt:
		if c.loopDepth == 0 {
			return fmt.Errorf("break outside loop")
		}
		c.jump(c.breakTarget)
		return nil
	case *ast.ContinueStmt:
		if c.loopDepth == 0 {
			return fmt.Errorf("continue outside loop")
		}
		c.jump(c.continueTarget)
		return nil
	case *ast.BlockStmt:
		for _, child := range s.Statements {
			if err := c.lowerStmt(child); err != nil {
				return err
			}
		}
		return nil

	default:
		return c.lowerer.lowerStmtWithCFG(stmt, c)
	}
}
func (l *lowerer) lowerStmtWithCFG(
	stmt ast.Stmt,
	c *cfgBuilder,
) error {
	switch s := stmt.(type) {
	case *ast.VariableDeclStmt:
		if s.Name == nil {
			return fmt.Errorf("variable declaration has no name")
		}
		value, err := c.lowerExpr(s.Value)
		if err != nil {
			return err
		}
		c.emit(Instruction{
			Op: OpStore, Type: s.TypeName,
			Name: s.Name.Name, Args: []ValueID{value},
		})
		return nil

	case *ast.AssignmentStmt:
		target, ok := s.Target.(*ast.Identifier)
		if !ok {
			return fmt.Errorf("unsupported assignment target %T", s.Target)
		}
		if s.Operator != "=" {
			return fmt.Errorf("unsupported assignment operator %q", s.Operator)
		}
		value, err := c.lowerExpr(s.Value)
		if err != nil {
			return err
		}
		c.emit(Instruction{
			Op: OpStore, Name: target.Name, Args: []ValueID{value},
		})
		return nil

	case *ast.ReturnStmt:
		var args []ValueID
		if s.Value != nil {
			value, err := c.lowerExpr(s.Value)
			if err != nil {
				return err
			}
			args = append(args, value)
		}
		c.emit(Instruction{Op: OpReturn, Args: args})
		return nil

	case *ast.ExpressionStmt:
		_, err := c.lowerExpr(s.Expression)
		return err

	default:
		return l.lowerStmt(stmt, &c.blocks[c.current].Instructions)
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
func containsConditionalExpr(stmt ast.Stmt) bool {
	var hasConditional func(ast.Expr) bool

	hasConditional = func(expr ast.Expr) bool {
		switch e := expr.(type) {
		case *ast.ConditionalExpr:
			return true
		case *ast.UnaryExpr:
			return hasConditional(e.Operand)
		case *ast.BinaryExpr:
			return hasConditional(e.Left) || hasConditional(e.Right)
		case *ast.CallExpr:
			if hasConditional(e.Callee) {
				return true
			}
			for _, arg := range e.Arguments {
				if hasConditional(arg) {
					return true
				}
			}
		case *ast.MemberExpr:
			return hasConditional(e.Object)
		case *ast.IndexExpr:
			return hasConditional(e.Object) || hasConditional(e.Index)
		case *ast.ArrayExpr:
			for _, item := range e.Elements {
				if hasConditional(item) {
					return true
				}
			}
		case *ast.TupleExpr:
			for _, item := range e.Elements {
				if hasConditional(item) {
					return true
				}
			}
		}
		return false
	}

	switch s := stmt.(type) {
	case *ast.VariableDeclStmt:
		return hasConditional(s.Value)
	case *ast.AssignmentStmt:
		return hasConditional(s.Target) || hasConditional(s.Value)
	case *ast.ExpressionStmt:
		return hasConditional(s.Expression)
	case *ast.ReturnStmt:
		return hasConditional(s.Value)
	}
	return false
}
