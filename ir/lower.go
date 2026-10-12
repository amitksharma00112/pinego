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

			var instructions []Instruction
			for _, bodyStmt := range s.Body.Statements {
				if err := l.lowerStmt(bodyStmt, &instructions); err != nil {
					return nil, fmt.Errorf(
						"function %q: %w", fn.Name, err,
					)
				}
			}

			fn.Blocks = []Block{{
				ID:           0,
				Instructions: instructions,
			}}
			result.Functions = append(result.Functions, fn)

		default:
			if err := l.lowerStmt(stmt, &result.Globals); err != nil {
				return nil, fmt.Errorf("global statement: %w", err)
			}
		}
	}

	if err := result.Validate(); err != nil {
		return nil, err
	}
	return result, nil
}

type lowerer struct {
	nextID ValueID
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
