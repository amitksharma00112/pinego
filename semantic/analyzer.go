package semantic

import (
	"fmt"
	"strings"

	"github.com/amitksharma00112/pinego/ast"
)

// Analyzer performs semantic checks on the PineGo AST.
type Analyzer struct {
	symbols             map[string]struct{}
	symbolTypes         map[string]string
	builtins            map[string]struct{}
	namespaces          map[string]struct{}
	builtinFunctions    map[string]struct{}
	builtinArgCounts    map[string]int
	knownTypes          map[string]struct{}
	functions           map[string]int
	functionParamTypes  map[string][]string
	functionReturnTypes map[string]string
	functionDepth       int
	loopDepth           int
	builtinParamTypes   map[string][]string
}

// New creates a semantic analyzer.
func New() *Analyzer {
	return &Analyzer{
		symbols: make(map[string]struct{}),
		builtins: map[string]struct{}{
			"open":   {},
			"high":   {},
			"low":    {},
			"close":  {},
			"volume": {},
		},
		namespaces: map[string]struct{}{
			"ta":    {},
			"math":  {},
			"str":   {},
			"array": {},
			"color": {},
			"input": {},
		},
		builtinFunctions: map[string]struct{}{
			"ta.sma": {},
		},
		builtinArgCounts: map[string]int{
			"ta.sma": 2,
		},
		knownTypes: map[string]struct{}{
			"int":    {},
			"float":  {},
			"bool":   {},
			"string": {},
			"color":  {},
		},
		symbolTypes:         make(map[string]string),
		functions:           make(map[string]int),
		functionParamTypes:  make(map[string][]string),
		functionReturnTypes: make(map[string]string),
		builtinParamTypes: map[string][]string{
			"ta.sma": {"float", "int"},
		},
	}
}

// Analyze checks the program and returns an error if it is semantically invalid.
func (a *Analyzer) Analyze(program *ast.Program) error {
	if program == nil {
		return fmt.Errorf("cannot analyze nil program")
	}

	for _, stmt := range program.Statements {
		if err := a.checkStatement(stmt); err != nil {
			return err
		}
	}

	return nil
}

func (a *Analyzer) checkStatement(stmt ast.Stmt) error {
	switch s := stmt.(type) {
	case *ast.AssignmentStmt:
		if err := a.checkExpression(s.Value); err != nil {
			return err
		}

		target, ok := s.Target.(*ast.Identifier)
		if !ok {
			return fmt.Errorf("invalid assignment target")
		}

		// "=" creates/initializes a variable.
		if s.Operator == "=" {
			if _, exists := a.symbols[target.Name]; exists {
				return fmt.Errorf(
					"variable %q already declared",
					target.Name,
				)
			}

			a.symbols[target.Name] = struct{}{}
			a.symbolTypes[target.Name] = a.inferExprType(s.Value)
			return nil
		}

		// ":=", "+=", "-=", "*=", "/=", "%=" are reassignments.
		// The variable must already exist.
		if _, exists := a.symbols[target.Name]; !exists {
			return fmt.Errorf(
				"undefined variable %q",
				target.Name,
			)
		}

		expectedType := a.symbolTypes[target.Name]
		actualType := a.inferExprType(s.Value)

		if s.Operator == "+=" ||
			s.Operator == "-=" ||
			s.Operator == "*=" ||
			s.Operator == "/=" ||
			s.Operator == "%=" {

			if !isNumericType(expectedType) || !isNumericType(actualType) {
				return fmt.Errorf(
					"compound assignment %q requires numeric types, got %s and %s",
					s.Operator,
					expectedType,
					actualType,
				)
			}
		}
		if expectedType != "" &&
			!a.isTypeCompatible(expectedType, actualType) {
			return fmt.Errorf(
				"type mismatch: cannot assign %s to %s",
				actualType,
				expectedType,
			)
		}

		return nil
	case *ast.WhileStmt:
		if err := a.checkExpression(s.Condition); err != nil {
			return err
		}

		conditionType := a.inferExprType(s.Condition)

		if conditionType != "bool" && conditionType != "unknown" {
			return fmt.Errorf(
				"while condition must be boolean, got %s",
				conditionType,
			)
		}

		// While body gets its own scope.
		outerSymbols := a.symbols
		outerTypes := a.symbolTypes

		a.symbols = cloneSymbols(outerSymbols)
		a.symbolTypes = cloneSymbolTypes(outerTypes)

		a.loopDepth++

		for _, stmt := range s.Body.Statements {
			if err := a.checkStatement(stmt); err != nil {
				a.loopDepth--
				a.symbols = outerSymbols
				a.symbolTypes = outerTypes
				return err
			}
		}

		a.loopDepth--

		// Variables created inside while must not escape.
		a.symbols = outerSymbols
		a.symbolTypes = outerTypes

		return nil
	case *ast.ForStmt:
		if err := a.checkExpression(s.From); err != nil {
			return err
		}

		if err := a.checkExpression(s.To); err != nil {
			return err
		}

		if s.Step != nil {
			if err := a.checkExpression(s.Step); err != nil {
				return err
			}
		}
		fromType := a.inferExprType(s.From)
		toType := a.inferExprType(s.To)

		if !isNumericType(fromType) {
			return fmt.Errorf(
				"for loop start must be numeric, got %s",
				fromType,
			)
		}

		if !isNumericType(toType) {
			return fmt.Errorf(
				"for loop end must be numeric, got %s",
				toType,
			)
		}

		if s.Step != nil {
			stepType := a.inferExprType(s.Step)

			if !isNumericType(stepType) {
				return fmt.Errorf(
					"for loop step must be numeric, got %s",
					stepType,
				)
			}
		}
		// The loop variable exists only inside the loop.
		outerSymbols := a.symbols
		outerTypes := a.symbolTypes

		a.symbols = cloneSymbols(outerSymbols)
		a.symbolTypes = cloneSymbolTypes(outerTypes)

		if s.Variable != nil {
			a.symbols[s.Variable.Name] = struct{}{}
			a.symbolTypes[s.Variable.Name] = "int"
		}

		a.loopDepth++

		for _, stmt := range s.Body.Statements {
			if err := a.checkStatement(stmt); err != nil {
				a.loopDepth--
				a.symbols = outerSymbols
				a.symbolTypes = outerTypes
				return err
			}
		}

		a.loopDepth--

		a.symbols = outerSymbols
		a.symbolTypes = outerTypes

		return nil
	case *ast.BreakStmt:
		if a.loopDepth == 0 {
			return fmt.Errorf("break statement outside loop")
		}

		return nil
	case *ast.ContinueStmt:
		if a.loopDepth == 0 {
			return fmt.Errorf("continue statement outside loop")
		}

		return nil
	case *ast.IfStmt:
		// Check the if condition in the current scope.
		if err := a.checkExpression(s.Condition); err != nil {
			return err
		}
		conditionType := a.inferExprType(s.Condition)

		if conditionType != "bool" && conditionType != "unknown" {
			return fmt.Errorf(
				"if condition must be boolean, got %s",
				conditionType,
			)
		}

		// Then block gets its own temporary scope.
		outerSymbols := a.symbols

		a.symbols = cloneSymbols(outerSymbols)

		for _, stmt := range s.Then.Statements {
			if err := a.checkStatement(stmt); err != nil {
				a.symbols = outerSymbols
				return err
			}
		}

		a.symbols = outerSymbols

		// Else-if gets its own scope.
		if s.ElseIf != nil {
			if err := a.checkStatement(s.ElseIf); err != nil {
				return err
			}
		}

		// Else gets its own scope.
		if s.Else != nil {
			a.symbols = cloneSymbols(outerSymbols)

			for _, stmt := range s.Else.Statements {
				if err := a.checkStatement(stmt); err != nil {
					a.symbols = outerSymbols
					return err
				}
			}

			a.symbols = outerSymbols
		}

		return nil
	case *ast.VariableDeclStmt:
		// Validate declared type first.
		if s.TypeName != "" {
			if _, exists := a.knownTypes[s.TypeName]; !exists {
				return fmt.Errorf(
					"unknown type %q",
					s.TypeName,
				)
			}

			// Check initializer type against declared type.
			valueType := a.inferExprType(s.Value)

			if !a.isTypeCompatible(s.TypeName, valueType) {
				return fmt.Errorf(
					"type mismatch: cannot assign %s to %s",
					valueType,
					s.TypeName,
				)
			}
		}

		if err := a.checkExpression(s.Value); err != nil {
			return err
		}

		if s.Name != nil {
			if _, exists := a.symbols[s.Name.Name]; exists {
				return fmt.Errorf(
					"variable %q already declared",
					s.Name.Name,
				)
			}

			a.symbols[s.Name.Name] = struct{}{}

			if s.TypeName != "" {
				a.symbolTypes[s.Name.Name] = s.TypeName
			} else {
				a.symbolTypes[s.Name.Name] = a.inferExprType(s.Value)
			}
		}

		return nil
	case *ast.FunctionDeclStmt:
		if a.functionDepth > 0 {
			return fmt.Errorf("nested function declaration is not allowed")
		}
		if s.Name == nil {
			return fmt.Errorf("function must have a name")
		}

		if _, exists := a.functions[s.Name.Name]; exists {
			return fmt.Errorf(
				"function %q already declared",
				s.Name.Name,
			)
		}

		// Register the function before checking its body.
		// This also allows recursive calls later.
		a.functions[s.Name.Name] = len(s.Parameters)

		paramTypes := make([]string, len(s.Parameters))

		for i, param := range s.Parameters {
			if param.TypeName == "" {
				continue
			}

			if _, exists := a.knownTypes[param.TypeName]; !exists {
				return fmt.Errorf(
					"unknown parameter type %q",
					param.TypeName,
				)
			}

			paramTypes[i] = param.TypeName
		}

		a.functionParamTypes[s.Name.Name] = paramTypes

		// A function gets its own local scope.
		outerSymbols := a.symbols
		outerTypes := a.symbolTypes

		a.symbols = cloneSymbols(outerSymbols)
		a.symbolTypes = cloneSymbolTypes(outerTypes)

		for _, param := range s.Parameters {
			if param.Name == nil {
				continue
			}

			name := param.Name.Name
			a.symbols[name] = struct{}{}

			if param.TypeName != "" {
				a.symbolTypes[name] = param.TypeName
			}
		}

		a.functionDepth++

		for _, stmt := range s.Body.Statements {
			if err := a.checkStatement(stmt); err != nil {
				a.functionDepth--
				a.symbols = outerSymbols
				a.symbolTypes = outerTypes
				return err
			}
		}

		a.functionDepth--
		if err := a.validateReturnTypes(s.Body); err != nil {
			return err
		}

		returnType := a.inferBlockReturnType(s.Body)

		a.functionReturnTypes[s.Name.Name] = returnType

		a.symbols = outerSymbols
		a.symbolTypes = outerTypes

		return nil

	case *ast.ExpressionStmt:
		return a.checkExpression(s.Expression)

	case *ast.ReturnStmt:
		if a.functionDepth == 0 {
			return fmt.Errorf("return statement outside function")
		}

		if s.Value == nil {
			return nil
		}

		return a.checkExpression(s.Value)

	default:
		return nil
	}
}

func (a *Analyzer) checkExpression(expr ast.Expr) error {
	switch e := expr.(type) {
	case *ast.Identifier:
		if _, ok := a.symbols[e.Name]; ok {
			return nil
		}

		if _, ok := a.builtins[e.Name]; ok {
			return nil
		}

		return fmt.Errorf(
			"undefined variable %q",
			e.Name,
		)

	case *ast.BinaryExpr:
		if err := a.checkExpression(e.Left); err != nil {
			return err
		}

		if err := a.checkExpression(e.Right); err != nil {
			return err
		}

		leftType := a.inferExprType(e.Left)
		rightType := a.inferExprType(e.Right)

		switch e.Operator {
		case "+", "-", "*", "/", "%":
			if !isNumericType(leftType) || !isNumericType(rightType) {
				return fmt.Errorf(
					"operator %q requires numeric operands, got %s and %s",
					e.Operator,
					leftType,
					rightType,
				)
			}

		case ">", ">=", "<", "<=":
			if !isNumericType(leftType) || !isNumericType(rightType) {
				return fmt.Errorf(
					"operator %q requires numeric operands, got %s and %s",
					e.Operator,
					leftType,
					rightType,
				)
			}

		case "==", "!=":
			if leftType != "unknown" &&
				rightType != "unknown" &&
				leftType != rightType &&
				!(isNumericType(leftType) && isNumericType(rightType)) {
				return fmt.Errorf(
					"operator %q cannot compare %s and %s",
					e.Operator,
					leftType,
					rightType,
				)
			}
		case "and", "or":
			if leftType != "bool" && leftType != "unknown" {
				return fmt.Errorf(
					"operator %q requires boolean operands, got %s and %s",
					e.Operator,
					leftType,
					rightType,
				)
			}

			if rightType != "bool" && rightType != "unknown" {
				return fmt.Errorf(
					"operator %q requires boolean operands, got %s and %s",
					e.Operator,
					leftType,
					rightType,
				)
			}
		}

		return nil

	case *ast.UnaryExpr:
		return a.checkExpression(e.Operand)

	case *ast.ConditionalExpr:
		if err := a.checkExpression(e.Condition); err != nil {
			return err
		}

		if err := a.checkExpression(e.Then); err != nil {
			return err
		}

		if err := a.checkExpression(e.Else); err != nil {
			return err
		}

		thenType := a.inferExprType(e.Then)
		elseType := a.inferExprType(e.Else)

		if thenType != "unknown" &&
			elseType != "unknown" &&
			thenType != elseType &&
			!(isNumericType(thenType) && isNumericType(elseType)) {
			return fmt.Errorf(
				"ternary branches have incompatible types: %s and %s",
				thenType,
				elseType,
			)
		}

		return nil

	case *ast.MemberExpr:
		if object, ok := e.Object.(*ast.Identifier); ok {
			if _, exists := a.namespaces[object.Name]; exists {
				name := object.Name + "." + e.Member

				if _, exists := a.builtinFunctions[name]; !exists {
					return fmt.Errorf(
						"unknown builtin function %q",
						name,
					)
				}

				return nil
			}
		}

		return a.checkExpression(e.Object)

	case *ast.IndexExpr:
		if err := a.checkExpression(e.Object); err != nil {
			return err
		}

		return a.checkExpression(e.Index)

	case *ast.CallExpr:
		if callee, ok := e.Callee.(*ast.Identifier); ok {
			expected, exists := a.functions[callee.Name]
			if !exists {
				return fmt.Errorf(
					"undefined function %q",
					callee.Name,
				)
			}

			if len(e.Arguments) != expected {
				return fmt.Errorf(
					"function %q expects %d arguments, got %d",
					callee.Name,
					expected,
					len(e.Arguments),
				)
			}
			paramTypes := a.functionParamTypes[callee.Name]

			for i, arg := range e.Arguments {
				if i >= len(paramTypes) || paramTypes[i] == "" {
					continue
				}

				actualType := a.inferExprType(arg)

				if !a.isTypeCompatible(paramTypes[i], actualType) {
					return fmt.Errorf(
						"function %q argument %d expects %s, got %s",
						callee.Name,
						i+1,
						paramTypes[i],
						actualType,
					)
				}
			}
		} else {
			if err := a.checkExpression(e.Callee); err != nil {
				return err
			}
		}

		// Check builtin function argument count.
		// Check builtin function argument count.
		if member, ok := e.Callee.(*ast.MemberExpr); ok {
			if object, ok := member.Object.(*ast.Identifier); ok {
				name := object.Name + "." + member.Member

				if expected, exists := a.builtinArgCounts[name]; exists {
					if len(e.Arguments) != expected {
						return fmt.Errorf(
							"builtin function %q expects %d arguments, got %d",
							name,
							expected,
							len(e.Arguments),
						)
					}
				}

				if expectedTypes, exists := a.builtinParamTypes[name]; exists {
					for i, expectedType := range expectedTypes {
						actualType := a.inferExprType(e.Arguments[i])

						if !a.isTypeCompatible(expectedType, actualType) {
							return fmt.Errorf(
								"builtin function %q argument %d expects %s, got %s",
								name,
								i+1,
								expectedType,
								actualType,
							)
						}
					}
				}
			}
		}

		for _, arg := range e.Arguments {
			if err := a.checkExpression(arg); err != nil {
				return err
			}
		}

		return nil

	case *ast.TupleExpr:
		for _, element := range e.Elements {
			if err := a.checkExpression(element); err != nil {
				return err
			}
		}

		return nil

	case *ast.NumberLiteral,
		*ast.StringLiteral,
		*ast.BoolLiteral,
		*ast.NALiteral,
		*ast.ColorLiteral:
		return nil

	default:
		return nil
	}
}

func cloneSymbols(source map[string]struct{}) map[string]struct{} {
	clone := make(map[string]struct{}, len(source))

	for name := range source {
		clone[name] = struct{}{}
	}

	return clone
}

func (a *Analyzer) inferExprType(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.Identifier:
		if typeName, exists := a.symbolTypes[e.Name]; exists {
			return typeName
		}

		if _, exists := a.builtins[e.Name]; exists {
			return "float"
		}

		return "unknown"

	case *ast.NumberLiteral:
		if strings.ContainsAny(e.Raw, ".eE") {
			return "float"
		}

		return "int"

	case *ast.StringLiteral:
		return "string"

	case *ast.BoolLiteral:
		return "bool"

	case *ast.ColorLiteral:
		return "color"

	case *ast.NALiteral:
		return "na"
	case *ast.IndexExpr:
		return a.inferExprType(e.Object)
	case *ast.UnaryExpr:
		operandType := a.inferExprType(e.Operand)

		switch e.Operator {
		case "not":
			return "bool"

		case "+", "-":
			return operandType
		}

		return "unknown"
	case *ast.ConditionalExpr:
		thenType := a.inferExprType(e.Then)
		elseType := a.inferExprType(e.Else)

		if thenType == "float" || elseType == "float" {
			return "float"
		}

		if thenType == "int" && elseType == "int" {
			return "int"
		}

		if thenType == elseType {
			return thenType
		}

		return "unknown"
	case *ast.CallExpr:
		// User-defined function call.
		if callee, ok := e.Callee.(*ast.Identifier); ok {
			if returnType, exists := a.functionReturnTypes[callee.Name]; exists {
				return returnType
			}

			return "unknown"
		}

		// Built-in namespace function.
		if member, ok := e.Callee.(*ast.MemberExpr); ok {
			if object, ok := member.Object.(*ast.Identifier); ok {
				name := object.Name + "." + member.Member

				switch name {
				case "ta.sma":
					return "float"
				}
			}
		}

		return "unknown"
	case *ast.BinaryExpr:
		leftType := a.inferExprType(e.Left)
		rightType := a.inferExprType(e.Right)

		switch e.Operator {
		case "+", "-", "*", "/", "%":
			if leftType == "float" || rightType == "float" {
				return "float"
			}

			if leftType == "int" && rightType == "int" {
				return "int"
			}

		case ">", ">=", "<", "<=", "==", "!=":
			return "bool"

		case "and", "or":
			return "bool"
		}

		return "unknown"
	default:
		return "unknown"
	}
}

func (a *Analyzer) isTypeCompatible(expected, actual string) bool {
	if actual == "na" || actual == "unknown" {
		return true
	}

	if expected == actual {
		return true
	}

	// Pine allows int values where float is expected.
	if expected == "float" && actual == "int" {
		return true
	}

	return false
}

func isNumericType(typeName string) bool {
	return typeName == "int" ||
		typeName == "float" ||
		typeName == "unknown"
}

func cloneSymbolTypes(source map[string]string) map[string]string {
	clone := make(map[string]string, len(source))

	for name, typeName := range source {
		clone[name] = typeName
	}

	return clone
}

func (a *Analyzer) inferBlockReturnType(block *ast.BlockStmt) string {
	if block == nil {
		return "unknown"
	}

	for i := len(block.Statements) - 1; i >= 0; i-- {
		if returnType := a.inferStatementReturnType(block.Statements[i]); returnType != "unknown" {
			return returnType
		}
	}

	return "unknown"
}

func (a *Analyzer) inferStatementReturnType(stmt ast.Stmt) string {
	switch s := stmt.(type) {
	case *ast.ReturnStmt:
		if s.Value == nil {
			return "unknown"
		}

		return a.inferExprType(s.Value)

	case *ast.ExpressionStmt:
		return a.inferExprType(s.Expression)

	case *ast.IfStmt:
		thenType := a.inferBlockReturnType(s.Then)
		elseType := "unknown"

		if s.ElseIf != nil {
			elseType = a.inferStatementReturnType(s.ElseIf)
		} else if s.Else != nil {
			elseType = a.inferBlockReturnType(s.Else)
		}

		if thenType == "unknown" {
			return elseType
		}

		if elseType == "unknown" {
			return thenType
		}

		if thenType == elseType {
			return thenType
		}

		if isNumericType(thenType) && isNumericType(elseType) {
			return "float"
		}
	}

	return "unknown"
}

func (a *Analyzer) validateReturnTypes(block *ast.BlockStmt) error {
	if block == nil {
		return nil
	}

	var expected string

	for _, stmt := range block.Statements {
		switch s := stmt.(type) {
		case *ast.ReturnStmt:
			if s.Value == nil {
				continue
			}

			current := a.inferExprType(s.Value)

			if expected == "" || expected == "unknown" {
				expected = current
				continue
			}

			if !returnTypesCompatible(expected, current) {
				return fmt.Errorf(
					"function return type mismatch: %s and %s",
					expected,
					current,
				)
			}

		case *ast.IfStmt:
			if err := a.validateReturnTypes(s.Then); err != nil {
				return err
			}

			if s.ElseIf != nil {
				if err := a.validateReturnTypes(
					&ast.BlockStmt{
						Statements: []ast.Stmt{s.ElseIf},
					},
				); err != nil {
					return err
				}
			}

			if err := a.validateReturnTypes(s.Else); err != nil {
				return err
			}

			// Compare return types from the two branches.
			thenType := a.inferBlockReturnType(s.Then)

			elseType := "unknown"

			if s.ElseIf != nil {
				elseType = a.inferStatementReturnType(s.ElseIf)
			} else if s.Else != nil {
				elseType = a.inferBlockReturnType(s.Else)
			}

			if thenType != "unknown" &&
				elseType != "unknown" &&
				!returnTypesCompatible(thenType, elseType) {
				return fmt.Errorf(
					"function return type mismatch: %s and %s",
					thenType,
					elseType,
				)
			}
		}
	}

	return nil
}

func returnTypesCompatible(left, right string) bool {
	if left == right {
		return true
	}

	if left == "unknown" || right == "unknown" {
		return true
	}

	// int and float are compatible.
	if (left == "int" || left == "float") &&
		(right == "int" || right == "float") {
		return true
	}

	return false
}
